package ethereum

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/mapprotocol/compass/internal/chain"
	"github.com/mapprotocol/compass/internal/constant"
	"github.com/mapprotocol/compass/pkg/util"

	"github.com/mapprotocol/compass/mapprotocol"

	"github.com/mapprotocol/compass/msg"

	eth "github.com/ethereum/go-ethereum"
	ethcommon "github.com/ethereum/go-ethereum/common"
	utils "github.com/mapprotocol/compass/shared/ethereum"
)

type Messenger struct {
	*chain.CommonSync
}

func NewMessenger(cs *chain.CommonSync) *Messenger {
	return &Messenger{
		CommonSync: cs,
	}
}

func (m *Messenger) Sync() error {
	m.Log.Debug("Starting listener...")
	go func() {
		err := m.sync()
		if err != nil {
			m.Log.Error("Polling blocks failed", "err", err)
		}
	}()

	return nil
}

// sync function of Messenger will poll for the latest block and listen the log information of transactions in the block
// Polling begins at the block defined in `m.Cfg.StartBlock`. Failed attempts to fetch the latest block or parse
// a block will be retried up to BlockRetryLimit times before continuing to the next block.
// However，an error in synchronizing the log will cause the entire program to block
func (m *Messenger) sync() error {
	var currentBlock = m.Cfg.StartBlock

	if m.Cfg.SyncToMap {
		// when listen to map there must be a 20 block confirmation at least
		big20 := big.NewInt(20)
		if m.BlockConfirmations.Cmp(big20) == -1 {
			m.BlockConfirmations = big20
		}
	}

	var retry = constant.BlockRetryLimit
	for {
		select {
		case <-m.Stop:
			return errors.New("polling terminated")
		default:
			// No more retries, goto next block
			if retry == 0 {
				m.Log.Error("Polling failed, retries exceeded")
				m.SysErr <- constant.ErrFatalPolling
				return nil
			}

			latestBlock, err := m.Conn.LatestBlock()
			if err != nil {
				m.Log.Error("Unable to get latest block", "block", currentBlock, "err", err)
				time.Sleep(constant.RetryLongInterval)
				continue
			}

			if m.Metrics != nil {
				m.Metrics.LatestKnownBlock.Set(float64(latestBlock.Int64()))
			}

			if m.Cfg.SyncToMap {
				left, right, err := mapprotocol.Get2MapVerifyRange(m.Cfg.Id)
				if err != nil {
					m.Log.Warn("Get2MapVerifyRange failed", "err", err)
				}
				if right != nil && right.Uint64() != 0 && right.Cmp(currentBlock) == -1 {
					m.Log.Info("currentBlock less than max verify range", "currentBlock", currentBlock, "maxVerify", right)
					time.Sleep(time.Minute)
					continue
				}

				if left != nil && left.Uint64() != 0 && left.Cmp(currentBlock) == 1 {
					currentBlock = left
					m.Log.Info("min verify range greater than currentBlock, set current to left", "currentBlock", currentBlock, "minVerify", left)
				}
			}

			// Sleep if the difference is less than BlockDelay; (latest - current) < BlockDelay
			if big.NewInt(0).Sub(latestBlock, currentBlock).Cmp(m.BlockConfirmations) == -1 {
				m.Log.Debug("Block not ready, will retry", "target", currentBlock, "latest", latestBlock)
				time.Sleep(constant.BlockRetryInterval)
				continue
			}
			// messager
			// Parse out events
			count, err := m.getEventsForBlock(currentBlock)
			if err != nil {
				m.Log.Error("Failed to get events for block", "block", currentBlock, "err", err)
				time.Sleep(constant.BlockRetryInterval)
				util.Alarm(context.Background(), fmt.Sprintf("map mos failed, err is %s", err.Error()))
				continue
			}

			// hold until all messages are handled
			_ = m.WaitUntilMsgHandled(count)

			// Write to block store. Not a critical operation, no need to retry
			err = m.BlockStore.StoreBlock(currentBlock)
			if err != nil {
				m.Log.Error("Failed to write latest block to blockstore", "block", currentBlock, "err", err)
			}
			if m.Metrics != nil {
				m.Metrics.BlocksProcessed.Inc()
				m.Metrics.LatestProcessedBlock.Set(float64(latestBlock.Int64()))
			}

			m.LatestBlock.Height = big.NewInt(0).Set(latestBlock)
			m.LatestBlock.LastUpdated = time.Now()

			// Goto next block and reset retry counter
			currentBlock.Add(currentBlock, big.NewInt(1))
			retry = constant.BlockRetryLimit
		}
	}
}

// getEventsForBlock looks for the deposit event in the latest block
func (m *Messenger) getEventsForBlock(latestBlock *big.Int) (int, error) {
	m.Log.Debug("Querying block for events", "block", latestBlock)
	query := m.buildQuery(m.Cfg.McsContract, m.Cfg.Events, latestBlock, latestBlock)
	// querying for logs
	logs, err := m.Conn.Client().FilterLogs(context.Background(), query)
	if err != nil {
		return 0, fmt.Errorf("unable to Filter Logs: %w", err)
	}

	m.Log.Debug("event", "latestBlock ", latestBlock, " logs ", len(logs))
	count := 0
	// read through the log events and handle their deposit event if handler is recognized
	for _, log := range logs {
		// evm event to msg
		var message msg.Message
		// getOrderId
		orderId := log.Data[:32]
		method := m.GetMethod(log.Topics[0])
		if m.Cfg.SyncToMap {
			// when syncToMap we need to assemble a tx proof
			txsHash, err := mapprotocol.GetTransactionsHashByBlockNumber(m.Conn.Client(), latestBlock)
			if err != nil {
				return 0, fmt.Errorf("unable to get tx hashes Logs: %w", err)
			}
			receipts, err := mapprotocol.GetReceiptsByTxsHash(m.Conn.Client(), txsHash)
			if err != nil {
				return 0, fmt.Errorf("unable to get receipts hashes Logs: %w", err)
			}
			payload, err := utils.ParseEthLogIntoSwapWithProofArgs(log, m.Cfg.McsContract, receipts, method, m.Cfg.Id, m.Cfg.MapChainID)
			if err != nil {
				return 0, fmt.Errorf("unable to Parse Log: %w", err)
			}

			msgPayload := []interface{}{payload, orderId, latestBlock.Uint64(), log.TxHash}
			message = msg.NewSwapWithProof(m.Cfg.Id, m.Cfg.MapChainID, msgPayload, m.MsgCh)
		} else if m.Cfg.Id == m.Cfg.MapChainID {
			// when listen from map we also need to assemble a tx prove in a different way
			header, err := m.Conn.Client().MAPHeaderByNumber(context.Background(), latestBlock)
			if err != nil {
				return 0, fmt.Errorf("unable to query header Logs: %w", err)
			}
			txsHash, err := mapprotocol.GetMapTransactionsHashByBlockNumber(m.Conn.Client(), latestBlock)
			if err != nil {
				return 0, fmt.Errorf("idSame unable to get tx hashes Logs: %w", err)
			}
			receipts, err := mapprotocol.GetReceiptsByTxsHash(m.Conn.Client(), txsHash)
			if err != nil {
				return 0, fmt.Errorf("unable to get receipts hashes Logs: %w", err)
			}
			//
			remainder := big.NewInt(0).Mod(latestBlock, big.NewInt(mapprotocol.EpochOfMap))
			if remainder.Cmp(mapprotocol.Big0) == 0 {
				lr, err := mapprotocol.GetLastReceipt(m.Conn.Client(), latestBlock)
				if err != nil {
					return 0, fmt.Errorf("unable to get last receipts in epoch last %w", err)
				}
				receipts = append(receipts, lr)
			}

			toChainID, payload, err := utils.AssembleMapProof(m.Conn.Client(), log, receipts, header, m.Cfg.MapChainID, method)
			if err != nil {
				return 0, fmt.Errorf("unable to Parse Log: %w", err)
			}

			if _, ok := mapprotocol.OnlineChaId[msg.ChainId(toChainID)]; !ok {
				m.Log.Info("Found a log that is not the current task ", "toChainID", toChainID)
				continue
			}

			if fn, ok := mapprotocol.Map2OtherVerifyRange[msg.ChainId(toChainID)]; ok {
				left, right, err := fn()
				if err != nil {
					m.Log.Warn("map chain Get2OtherVerifyRange failed", "err", err)
				}
				if left != nil && left.Uint64() != 0 && left.Cmp(latestBlock) == 1 {
					m.Log.Info("min verify range greater than currentBlock, skip ", "currentBlock", latestBlock, "minVerify", left)
					continue
				}
				if right != nil && right.Uint64() != 0 && right.Cmp(latestBlock) == -1 {
					m.Log.Info("currentBlock less than max verify range", "currentBlock", latestBlock, "maxVerify", right)
					time.Sleep(time.Minute * 3)
				}
			}

			msgPayload := []interface{}{payload, orderId, latestBlock.Uint64(), log.TxHash, method}
			message = msg.NewSwapWithMapProof(m.Cfg.MapChainID, msg.ChainId(toChainID), msgPayload, m.MsgCh)
		}

		m.Log.Info("Event found", "BlockNumber", log.BlockNumber, "txHash", log.TxHash, "logIdx", log.Index, "orderId", ethcommon.Bytes2Hex(orderId))
		err = m.Router.Send(message)
		if err != nil {
			m.Log.Error("subscription error: failed to route message", "err", err)
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
