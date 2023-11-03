package bttc

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"time"

	metrics "github.com/ChainSafe/chainbridge-utils/metrics/types"
	"github.com/ChainSafe/log15"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	connection "github.com/mapprotocol/compass/connections/ethereum"
	"github.com/mapprotocol/compass/core"
	"github.com/mapprotocol/compass/internal/chain"
	"github.com/mapprotocol/compass/internal/tx"
	"github.com/mapprotocol/compass/mapprotocol"
	"github.com/mapprotocol/compass/msg"
	utils "github.com/mapprotocol/compass/shared/ethereum"
)

var (
	systemAddr = common.HexToAddress("0x0000000000000000000000000000000000001010")
)

func NewChain(chainCfg *core.ChainConfig, logger log15.Logger, sysErr chan<- error, m *metrics.ChainMetrics,
	role mapprotocol.Role) (core.Chain, error) {
	return chain.New(chainCfg, logger, sysErr, m, role, connection.NewConnection,
		chain.OptOfSync2Map(syncHeaderToMap),
		chain.OptOfInitHeight(mapprotocol.HeaderOneCount),
		chain.OptOfMos(mosHandler),
	)
}

func syncHeaderToMap(m *chain.Maintainer, latestBlock *big.Int) error {
	remainder := big.NewInt(0).Mod(new(big.Int).Sub(latestBlock, big.NewInt(mapprotocol.HeaderLenOfBttc)), big.NewInt(mapprotocol.HeaderCountOfBttc))
	if remainder.Cmp(mapprotocol.Big0) != 0 {
		return nil
	}
	//syncedHeight, err := mapprotocol.Get2MapHeight(m.Cfg.Id)
	syncedHeight, err := mapprotocol.Get2MapByLight()
	if err != nil {
		m.Log.Error("Get current synced Height failed", "err", err)
		return err
	}
	if latestBlock.Cmp(syncedHeight) <= 0 {
		m.Log.Info("CurrentBlock less than synchronized headerHeight", "synced height", syncedHeight,
			"current height", latestBlock)
		return nil
	}

	m.Log.Info("Find sync block", "current height", latestBlock)
	startBlock := new(big.Int).Sub(latestBlock, new(big.Int).SetInt64(mapprotocol.HeaderLenOfBttc+1))
	headers := make([]*types.Header, mapprotocol.HeaderLenOfBttc)
	for i := 0; i < int(mapprotocol.HeaderLenOfBttc); i++ {
		headerHeight := new(big.Int).Add(startBlock, new(big.Int).SetInt64(int64(i)))
		header, err := m.Conn.Client().HeaderByNumber(context.Background(), headerHeight)
		if err != nil {
			return err
		}
		headers[i] = header
	}

	mHeaders := make([]BlockHeader, 0, len(headers))
	for _, h := range headers {
		mHeaders = append(mHeaders, convertHeader(h))
	}

	input, err := mapprotocol.Bttc.Methods[mapprotocol.MethodOfGetHeadersBytes].Inputs.Pack(mHeaders)
	if err != nil {
		m.Log.Error("Failed to abi pack", "err", err)
		return err
	}

	id := big.NewInt(0).SetUint64(uint64(m.Cfg.Id))
	msgpayload := []interface{}{id, input}
	message := msg.NewSyncToMap(m.Cfg.Id, m.Cfg.MapChainID, msgpayload, m.MsgCh)

	err = m.Router.Send(message)
	if err != nil {
		m.Log.Error("Subscription error: failed to route message", "err", err)
		return err
	}

	err = m.WaitUntilMsgHandled(1)
	if err != nil {
		return err
	}
	return nil
}

func mosHandler(m *chain.Messenger, latestBlock *big.Int) (int, error) {
	if !m.Cfg.SyncToMap {
		return 0, nil
	}
	count := 0
	for idx, topic := range m.Cfg.Events {
		logs, err := filterLogs(m.Cfg.Endpoint, topic.GetTopic().String(), latestBlock)
		if err != nil {
			return 0, fmt.Errorf("unable to Filter Logs: %w", err)
		}

		m.Log.Debug("event", "latestBlock ", latestBlock, " logs ", len(logs))
		for _, log := range logs {
			var message msg.Message
			orderId := log.Data[:32]
			method := m.GetMethod(common.HexToHash(log.Topics[0]))
			txsHash, err := tx.GetTxsHashByBlockNumber(m.Conn.Client(), latestBlock)
			if err != nil {
				return 0, fmt.Errorf("unable to get tx hashes Logs: %w", err)
			}
			allR, cullSys, err := getReceiptsAndTxs(m, txsHash)
			if err != nil {
				return 0, fmt.Errorf("unable to get receipts hashes Logs: %w", err)
			}

			headers := make([]*types.Header, mapprotocol.HeaderLenOfBttc)
			for i := 0; i < int(mapprotocol.HeaderLenOfBttc); i++ {
				headerHeight := new(big.Int).Add(latestBlock, new(big.Int).SetInt64(int64(i)))
				tmp, err := m.Conn.Client().HeaderByNumber(context.Background(), headerHeight)
				if err != nil {
					return 0, fmt.Errorf("getHeader failed, err is %v", err)
				}
				headers[i] = tmp
			}

			mHeaders := make([]BlockHeader, 0, len(headers))
			for _, h := range headers {
				mHeaders = append(mHeaders, convertHeader(h))
			}

			payload, err := AssembleProof(mHeaders, 0, m.Cfg.Id, allR, cullSys, method)
			if err != nil {
				return 0, fmt.Errorf("unable to Parse Log: %w", err)
			}

			msgPayload := []interface{}{payload, orderId, latestBlock.Uint64(), log.TransactionHash}
			message = msg.NewSwapWithProof(m.Cfg.Id, m.Cfg.MapChainID, msgPayload, m.MsgCh)
			message.Idx = idx

			m.Log.Info("Event found", "BlockNumber", log.BlockNumber, "txHash", log.TransactionHash, "logIdx", log.TransactionIndex)
			err = m.Router.Send(message)
			if err != nil {
				m.Log.Error("subscription error: failed to route message", "err", err)
			}
			count++
		}
	}
	//}

	return count, nil
}

func getReceiptsAndTxs(m *chain.Messenger, txsHash []common.Hash) ([]*types.Receipt, []*types.Receipt, error) {
	var (
		rs      = make([]*types.Receipt, 0, len(txsHash))
		cullSys = make([]*types.Receipt, 0, len(txsHash))
	)
	for _, h := range txsHash {
		r, err := m.Conn.Client().TransactionReceipt(context.Background(), h)
		if err != nil {
			if err.Error() == "not found" {
				continue
			}
			return nil, nil, err
		}
		if len(txsHash) > 1000 {
			time.Sleep(time.Millisecond * 10)
		}
		rs = append(rs, r)

		oneTx, _, err := m.Conn.Client().TransactionByHash(context.Background(), h)
		if err != nil {
			if err.Error() == "not found" {
				continue
			}
			return nil, nil, err
		}
		message, err := oneTx.AsMessage(types.NewEIP155Signer(big.NewInt(int64(m.Cfg.Id))), nil)
		if err != nil {
			return nil, nil, err
		}
		m.Log.Info("check address", "hash", oneTx.Hash(), "from", message.From(), "to", oneTx.To())
		if oneTx.To().String() == utils.ZeroAddress.String() && message.From() == utils.ZeroAddress {
			continue
		}
		cullSys = append(cullSys, r)
	}
	return rs, cullSys, nil
}

func filterLogs(endpoint, topic string, latestBlock *big.Int) ([]Log, error) {
	url := fmt.Sprintf("%s&topic0=%s&fromBlock=%v&toBlock=%v", endpoint, topic, latestBlock.String(), latestBlock.String())
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}

	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != 299 {
		return nil, fmt.Errorf("getLogs back code is (%v)", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	ret := &Logs{}
	err = json.Unmarshal(body, ret)
	if err != nil {
		return nil, err
	}

	return ret.Result, nil
}

type Logs struct {
	Status  string `json:"status"`
	Message string `json:"message"`
	Result  []Log  `json:"result"`
}

type Log struct {
	Address          string   `json:"address"`
	Topics           []string `json:"topics"`
	Data             string   `json:"data"`
	BlockNumber      string   `json:"blockNumber"`
	BlockHash        string   `json:"blockHash"`
	TimeStamp        string   `json:"timeStamp"`
	GasPrice         string   `json:"gasPrice"`
	GasUsed          string   `json:"gasUsed"`
	LogIndex         string   `json:"logIndex"`
	TransactionHash  string   `json:"transactionHash"`
	TransactionIndex string   `json:"transactionIndex"`
}
