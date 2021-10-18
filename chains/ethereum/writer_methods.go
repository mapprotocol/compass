// Copyright 2020 ChainSafe Systems
// SPDX-License-Identifier: LGPL-3.0-only

package ethereum

import (
	"context"
	"errors"
	"math/big"
	"time"

	log "github.com/ChainSafe/log15"
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

// watchThenExecute watches for the latest block and executes once the matching finalized event is found
func (w *writer) watchThenExecute(m msg.Message, data []byte, dataHash [32]byte, latestBlock *big.Int) {
	w.log.Info("Watching for finalization event", "src", m.Source, "nonce", m.DepositNonce)

	// watching for the latest block, querying and matching the finalized event will be retried up to ExecuteBlockWatchLimit times
	for i := 0; i < ExecuteBlockWatchLimit; i++ {
		select {
		case <-w.stop:
			return
		default:
			// watch for the lastest block, retry up to BlockRetryLimit times
			for waitRetrys := 0; waitRetrys < BlockRetryLimit; waitRetrys++ {
				err := w.conn.WaitForBlock(latestBlock, w.cfg.blockConfirmations)
				if err != nil {
					w.log.Error("Waiting for block failed", "err", err)
					// Exit if retries exceeded
					if waitRetrys+1 == BlockRetryLimit {
						w.log.Error("Waiting for block retries exceeded, shutting down")
						w.sysErr <- ErrFatalQuery
						return
					}
				} else {
					break
				}
			}

			// query for logs
			query := buildQuery(w.cfg.bridgeContract, utils.ProposalEvent, latestBlock, latestBlock)
			evts, err := w.conn.Client().FilterLogs(context.Background(), query)
			if err != nil {
				w.log.Error("Failed to fetch logs", "err", err)
				return
			}

			// execute the proposal once we find the matching finalized event
			for _, evt := range evts {
				sourceId := evt.Topics[1].Big().Uint64()
				depositNonce := evt.Topics[2].Big().Uint64()
				status := evt.Topics[3].Big().Uint64()

				if m.Source == msg.ChainId(sourceId) &&
					m.DepositNonce.Big().Uint64() == depositNonce &&
					utils.IsFinalized(uint8(status)) {
					// todo
					//w.executeProposal(m, data, dataHash)
					return
				} else {
					w.log.Trace("Ignoring event", "src", sourceId, "nonce", depositNonce)
				}
			}
			w.log.Trace("No finalization event found in current block", "block", latestBlock, "src", m.Source, "nonce", m.DepositNonce)
			latestBlock = latestBlock.Add(latestBlock, big.NewInt(1))
		}
	}
	log.Warn("Block watch limit exceeded, skipping execution", "source", m.Source, "dest", m.Destination, "nonce", m.DepositNonce)
}
