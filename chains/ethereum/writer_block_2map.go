package ethereum

import (
	"math/big"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/rlp"
	"github.com/mapprotocol/compass/mapprotocol"
	"github.com/mapprotocol/compass/msg"
)

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

			param := struct {
				From    *big.Int
				To      *big.Int
				Headers []byte
			}{
				From:    src,
				To:      dest,
				Headers: marshal,
			}

			bytes, err := rlp.EncodeToBytes(param)
			if err != nil {
				w.log.Error("rlp EncodeToBytes failed ", "err", err)
				return false
			}

			// save header data
			data, err := mapprotocol.SaveHeaderTxData(bytes)
			if err != nil {
				w.log.Error("Failed to pack abi data", "err", err)
				w.conn.UnlockOpts()
				return false
			}
			tx, err := w.sendTx(&mapprotocol.RelayerAddress, nil, data)
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
			} else if err.Error() == ErrNonceTooLow.Error() || err.Error() == ErrTxUnderpriced.Error() {
				w.log.Error("Sync Header to map Nonce too low, will retry")
				time.Sleep(TxRetryInterval)
			} else if strings.Index(err.Error(), "EOF") != -1 { // When requesting the lightNode to return EOF, it indicates that there may be a problem with the network and it needs to be retried
				w.log.Error("Sync Header to map encounter EOF, will retry")
				time.Sleep(TxRetryInterval)
			} else {
				w.log.Warn("Sync Header to map Execution failed, header may already been synced", "gasLimit", gasLimit, "gasPrice", gasPrice, "err", err)
				m.DoneCh <- struct{}{}
				return true
			}
		}
	}
	w.log.Error("Sync Header to map Submission of Sync Header transaction failed", "source", m.Source, "dest", m.Destination, "depositNonce", m.DepositNonce)
	w.sysErr <- ErrFatalTx
	return false
}
