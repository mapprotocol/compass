package writer

import (
	"github.com/mapprotocol/compass/internal/constant"
	"github.com/mapprotocol/compass/msg"
	"strings"
	"time"
)

// execMap2OtherMsg executes sync msg, and send tx to the destination blockchain
func (w *Writer) execMap2OtherMsg(m msg.Message) bool {
	//return w.callContractWithMsg(,  m)
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
			tx, err := w.sendTx(&w.cfg.LightNode, nil, m.Payload[0].([]byte))
			w.conn.UnlockOpts()
			if err == nil {
				// message successfully handled
				w.log.Info("Sync Map Header to other chain tx execution", "tx", tx.Hash(), "src", m.Source, "dst", m.Destination)
				// waited till successful mined
				err = w.blockForPending(tx.Hash())
				if err != nil {
					w.log.Warn("Sync Map Header to other chain blockForPending error", "err", err)
				} else {
					err = w.txStatus(tx.Hash())
					if err != nil {
						w.log.Warn("TxHash Status is not successful, will retry", "err", err)
					} else {
						m.DoneCh <- struct{}{}
						return true
					}
				}
			} else if strings.Index(err.Error(), constant.EthOrderExist) != -1 {
				w.log.Info(constant.EthOrderExistPrint, "err", err)
				m.DoneCh <- struct{}{}
				return true
			} else if strings.Index(err.Error(), constant.HeaderIsHave) != -1 {
				w.log.Info(constant.HeaderIsHavePrint, "err", err)
				m.DoneCh <- struct{}{}
				return true
			} else if strings.Index(err.Error(), "EOF") != -1 {
				w.log.Error("Sync Header to map encounter EOF, will retry")
			} else if err.Error() == constant.ErrNonceTooLow.Error() || err.Error() == constant.ErrTxUnderpriced.Error() {
				w.log.Error("Sync Map Header to other chain Nonce too low, will retry")
			} else if strings.Index(err.Error(), constant.NotEnoughGas) != -1 {
				w.log.Error(constant.NotEnoughGasPrint)
			} else {
				w.log.Warn("Sync Map Header to other chain Execution failed, header may already been synced",
					"gasLimit", gasLimit, "gasPrice", gasPrice, "err", err)
			}
			time.Sleep(constant.TxRetryInterval)
		}
	}
	//w.log.Error("Sync Map Header to other chain Submission of Sync MapHeader transaction failed", "source", m.Source,
	//	"dest", m.Destination, "depositNonce", m.DepositNonce)
	//w.sysErr <- constant.ErrFatalTx
	//return false
}
