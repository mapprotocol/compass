package chain

import (
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
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
	data, err := request(fmt.Sprintf("%s/%s?%s", m.Cfg.FilterHost, constant.FilterUrl, fmt.Sprintf("id=%d&project_id=%d&chain_id=%d&topic=%s&limit=1",
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
		idx := m.match(ele.ContractAddress)
		if idx == -1 {
			m.Log.Info("Filter Log Address Not Match", "id", ele.Id, "address", ele.ContractAddress)
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

func request(urlPath string) (interface{}, error) {
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
	if ret.Code != http.StatusOK {
		return nil, fmt.Errorf("request code is not success, msg is %s", ret.Message)
	}

	return ret.Data, nil
}
