// Copyright 2020 ChainSafe Systems
// SPDX-License-Identifier: LGPL-3.0-only

package ethereum

import (
	"errors"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/mapprotocol/compass/msg"
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

// executeMsg executes the msg, and send tx to the blockchain
func (w *writer) executeMsg(m msg.Message) {
	for i := 0; i < TxRetryLimit; i++ {
		select {
		case <-w.stop:
			return
		default:
			err := w.conn.LockAndUpdateOpts()
			if err != nil {
				w.log.Error("Failed to update nonce", "err", err)
				return
			}
			// These store the gas limit and price before a transaction is sent for logging in case of a failure
			// This is necessary as tx will be nil in the case of an error when sending VoteProposal()
			gasLimit := w.conn.Opts().GasLimit
			gasPrice := w.conn.Opts().GasPrice
			nonce := w.conn.Opts().Nonce

			// todo # sendtx using general method
			tx := types.NewTransaction(nonce.Uint64(), w.cfg.bridgeContract, big.NewInt(0), gasLimit, gasPrice, nil)

			// err = w.conn.SendTransaction(context.Background(), signedTx)
			// if err != nil {
			// log.Fatal(err)
			// }

			// tx, err := w.bridgeContract.ExecuteProposal(
			// 	w.conn.Opts(),
			// 	uint8(m.Source),
			// 	uint64(m.DepositNonce),
			// 	data,
			// 	m.ResourceId,
			// )
			w.conn.UnlockOpts()

			if err == nil {
				w.log.Info("Submitted proposal execution", "tx", tx.Hash(), "src", m.Source, "dst", m.Destination, "nonce", m.DepositNonce, "gasPrice", tx.GasPrice().String())
				return
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
}
