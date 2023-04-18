package klaytn

import (
	"context"
	"fmt"
	"math/big"

	metrics "github.com/ChainSafe/chainbridge-utils/metrics/types"
	"github.com/ChainSafe/log15"
	ethcommon "github.com/ethereum/go-ethereum/common"
	connection "github.com/mapprotocol/compass/connections/ethereum"
	"github.com/mapprotocol/compass/core"
	"github.com/mapprotocol/compass/internal/chain"
	"github.com/mapprotocol/compass/internal/klaytn"
	"github.com/mapprotocol/compass/internal/tx"
	"github.com/mapprotocol/compass/mapprotocol"
	"github.com/mapprotocol/compass/msg"
)

var (
	kClient = &klaytn.Client{}
)

func InitializeChain(chainCfg *core.ChainConfig, logger log15.Logger, sysErr chan<- error, m *metrics.ChainMetrics,
	role mapprotocol.Role) (core.Chain, error) {
	err := connectKClient(chainCfg.Endpoint)
	if err != nil {
		return nil, err
	}

	return chain.New(chainCfg, logger, sysErr, m, role, connection.NewConnection, chain.OptOfSync2Map(syncHeaderToMap),
		chain.OptOfInitHeight(mapprotocol.HeaderCountOfBsc), chain.OptOfMos(mosHandler))
}

func connectKClient(endpoint string) error {
	kc, err := klaytn.DialHttp(endpoint, true)
	if err != nil {
		return err
	}
	kClient = kc
	return nil
}

func syncHeaderToMap(m *chain.Maintainer, latestBlock *big.Int) error {
	remainder := big.NewInt(0).Mod(new(big.Int).Sub(latestBlock, new(big.Int).SetInt64(mapprotocol.HeaderCountOfKlaytn-1)),
		big.NewInt(mapprotocol.EpochOfKlaytn))
	fmt.Println("latestBlock ------------- ", latestBlock, "remainder", remainder)
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

	headers := make([]klaytn.Header, mapprotocol.HeaderCountOfKlaytn)
	for i := 0; i < mapprotocol.HeaderCountOfKlaytn; i++ {
		headerHeight := new(big.Int).Sub(latestBlock, new(big.Int).SetInt64(int64(i)))
		header, err := m.Conn.Client().HeaderByNumber(context.Background(), headerHeight)
		if err != nil {
			return err
		}
		hKheader, err := kClient.BlockByNumber(context.Background(), headerHeight)
		if err != nil {
			return err
		}

		headers[mapprotocol.HeaderCountOfKlaytn-i-1] = klaytn.ConvertContractHeader(header, hKheader)
	}

	input, err := mapprotocol.Klaytn.Methods[mapprotocol.MethodOfGetHeadersBytes].Inputs.Pack(headers)
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
	m.Log.Debug("Querying block for events", "block", latestBlock)
	query := m.BuildQuery(m.Cfg.McsContract, m.Cfg.Events, latestBlock, latestBlock)
	// querying for logs
	logs, err := m.Conn.Client().FilterLogs(context.Background(), query)
	if err != nil {
		return 0, fmt.Errorf("unable to Filter Logs: %w", err)
	}

	m.Log.Debug("event", "latestBlock ", latestBlock, " logs ", len(logs))
	count := 0
	// read through the log events and handle their deposit event if handler is recognized
	for _, log := range logs {
		// evm event to msg
		var message msg.Message
		// getOrderId
		orderId := log.Data[:32]
		method := m.GetMethod(log.Topics[0])
		// when syncToMap we need to assemble a tx proof
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

		payload, err := klaytn.AssembleProof(klaytn.ConvertContractHeader(header, kHeader), log, m.Cfg.Id, receipts, method)
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
