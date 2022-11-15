package ethereum

import (
	"context"
	"errors"
	"math/big"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/mapprotocol/compass/msg"
)

const (
	TxRetryInterval = time.Second * 2 // TxRetryInterval Time between retrying a failed tx
	TxRetryLimit    = 10              // TxRetryLimit Maximum number of tx retries before exiting
)

var (
	ErrNonceTooLow   = errors.New("nonce too low")
	ErrTxUnderpriced = errors.New("replacement transaction underpriced")
	ErrFatalTx       = errors.New("submission of transaction failed")
)

// exeSyncMapMsg executes sync msg, and send tx to the destination blockchain
func (w *writer) exeSyncMapMsg(m msg.Message) bool {
	for {
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

			tx, err := w.sendTx(&w.cfg.lightNode, nil, m.Payload[0].([]byte))
			w.conn.UnlockOpts()
			if err == nil {
				// message successfully handled
				w.log.Info("Sync Map Header to other chain tx execution", "tx", tx.Hash(), "src", m.Source, "dst", m.Destination)
				// waited till successful mined
				err = w.blockForPending(tx.Hash())
				if err != nil {
					w.log.Warn("Sync Map Header to other chain blockForPending error", "err", err)
				}
				m.DoneCh <- struct{}{}
				return true
			} else if strings.Index(err.Error(), "EOF") != -1 {
				w.log.Error("Sync Header to map encounter EOF, will retry")
			} else if err.Error() == ErrNonceTooLow.Error() || err.Error() == ErrTxUnderpriced.Error() {
				w.log.Error("Sync Map Header to other chain Nonce too low, will retry")
			} else if strings.Index(err.Error(), "insufficient funds for gas * price + value") != -1 {
				w.log.Error("insufficient funds for gas * price + value, will retry")
			} else {
				w.log.Warn("Sync Map Header to other chain Execution failed", "gasLimit", gasLimit, "gasPrice", gasPrice, "err", err)
			}
			time.Sleep(TxRetryInterval)
		}
	}
	//w.log.Error("Sync Map Header to other chain Submission of Sync MapHeader transaction failed", "source", m.Source, "dest", m.Destination, "depositNonce", m.DepositNonce)
	//w.sysErr <- ErrFatalTx
	//return false
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

// this function will block for the txhash given
func (w *writer) blockForPending(txHash common.Hash) error {
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
