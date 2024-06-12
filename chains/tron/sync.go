package tron

import (
	"context"
	"fmt"
	"math/big"
	"strconv"
	"time"

	eth "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethdb/memorydb"
	"github.com/ethereum/go-ethereum/trie"
	"github.com/mapprotocol/compass/core"
	"github.com/mapprotocol/compass/internal/chain"
	"github.com/mapprotocol/compass/internal/constant"
	"github.com/mapprotocol/compass/internal/proof"
	"github.com/mapprotocol/compass/internal/tx"
	"github.com/mapprotocol/compass/mapprotocol"
	"github.com/mapprotocol/compass/msg"
	"github.com/mapprotocol/compass/pkg/ethclient"
	"github.com/mapprotocol/compass/pkg/util"
	"github.com/pkg/errors"

	"github.com/ChainSafe/log15"
	"github.com/mapprotocol/compass/chains"
)

type Maintainer struct {
	Log log15.Logger
}

func NewMaintainer(log log15.Logger) *Maintainer {
	return &Maintainer{Log: log}
}

func (m *Maintainer) Sync() error {
	m.Log.Debug("Starting listener...")
	go func() {
		time.Sleep(time.Hour * 2400)
	}()

	return nil
}

func (m *Maintainer) SetRouter(r chains.Router) {

}

type Handler func(*sync, *big.Int) (int, error)

type sync struct {
	*chain.CommonSync
	handler Handler
	conn    core.Connection
}

func newSync(cs *chain.CommonSync, handler Handler, conn core.Connection) *sync {
	return &sync{CommonSync: cs, handler: handler, conn: conn}
}

func (m *sync) Sync() error {
	m.Log.Info("Starting listener...")
	if !m.Cfg.SyncToMap {
		time.Sleep(time.Hour * 2400)
		return nil
	}
	var currentBlock = m.Cfg.StartBlock

	select {
	case <-m.Stop:
		return errors.New("polling terminated")
	default:
		for {
			latestBlock, err := m.conn.LatestBlock()
			if err != nil {
				m.Log.Error("Unable to get latest block", "err", err)
				time.Sleep(constant.RetryLongInterval)
				continue
			}

			if big.NewInt(0).Sub(latestBlock, currentBlock).Cmp(m.BlockConfirmations) == -1 {
				m.Log.Debug("Block not ready, will retry", "currentBlock", currentBlock, "latest", latestBlock)
				time.Sleep(constant.BalanceRetryInterval)
				continue
			}

			count, err := m.handler(m, currentBlock)
			if err != nil {
				m.Log.Error("Failed to get events for block", "block", currentBlock, "err", err)
				time.Sleep(constant.BlockRetryInterval)
				util.Alarm(context.Background(), fmt.Sprintf("mos failed, chain=%s, err is %s", m.Cfg.Name, err.Error()))
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

func messengerHandler(m *sync, current *big.Int) (int, error) {
	m.Log.Debug("Querying block for events", "block", current)
	count := 0
	for idx, addr := range m.Cfg.TronContract {
		query := m.BuildQuery(addr, m.Cfg.Events[0:0], current, current)
		query = eth.FilterQuery{
			FromBlock: current,
			ToBlock:   current,
		}
		logs, err := m.Conn.Client().FilterLogs(context.Background(), query)
		if err != nil {
			return 0, fmt.Errorf("unable to Filter Logs: %w", err)
		}

		if len(logs) == 0 {
			continue
		}

		key := strconv.FormatUint(uint64(m.Cfg.Id), 10) + "_" + current.String()
		for _, l := range logs {
			if !existTopic(l.Topics[0], m.Cfg.Events) {
				m.Log.Debug("ignore log, because topics not match", "blockNumber", l.BlockNumber, "logTopic", l.Topics[0])
				continue
			}
			orderId := l.Data[:32]
			var (
				message  msg.Message
				receipts []*types.Receipt
			)
			if v, ok := proof.CacheReceipt[key]; ok {
				receipts = v
				m.Log.Info("use cache receipt", "latestBlock ", current, "txHash", l.TxHash)
			} else {
				txsHash, err := getTxsByBN(m.Conn.Client(), current)
				if err != nil {
					return 0, fmt.Errorf("unable to get tx hashes Logs: %w", err)
				}
				receipts, err = tx.GetReceiptsByTxsHash(m.Conn.Client(), txsHash)
				if err != nil {
					return 0, fmt.Errorf("unable to get receipts hashes Logs: %w", err)
				}
				proof.CacheReceipt[key] = receipts
			}
			if l.Topics[0].Hex() == constant.TopicsOfSwapInVerified {
				logIdx := 0
				for i, ele := range receipts[l.TxIndex].Logs {
					if ele.Index != l.Index {
						continue
					}
					logIdx = i
				}

				m.Log.Info("Event found SwapInVerified", "block", current, "txHash", l.TxHash, "idx", l.Index,
					"logIdx", logIdx, "txIdx", l.TxIndex, "all", len(receipts[l.TxIndex].Logs))
				data, err := mapprotocol.Mcs.Events[mapprotocol.EventOfSwapInVerified].Inputs.UnpackValues(l.Data)
				if err != nil {
					return 0, errors.Wrap(err, "swapIn unpackData failed")
				}

				input, _ := mapprotocol.Mcs.Pack(mapprotocol.MtdOfSwapInVerifiedWithIndex, data[0].([]byte), big.NewInt(int64(logIdx)))
				msgPayload := []interface{}{input, orderId, l.BlockNumber, l.TxHash, mapprotocol.MtdOfSwapInVerifiedWithIndex}
				message = msg.NewSwapWithMapProof(m.Cfg.MapChainID, m.Cfg.Id, msgPayload, m.MsgCh)
			} else {

				method := m.GetMethod(l.Topics[0])
				toChainID, _ := strconv.ParseUint(mapprotocol.MapId, 10, 64)
				m.Log.Info("Event found", "block", current, "txHash", l.TxHash, "logIdx", l.Index, "orderId", common.Bytes2Hex(orderId))
				proofType, err := chain.PreSendTx(idx, uint64(m.Cfg.Id), toChainID, current, orderId)
				if errors.Is(err, chain.OrderExist) {
					m.Log.Info("This orderId exist", "block", current, "txHash", l.TxHash, "orderId", common.Bytes2Hex(orderId))
					continue
				}
				if err != nil {
					return 0, err
				}

				tmp := l
				input, err := assembleProof(&tmp, receipts, method, m.Cfg.Id, proofType)
				if err != nil {
					return 0, err
				}

				message = msg.NewSwapWithProof(m.Cfg.Id, m.Cfg.MapChainID, []interface{}{input, orderId, l.BlockNumber, l.TxHash}, m.MsgCh)
			}
			err = m.Router.Send(message)
			if err != nil {
				m.Log.Error("subscription error: failed to route message", "err", err)
				return 0, nil
			}
			count++
		}
	}

	return count, nil
}

func existTopic(target common.Hash, dst []constant.EventSig) bool {
	for _, d := range dst {
		if target == d.GetTopic() {
			return true
		}
	}
	return false
}

func oracleHandler(m *sync, latestBlock *big.Int) (int, error) {
	query := m.BuildQuery(m.Cfg.OracleNode, m.Cfg.Events[:1], latestBlock, latestBlock)
	logs, err := m.Conn.Client().FilterLogs(context.Background(), query)
	if err != nil {
		return 0, fmt.Errorf("sync unable to Filter Logs: %w", err)
	}
	if len(logs) == 0 {
		return 0, nil
	}
	m.Log.Info("Find log", "block", latestBlock, "log", len(logs))
	txsHash, err := getTxsByBN(m.Conn.Client(), latestBlock)
	if err != nil {
		return 0, fmt.Errorf("unable to get tx hashes Logs: %w", err)
	}
	receipts, err := tx.GetReceiptsByTxsHash(m.Conn.Client(), txsHash)
	if err != nil {
		return 0, fmt.Errorf("unable to get receipts hashes Logs: %w", err)
	}
	tr, _ := trie.New(common.Hash{}, trie.NewDatabase(memorydb.New()))
	tr = proof.DeriveTire(types.Receipts(receipts), tr)
	m.Log.Info("oracle tron receipt", "blockNumber", latestBlock, "hash", tr.Hash())
	receiptHash := tr.Hash()
	input, err := mapprotocol.OracleAbi.Methods[mapprotocol.MethodOfPropose].Inputs.Pack(latestBlock, receiptHash)

	id := big.NewInt(0).SetUint64(uint64(m.Cfg.Id))
	message := msg.NewSyncToMap(m.Cfg.Id, m.Cfg.MapChainID, []interface{}{id, input}, m.MsgCh)
	err = m.Router.Send(message)
	if err != nil {
		m.Log.Error("subscription error: failed to route message", "err", err)
		return 0, nil
	}

	return 1, nil
}

func getTxsByBN(conn *ethclient.Client, number *big.Int) ([]common.Hash, error) {
	block, err := conn.TronBlockByNumber(context.Background(), number)
	if err != nil {
		return nil, err
	}

	txs := make([]common.Hash, 0, len(block.Transactions))
	for _, tmp := range block.Transactions {
		ele := common.HexToHash(tmp.Hash)
		txs = append(txs, ele)
	}
	return txs, nil
}

func returnEnergy(conn *Connection, cs *chain.CommonSync, cfg *Config) {
	//msgCh := make(chan struct{})
	for {
		acc, err := conn.cli.GetAccountResource(cs.Cfg.From)
		if err != nil {
			cs.Log.Error("Return energy, GetAccountResource failed", "err", err)
			continue
		}
		overage := acc.EnergyLimit - acc.EnergyUsed
		cs.Log.Info("Return energy, account energy detail", "account", cs.Cfg.From, "all", acc.EnergyLimit, "used", acc.EnergyUsed)
		if acc.EnergyLimit == 0 {
			cs.Log.Info("Return energy, energy is zero, maybe not rent")
			continue
		}
		if overage > mcsEnergy {
			cs.Log.Info("Return energy, energy gather than once mcs, don return")
			continue
		}
		account, err := conn.cli.GetAccount(cs.Cfg.From)
		if err != nil {
			cs.Log.Error("Return energy, GetAccount failed", "err", err)
			continue
		}
		balance, _ := big.NewFloat(0).Quo(big.NewFloat(0).SetInt64(account.Balance), wei).Float64()
		cs.Log.Info("Return energy, will Return, account bal detail", "account", cs.Cfg.From, "trx", balance)
		// return
		//input, err := mapprotocol.TronAbi.Pack("returnResource", cfg.EthFrom, big.NewInt(162932000000), big.NewInt(1))
		//if err != nil {
		//	cs.Log.Error("Return energy, GetAccount failed", "err", err)
		//	continue
		//}
		cs.Log.Info("Return energy, send msg")
		//message := msg.NewReturnEnergy(cs.Cfg.Id, cs.Cfg.Id, []interface{}{input}, msgCh)
		//err = cs.Router.Send(message)
		//if err != nil {
		//	cs.Log.Error("Subscription error: failed to route message", "err", err)
		//	continue
		//}
		//<-msgCh
		//cs.Log.Info("Return energy Success")
		time.Sleep(time.Minute)
	}
}
