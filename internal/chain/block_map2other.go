package chain

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/mapprotocol/compass/mapprotocol"

	"github.com/mapprotocol/compass/internal/constant"
	"github.com/mapprotocol/compass/msg"
	"github.com/mapprotocol/compass/pkg/util"
)

// execMap2OtherMsg executes sync msg, and send tx to the destination blockchain
func (w *Writer) execMap2OtherMsg(m msg.Message) bool {
	var (
		errorCount int64
		needNonce  = true
	)
	for {
		select {
		case <-w.stop:
			return false
		default:
			err := w.conn.LockAndUpdateOpts(needNonce)
			if err != nil {
				w.log.Error("Failed to update nonce", "err", err)
				time.Sleep(constant.TxRetryInterval)
				continue
			}
			// These store the gas limit and price before a transaction is sent for logging in case of a failure
			// This is necessary as tx will be nil in the case of an error when sending VoteProposal()
			tx, err := w.sendTx(&w.cfg.LightNode, nil, m.Payload[0].([]byte))
			w.conn.UnlockOpts()
			if err == nil {
				// message successfully handled
				w.log.Info("Sync Map Header to other chain tx execution", "tx", tx.Hash(), "src", m.Source, "dst", m.Destination, "needNonce", needNonce, "nonce", w.conn.Opts().Nonce)
				err = w.txStatus(tx.Hash())
				if err != nil {
					w.log.Warn("TxHash Status is not successful, will retry", "err", err)
				} else {
					m.DoneCh <- struct{}{}
					return true
				}
			} else {
				for e := range constant.IgnoreError {
					if strings.Index(err.Error(), e) != -1 {
						w.log.Info("Ignore This Error, Continue to the next", "id", m.Destination, "err", err)
						m.DoneCh <- struct{}{}
						return true
					}
				}
				w.log.Warn("Sync Map Header to other chain Execution failed, header may already been synced", "id", m.Destination, "err", err)
			}
			needNonce = w.needNonce(err)
			errorCount++
			if errorCount >= 10 {
				util.Alarm(context.Background(), fmt.Sprintf("map2%s updateHeader failed, err is %s", mapprotocol.OnlineChaId[m.Destination], err.Error()))
				errorCount = 0
			}
			time.Sleep(constant.TxRetryInterval)
		}
	}
}
