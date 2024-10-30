package ton

import (
	"context"
	"encoding/json"
	"fmt"
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
	"math/big"
	"strings"
	"time"
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
		}
		//else {
		//	err = m.sync()
		//}
		if err != nil {
			m.Log.Error("Polling blocks failed", "err", err)
		}
	}()

	return nil
}

func (m *Messenger) filter() error {
	time.Sleep(time.Hour * 48)
	for {
		select {
		case <-m.Stop:
			return errors.New("filter polling terminated")
		default:
			count, err := m.filterMosHandler()
			if err != nil {
				if errors.Is(err, chain.NotVerifyAble) {
					time.Sleep(constant.ThirtySecondInterval)
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

func (m *Messenger) match(target string) int { // todo ton
	return -1
}

func (m *Messenger) filterMosHandler() (int, error) {
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
		idx := m.match(ele.ContractAddress)
		if idx == -1 {
			m.Log.Info("Filter Log Address Not Match", "id", ele.Id, "address", ele.ContractAddress)
			continue
		}

		//  todo ton 自己拼log
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

// todo ton 根据自己的log，去拼proof
func log2Msg(m *Messenger, log *types.Log, idx int) (int, error) {
	orderId := log.Topics[1]
	method := m.GetMethod(log.Topics[0])
	blockNumber := big.NewInt(0).SetUint64(log.BlockNumber)
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

	proofType, err := chain.PreSendTx(idx, uint64(m.Cfg.Id), uint64(m.Cfg.MapChainID), big.NewInt(0).SetUint64(log.BlockNumber), orderId.Bytes())
	if errors.Is(err, chain.OrderExist) {
		m.Log.Info("This txHash order exist", "txHash", log.TxHash)
		return 0, nil
	}
	if errors.Is(err, chain.NotVerifyAble) {
		m.Log.Info("CurrentBlock not verify", "txHash", log.TxHash)
		return 0, err
	}
	if err != nil {
		return 0, err
	}
	var sign [][]byte
	if proofType == constant.ProofTypeOfNewOracle || proofType == constant.ProofTypeOfLogOracle {
		ret, err := chain.Signer(m.Conn.Client(), uint64(m.Cfg.Id), uint64(m.Cfg.MapChainID), log, proofType)
		if err != nil {
			return 0, err
		}
		if !ret.CanVerify {
			return 0, chain.NotVerifyAble
		}
		sign = ret.Signatures
	}
	m.Log.Info("Event found", "txHash", log.TxHash, "orderId", orderId, "method", method, "proofType", proofType)

	var orderId32 [32]byte
	for i, v := range orderId {
		orderId32[i] = v
	}

	payload, err := eth2.AssembleProof(*eth2.ConvertHeader(header), log, receipts, method, m.Cfg.Id, proofType, sign, orderId32)
	if err != nil {
		return 0, fmt.Errorf("unable to Parse Log: %w", err)
	}

	msgPayload := []interface{}{payload, orderId32, log.BlockNumber, log.TxHash}
	message := msg.NewSwapWithProof(m.Cfg.Id, m.Cfg.MapChainID, msgPayload, m.MsgCh) // todo ton 这一行之后不要改
	message.Idx = idx

	err = m.Router.Send(message)
	if err != nil {
		m.Log.Error("Subscription error: failed to route message", "err", err)
	}
	return 1, nil
}
