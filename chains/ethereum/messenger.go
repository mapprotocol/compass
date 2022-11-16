package ethereum

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/mapprotocol/compass/mapprotocol"

	"github.com/mapprotocol/compass/msg"

	eth "github.com/ethereum/go-ethereum"
	ethcommon "github.com/ethereum/go-ethereum/common"
	utils "github.com/mapprotocol/compass/shared/ethereum"
)

type Messenger struct {
	*CommonSync
}

func NewMessenger(cs *CommonSync) *Messenger {
	return &Messenger{
		CommonSync: cs,
	}
}

func (m *Messenger) Sync() error {
	m.log.Debug("Starting listener...")
	go func() {
		err := m.sync()
		if err != nil {
			m.log.Error("Polling blocks failed", "err", err)
		}
	}()

	return nil
}

// sync function of Messenger will poll for the latest block and listen the log information of transactions in the block
// Polling begins at the block defined in `m.cfg.startBlock`. Failed attempts to fetch the latest block or parse
// a block will be retried up to BlockRetryLimit times before continuing to the next block.
// However，an error in synchronizing the log will cause the entire program to block
func (m *Messenger) sync() error {
	var currentBlock = m.cfg.startBlock

	if m.cfg.syncToMap {
		// when listen to map there must be a 20 block confirmation at least
		big20 := big.NewInt(20)
		if m.blockConfirmations.Cmp(big20) == -1 {
			m.blockConfirmations = big20
		}
	}

	var retry = BlockRetryLimit
	for {
		select {
		case <-m.stop:
			return errors.New("polling terminated")
		default:
			// No more retries, goto next block
			if retry == 0 {
				m.log.Error("Polling failed, retries exceeded")
				m.sysErr <- ErrFatalPolling
				return nil
			}

			latestBlock, err := m.conn.LatestBlock()
			if err != nil {
				m.log.Error("Unable to get latest block", "block", currentBlock, "err", err)
				retry--
				time.Sleep(BlockRetryInterval)
				continue
			}

			if m.metrics != nil {
				m.metrics.LatestKnownBlock.Set(float64(latestBlock.Int64()))
			}

			// Sleep if the difference is less than BlockDelay; (latest - current) < BlockDelay
			if big.NewInt(0).Sub(latestBlock, currentBlock).Cmp(m.blockConfirmations) == -1 {
				m.log.Debug("Block not ready, will retry", "target", currentBlock, "latest", latestBlock)
				time.Sleep(BlockRetryInterval)
				continue
			}
			// messager
			// Parse out events
			count, err := m.getEventsForBlock(currentBlock)
			if err != nil {
				m.log.Error("Failed to get events for block", "block", currentBlock, "err", err)
				retry--
				continue
			}

			// hold until all messages are handled
			_ = m.waitUntilMsgHandled(count)

			// Write to block store. Not a critical operation, no need to retry
			err = m.blockStore.StoreBlock(currentBlock)
			if err != nil {
				m.log.Error("Failed to write latest block to blockstore", "block", currentBlock, "err", err)
			}
			if m.metrics != nil {
				m.metrics.BlocksProcessed.Inc()
				m.metrics.LatestProcessedBlock.Set(float64(latestBlock.Int64()))
			}

			m.latestBlock.Height = big.NewInt(0).Set(latestBlock)
			m.latestBlock.LastUpdated = time.Now()

			// Goto next block and reset retry counter
			currentBlock.Add(currentBlock, big.NewInt(1))
			retry = BlockRetryLimit
		}
	}
}

// getEventsForBlock looks for the deposit event in the latest block
func (m *Messenger) getEventsForBlock(latestBlock *big.Int) (int, error) {
	m.log.Debug("Querying block for events", "block", latestBlock)
	query := m.buildQuery(m.cfg.mcsContract, m.cfg.events, latestBlock, latestBlock)

	// querying for logs
	logs, err := m.conn.Client().FilterLogs(context.Background(), query)
	if err != nil {
		return 0, fmt.Errorf("unable to Filter Logs: %w", err)
	}

	count := 0
	// read through the log events and handle their deposit event if handler is recognized
	for _, log := range logs {
		// evm event to msg
		var message msg.Message
		// getOrderId
		orderId := log.Data[64:96]
		if m.cfg.syncToMap {
			method := mapprotocol.MethodOfTransferIn
			if log.Topics[0] != mapprotocol.HashOfTransferIn {
				method = mapprotocol.MethodOfDepositIn
			}
			// when syncToMap we need to assemble a tx proof
			txsHash, err := getTransactionsHashByBlockNumber(m.conn.Client(), latestBlock)
			if err != nil {
				return 0, fmt.Errorf("unable to get tx hashes Logs: %w", err)
			}
			receipts, err := getReceiptsByTxsHash(m.conn.Client(), txsHash)
			if err != nil {
				return 0, fmt.Errorf("unable to get receipts hashes Logs: %w", err)
			}
			payload, err := utils.ParseEthLogIntoSwapWithProofArgs(log, m.cfg.mcsContract, receipts, method, m.cfg.id, m.cfg.mapChainID)
			if err != nil {
				return 0, fmt.Errorf("unable to Parse Log: %w", err)
			}

			msgPayload := []interface{}{payload, orderId, latestBlock.Uint64(), log.TxHash}
			message = msg.NewSwapWithProof(m.cfg.id, m.cfg.mapChainID, msgPayload, m.msgCh)
		} else if m.cfg.id == m.cfg.mapChainID {
			// when listen from map we also need to assemble a tx prove in a different way
			header, err := m.conn.Client().MAPHeaderByNumber(context.Background(), latestBlock)
			if err != nil {
				return 0, fmt.Errorf("unable to query header Logs: %w", err)
			}
			txsHash, err := getMapTransactionsHashByBlockNumber(m.conn.Client(), latestBlock)
			if err != nil {
				return 0, fmt.Errorf("idSame unable to get tx hashes Logs: %w", err)
			}
			receipts, err := getReceiptsByTxsHash(m.conn.Client(), txsHash)
			if err != nil {
				return 0, fmt.Errorf("unable to get receipts hashes Logs: %w", err)
			}
			fromChainID, toChainID, payload, err := utils.ParseMapLogIntoSwapWithProofArgsV2(m.conn.Client(), log, receipts, header)
			if err != nil {
				return 0, fmt.Errorf("unable to Parse Log: %w", err)
			}

			if _, ok := mapprotocol.OnlineChaId[msg.ChainId(toChainID)]; !ok {
				m.log.Debug("Found a log that is not the current task ", "toChainID", toChainID)
				continue
			}
			msgPayload := []interface{}{payload, orderId, latestBlock.Uint64(), log.TxHash}
			message = msg.NewSwapWithMapProof(msg.ChainId(fromChainID), msg.ChainId(toChainID), msgPayload, m.msgCh)
		}

		m.log.Info("Event found", "BlockNumber", log.BlockNumber, "txHash", log.TxHash, "logIdx", log.Index, "orderId", ethcommon.Bytes2Hex(orderId))
		err = m.router.Send(message)
		if err != nil {
			m.log.Error("subscription error: failed to route message", "err", err)
		}
		count++
	}

	return count, nil
}

// buildQuery constructs a query for the bridgeContract by hashing sig to get the event topic
func (m *Messenger) buildQuery(contract ethcommon.Address, sig []utils.EventSig, startBlock *big.Int, endBlock *big.Int) eth.FilterQuery {
	topics := make([]ethcommon.Hash, 0, len(sig))
	for _, s := range sig {
		topics = append(topics, s.GetTopic())
	}
	query := eth.FilterQuery{
		FromBlock: startBlock,
		ToBlock:   endBlock,
		Addresses: []ethcommon.Address{contract},
		Topics:    [][]ethcommon.Hash{topics},
	}
	return query
}
