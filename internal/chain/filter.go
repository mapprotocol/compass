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
	"github.com/mapprotocol/compass/internal/constant"
	"github.com/mapprotocol/compass/internal/stream"
	"github.com/pkg/errors"
)

func (m *Messenger) filterMosHandler(latestBlock uint64) (int, uint64, error) {
	count := 0
	progressBlock := uint64(0)
	topic := ""
	for idx, ele := range m.Cfg.Events {
		topic += ele.GetTopic().Hex()
		if idx != len(m.Cfg.Events)-1 {
			topic += ","
		}
	}
	data, err := RequestWithAPIKey(fmt.Sprintf("%s/%s?%s", m.Cfg.FilterHost, constant.FilterUrl,
		fmt.Sprintf("id=%d&project_id=%d&chain_id=%d&topic=%s&limit=1",
			m.Cfg.StartBlock.Int64(), constant.ProjectOfMsger, m.Cfg.Id, topic)), m.Cfg.FilterAPIKey)
	if err != nil {
		return 0, progressBlock, err
	}
	listData, err := json.Marshal(data)
	if err != nil {
		return 0, progressBlock, errors.Wrap(err, "marshal resp.Data failed")
	}
	back := stream.MosListResp{}
	err = json.Unmarshal(listData, &back)
	if err != nil {
		return 0, progressBlock, err
	}
	if len(back.List) == 0 {
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
			return 0, progressBlock, err
		}
		count += send
		m.Cfg.StartBlock = big.NewInt(ele.Id)
	}

	return count, progressBlock, nil
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
		data, err = RequestWithAPIKey(fmt.Sprintf("%s/%s?%s", m.Cfg.FilterHost, constant.FilterUrl,
			fmt.Sprintf("id=%d&project_id=%d&chain_id=%d&topic=%s&limit=1",
				m.Cfg.StartBlock.Int64(), pid, m.Cfg.Id, topic)), m.Cfg.FilterAPIKey)
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
	return RequestWithAPIKey(urlPath, "")
}

func RequestWithAPIKey(urlPath, apiKey string) (interface{}, error) {
	req, err := http.NewRequest(http.MethodGet, urlPath, nil)
	if err != nil {
		return nil, errors.Wrap(err, "new request failed")
	}
	if apiKey != "" {
		req.Header.Set("X-API-Key", apiKey)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "request get failed")
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusUnauthorized {
		return nil, fmt.Errorf("request unauthorized")
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
