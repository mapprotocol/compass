package tron

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/mapprotocol/compass/internal/mapprotocol"
	"github.com/mapprotocol/compass/pkg/msg"
	"math/big"
	"strconv"
	"strings"
	"time"

	"github.com/ChainSafe/log15"
	eth "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethdb/memorydb"
	"github.com/ethereum/go-ethereum/trie"
	"github.com/mapprotocol/compass/core"
	"github.com/mapprotocol/compass/internal/chain"
	"github.com/mapprotocol/compass/internal/constant"
	"github.com/mapprotocol/compass/internal/proof"
	"github.com/mapprotocol/compass/internal/stream"
	"github.com/mapprotocol/compass/internal/tx"
	"github.com/mapprotocol/compass/pkg/ethclient"
	"github.com/mapprotocol/compass/pkg/util"
	"github.com/pkg/errors"
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

func (m *Maintainer) SetRouter(r core.Router) {

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
	go func() {
		err := m.sync()
		if err != nil {
			m.Log.Error("Polling blocks failed", "err", err)
			panic(err)
		}
	}()
	return nil
}

func (m *sync) sync() error {
	if m.Cfg.Filter {
		err := m.filter()
		if err != nil {
			m.Log.Error("Polling blocks failed", "err", err)
			return err
		}
		return nil
	}

	var currentBlock = m.Cfg.StartBlock
	for {
		select {
		case <-m.Stop:
			return errors.New("polling terminated")
		default:
			latestBlock, err := m.conn.LatestBlock()
			if err != nil {
				m.Log.Error("Unable to get latest block", "err", err)
				time.Sleep(constant.QueryRetryInterval)
				continue
			}

			if big.NewInt(0).Sub(latestBlock, currentBlock).Cmp(m.BlockConfirmations) == -1 {
				m.Log.Debug("Block not ready, will retry", "currentBlock", currentBlock, "latest", latestBlock)
				time.Sleep(constant.BlockRetryInterval)
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

func (m *sync) filter() error {
	for {
		select {
		case <-m.Stop:
			return errors.New("polling terminated")
		default:
			latestBlock, err := m.FilterLatestBlock()
			if err != nil {
				m.Log.Error("Unable to get latest block", "err", err)
				time.Sleep(constant.QueryRetryInterval)
				continue
			}

			count, err := m.handler(m, latestBlock)
			if m.Cfg.SkipError && errors.Is(err, chain.NotVerifyAble) {
				m.Log.Info("Block not verify, will ignore", "startBlock", m.Cfg.StartBlock)
				m.Cfg.StartBlock = m.Cfg.StartBlock.Add(m.Cfg.StartBlock, big.NewInt(1))
				err = m.BlockStore.StoreBlock(m.Cfg.StartBlock)
				continue
			}
			if err != nil {
				m.Log.Error("Filter Failed to get events for block", "err", err)
				if errors.Is(err, chain.NotVerifyAble) {
					time.Sleep(constant.BlockRetryInterval)
					continue
				}
				util.Alarm(context.Background(), fmt.Sprintf("filter mos failed, chain=%s, err is %s", m.Cfg.Name, err.Error()))
				time.Sleep(constant.BlockRetryInterval)
				continue
			}

			_ = m.WaitUntilMsgHandled(count)
			err = m.BlockStore.StoreBlock(m.Cfg.StartBlock)
			if err != nil {
				m.Log.Error("Filter Failed to write latest block to blockStore", "err", err)
			}

			time.Sleep(constant.MessengerInterval)
		}
	}
}

func filterMos(m *sync, latestBlock *big.Int) (int, error) {
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
		return 0, nil
	}

	for _, ele := range back.List {
		idx := m.Match(ele.ContractAddress)
		if idx == -1 {
			m.Log.Info("Filter Log Address Not Match", "id", ele.Id, "address", ele.ContractAddress)
			m.Cfg.StartBlock = big.NewInt(ele.Id)
			continue
		}
		if latestBlock.Uint64()-ele.BlockNumber < m.BlockConfirmations.Uint64() {
			m.Log.Info("Block not ready, will retry", "currentBlock", ele.BlockNumber, "latest", latestBlock)
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

func messenger(m *sync, current *big.Int) (int, error) {
	count := 0
	for idx, addr := range m.Cfg.McsContract {
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
		for _, l := range logs {
			tmp := l
			stage, err := log2Msg(m, &tmp, idx)
			if err != nil {
				return 0, err
			}
			count += stage
		}
	}

	return count, nil
}

func log2Msg(m *sync, l *types.Log, idx int) (int, error) {
	if !existTopic(l.Topics[0], m.Cfg.Events) {
		m.Log.Info("ignore log, because topics not match", "blockNumber", l.BlockNumber, "logTopic", l.Topics[0])
		return 0, nil
	}

	var (
		orderId32 [32]byte
		message   msg.Message
		orderId   = l.Topics[1]
		receipts  []*types.Receipt
		current   = big.NewInt(0).SetUint64(l.BlockNumber)
		key       = strconv.FormatUint(uint64(m.Cfg.Id), 10) + "_" + current.String()
	)
	for i, v := range orderId {
		orderId32[i] = v
	}
	if v, ok := proof.CacheReceipt.Get(key); ok {
		receipts = v.([]*types.Receipt)
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
		proof.CacheReceipt.Add(key, receipts)
	}

	method := m.GetMethod(l.Topics[0])
	m.Log.Info("Event found", "txHash", l.TxHash, "logIdx", l.Index, "orderId", orderId, "cIdx", idx)
	proofType, err := chain.PreSendTx(idx, uint64(m.Cfg.Id), uint64(m.Cfg.MapChainID), current, orderId.Bytes())
	if errors.Is(err, chain.OrderExist) {
		m.Log.Info("This orderId exist", "txHash", l.TxHash, "orderId", orderId)
		return 0, nil
	}
	if err != nil {
		return 0, err
	}

	tmp := l
	input, err := assembleProof(tmp, receipts, method, m.Cfg.Id, m.Cfg.MapChainID, proofType, orderId32)
	if err != nil {
		return 0, err
	}
	message = msg.NewSwapWithProof(m.Cfg.Id, m.Cfg.MapChainID, []interface{}{input, orderId32, l.BlockNumber, l.TxHash}, m.MsgCh)
	err = m.Router.Send(message)
	if err != nil {
		m.Log.Error("subscription error: failed to route message", "err", err)
		return 0, nil
	}
	return 1, nil
}

func existTopic(target common.Hash, dst []constant.EventSig) bool {
	for _, d := range dst {
		if target == d.GetTopic() {
			return true
		}
	}
	return false
}

func filterOracle(m *sync, latestBlock *big.Int) (int, error) {
	topic := ""
	for idx, ele := range m.Cfg.Events {
		topic += ele.GetTopic().Hex()
		if idx != len(m.Cfg.Events)-1 {
			topic += ","
		}
	}

	tmp := int64(0)
	defer func() {
		if tmp == 0 {
			return
		}
		if tmp > m.Cfg.StartBlock.Int64() {
			m.Cfg.StartBlock = big.NewInt(tmp)
		}
	}()

	data, err := chain.Request(fmt.Sprintf("%s/%s?%s", m.Cfg.FilterHost, constant.FilterUrl,
		fmt.Sprintf("id=%d&project_id=%d&chain_id=%d&topic=%s&limit=1",
			m.Cfg.StartBlock.Int64(), constant.ProjectOfOracle, m.Cfg.Id, topic)))

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
		return 0, nil
	}

	for _, ele := range back.List {
		if m.Cfg.OracleNode.Hex() != ele.ContractAddress {
			m.Log.Info("Filter Log Address Not Match", "id", ele.Id, "address", ele.ContractAddress)
			tmp = ele.Id
			continue
		}

		split := strings.Split(ele.Topic, ",")
		topics := make([]common.Hash, 0, len(split))
		for _, sp := range split {
			topics = append(topics, common.HexToHash(sp))
		}
		log := types.Log{
			Address:     common.HexToAddress(ele.ContractAddress),
			Topics:      topics,
			Data:        common.Hex2Bytes(ele.LogData),
			BlockNumber: ele.BlockNumber,
			TxHash:      common.HexToHash(ele.TxHash),
			TxIndex:     ele.TxIndex,
			BlockHash:   common.HexToHash(ele.BlockHash),
			Index:       ele.LogIndex,
		}
		_, err = log2Oracle(m, &log)
		if err != nil {
			return 0, err
		}
		tmp = ele.Id
	}

	return 1, nil
}

func oracle(m *sync, latestBlock *big.Int) (int, error) {
	query := m.BuildQuery(m.Cfg.OracleNode, m.Cfg.Events[:1], latestBlock, latestBlock)
	logs, err := m.Conn.Client().FilterLogs(context.Background(), query)
	if err != nil {
		return 0, fmt.Errorf("sync unable to Filter Logs: %w", err)
	}
	if len(logs) == 0 {
		return 0, nil
	}
	m.Log.Info("Find log", "block", latestBlock, "log", len(logs))
	total := 0
	for _, log := range logs {
		ele := log
		count, err := log2Oracle(m, &ele)
		if err != nil {
			return 0, err
		}
		total += count
	}

	return total, nil
}

func log2Oracle(m *sync, l *types.Log) (int, error) {
	latestBlock := big.NewInt(0).SetUint64(l.BlockNumber)
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
	m.Log.Info("Tron Oracle receipt", "blockNumber", latestBlock, "hash", tr.Hash())
	receiptHash := tr.Hash()
	ret, err := chain.MulSignInfo(0, uint64(m.Cfg.MapChainID))
	if err != nil {
		return 0, err
	}

	input, err := mapprotocol.PackAbi.Methods[mapprotocol.MethodOfSolidityPack].Inputs.Pack(receiptHash, ret.Version, latestBlock, big.NewInt(int64(m.Cfg.Id)))
	if err != nil {
		return 0, err
	}

	message := msg.NewProposal(m.Cfg.Id, m.Cfg.MapChainID, []interface{}{input, &receiptHash, latestBlock}, m.MsgCh)
	err = m.Router.Send(message)
	if err != nil {
		m.Log.Error("Subscription error: failed to route message", "err", err)
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

func getSigner(log *types.Log, receiptHash common.Hash, selfId, toChainID uint64) (*chain.ProposalInfoResp, error) {
	bn := big.NewInt(int64(log.BlockNumber))
	ret, err := chain.MulSignInfo(0, toChainID)
	if err != nil {
		return nil, err
	}

	piRet, err := chain.ProposalInfo(0, selfId, toChainID, bn, receiptHash, ret.Version)
	if err != nil {
		return nil, err
	}
	if !piRet.CanVerify {
		return nil, chain.NotVerifyAble
	}
	return piRet, nil
}
