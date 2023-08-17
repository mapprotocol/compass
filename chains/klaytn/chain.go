package klaytn

import (
	"context"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/mapprotocol/compass/internal/tx"

	"github.com/ethereum/go-ethereum/rlp"

	"github.com/klaytn/klaytn/common"

	metrics "github.com/ChainSafe/chainbridge-utils/metrics/types"
	"github.com/ChainSafe/log15"
	ethcommon "github.com/ethereum/go-ethereum/common"
	connection "github.com/mapprotocol/compass/connections/ethereum"
	"github.com/mapprotocol/compass/core"
	"github.com/mapprotocol/compass/internal/chain"
	"github.com/mapprotocol/compass/internal/klaytn"
	"github.com/mapprotocol/compass/mapprotocol"
	"github.com/mapprotocol/compass/msg"
	"github.com/mapprotocol/compass/pkg/ethclient"
)

var (
	kClient = &klaytn.Client{}
)

func connectKClient(endpoint string) error {
	kc, err := klaytn.DialHttp(endpoint, true)
	if err != nil {
		return err
	}
	kClient = kc
	return nil
}

func InitializeChain(chainCfg *core.ChainConfig, logger log15.Logger, sysErr chan<- error, m *metrics.ChainMetrics,
	role mapprotocol.Role) (core.Chain, error) {
	err := connectKClient(chainCfg.Endpoint)
	if err != nil {
		return nil, err
	}

	return chain.New(chainCfg, logger, sysErr, m, role, connection.NewConnection,
		chain.OptOfSync2Map(syncHeaderToMap),
		chain.OptOfMos(mosHandler))
}

func syncHeaderToMap(m *chain.Maintainer, latestBlock *big.Int) error {
	if err := syncValidatorHeader(m, latestBlock); err != nil {
		return err
	}

	if err := syncHeader(m, latestBlock); err != nil {
		return err
	}

	return nil
}

func syncValidatorHeader(m *chain.Maintainer, latestBlock *big.Int) error {
	kHeader, err := kClient.BlockByNumber(context.Background(), latestBlock)
	if err != nil {
		return err
	}

	if kHeader.VoteData == "0x" {
		return nil
	}
	data := common.Hex2Bytes(strings.TrimPrefix(kHeader.VoteData, klaytn.PrefixOfHex))
	gVote := new(klaytn.GovernanceVote)
	err = rlp.DecodeBytes(data, gVote)
	if err != nil {
		m.Log.Error("Failed to decode a vote", "number", kHeader.Number, "key", gVote.Key, "value", gVote.Value, "validator", gVote.Validator)
		return err
	}

	if gVote.Key != "addvalidator" && gVote.Key != "removevalidator" {
		return nil
	}

	time.Sleep(time.Second)
	m.Log.Info("Send Validator Header", "blockHeight", latestBlock, "voteData", kHeader.VoteData)
	return sendSyncHeader(m, latestBlock, 2)
}

func syncHeader(m *chain.Maintainer, latestBlock *big.Int) error {
	remainder := big.NewInt(0).Mod(latestBlock, big.NewInt(mapprotocol.EpochOfKlaytn))
	if remainder.Cmp(mapprotocol.Big0) != 0 {
		return nil
	}

	m.Log.Info("Find sync block", "current height", latestBlock)
	//syncedHeight, err := mapprotocol.Get2MapByLight()
	syncedHeight, err := mapprotocol.Get2MapHeight(m.Cfg.Id)
	if err != nil {
		m.Log.Error("Get current synced Height failed", "err", err)
		return err
	}
	if latestBlock.Cmp(syncedHeight) <= 0 {
		m.Log.Info("CurrentBlock less than synchronized headerHeight", "synced height", syncedHeight,
			"current height", latestBlock)
		return nil
	}

	return sendSyncHeader(m, latestBlock, mapprotocol.HeaderCountOfKlaytn)
}

func sendSyncHeader(m *chain.Maintainer, latestBlock *big.Int, count int) error {
	headers, err := assembleHeader(m.Conn.Client(), latestBlock, count)
	if err != nil {
		return err
	}

	input, err := mapprotocol.Klaytn.Methods[mapprotocol.MethodOfGetHeadersBytes].Inputs.Pack(headers)
	if err != nil {
		m.Log.Error("Failed to abi pack", "err", err)
		return err
	}

	fmt.Println("input -------------- ", "0x"+ethcommon.Bytes2Hex(input))
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

func assembleHeader(client *ethclient.Client, latestBlock *big.Int, count int) ([]klaytn.Header, error) {
	headers := make([]klaytn.Header, count)
	for i := 0; i < count; i++ {
		headerHeight := new(big.Int).Add(latestBlock, new(big.Int).SetInt64(int64(i)))
		header, err := client.HeaderByNumber(context.Background(), headerHeight)
		if err != nil {
			return nil, err
		}
		hKheader, err := kClient.BlockByNumber(context.Background(), headerHeight)
		if err != nil {
			return nil, err
		}

		headers[count-count+i] = klaytn.ConvertContractHeader(header, hKheader)
	}

	return headers, nil
}

func mosHandler(m *chain.Messenger, latestBlock *big.Int) (int, error) {
	if !m.Cfg.SyncToMap {
		return 0, nil
	}
	m.Log.Debug("Querying block for events", "block", latestBlock)
	query := m.BuildQuery(m.Cfg.McsContract, m.Cfg.Events, latestBlock, latestBlock)
	// querying for logs
	logs, err := m.Conn.Client().FilterLogs(context.Background(), query)
	if err != nil {
		return 0, fmt.Errorf("unable to Filter Logs: %w", err)
	}

	m.Log.Debug("Event", "latestBlock ", latestBlock, " logs ", len(logs))
	count := 0
	for _, log := range logs {
		var message msg.Message
		orderId := log.Data[:32]
		method := m.GetMethod(log.Topics[0])
		txsHash, err := klaytn.GetTxsHashByBlockNumber(kClient, latestBlock)
		if err != nil {
			return 0, fmt.Errorf("unable to get tx hashes Logs: %w", err)
		}
		receipts, err := tx.GetReceiptsByTxsHash(m.Conn.Client(), txsHash)
		if err != nil {
			return 0, fmt.Errorf("unable to get receipts hashes Logs: %w", err)
		}
		// get block
		header, err := m.Conn.Client().HeaderByNumber(context.Background(), latestBlock)
		if err != nil {
			return 0, err
		}
		kHeader, err := kClient.BlockByNumber(context.Background(), latestBlock)
		if err != nil {
			return 0, err
		}

		payload, err := klaytn.AssembleProof(kClient, klaytn.ConvertContractHeader(header, kHeader), log, m.Cfg.Id, receipts, method)
		if err != nil {
			return 0, fmt.Errorf("unable to Parse Log: %w", err)
		}

		msgPayload := []interface{}{payload, orderId, latestBlock.Uint64(), log.TxHash}
		message = msg.NewSwapWithProof(m.Cfg.Id, m.Cfg.MapChainID, msgPayload, m.MsgCh)

		m.Log.Info("Event found", "BlockNumber", log.BlockNumber, "txHash", log.TxHash, "logIdx", log.Index,
			"orderId", ethcommon.Bytes2Hex(orderId))
		err = m.Router.Send(message)
		if err != nil {
			m.Log.Error("Subscription error: failed to route message", "err", err)
		}
		count++
	}

	return count, nil
}
