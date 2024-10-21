package chain

import (
	"context"
	"encoding/binary"
	"fmt"
	"math/big"
	"time"

	"github.com/mapprotocol/compass/msg"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethdb/memorydb"
	"github.com/ethereum/go-ethereum/trie"
	"github.com/mapprotocol/compass/internal/constant"
	"github.com/mapprotocol/compass/internal/proof"
	"github.com/mapprotocol/compass/internal/tx"
	"github.com/mapprotocol/compass/mapprotocol"
	"github.com/mapprotocol/compass/pkg/ethclient"
	"github.com/mapprotocol/compass/pkg/util"
	"github.com/pkg/errors"
)

type Oracle struct {
	*CommonSync
}

func NewOracle(cs *CommonSync) *Oracle {
	return &Oracle{
		CommonSync: cs,
	}
}

func (m *Oracle) Sync() error {
	m.Log.Debug("Starting listener...")
	go func() {
		if !m.Cfg.SyncToMap && m.Cfg.Id != m.Cfg.MapChainID {
			time.Sleep(time.Hour * 2400)
			return
		}
		if m.Cfg.Filter {
			err := m.filter()
			if err != nil {
				m.Log.Error("Filter Polling blocks failed", "err", err)
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

func (m *Oracle) sync() error {
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
				m.Log.Debug("Block not ready, will retry", "currentBlock", currentBlock, "latest", latestBlock)
				time.Sleep(constant.BalanceRetryInterval)
				continue
			}

			err = m.oracleHandler(m, currentBlock)
			if err != nil {
				m.Log.Error("Failed to get events for block", "block", currentBlock, "err", err)
				time.Sleep(constant.BlockRetryInterval)
				util.Alarm(context.Background(), fmt.Sprintf("oracle failed, chain=%s, err is %s", m.Cfg.Name, err.Error()))
				continue
			}

			err = m.BlockStore.StoreBlock(currentBlock)
			if err != nil {
				m.Log.Error("Failed to write latest block to blockStore", "block", currentBlock, "err", err)
			}

			currentBlock.Add(currentBlock, big.NewInt(1))
			if latestBlock.Int64()-currentBlock.Int64() <= m.Cfg.BlockConfirmations.Int64() {
				time.Sleep(constant.MessengerInterval)
			}
		}
	}
}

func (m *Oracle) filter() error {
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
			err = m.filterOracle(latestBlock.Uint64())
			if err != nil {
				m.Log.Error("Failed to get events for block", "err", err)
				time.Sleep(constant.BlockRetryInterval)
				util.Alarm(context.Background(), fmt.Sprintf("oracle failed, chain=%s, err is %s", m.Cfg.Name, err.Error()))
				continue
			}

			err = m.BlockStore.StoreBlock(m.Cfg.StartBlock)
			if err != nil {
				m.Log.Error("Filter Failed to write latest block to blockstore", "err", err)
			}

			time.Sleep(constant.MessengerInterval)
		}
	}
}

func DefaultOracleHandler(m *Oracle, currentBlock *big.Int) error {
	m.Log.Debug("Querying block for events", "block", currentBlock)
	query := m.BuildQuery(m.Cfg.LightNode, m.Cfg.Events, currentBlock, currentBlock)
	// querying for logs
	logs, err := m.Conn.Client().FilterLogs(context.Background(), query)
	if err != nil {
		return fmt.Errorf("oracle unable to Filter Logs: %w", err)
	}
	if len(logs) == 0 {
		return nil
	}
	err = log2Oracle(m, logs, currentBlock)
	if err != nil {
		return err
	}

	return nil
}

func log2Oracle(m *Oracle, logs []types.Log, blockNumber *big.Int) error {
	count := 0
	header, err := m.Conn.Client().HeaderByNumber(context.Background(), blockNumber)
	if err != nil {
		return fmt.Errorf("oracle get header failed, err: %w", err)
	}
	receiptHash, err := generateReceipt(m.Conn.Client(), int64(m.Cfg.Id), blockNumber)
	if err != nil {
		return fmt.Errorf("oracle generate receipt failed, err is %w", err)
	}
	if receiptHash != nil {
		header.ReceiptHash = *receiptHash
	}

	m.Log.Info("Find log", "block", blockNumber, "logs", len(logs), "receipt", header.ReceiptHash)
	id := big.NewInt(int64(m.Cfg.Id))
	for _, log := range logs {
		toChainID := uint64(m.Cfg.MapChainID)
		if m.Cfg.Id == m.Cfg.MapChainID {
			toChainID = binary.BigEndian.Uint64(log.Topics[1][len(logs[0].Topics[1])-8:])
			if _, ok := mapprotocol.OnlineChaId[msg.ChainId(toChainID)]; !ok {
				m.Log.Info("Map Oracle Found a log that is not the current task", "blockNumber", log.BlockNumber, "toChainID", toChainID)
				continue
			}
		}

		ret, err := MulSignInfo(0, uint64(m.Cfg.Id), uint64(m.Cfg.Id))
		if err != nil {
			return err
		}
		m.Log.Info("MulSignInfo success", "ret", ret)
		pack, err := mapprotocol.PackAbi.Methods[mapprotocol.MethodOfSolidityPack].Inputs.Pack(header.ReceiptHash, ret.Version, blockNumber, id)
		if err != nil {
			return err
		}

		err = m.Router.Send(msg.NewProposal(m.Cfg.Id, m.Cfg.MapChainID, []interface{}{pack, header.ReceiptHash, blockNumber}, m.MsgCh))
		if err != nil {
			m.Log.Error("Proposal error: failed to route message", "err", err)
			return err
		}
		count++

	}

	err = m.WaitUntilMsgHandled(count)
	if err != nil {
		return err
	}
	return nil
}

func generateReceipt(cli *ethclient.Client, selfId int64, latestBlock *big.Int) (*common.Hash, error) {
	if !exist(selfId, []int64{constant.MerlinChainId, constant.CfxChainId, constant.ZkSyncChainId}) {
		return nil, nil
	}
	txsHash, err := mapprotocol.GetTxsByBn(cli, latestBlock)
	if err != nil {
		return nil, fmt.Errorf("unable to get tx hashes Logs: %w", err)
	}
	receipts, err := tx.GetReceiptsByTxsHash(cli, txsHash)
	if err != nil {
		return nil, fmt.Errorf("unable to get receipts hashes Logs: %w", err)
	}
	tr, _ := trie.New(common.Hash{}, trie.NewDatabase(memorydb.New()))
	tr = proof.DeriveTire(types.Receipts(receipts), tr)
	ret := tr.Hash()
	return &ret, nil
}

func exist(target int64, dst []int64) bool {
	for _, d := range dst {
		if target == d {
			return true
		}
	}
	return false
}
