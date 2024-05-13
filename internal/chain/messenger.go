package chain

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/mapprotocol/compass/mapprotocol"
	"github.com/mapprotocol/compass/msg"
	"math/big"
	"strconv"
	"strings"
	"time"

	"github.com/mapprotocol/compass/internal/constant"
	"github.com/mapprotocol/compass/pkg/util"
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
	m.Log.Debug("Starting listener...")
	go func() {
		if !m.Cfg.SyncToMap && m.Cfg.Id != m.Cfg.MapChainID {
			time.Sleep(time.Hour * 2400)
			return
		}
		if m.Cfg.Filter {
			err := m.filter()
			if err != nil {
				m.Log.Error("Polling blocks failed", "err", err)
			}
			return
		}
		err := m.sync()
		if err != nil {
			m.Log.Error("Polling blocks failed", "err", err)
		}
	}()

	return nil
}

// sync function of Messenger will poll for the latest block and listen the log information of transactions in the block
// Polling begins at the block defined in `m.Cfg.startBlock`. Failed attempts to fetch the latest block or parse
// a block will be retried up to BlockRetryLimit times before continuing to the next block.
// However，an error in synchronizing the log will cause the entire program to block
func (m *Messenger) sync() error {
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
				m.Log.Debug("Block not ready, will retry", "currentBlock", currentBlock, "latest", latestBlock)
				time.Sleep(constant.BalanceRetryInterval)
				continue
			}
			count, err := m.mosHandler(m, currentBlock)
			if err != nil {
				if errors.Is(err, NotVerifyAble) {
					time.Sleep(constant.BalanceRetryInterval)
					continue
				}
				m.Log.Error("Failed to get events for block", "block", currentBlock, "err", err)
				util.Alarm(context.Background(), fmt.Sprintf("mos failed, chain=%s, err is %s", m.Cfg.Name, err.Error()))
				time.Sleep(constant.BlockRetryInterval)
				continue
			}

			_ = m.WaitUntilMsgHandled(count)
			err = m.BlockStore.StoreBlock(currentBlock)
			if err != nil {
				m.Log.Error("Failed to write latest block to blockstore", "block", currentBlock, "err", err)
			}

			currentBlock.Add(currentBlock, big.NewInt(1))
			if latestBlock.Int64()-currentBlock.Int64() <= m.Cfg.BlockConfirmations.Int64() {
				time.Sleep(constant.MessengerInterval)
			}
		}
	}
}

func (m *Messenger) filter() error {
	for {
		select {
		case <-m.Stop:
			return errors.New("polling terminated")
		default:
			data, err := request(fmt.Sprintf("%s/%s", m.Cfg.FilterHost, fmt.Sprintf("%s?chain_id=%d", constant.FilterBlockUrl, m.Cfg.Id)))
			if err != nil {
				m.Log.Error("Unable to get latest block", "err", err)
				time.Sleep(constant.BlockRetryInterval)
				continue
			}
			latestBlock, _ := big.NewInt(0).SetString(data.(string), 10)
			count, err := m.filterMosHandler(latestBlock.Uint64())
			if err != nil {
				if errors.Is(err, NotVerifyAble) {
					time.Sleep(constant.BalanceRetryInterval)
					continue
				}
				m.Log.Error("Filter Failed to get events for block", "err", err)
				util.Alarm(context.Background(), fmt.Sprintf("mos failed, chain=%s, err is %s", m.Cfg.Name, err.Error()))
				time.Sleep(constant.BlockRetryInterval)
				continue
			}

			// hold until all messages are handled
			_ = m.WaitUntilMsgHandled(count)
			err = m.BlockStore.StoreBlock(m.Cfg.StartBlock)
			if err != nil {
				m.Log.Error("Filter Failed to write latest block to blockstore", "err", err)
			}

			time.Sleep(constant.BlockRetryInterval)
		}
	}
}

func defaultMosHandler(m *Messenger, blockNumber *big.Int) (int, error) {
	count := 0
	for idx, addr := range m.Cfg.McsContract {
		query := m.BuildQuery(addr, m.Cfg.Events, blockNumber, blockNumber)
		logs, err := m.Conn.Client().FilterLogs(context.Background(), query)
		if err != nil {
			return 0, fmt.Errorf("unable to Filter Logs: %w", err)
		}

		m.Log.Debug("event", "blockNumber ", blockNumber, " logs ", len(logs))
		for _, log := range logs {
			ele := log
			send, err := log2Msg(m, &ele, idx)
			if err != nil {
				return 0, err
			}
			count += send
		}
	}
	return count, nil
}

func log2Msg(m *Messenger, log *types.Log, idx int) (int, error) {
	var (
		proofType int64
		toChainID uint64
		err       error
	)

	if log.Topics[0].Hex() == constant.TopicsOfSwapInVerified {
		proofType = 3
	} else {
		orderId := log.Data[:32]
		toChainID, _ = strconv.ParseUint(mapprotocol.MapId, 10, 64)
		if m.Cfg.Id == m.Cfg.MapChainID {
			toChainID = binary.BigEndian.Uint64(log.Topics[2][len(log.Topics[2])-8:])
		}
		chainName, ok := mapprotocol.OnlineChaId[msg.ChainId(toChainID)]
		if !ok {
			m.Log.Info("Map Found a log that is not the current task ", "blockNumber", log.BlockNumber, "toChainID", toChainID)
			return 0, nil
		}
		if strings.ToLower(chainName) == "near" {
			proofType = 1
		} else if strings.ToLower(chainName) == "tron" {
			proofType = 3
		} else {
			m.Log.Info("Event found", "txHash", log.TxHash, "logIdx", log.Index, "toChainID", toChainID, "orderId", common.Bytes2Hex(orderId))
			proofType, err = PreSendTx(idx, uint64(m.Cfg.Id), toChainID, big.NewInt(0).SetUint64(log.BlockNumber), orderId)
			if errors.Is(err, OrderExist) {
				m.Log.Info("This txHash order exist", "txHash", log.TxHash)
				return 0, nil
			}
			if errors.Is(err, NotVerifyAble) {
				m.Log.Info("CurrentBlock not verify", "txHash", log.TxHash)
				return 0, err
			}
			m.Log.Info("Event found", "txHash", log.TxHash, "proofType", proofType)
			if err != nil {
				return 0, err
			}
		}
	}

	tmpLog := log
	message, err := m.assembleProof(m, tmpLog, proofType, toChainID)
	if err != nil {
		return 0, err
	}
	message.Idx = idx
	time.Sleep(time.Minute)
	err = m.Router.Send(*message)
	if err != nil {
		m.Log.Error("Subscription error: failed to route message", "err", err)
		return 0, err
	}
	return 1, nil
}
