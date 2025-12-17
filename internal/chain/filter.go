package chain

import (
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"sort"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/mapprotocol/compass/internal/butter"
	"github.com/mapprotocol/compass/internal/constant"
	"github.com/mapprotocol/compass/internal/stream"
	"github.com/pkg/errors"
)

func (m *Messenger) filterMosHandler(latestBlock uint64) (int, error) {
	count := 0
	topic := ""
	for idx, ele := range m.Cfg.Events {
		topic += ele.GetTopic().Hex()
		if idx != len(m.Cfg.Events)-1 {
			topic += ","
		}
	}
	data, err := Request(fmt.Sprintf("%s/%s?%s", m.Cfg.FilterHost, constant.FilterUrl,
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
		if latestBlock-ele.BlockNumber < m.BlockConfirmations.Uint64() {
			m.Log.Debug("Block not ready, will retry", "currentBlock", ele.BlockNumber, "latest", latestBlock)
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

		var isBlock bool
		if m.Cfg.Id == m.Cfg.MapChainID {
			relay, err := DecodeMessageRelay(log.Topics, log.Data)
			if err != nil {
				return 0, err
			}
			isBlock, err = butter.BlockedAccount(m.Cfg.PriceHost, string(m.Cfg.Id),
				relay.Sender, common.BytesToAddress(relay.Receiver).String())
			if err != nil {
				return 0, err
			}
			if isBlock {
				m.Log.Info("Ignore this tx, beacuse addr is blocked",
					"tx", ele.TxHash, "sender", relay.Sender,
					"receiver", common.BytesToAddress(relay.Receiver))
			}
		} else {
			initiator, from, err := DecodeMessageOut(log)
			if err != nil {
				return 0, err
			}
			isBlock, err = butter.BlockedAccount(m.Cfg.PriceHost, string(m.Cfg.Id),
				initiator.String(), from.String())
			if err != nil {
				return 0, err
			}
			if isBlock {
				m.Log.Info("Ignore this tx, beacuse addr is blocked",
					"tx", ele.TxHash, "initiator", initiator, "from", from)
			}
		}
		if isBlock {
			m.Cfg.StartBlock = big.NewInt(ele.Id)
			m.Log.Info("Ignore this tx, beacuse addr is blocked",
				"tx", ele.TxHash)
			continue
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

func (m *Oracle) filterOracle() error {
	topic := ""
	for idx, ele := range m.Cfg.Events {
		topic += ele.GetTopic().Hex()
		if idx != len(m.Cfg.Events)-1 {
			topic += ","
		}
	}

	tmp := []int{}
	var err error
	defer func() {
		if len(tmp) == 0 {
			return
		}
		if err != nil {
			return
		}
		sort.Ints(tmp) // less - big
		if int64(tmp[0]) > m.Cfg.StartBlock.Int64() {
			m.Cfg.StartBlock = big.NewInt(int64(tmp[0]))
		}
	}()
	for _, pid := range []int64{constant.ProjectOfOracle, constant.ProjectOfMsger} {
		var data interface{}
		data, err = Request(fmt.Sprintf("%s/%s?%s", m.Cfg.FilterHost, constant.FilterUrl,
			fmt.Sprintf("id=%d&project_id=%d&chain_id=%d&topic=%s&limit=1",
				m.Cfg.StartBlock.Int64(), pid, m.Cfg.Id, topic)))
		if err != nil {
			return err
		}
		var listData []byte
		listData, err = json.Marshal(data)
		if err != nil {
			return errors.Wrap(err, "marshal resp.Data failed")
		}
		back := stream.MosListResp{}
		err = json.Unmarshal(listData, &back)
		if err != nil {
			return err
		}
		if len(back.List) == 0 {
			continue
		}

		for _, ele := range back.List {
			idx := m.Match(ele.ContractAddress) // 新版 oracle
			if idx == -1 {
				m.Log.Info("Filter Log Address Not Match", "id", ele.Id, "address", ele.ContractAddress)
				tmp = append(tmp, int(ele.Id))
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
			err = log2Oracle(m, []types.Log{log}, big.NewInt(0).SetUint64(ele.BlockNumber), ele.Id)
			if err != nil {
				return err
			}
			tmp = append(tmp, int(ele.Id))
		}
	}
	return nil
}

func Request(urlPath string) (interface{}, error) {
	resp, err := http.Get(urlPath)
	if err != nil {
		return nil, errors.Wrap(err, "request get failed")
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrap(err, "readAll failed")
	}
	ret := stream.CommonResp{}
	err = json.Unmarshal(data, &ret)
	if err != nil {
		return nil, errors.Wrap(err, "unMarshal resp failed")
	}
	if ret.Code != http.StatusOK && ret.Code != 2000 {
		msg := ret.Message
		if msg == "" {
			msg = ret.Msg
		}
		return nil, fmt.Errorf("request code is not success, msg is %s", msg)
	}

	return ret.Data, nil
}
