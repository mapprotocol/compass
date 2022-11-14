package near

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/mapprotocol/compass/internal/near"
	"github.com/mapprotocol/compass/msg"
	"github.com/mapprotocol/near-api-go/pkg/client"
	"github.com/mapprotocol/near-api-go/pkg/types"
	"github.com/mapprotocol/near-api-go/pkg/types/action"
	"github.com/mapprotocol/near-api-go/pkg/types/hash"
)

const (
	// TxRetryLimit Maximum number of tx retries before exiting
	TxRetryLimit = 10
	// TxRetryInterval Time between retrying a failed tx
	TxRetryInterval              = time.Second * 2
	AbiMethodOfUpdateBlockHeader = "update_block_header"
	AbiMethodOfNew               = "new"
	AbiMethodOfGetHeaderHeight   = "get_header_height"
	AbiMethodOfTransferIn        = "transfer_in"
)

var (
	ErrNonceTooLow   = errors.New("nonce too low")
	ErrFatalTx       = errors.New("submission of transaction failed")
	ErrTxUnderpriced = errors.New("replacement transaction underpriced")
)

// exeSyncMapMsg executes sync msg, and send tx to the destination blockchain
func (w *writer) exeSyncMapMsg(m msg.Message) bool {
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

			txHash, err := w.sendTx(w.cfg.lightNode, AbiMethodOfUpdateBlockHeader, m.Payload[0].([]byte))
			w.conn.UnlockOpts()
			if err == nil {
				// message successfully handled
				w.log.Info("Sync MapHeader to Near tx execution", "tx", txHash.String(), "src", m.Source, "dst", m.Destination)
				m.DoneCh <- struct{}{}
				return true
			} else if err.Error() == ErrNonceTooLow.Error() || err.Error() == ErrTxUnderpriced.Error() {
				w.log.Error("Nonce too low, will retry", "err", err)
				time.Sleep(TxRetryInterval)
			} else if strings.Index(err.Error(), "EOF") != -1 || strings.Index(err.Error(), "unexpected end of JSON input") != -1 { // When requesting the lightNode to return EOF, it indicates that there may be a problem with the network and it needs to be retried
				w.log.Error("Sync Header to map encounter EOF, will retry")
				time.Sleep(TxRetryInterval)
			} else {
				w.log.Warn("Execution failed will retry", "err", err)
				m.DoneCh <- struct{}{}
				return true
			}
		}
	}
	w.log.Error("Submission of Sync MapHeader transaction failed", "source", m.Source, "dest", m.Destination, "depositNonce", m.DepositNonce)
	w.sysErr <- ErrFatalTx
	return false
}

// exeSwapMsg executes swap msg, and send tx to the destination blockchain
func (w *writer) exeSwapMsg(m msg.Message) bool {
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

			// sendtx using general method
			txHash, err := w.sendTx(w.cfg.mcsContract, AbiMethodOfTransferIn, m.Payload[0].([]byte))
			w.conn.UnlockOpts()
			if err == nil {
				// message successfully handled
				w.log.Info("Submitted cross tx execution", "txHash", txHash.String(), "src", m.Source, "dst", m.Destination, "nonce", m.DepositNonce)
				m.DoneCh <- struct{}{}
				return true
			} else if err.Error() == ErrNonceTooLow.Error() || err.Error() == ErrTxUnderpriced.Error() {
				w.log.Error("Nonce too low, will retry")
				time.Sleep(TxRetryInterval)
			} else if strings.Index(err.Error(), "EOF") != -1 || strings.Index(err.Error(), "unexpected end of JSON input") != -1 { // When requesting the lightNode to return EOF, it indicates that there may be a problem with the network and it needs to be retried
				w.log.Error("Sync Header to map encounter EOF, will retry", "err", err)
				time.Sleep(TxRetryInterval)
			} else if strings.Index(err.Error(), "the event with order id") != -1 && strings.Index(err.Error(), "is used") != -1 {
				w.log.Info("Execution failed, tx may already be complete", "back", err)
				m.DoneCh <- struct{}{}
				return true
			} else {
				w.log.Warn("Execution failed, tx may already be complete", "err", err)
				time.Sleep(TxRetryInterval)
				m.DoneCh <- struct{}{}
				return true
			}
		}
	}
	w.log.Error("Submission of Execute transaction failed", "source", m.Source, "dest", m.Destination, "depositNonce", m.DepositNonce)
	w.sysErr <- ErrFatalTx
	return false
}

// sendTx send tx to an address with value and input data
func (w *writer) sendTx(toAddress string, method string, input []byte) (hash.CryptoHash, error) {
	w.log.Info("sendTx", "toAddress", toAddress)
	ctx := client.ContextWithKeyPair(context.Background(), *w.conn.Keypair())
	b := types.Balance{}
	if method == AbiMethodOfTransferIn {
		b, _ = types.BalanceFromString(near.Deposit)
	}
	res, err := w.conn.Client().TransactionSendAwait(
		ctx,
		w.cfg.from,
		toAddress,
		[]action.Action{
			action.NewFunctionCall(method, input, near.NewFunctionCallGas, b),
		},
		client.WithLatestBlock(),
		client.WithKeyPair(*w.conn.Keypair()),
	)
	if err != nil {
		return hash.CryptoHash{}, fmt.Errorf("failed to do txn: %w", err)
	}
	w.log.Debug("sendTx success", "res", res)
	if len(res.Status.Failure) != 0 {
		return hash.CryptoHash{}, fmt.Errorf("back resp failed, err is %s", string(res.Status.Failure))
	}
	return res.Transaction.Hash, nil
}
