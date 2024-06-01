package eth2

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"strconv"
	"time"

	ethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/mapprotocol/compass/mapprotocol"

	"github.com/mapprotocol/compass/internal/chain"
	"github.com/mapprotocol/compass/internal/constant"
	"github.com/mapprotocol/compass/internal/eth2"
	"github.com/mapprotocol/compass/internal/tx"
	"github.com/mapprotocol/compass/pkg/util"

	"github.com/mapprotocol/compass/msg"
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
	if !m.Cfg.SyncToMap {
		time.Sleep(time.Hour * 2400)
	}
	var currentBlock = m.Cfg.StartBlock

	for {
		select {
		case <-m.Stop:
			return errors.New("polling terminated")
		default:
			latestBlock, err := m.Conn.LatestBlock()
			if err != nil {
				m.Log.Error("Unable to get latest block", "block", currentBlock, "err", err)
				time.Sleep(constant.RetryLongInterval)
				continue
			}

			if big.NewInt(0).Sub(latestBlock, currentBlock).Cmp(m.BlockConfirmations) == -1 {
				m.Log.Debug("Block not ready, will retry", "target", currentBlock, "latest", latestBlock)
				time.Sleep(constant.BalanceRetryInterval)
				continue
			}
			count, err := m.getEventsForBlock(currentBlock)
			if err != nil {
				m.Log.Error("Failed to get events for block", "block", currentBlock, "err", err)
				time.Sleep(constant.BlockRetryInterval)
				util.Alarm(context.Background(), fmt.Sprintf("eth2 mos failed, err is %s", err.Error()))
				continue
			}

			// hold until all messages are handled
			_ = m.WaitUntilMsgHandled(count)

			// Write to block store. Not a critical operation, no need to retry
			err = m.BlockStore.StoreBlock(currentBlock)
			if err != nil {
				m.Log.Error("Failed to write latest block to blockstore", "block", currentBlock, "err", err)
			}

			// Goto next block and reset retry counter
			currentBlock.Add(currentBlock, big.NewInt(1))
			if latestBlock.Int64()-currentBlock.Int64() <= m.Cfg.BlockConfirmations.Int64() {
				time.Sleep(time.Second * 10)
			} else {
				time.Sleep(time.Millisecond * 20)
			}
		}
	}
}

// getEventsForBlock looks for the deposit event in the latest block
func (m *Messenger) getEventsForBlock(latestBlock *big.Int) (int, error) {
	count := 0
	for idx, addr := range m.Cfg.McsContract {
		query := m.BuildQuery(addr, m.Cfg.Events, latestBlock, latestBlock)
		logs, err := m.Conn.Client().FilterLogs(context.Background(), query)
		if err != nil {
			return 0, fmt.Errorf("unable to Filter Logs: %w", err)
		}

		for _, log := range logs {
			// evm event to msg
			var message msg.Message
			orderId := log.Data[:32]
			toChainID, _ := strconv.ParseUint(mapprotocol.MapId, 10, 64)
			if _, ok := mapprotocol.OnlineChaId[msg.ChainId(toChainID)]; !ok {
				m.Log.Info("Map Found a log that is not the current task ", "blockNumber", log.BlockNumber, "toChainID", toChainID)
				continue
			}
			m.Log.Info("Event found", "BlockNumber", log.BlockNumber, "txHash", log.TxHash, "orderId", ethcommon.Bytes2Hex(orderId))
			if err != nil {
				return 0, err
			}
			method := m.GetMethod(log.Topics[0])
			header, err := m.Conn.Client().EthLatestHeaderByNumber(m.Cfg.Endpoint, latestBlock)
			if err != nil {
				return 0, err
			}
			// when syncToMap we need to assemble a tx proof
			txsHash, err := mapprotocol.GetTxsByBn(m.Conn.Client(), latestBlock)
			if err != nil {
				return 0, fmt.Errorf("unable to get tx hashes Logs: %w", err)
			}
			receipts, err := tx.GetReceiptsByTxsHash(m.Conn.Client(), txsHash)
			if err != nil {
				return 0, fmt.Errorf("unable to get receipts hashes Logs: %w", err)
			}
			payload, err := eth2.AssembleProof(*eth2.ConvertHeader(header), log, receipts, method, m.Cfg.Id, constant.ProofTypeOfOracle)
			if err != nil {
				return 0, fmt.Errorf("unable to Parse Log: %w", err)
			}

			msgPayload := []interface{}{payload, orderId, latestBlock.Uint64(), log.TxHash}
			message = msg.NewSwapWithProof(m.Cfg.Id, m.Cfg.MapChainID, msgPayload, m.MsgCh)
			message.Idx = idx

			err = m.Router.Send(message)
			if err != nil {
				m.Log.Error("Subscription error: failed to route message", "err", err)
			}
			count++
		}
	}

	return count, nil
}
