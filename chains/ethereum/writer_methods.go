// Copyright 2021 Compass Systems
// SPDX-License-Identifier: LGPL-3.0-only

package ethereum

import (
	"context"
	"errors"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/mapprotocol/compass/mapprotocol"
	"github.com/mapprotocol/compass/msg"
	utils "github.com/mapprotocol/compass/shared/ethereum"
)

// Number of blocks to wait for an finalization event
const ExecuteBlockWatchLimit = 100

// Time between retrying a failed tx
const TxRetryInterval = time.Second * 2

// Maximum number of tx retries before exiting
const TxRetryLimit = 10

var ErrNonceTooLow = errors.New("nonce too low")
var ErrTxUnderpriced = errors.New("replacement transaction underpriced")
var ErrFatalTx = errors.New("submission of transaction failed")
var ErrFatalQuery = errors.New("query of chain state failed")

// exeSwapMsg executes swap msg, and send tx to the destination blockchain
func (w *writer) exeSwapMsg(m msg.Message) bool {
	return w.callContractWithMsg(w.cfg.bridgeContract, utils.SwapIn, m)
}

// func (w *writer) exeSwapWithProofMsg(m msg.Message) bool {
// 	return w.callContractWithMsg(w.cfg.bridgeContract, utils.SwapInWithProof, m)
// }

// callContractWithMsg call contract using address and function signature with message info
func (w *writer) callContractWithMsg(addr common.Address, funcSignature string, m msg.Message) bool {
	for i := 0; i < TxRetryLimit; i++ {
		select {
		case <-w.stop:
			return false
		default:
			err := w.conn.LockAndUpdateOpts()
			if err != nil {
				w.log.Error("Failed to update nonce", "err", err)
				return false
			}
			// These store the gas limit and price before a transaction is sent for logging in case of a failure
			// This is necessary as tx will be nil in the case of an error when sending VoteProposal()
			gasLimit := w.conn.Opts().GasLimit
			gasPrice := w.conn.Opts().GasPrice

			// sendtx using general method
			data := utils.ComposeMsgPayloadWithSignature(funcSignature, m.Payload)
			tx, err := w.sendTx(&addr, nil, data)

			w.conn.UnlockOpts()

			if err == nil {
				// message successfully handled
				w.log.Info("Submitted cross tx execution", "tx", tx.Hash(), "src", m.Source, "dst", m.Destination, "nonce", m.DepositNonce, "gasPrice", tx.GasPrice().String())
				m.DoneCh <- struct{}{}
				return true
			} else if err.Error() == ErrNonceTooLow.Error() || err.Error() == ErrTxUnderpriced.Error() {
				w.log.Error("Nonce too low, will retry")
				time.Sleep(TxRetryInterval)
			} else {
				w.log.Warn("Execution failed, tx may already be complete", "gasLimit", gasLimit, "gasPrice", gasPrice, "err", err)
				time.Sleep(TxRetryInterval)
			}
		}
	}
	w.log.Error("Submission of Execute transaction failed", "source", m.Source, "dest", m.Destination, "depositNonce", m.DepositNonce)
	w.sysErr <- ErrFatalTx
	return false
}

// sendTx send tx to an address with value and input data
func (w *writer) sendTx(toAddress *common.Address, value *big.Int, input []byte) (*types.Transaction, error) {
	gasPrice := w.conn.Opts().GasPrice
	nonce := w.conn.Opts().Nonce
	from := w.conn.Keypair().CommonAddress()

	msg := ethereum.CallMsg{
		From:     from,
		To:       toAddress,
		GasPrice: gasPrice,
		Value:    value,
		Data:     input,
	}
	w.log.Debug("eth CallMsg", "msg", msg)
	gasLimit, err := w.conn.Client().EstimateGas(context.Background(), msg)
	if err != nil {
		w.log.Error("EstimateGas failed", "error:", err.Error())
		return nil, err
	}

	// td interface
	var td types.TxData
	// EIP-1559
	if gasPrice != nil {
		// legacy branch
		td = &types.LegacyTx{
			Nonce:    nonce.Uint64(),
			Value:    value,
			To:       toAddress,
			Gas:      gasLimit,
			GasPrice: gasPrice,
			Data:     input,
		}
	} else {
		// london branch
		td = &types.DynamicFeeTx{
			Nonce:     nonce.Uint64(),
			Value:     value,
			To:        toAddress,
			Gas:       gasLimit,
			GasTipCap: w.conn.Opts().GasTipCap,
			GasFeeCap: w.conn.Opts().GasFeeCap,
			Data:      input,
		}
	}

	tx := types.NewTx(td)
	chainID := big.NewInt(int64(w.cfg.id))
	privateKey := w.conn.Keypair().PrivateKey()

	signedTx, err := types.SignTx(tx, types.NewLondonSigner(chainID), privateKey)
	if err != nil {
		w.log.Error("SignTx failed", "error:", err.Error())
		return nil, err
	}

	err = w.conn.Client().SendTransaction(context.Background(), signedTx)
	if err != nil {
		w.log.Error("SendTransaction failed", "error:", err.Error())
		return nil, err
	}
	return signedTx, nil
}

// exeSyncMsg executes sync msg, and send tx to the destination blockchain
func (w *writer) exeSyncMsg(m msg.Message) bool {
	//return w.callContractWithMsg(,  m)
	for i := 0; i < TxRetryLimit; i++ {
		select {
		case <-w.stop:
			return false
		default:
			err := w.conn.LockAndUpdateOpts()
			if err != nil {
				w.log.Error("Failed to update nonce", "err", err)
				return false
			}
			// These store the gas limit and price before a transaction is sent for logging in case of a failure
			// This is necessary as tx will be nil in the case of an error when sending VoteProposal()
			gasLimit := w.conn.Opts().GasLimit
			gasPrice := w.conn.Opts().GasPrice

			src := big.NewInt(int64(m.Source))
			dest := big.NewInt(int64(m.Destination))
			marshal, _ := m.Payload[0].([]byte)

			// save header data
			data, err := mapprotocol.SaveHeaderTxData(src, dest, marshal)
			if err != nil {
				w.log.Error("Failed to pack abi data", "err", err)
				w.conn.UnlockOpts()
				return false
			}
			tx, err := w.sendTx(&mapprotocol.RelayerAddress, nil, data)

			w.conn.UnlockOpts()

			if err == nil {
				// message successfully handled
				w.log.Info("Sync Header tx execution", "tx", tx.Hash(), "src", m.Source, "dst", m.Destination)
				// waited till successful mined
				err = w.blockForPending(tx.Hash())
				if err != nil {
					w.log.Warn("blockForPending error", "err", err)
				}
				m.DoneCh <- struct{}{}
				return true
			} else if err.Error() == ErrNonceTooLow.Error() || err.Error() == ErrTxUnderpriced.Error() {
				w.log.Error("Nonce too low, will retry")
				time.Sleep(TxRetryInterval)
			} else {
				w.log.Warn("Execution failed, header may already been synced", "gasLimit", gasLimit, "gasPrice", gasPrice, "err", err)
				m.DoneCh <- struct{}{}
				return true
			}
		}
	}
	w.log.Error("Submission of Sync Header transaction failed", "source", m.Source, "dest", m.Destination, "depositNonce", m.DepositNonce)
	w.sysErr <- ErrFatalTx
	return false
}

// this function will block for the txhash given
func (w *writer) blockForPending(txHash common.Hash) error {
	for {
		_, isPending, err := w.conn.Client().TransactionByHash(context.Background(), txHash)
		if err != nil {
			return err
		}

		if isPending {
			w.log.Info("tx is pending, please wait...")
			time.Sleep(time.Millisecond * 900)
		} else {
			break
		}
	}
	return nil
}
