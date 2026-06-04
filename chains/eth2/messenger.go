package eth2

import (
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/mapprotocol/compass/internal/chain"
	"github.com/mapprotocol/compass/internal/constant"
	"github.com/mapprotocol/compass/internal/eth2"
	"github.com/mapprotocol/compass/internal/mapprotocol"
	"github.com/mapprotocol/compass/internal/tx"
	"github.com/mapprotocol/compass/pkg/msg"
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
				time.Sleep(constant.QueryRetryInterval)
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
	opts := chain.DefaultFilterRunnerOptions()
	opts.SkipMissingField = false
	return (&chain.FilterRunner{
		Sync:      m.CommonSync,
		Client:    m.FilterClient(),
		Processor: m,
		Options:   opts,
	}).Run()
}

func (m *Messenger) HandleFilterBlock(latestBlock uint64) (int, uint64, error) {
	return m.filterMosHandler(latestBlock)
}

func (m *Messenger) filterMosHandler(latestBlock uint64) (int, uint64, error) {
	count := 0
	progressBlock := uint64(0)
	topic := chain.BuildFilterTopic(m.Cfg.Events)
	back, err := m.ListMosLogs(constant.ProjectOfMsger, topic, 1)
	if err != nil {
		return 0, progressBlock, err
	}
	if len(back.List) == 0 {
		time.Sleep(constant.QueryRetryInterval)
		return 0, latestBlock, nil
	}

	for _, ele := range back.List {
		progressBlock = ele.BlockNumber
		idx := m.Match(ele.ContractAddress)
		if idx == -1 {
			m.Log.Info("Filter Log Address Not Match", "id", ele.Id, "address", ele.ContractAddress)
			m.Cfg.StartBlock = big.NewInt(ele.Id)
			continue
		}
		if latestBlock-ele.BlockNumber < m.BlockConfirmations.Uint64() {
			m.Log.Debug("Block not ready, will retry", "currentBlock", ele.BlockNumber, "latest", latestBlock)
			time.Sleep(constant.BalanceRetryInterval)
			continue
		}

		log := chain.MosRespToEthLog(ele)

		send, err := log2Msg(m, log, idx)
		if err != nil {
			return 0, progressBlock, err
		}
		count += send
		m.Cfg.StartBlock = big.NewInt(ele.Id)
	}

	return count, progressBlock, nil
}

func log2Msg(m *Messenger, log *types.Log, idx int) (int, error) {
	orderId := log.Topics[1]
	method := m.GetMethod(log.Topics[0])
	blockNumber := big.NewInt(0).SetUint64(log.BlockNumber)

	prepared, err := chain.NewMessageGate(m.CommonSync).Prepare(log, chain.MessageGateOptions{
		Idx:         idx,
		ToChainID:   uint64(m.Cfg.MapChainID),
		OrderID:     orderId,
		MapChainLog: false,
		DoPreSend:   true,
		RequireSign: true,
		LogPrefix:   "Eth Msger",
	})
	if errors.Is(err, chain.OrderIgnored) || errors.Is(err, chain.OrderExist) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	log = prepared.Log

	header, err := m.Conn.Client().EthLatestHeaderByNumber(m.Cfg.Endpoint, blockNumber)
	if err != nil {
		return 0, err
	}
	txsHash, err := mapprotocol.GetTxsByBn(m.Conn.Client(), blockNumber)
	if err != nil {
		return 0, fmt.Errorf("unable to get tx hashes Logs: %w", err)
	}
	receipts, err := tx.GetReceiptsByTxsHash(m.Conn.Client(), txsHash)
	if err != nil {
		return 0, fmt.Errorf("unable to get receipts hashes Logs: %w", err)
	}

	m.Log.Info("Event found", "txHash", log.TxHash, "orderId", orderId, "method", method, "proofType", prepared.ProofType)
	payload, err := eth2.AssembleProof(*eth2.ConvertHeader(header), log, receipts, method, m.Cfg.Id, prepared.ProofType, prepared.Sign)
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
