package eth2

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/mapprotocol/compass/internal/chain"
	"github.com/mapprotocol/compass/internal/constant"
	"github.com/mapprotocol/compass/internal/eth2"
	"github.com/mapprotocol/compass/internal/stream"
	"github.com/mapprotocol/compass/internal/tx"
	"github.com/mapprotocol/compass/mapprotocol"
	"github.com/mapprotocol/compass/msg"
	"github.com/mapprotocol/compass/pkg/util"
	"github.com/pkg/errors"
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
		var err error
		if m.Cfg.Filter {
			err = m.filter()
		} else {
			err = m.sync()
		}
		if err != nil {
			m.Log.Error("Polling blocks failed", "err", err)
		}
	}()

	return nil
}

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
				util.Alarm(context.Background(), fmt.Sprintf("eth2 mos failed, err is %s", err.Error()))
				time.Sleep(constant.BlockRetryInterval)
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
			tmp := log
			send, err := log2Msg(m, &tmp, idx)
			if err != nil {
				return 0, err
			}
			count += send
		}
	}

	return count, nil
}

func (m *Messenger) filter() error {
	for {
		select {
		case <-m.Stop:
			return errors.New("filter polling terminated")
		default:
			latestBlock, err := m.FilterLatestBlock()
			if err != nil {
				m.Log.Error("Unable to get latest block", "err", err)
				time.Sleep(constant.BlockRetryInterval)
				continue
			}
			count, err := m.filterMosHandler(latestBlock.Uint64())
			if err != nil {
				if errors.Is(err, chain.NotVerifyAble) {
					time.Sleep(constant.BalanceRetryInterval)
					continue
				}
				m.Log.Error("Filter Failed to get events for block", "err", err)
				util.Alarm(context.Background(), fmt.Sprintf("filter mos failed, chain=%s, err is %s", m.Cfg.Name, err.Error()))
				time.Sleep(constant.BlockRetryInterval)
				continue
			}

			// hold until all messages are handled
			_ = m.WaitUntilMsgHandled(count)
			err = m.BlockStore.StoreBlock(m.Cfg.StartBlock)
			if err != nil {
				m.Log.Error("Filter Failed to write latest block to blockStore", "err", err)
			}

			time.Sleep(constant.MessengerInterval)
		}
	}
}

func (m *Messenger) filterMosHandler(latestBlock uint64) (int, error) {
	count := 0
	topic := ""
	for idx, ele := range m.Cfg.Events {
		topic += ele.GetTopic().Hex()
		if idx != len(m.Cfg.Events)-1 {
			topic += ","
		}
	}
	data, err := chain.Request(fmt.Sprintf("%s/%s?%s", m.Cfg.FilterHost, constant.FilterUrl,
		fmt.Sprintf("id=%d&project_id=%d&chain_id=%d&topic=%s&limit=1",
			m.Cfg.StartBlock.Int64(), constant.ProjectOfMsger, m.Cfg.Id, topic)))
	if err != nil {
		return 0, err
	}
	listData, err := json.Marshal(data)
	if err != nil {
		return 0, errors.Wrap(err, "marshal resp.Data failed")
	}
	back := stream.MosListResp{}
	err = json.Unmarshal(listData, &back)
	if err != nil {
		return 0, err
	}
	if len(back.List) == 0 {
		time.Sleep(constant.QueryRetryInterval)
		return 0, nil
	}

	for _, ele := range back.List {
		idx := m.Match(ele.ContractAddress)
		if idx == -1 {
			m.Log.Info("Filter Log Address Not Match", "id", ele.Id, "address", ele.ContractAddress)
			//m.Cfg.StartBlock = big.NewInt(ele.Id)
			continue
		}
		if latestBlock-ele.BlockNumber < m.BlockConfirmations.Uint64() {
			m.Log.Debug("Block not ready, will retry", "currentBlock", ele.BlockNumber, "latest", latestBlock)
			time.Sleep(constant.BalanceRetryInterval)
			continue
		}

		split := strings.Split(ele.Topic, ",")
		topics := make([]common.Hash, 0, len(split))
		for _, sp := range split {
			topics = append(topics, common.HexToHash(sp))
		}
		log := &types.Log{
			Address:     common.HexToAddress(ele.ContractAddress),
			Topics:      topics,
			Data:        common.Hex2Bytes(ele.LogData),
			BlockNumber: ele.BlockNumber,
			TxHash:      common.HexToHash(ele.TxHash),
			TxIndex:     ele.TxIndex,
			BlockHash:   common.HexToHash(ele.BlockHash),
			Index:       ele.LogIndex,
		}
		send, err := log2Msg(m, log, idx)
		if err != nil {
			return 0, err
		}
		count += send
		m.Cfg.StartBlock = big.NewInt(ele.Id)
	}

	return count, nil
}

func log2Msg(m *Messenger, log *types.Log, idx int) (int, error) {
	orderId := log.Data[:32]
	method := m.GetMethod(log.Topics[0])
	blockNumber := big.NewInt(0).SetUint64(log.BlockNumber)
	m.Log.Info("Event found", "BlockNumber", log.BlockNumber, "txHash", log.TxHash, "orderId", common.Bytes2Hex(orderId))
	header, err := m.Conn.Client().EthLatestHeaderByNumber(m.Cfg.Endpoint, blockNumber)
	if err != nil {
		return 0, err
	}
	// when syncToMap we need to assemble a tx proof
	txsHash, err := mapprotocol.GetTxsByBn(m.Conn.Client(), blockNumber)
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

	msgPayload := []interface{}{payload, orderId, log.BlockNumber, log.TxHash}
	message := msg.NewSwapWithProof(m.Cfg.Id, m.Cfg.MapChainID, msgPayload, m.MsgCh)
	message.Idx = idx

	err = m.Router.Send(message)
	if err != nil {
		m.Log.Error("Subscription error: failed to route message", "err", err)
	}
	return 1, nil
}
