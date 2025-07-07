package chain

import (
	"context"
	"crypto/ecdsa"
	"encoding/binary"
	"fmt"
	"github.com/mapprotocol/compass/internal/mapprotocol"
	"github.com/mapprotocol/compass/pkg/msg"
	"math/big"
	"time"

	eth "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethdb/memorydb"
	"github.com/ethereum/go-ethereum/trie"
	"github.com/mapprotocol/compass/internal/constant"
	"github.com/mapprotocol/compass/internal/proof"
	"github.com/mapprotocol/compass/internal/tx"
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
				m.Log.Debug("Block not ready, will retry", "currentBlock", currentBlock, "latest", latestBlock, "sub", big.NewInt(0).Sub(latestBlock, currentBlock))
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
			err := m.filterOracle()
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

func BuildQuery(contract []common.Address, sig []constant.EventSig, startBlock *big.Int, endBlock *big.Int) eth.FilterQuery {
	topics := make([]common.Hash, 0, len(sig))
	for _, s := range sig {
		topics = append(topics, s.GetTopic())
	}
	query := eth.FilterQuery{
		FromBlock: startBlock,
		ToBlock:   endBlock,
		Addresses: contract,
		Topics:    [][]common.Hash{topics},
	}
	return query
}

func DefaultOracleHandler(m *Oracle, currentBlock *big.Int) error {
	//  区分
	query := BuildQuery(append(m.Cfg.McsContract, m.Cfg.LightNode), m.Cfg.Events, currentBlock, currentBlock)
	// querying for logs
	logs, err := m.Conn.Client().FilterLogs(context.Background(), query)
	if err != nil {
		return fmt.Errorf("oracle unable to Filter Logs: %w", err)
	}
	if len(logs) == 0 {
		return nil
	}
	m.Log.Info("Querying block for events", "block", currentBlock, "logs", len(logs))
	err = log2Oracle(m, logs, currentBlock, 0)
	if err != nil {
		return err
	}

	return nil
}

func getToChainId(topics []common.Hash) uint64 {
	var ret uint64
	if topics[0] == mapprotocol.TopicOfManagerNotifySend {
		ret = binary.BigEndian.Uint64(topics[1][len(topics[1])-8:])
	} else {
		ret = big.NewInt(0).SetBytes(topics[2].Bytes()[8:16]).Uint64()
	}
	return ret
}

func log2Oracle(m *Oracle, logs []types.Log, blockNumber *big.Int, filterId int64) error {
	count := 0
	var (
		err     error
		receipt *common.Hash
	)
	id := big.NewInt(int64(m.Cfg.Id))
	for _, log := range logs {
		toChainID := uint64(m.Cfg.MapChainID)
		if m.Cfg.Id == m.Cfg.MapChainID {
			toChainID = getToChainId(log.Topics)
			if _, ok := mapprotocol.OnlineChaId[msg.ChainId(toChainID)]; !ok {
				m.Log.Info("Map Oracle Found a log that is not the current task", "blockNumber", log.BlockNumber, "toChainID", toChainID)
				continue
			}
		}

		nodeType := new(big.Int)
		if m.Cfg.Id == m.Cfg.MapChainID {
			nodeType, err = GetMap2OtherNodeType(0, toChainID)
			if err != nil {
				return errors.Wrap(err, "Get2OtherNodeType failed")
			}
		} else {
			nodeType, err = mapprotocol.GetNodeTypeByManager(mapprotocol.MethodOfNodeType, big.NewInt(int64(m.Cfg.Id)))
			if err != nil {
				return err
			}
		}

		m.Log.Info("Oracle model get node type is", "blockNumber", blockNumber, "nodeType", nodeType, "topic", log.Topics[0], "filterId", filterId)
		tmp := log
		targetBn := blockNumber
		rpcReceipt, err := m.Conn.Client().TransactionReceipt(context.Background(), log.TxHash)
		if err != nil {
			return err
		}
		if !matchLog(rpcReceipt.Logs, &tmp) {
			m.Log.Info("Oracle receipt log not match", "blockNumber", blockNumber)
			return errors.New("Oracle model get node type is")
		}

		blockHash := ""
		for i := int64(3); i > 0; i-- {
			willBlock, err := m.Conn.Client().MAPBlockByNumber(context.Background(), big.NewInt(targetBn.Int64()+i))
			if err != nil {
				return err
			}
			m.Log.Debug("Oracle getBlock", "willBlock.Number", willBlock.Number, "logNumber", big.NewInt(0).SetUint64(log.BlockNumber).Text(16))
			if willBlock.Number == "0x"+big.NewInt(0).SetUint64(log.BlockNumber+1).Text(16) {
				blockHash = willBlock.ParentHash
			}
		}
		m.Log.Info("Oracle model log blockHash", "blockHash", blockHash)
		switch nodeType.Int64() {
		case constant.ProofTypeOfNewOracle: // mpt
			if log.Topics[0] != mapprotocol.TopicOfClientNotify && log.Topics[0] != mapprotocol.TopicOfManagerNotifySend {
				m.Log.Info("Oracle model ignore this topic", "blockNumber", blockNumber)
				continue
			}

			header, err := m.Conn.Client().HeaderByHash(context.Background(), common.HexToHash(blockHash))
			if err != nil {
				return fmt.Errorf("oracle get header failed, err: %w", err)
			}
			receipt = &header.ReceiptHash
			genRece, err := genMptReceipt(m.Conn.Client(), int64(m.Cfg.Id), blockNumber)
			if genRece != nil {
				receipt = genRece
			}
			log.Index = 0
		case constant.ProofTypeOfLogOracle: // log
			if log.Topics[0] == mapprotocol.TopicOfClientNotify || log.Topics[0] == mapprotocol.TopicOfManagerNotifySend {
				m.Log.Info("Oracle model ignore this topic", "blockNumber", blockNumber)
				continue
			}
			receipt, err = GenLogReceipt(&tmp)
			idx := log.Index
			targetBn = proof.GenLogBlockNumber(blockNumber, idx) // update block number
		default:
			m.Log.Info("Oracle model ignore this tx, because this model type", "blockNumber", blockNumber, "nodeType", nodeType.Int64())
			return nil
		}
		if err != nil {
			return fmt.Errorf("oracle generate receipt failed, err is %w", err)
		}

		m.Log.Info("Find log", "block", blockNumber, "logIndex", log.Index, "receipt", receipt, "targetBn", targetBn)
		ret, err := MulSignInfo(0, uint64(m.Cfg.MapChainID))
		if err != nil {
			return err
		}
		pack, err := mapprotocol.PackAbi.Methods[mapprotocol.MethodOfSolidityPack].Inputs.Pack(receipt, ret.Version, targetBn, id)
		if err != nil {
			return err
		}

		err = m.Router.Send(msg.NewProposal(m.Cfg.Id, m.Cfg.MapChainID, []interface{}{pack, receipt, targetBn}, m.MsgCh))
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

// GenLogReceipt generate log receipt
func GenLogReceipt(log *types.Log) (*common.Hash, error) {
	recePack := make([]byte, 0)
	recePack = append(recePack, log.Address.Bytes()...)
	recePack = append(recePack, []byte{0, 0, 0, 0}...)
	recePack = append(recePack, Completion(big.NewInt(int64(len(log.Topics))).Bytes(), 4)...)
	recePack = append(recePack, Completion(big.NewInt(int64(len(log.Data))).Bytes(), 4)...)
	for _, tp := range log.Topics {
		recePack = append(recePack, tp.Bytes()...)
	}
	recePack = append(recePack, log.Data...)
	receipt := common.BytesToHash(crypto.Keccak256(recePack))
	return &receipt, nil
}

func Completion(bytes []byte, number int) []byte {
	ret := make([]byte, 0, number)
	for i := 0; i < number-len(bytes); i++ {
		ret = append(ret, byte(0))
	}
	ret = append(ret, bytes...)
	return ret
}

func genMptReceipt(cli *ethclient.Client, selfId int64, latestBlock *big.Int) (*common.Hash, error) {
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

func GetMap2OtherNodeType(idx int, toChainID uint64) (*big.Int, error) {
	switch toChainID {
	case constant.TronChainId, constant.SolTestChainId, constant.SolMainChainId:
		return big.NewInt(5), nil
	default:

	}
	if toChainID == constant.TonChainId {
		return big.NewInt(5), nil
	}
	call, ok := mapprotocol.LightNodeMapping[msg.ChainId(toChainID)]
	if !ok {
		return nil, ContractNotExist
	}

	ret := new(big.Int)
	err := call.Call(mapprotocol.MethodOfNodeType, &ret, idx)
	if err != nil {
		return nil, err
	}
	return ret, nil
}

func ExternalOracleInput(selfId, nodeType int64, log *types.Log, client *ethclient.Client, pri *ecdsa.PrivateKey) ([]byte, error) {
	var (
		err         error
		blockNumber = big.NewInt(int64(log.BlockNumber))
		receipt     *common.Hash
		targetBn    = blockNumber
	)
	switch nodeType {
	case constant.ProofTypeOfNewOracle: //mpt
		if log.Topics[0] != mapprotocol.TopicOfClientNotify && log.Topics[0] != mapprotocol.TopicOfManagerNotifySend {
			return nil, ContractNotExist
		}
		header, err := client.HeaderByNumber(context.Background(), blockNumber)
		if err != nil {
			return nil, fmt.Errorf("oracle get header failed, err: %w", err)
		}
		receipt = &header.ReceiptHash
		genRece, err := genMptReceipt(client, selfId, blockNumber)
		if genRece != nil {
			receipt = genRece
		}
		log.Index = 0
	case constant.ProofTypeOfLogOracle:
		if log.Topics[0] == mapprotocol.TopicOfClientNotify || log.Topics[0] == mapprotocol.TopicOfManagerNotifySend {
			return nil, ContractNotExist
		}
		receipt, err = GenLogReceipt(log)
		idx := log.Index
		targetBn = proof.GenLogBlockNumber(blockNumber, idx)
	default:
		panic("unhandled default case")
	}
	if err != nil {
		return nil, fmt.Errorf("oracle generate receipt failed, err is %w", err)
	}
	ret, err := MulSignInfo(0, uint64(constant.MapChainId))
	if err != nil {
		return nil, err
	}
	pack, err := mapprotocol.PackAbi.Methods[mapprotocol.MethodOfSolidityPack].Inputs.Pack(receipt, ret.Version,
		targetBn, big.NewInt(selfId))
	if err != nil {
		return nil, err
	}

	hash := common.Bytes2Hex(crypto.Keccak256(pack))
	sign, err := personalSign(string(common.Hex2Bytes(hash)), pri)
	if err != nil {
		return nil, err
	}
	var fixedHash [32]byte
	for i, v := range receipt {
		fixedHash[i] = v
	}

	data, err := mapprotocol.SignerAbi.Pack(mapprotocol.MethodOfPropose, big.NewInt(selfId), targetBn, fixedHash, sign)
	if err != nil {
		return nil, err
	}

	return data, nil
}

func matchLog(source []*types.Log, targetLog *types.Log) bool {
	for _, l := range source {
		if l.TxHash.Hex() == targetLog.TxHash.Hex() && l.BlockNumber == targetLog.BlockNumber &&
			l.BlockHash.Hex() == targetLog.BlockHash.Hex() && common.Bytes2Hex(l.Data) == common.Bytes2Hex(targetLog.Data) &&
			l.Index == targetLog.Index && l.TxIndex == targetLog.TxIndex {
			return true
		}
	}
	return false
}
