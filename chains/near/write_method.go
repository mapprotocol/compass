package near

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/mapprotocol/compass/internal/near"

	"github.com/eteu-technologies/near-api-go/pkg/types"

	"github.com/eteu-technologies/near-api-go/pkg/client"
	"github.com/eteu-technologies/near-api-go/pkg/types/action"

	"github.com/eteu-technologies/near-api-go/pkg/types/hash"
	"github.com/mapprotocol/compass/msg"
)

const (
	// TxRetryLimit Maximum number of tx retries before exiting
	TxRetryLimit = 10
	// TxRetryInterval Time between retrying a failed tx
	TxRetryInterval              = time.Second * 2
	AbiMethodOfUpdateBlockHeader = "update_block_header"
	AbiMethodOfNew               = "new"
	AbiMethodOfGetHeaderHeight   = "get_header_height"
)

var (
	ErrNonceTooLow   = errors.New("nonce too low")
	ErrFatalTx       = errors.New("submission of transaction failed")
	ErrTxUnderpriced = errors.New("replacement transaction underpriced")
)

// exeSyncMapMsg executes sync msg, and send tx to the destination blockchain
func (w *writer) exeSyncMapMsg(m msg.Message) bool {
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
			// gasLimit := w.conn.Opts().GasLimit
			// gasPrice := w.conn.Opts().GasPrice

			txHash, err := w.sendTx(w.cfg.lightNode, nil, m.Payload[0].([]byte))
			w.conn.UnlockOpts()
			if err == nil {
				// message successfully handled
				w.log.Info("Sync MapHeader to Near tx execution", "tx", txHash.String(), "src", m.Source, "dst", m.Destination)
				// waited till successful mined
				err = w.blockForPending(txHash)
				if err != nil {
					w.log.Warn("blockForPending error", "err", err)
				}
				m.DoneCh <- struct{}{}
				return true
			} else if err.Error() == ErrNonceTooLow.Error() || err.Error() == ErrTxUnderpriced.Error() {
				w.log.Error("Nonce too low, will retry")
				time.Sleep(TxRetryInterval)
			} else {
				w.log.Warn("Execution failed ", "err", err)
				m.DoneCh <- struct{}{}
				return true
			}
		}
	}
	w.log.Error("Submission of Sync MapHeader transaction failed", "source", m.Source, "dest", m.Destination, "depositNonce", m.DepositNonce)
	w.sysErr <- ErrFatalTx
	return false
}

// sendTx send tx to an address with value and input data
func (w *writer) sendTx(toAddress string, value *big.Int, input []byte) (hash.CryptoHash, error) {
	w.log.Info("sendTx", "input", string(input))
	ctx := client.ContextWithKeyPair(context.Background(), *w.conn.Keypair())
	w.log.Info("sendTx  request")
	res, err := w.conn.Client().TransactionSendAwait(
		ctx,
		w.cfg.from,
		toAddress,
		[]action.Action{
			action.NewFunctionCall(AbiMethodOfUpdateBlockHeader, input, near.NewFunctionCallGas,
				types.Balance{}), // todo deposit
		},
		client.WithLatestBlock(),
		client.WithKeyPair(*w.conn.Keypair()),
	)
	w.log.Info("sendTx  request done")
	if err != nil {
		return hash.CryptoHash{}, fmt.Errorf("failed to do txn: %w", err)
	}
	if len(res.Status.Failure) != 0 {
		return hash.CryptoHash{}, fmt.Errorf("back resp failed, err is %s", string(res.Status.Failure))
	}
	//w.log.Info("sendTx success", "res", res)
	fmt.Printf("sendTx resp is %+v", res)
	return res.Transaction.Hash, nil
}

// this function will block for the txhash given
func (w *writer) blockForPending(txHash hash.CryptoHash) error {
	for {
		resp, err := w.conn.Client().TransactionStatus(context.Background(), txHash, w.cfg.from)
		if err != nil {
			return err
		}

		w.log.Info("blockForPending ", "resp", resp)

		if resp.Status.SuccessValue != "" { // todo modify check condition
			w.log.Info("tx is pending, please wait...")
			time.Sleep(time.Millisecond * 900)
			continue
		}
		w.log.Info("tx is successful", "tx", txHash)
		break
	}
	return nil
}
