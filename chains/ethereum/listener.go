// Copyright 2021 Compass Systems
// SPDX-License-Identifier: LGPL-3.0-only

package ethereum

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/pkg/math"

	metrics "github.com/ChainSafe/chainbridge-utils/metrics/types"
	"github.com/ChainSafe/log15"
	eth "github.com/ethereum/go-ethereum"
	ethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/mapprotocol/compass/blockstore"
	"github.com/mapprotocol/compass/chains"
	"github.com/mapprotocol/compass/mapprotocol"
	"github.com/mapprotocol/compass/msg"
	utils "github.com/mapprotocol/compass/shared/ethereum"
)

var BlockRetryInterval = time.Second * 5
var BlockRetryLimit = 5
var ErrFatalPolling = errors.New("listener block polling failed")

type listener struct {
	cfg                Config
	conn               Connection
	router             chains.Router
	log                log15.Logger
	blockstore         blockstore.Blockstorer
	stop               <-chan int
	sysErr             chan<- error // Reports fatal error to core
	latestBlock        metrics.LatestBlock
	metrics            *metrics.ChainMetrics
	blockConfirmations *big.Int
	msgCh              chan struct{} // wait for msg handles
	syncedHeight       *big.Int      // used to record syncd height in map chain when sync is on
}

// NewListener creates and returns a listener
func NewListener(conn Connection, cfg *Config, log log15.Logger, bs blockstore.Blockstorer, stop <-chan int, sysErr chan<- error, m *metrics.ChainMetrics) *listener {
	return &listener{
		cfg:                *cfg,
		conn:               conn,
		log:                log,
		blockstore:         bs,
		stop:               stop,
		sysErr:             sysErr,
		latestBlock:        metrics.LatestBlock{LastUpdated: time.Now()},
		metrics:            m,
		blockConfirmations: cfg.blockConfirmations,
		msgCh:              make(chan struct{}),
	}
}

// sets the router
func (l *listener) setRouter(r chains.Router) {
	l.router = r
}

// start registers all subscriptions provided by the config
func (l *listener) start() error {
	l.log.Debug("Starting listener...")

	go func() {
		err := l.pollBlocks()
		if err != nil {
			l.log.Error("Polling blocks failed", "err", err)
		}
	}()

	return nil
}

// pollBlocks will poll for the latest block and proceed to parse the associated events as it sees new blocks.
// Polling begins at the block defined in `l.cfg.startBlock`. Failed attempts to fetch the latest block or parse
// a block will be retried up to BlockRetryLimit times before continuing to the next block.
func (l *listener) pollBlocks() error {
	var currentBlock = l.cfg.startBlock
	l.log.Info("Polling Blocks...", "block", currentBlock)

	// check whether needs quick sync
	if l.cfg.syncToMap {
		syncedHeight, _, err := mapprotocol.GetCurrentNumberAbi(ethcommon.HexToAddress(l.cfg.from))
		if err != nil {
			l.log.Error("Get synced Height failed")
			return err
		}

		l.log.Info("Check Sync Status...", "synced", syncedHeight)
		l.syncedHeight = syncedHeight

		// when sync to map there must be a 20 block confirmation at least
		big20 := big.NewInt(20)
		if l.blockConfirmations.Cmp(big20) == -1 {
			l.blockConfirmations = big20
		}
		// fix the currentBlock Number
		currentBlock = big.NewInt(0).Sub(currentBlock, l.blockConfirmations)

		if currentBlock.Cmp(l.syncedHeight) == 1 {
			//sync and start block differs too much perform a fast synced
			l.log.Info("Perform fast sync to catch up...")
			err = l.batchSyncHeadersTo(big.NewInt(0).Sub(currentBlock, mapprotocol.Big1))
			if err != nil {
				l.log.Error("Fast batch sync failed")
				return err
			}
		}
	}

	var retry = BlockRetryLimit
	for {
		select {
		case <-l.stop:
			return errors.New("polling terminated")
		default:
			// No more retries, goto next block
			if retry == 0 {
				l.log.Error("Polling failed, retries exceeded")
				l.sysErr <- ErrFatalPolling
				return nil
			}

			latestBlock, err := l.conn.LatestBlock()
			if err != nil {
				l.log.Error("Unable to get latest block", "block", currentBlock, "err", err)
				retry--
				time.Sleep(BlockRetryInterval)
				continue
			}

			if l.metrics != nil {
				l.metrics.LatestKnownBlock.Set(float64(latestBlock.Int64()))
			}

			// Sleep if the difference is less than BlockDelay; (latest - current) < BlockDelay
			if big.NewInt(0).Sub(latestBlock, currentBlock).Cmp(l.blockConfirmations) == -1 {
				l.log.Debug("Block not ready, will retry", "target", currentBlock, "latest", latestBlock)
				time.Sleep(BlockRetryInterval)
				continue
			}

			// Sync headers to Map
			if l.cfg.syncToMap {
				// sync when catchup
				offsetCurrentBlock := big.NewInt(0).Sub(currentBlock, l.blockConfirmations)
				if offsetCurrentBlock.Cmp(l.syncedHeight) == 1 {
					l.log.Info("Sync Header to Map Chain", "target", currentBlock)
					err = l.syncHeaderToMapChain(currentBlock)
					if err != nil {
						l.log.Error("Failed to sync header for block", "block", currentBlock, "err", err)
						retry--
						continue
					}
				}
			}

			// Parse out events
			count, err := l.getEventsForBlock(currentBlock)
			if err != nil {
				l.log.Error("Failed to get events for block", "block", currentBlock, "err", err)
				retry--
				continue
			}

			// hold until all messages are handled
			l.waitUntilMsgHandled(count)

			// Write to block store. Not a critical operation, no need to retry
			err = l.blockstore.StoreBlock(currentBlock)
			if err != nil {
				l.log.Error("Failed to write latest block to blockstore", "block", currentBlock, "err", err)
			}

			if l.metrics != nil {
				l.metrics.BlocksProcessed.Inc()
				l.metrics.LatestProcessedBlock.Set(float64(latestBlock.Int64()))
			}

			l.latestBlock.Height = big.NewInt(0).Set(latestBlock)
			l.latestBlock.LastUpdated = time.Now()

			// Goto next block and reset retry counter
			currentBlock.Add(currentBlock, big.NewInt(1))
			retry = BlockRetryLimit
		}
	}
}

// getEventsForBlock looks for the deposit event in the latest block
func (l *listener) getEventsForBlock(latestBlock *big.Int) (int, error) {
	l.log.Debug("Querying block for events", "block", latestBlock)
	query := buildQuery(l.cfg.bridgeContract, utils.SwapOut, latestBlock, latestBlock)

	// querying for logs
	logs, err := l.conn.Client().FilterLogs(context.Background(), query)
	if err != nil {
		return 0, fmt.Errorf("unable to Filter Logs: %w", err)
	}

	// read through the log events and handle their deposit event if handler is recognized
	for _, log := range logs {
		// evm event to msg
		var m msg.Message
		if l.cfg.syncToMap {
			// when syncToMap we need to assemble a tx proof
			txsHash, err := getTransactionsHashByBlockNumber(l.conn.Client(), latestBlock)
			if err != nil {
				return 0, fmt.Errorf("unable to get tx hashes Logs: %w", err)
			}
			receipts, err := getReceiptsByTxsHash(l.conn.Client(), txsHash)
			if err != nil {
				return 0, fmt.Errorf("unable to get receipts hashes Logs: %w", err)
			}

			fromChainID, toChainID, payload, err := utils.ParseEthLogIntoSwapWithProofArgs(log, l.cfg.bridgeContract, receipts)
			if err != nil {
				return 0, fmt.Errorf("unable to Parse Log: %w", err)
			}

			msgpayload := []interface{}{payload}
			m = msg.NewSwapWithProof(msg.ChainId(fromChainID), msg.ChainId(toChainID), msgpayload, l.msgCh)

		} else {
			fromChainID, toChainID, payload, err := utils.ParseEthLogIntoSwapArgs(log, l.cfg.bridgeContract)
			if err != nil {
				return 0, fmt.Errorf("unable to Parse SwapArgs Log: %w", err)
			}

			msgpayload := []interface{}{payload}
			m = msg.NewSwapTransfer(msg.ChainId(fromChainID), msg.ChainId(toChainID), msgpayload, l.msgCh)
		}

		l.log.Info("Event found", "BlockNumber", log.BlockNumber, "txHash", log.TxHash, "logIdx", log.Index)
		err = l.router.Send(m)
		if err != nil {
			l.log.Error("subscription error: failed to route message", "err", err)
		}
	}

	return len(logs), nil
}

// buildQuery constructs a query for the bridgeContract by hashing sig to get the event topic
func buildQuery(contract ethcommon.Address, sig utils.EventSig, startBlock *big.Int, endBlock *big.Int) eth.FilterQuery {
	query := eth.FilterQuery{
		FromBlock: startBlock,
		ToBlock:   endBlock,
		Addresses: []ethcommon.Address{contract},
		Topics: [][]ethcommon.Hash{
			{sig.GetTopic()},
		},
	}
	return query
}

// waitUntilMsgHandled this function will block untill message is handled
func (l *listener) waitUntilMsgHandled(counter int) error {
	l.log.Debug("waitUntilMsgHandled", "counter", counter)
	for counter > 0 {
		<-l.msgCh
		counter -= 1
	}
	return nil
}

// syncHeaderToMapChain sync header from current chain to Map chain
func (l *listener) syncHeaderToMapChain(latestBlock *big.Int) error {
	header, err := l.conn.Client().HeaderByNumber(context.Background(), latestBlock)
	if err != nil {
		return err
	}
	chains := []types.Header{*header}
	marshal, _ := rlp.EncodeToBytes(chains)

	msgpayload := []interface{}{marshal}
	m := msg.NewSyncToMap(l.cfg.id, l.cfg.mapChainID, msgpayload, l.msgCh)

	err = l.router.Send(m)
	if err != nil {
		l.log.Error("subscription error: failed to route message", "err", err)
		return nil
	}

	err = l.waitUntilMsgHandled(1)
	if err != nil {
		return err
	}
	return nil
}

func (l *listener) batchSyncHeadersTo(height *big.Int) error {
	// batch
	var batch = big.NewInt(20)
	chains := make([]types.Header, batch.Int64())
	var heightDiff = big.NewInt(0)
	for l.syncedHeight.Cmp(height) == -1 {
		chains = chains[:0]
		heightDiff.Sub(height, l.syncedHeight)
		loop := math.MinBigInt(batch, heightDiff)
		for i := int64(1); i <= loop.Int64(); i++ {
			calcHeight := big.NewInt(0).Add(l.syncedHeight, big.NewInt(i))

			header, err := l.conn.Client().HeaderByNumber(context.Background(), calcHeight)
			if err != nil {
				return err
			}
			chains = append(chains, *header)
		}

		marshal, _ := rlp.EncodeToBytes(chains)
		msgpayload := []interface{}{marshal}
		m := msg.NewSyncToMap(l.cfg.id, l.cfg.mapChainID, msgpayload, l.msgCh)

		err := l.router.Send(m)
		if err != nil {
			l.log.Error("subscription error: failed to route message", "err", err)
			return err
		}

		err = l.waitUntilMsgHandled(1)
		if err != nil {
			return err
		}

		l.syncedHeight = l.syncedHeight.Add(l.syncedHeight, loop)
		l.log.Info("Headers synced...", "height", l.syncedHeight)
	}

	l.log.Info("Batch sync finished", "height", height, "syncHeight", l.syncedHeight)
	return nil
}
