package writer

import (
	"context"
	"errors"
	"math/big"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/mapprotocol/compass/internal/constant"
	"github.com/mapprotocol/compass/mapprotocol"
	"github.com/mapprotocol/compass/msg"
)

// execToMapMsg executes sync msg, and send tx to the destination blockchain
// the current function is only responsible for sending messages and is not responsible for processing data formats，
func (w *Writer) execToMapMsg(m msg.Message) bool {
	//return w.callContractWithMsg(,  m)
	for {
		select {
		case <-w.stop:
			return false
		default:
			err := w.conn.LockAndUpdateOpts()
			if err != nil {
				w.log.Error("BlockToMap Failed to update nonce", "err", err)
				time.Sleep(constant.TxRetryInterval)
				continue
			}
			// These store the gas limit and price before a transaction is sent for logging in case of a failure
			// This is necessary as tx will be nil in the case of an error when sending VoteProposal()
			gasLimit := w.conn.Opts().GasLimit
			gasPrice := w.conn.Opts().GasPrice

			id, _ := m.Payload[0].(*big.Int)
			marshal, _ := m.Payload[1].([]byte)
			// save header data
			data, err := mapprotocol.PackInput(mapprotocol.LightManger, mapprotocol.MethodUpdateBlockHeader, id, marshal)
			//data, err := mapprotocol.PackInput(mapprotocol.Bsc, mapprotocol.MethodUpdateBlockHeader, marshal)
			if err != nil {
				w.log.Error("block2Map Failed to pack abi data", "err", err)
				w.conn.UnlockOpts()
				return false
			}
			tx, err := w.sendTx(&w.cfg.LightNode, nil, data)
			w.conn.UnlockOpts()
			if err == nil {
				// message successfully handled
				w.log.Info("Sync Header to map tx execution", "tx", tx.Hash(), "src", m.Source, "dst", m.Destination)
				time.Sleep(time.Second * 2)
				// waited till successful mined
				err = w.blockForPending(tx.Hash())
				if err != nil {
					w.log.Warn("Sync Header to map blockForPending error", "err", err)
				}
				m.DoneCh <- struct{}{}
				return true
			} else if strings.Index(err.Error(), constant.EthOrderExist) != -1 {
				w.log.Info(constant.EthOrderExistPrint, "err", err)
				m.DoneCh <- struct{}{}
				return true
			} else if err.Error() == constant.ErrNonceTooLow.Error() || err.Error() == constant.ErrTxUnderpriced.Error() {
				w.log.Error("Sync Header to map Nonce too low, will retry")
			} else if strings.Index(err.Error(), "EOF") != -1 { // When requesting the lightNode to return EOF, it indicates that there may be a problem with the network and it needs to be retried
				w.log.Error("Sync Header to map encounter EOF, will retry")
			} else if strings.Index(err.Error(), "max fee per gas less than block base fee") != -1 {
				w.log.Error("gas maybe less than base fee, will retry")
			} else if strings.Index(err.Error(), constant.NotEnoughGas) != -1 {
				w.log.Error(constant.NotEnoughGasPrint)
			} else {
				w.log.Warn("Sync Header to map Execution failed, header may already been synced", "gasLimit", gasLimit, "gasPrice", gasPrice, "err", err)
			}
			time.Sleep(constant.TxRetryInterval)
		}
	}
	//w.log.Error("Sync Header to map Submission of Sync Header transaction failed", "source", m.Source, "dest", m.Destination, "depositNonce", m.DepositNonce)
	//w.sysErr <- constant.ErrFatalTx
	//return false
}

// sendTx send tx to an address with value and input data
func (w *Writer) sendTx(toAddress *common.Address, value *big.Int, input []byte) (*types.Transaction, error) {
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
	w.log.Debug("eth CallMsg", "toAddress", toAddress)
	gasLimit, err := w.conn.Client().EstimateGas(context.Background(), msg)
	if err != nil {
		w.log.Error("EstimateGas failed sendTx", "error:", err.Error())
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
	chainID := big.NewInt(int64(w.cfg.Id))
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

// this function will block for the txhash given
func (w *Writer) blockForPending(txHash common.Hash) error {
	for {
		_, isPending, err := w.conn.Client().TransactionByHash(context.Background(), txHash)
		if err != nil {
			w.log.Info("blockForPending tx is temporary not found", "err is not found?", errors.Is(err, errors.New("not found")), "err", err)
			if strings.Index(err.Error(), "not found") != -1 {
				w.log.Info("tx is temporary not found, please wait...", "tx", txHash)
				time.Sleep(time.Millisecond * 900)
				continue
			}
			return err
		}

		if isPending {
			w.log.Info("tx is pending, please wait...")
			time.Sleep(time.Millisecond * 900)
			continue
		}
		w.log.Info("tx is successful", "tx", txHash)
		break
	}
	return nil
}
