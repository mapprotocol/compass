package chain

import (
	"encoding/json"
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/mapprotocol/compass/internal/constant"
	"github.com/mapprotocol/compass/internal/stream"
	"github.com/pkg/errors"
)

type FilterListRequest struct {
	StartID   *big.Int
	ProjectID int64
	ChainID   int64
	Topic     string
	Limit     int
}

type FilterClient interface {
	LatestBlock(chainID int64) (*big.Int, error)
	MaxID(chainID int64) (*big.Int, error)
	ListMosLogs(req FilterListRequest) (*stream.MosListResp, error)
}

type RadarFilterClient struct {
	Host   string
	APIKey string
}

func NewRadarFilterClient(host, apiKey string) *RadarFilterClient {
	return &RadarFilterClient{Host: host, APIKey: apiKey}
}

func (c *RadarFilterClient) LatestBlock(chainID int64) (*big.Int, error) {
	data, err := RequestWithAPIKey(fmt.Sprintf("%s/%s", c.Host, fmt.Sprintf("%s?chain_id=%d", constant.FilterBlockUrl, chainID)), c.APIKey)
	if err != nil {
		return nil, err
	}
	latestBlock, ok := big.NewInt(0).SetString(data.(string), 10)
	if !ok {
		return nil, fmt.Errorf("get latest failed, block is %v", data)
	}
	return latestBlock, nil
}

func (c *RadarFilterClient) MaxID(chainID int64) (*big.Int, error) {
	data, err := RequestWithAPIKey(fmt.Sprintf("%s/%s", c.Host, fmt.Sprintf("%s?chain_id=%d", constant.FilterMaxIDUrl, chainID)), c.APIKey)
	if err != nil {
		return nil, err
	}
	maxID, err := parseFilterNumber(data)
	if err != nil {
		return nil, fmt.Errorf("get filter max id failed: %w", err)
	}
	return maxID, nil
}

func (c *RadarFilterClient) ListMosLogs(req FilterListRequest) (*stream.MosListResp, error) {
	limit := req.Limit
	if limit <= 0 {
		limit = 1
	}
	data, err := RequestWithAPIKey(fmt.Sprintf("%s/%s?%s", c.Host, constant.FilterUrl,
		fmt.Sprintf("id=%d&project_id=%d&chain_id=%d&topic=%s&limit=%d",
			req.StartID.Int64(), req.ProjectID, req.ChainID, req.Topic, limit)), c.APIKey)
	if err != nil {
		return nil, err
	}
	listData, err := json.Marshal(data)
	if err != nil {
		return nil, errors.Wrap(err, "marshal resp.Data failed")
	}
	back := stream.MosListResp{}
	if err := json.Unmarshal(listData, &back); err != nil {
		return nil, err
	}
	return &back, nil
}

func BuildFilterTopic(events []constant.EventSig) string {
	topic := ""
	for idx, ele := range events {
		topic += ele.GetTopic().Hex()
		if idx != len(events)-1 {
			topic += ","
		}
	}
	return topic
}

func BuildRawFilterTopic(events []string) string {
	topic := ""
	for idx, ele := range events {
		topic += ele
		if idx != len(events)-1 {
			topic += ","
		}
	}
	return topic
}

func MosRespToEthLog(ele *stream.GetMosResp) *types.Log {
	topics := SplitTopics(ele.Topic)
	return &types.Log{
		Address:     common.HexToAddress(ele.ContractAddress),
		Topics:      topics,
		Data:        common.Hex2Bytes(ele.LogData),
		BlockNumber: ele.BlockNumber,
		TxHash:      common.HexToHash(ele.TxHash),
		TxIndex:     ele.TxIndex,
		BlockHash:   common.HexToHash(ele.BlockHash),
		Index:       ele.LogIndex,
	}
}

func SplitTopics(topic string) []common.Hash {
	if topic == "" {
		return nil
	}
	parts := strings.Split(topic, ",")
	topics := make([]common.Hash, 0, len(parts))
	for _, sp := range parts {
		topics = append(topics, common.HexToHash(sp))
	}
	return topics
}
