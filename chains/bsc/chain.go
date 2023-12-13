package bsc

import (
	"context"
	"fmt"
	metrics "github.com/ChainSafe/chainbridge-utils/metrics/types"
	"github.com/ChainSafe/log15"
	ethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	connection "github.com/mapprotocol/compass/connections/ethereum"
	"github.com/mapprotocol/compass/core"
	"github.com/mapprotocol/compass/internal/bsc"
	"github.com/mapprotocol/compass/internal/chain"
	"github.com/mapprotocol/compass/internal/tx"
	"github.com/mapprotocol/compass/mapprotocol"
	"github.com/mapprotocol/compass/msg"
	"math/big"
)

func InitializeChain(chainCfg *core.ChainConfig, logger log15.Logger, sysErr chan<- error, m *metrics.ChainMetrics,
	role mapprotocol.Role) (core.Chain, error) {
	return chain.New(chainCfg, logger, sysErr, m, role, connection.NewConnection, chain.OptOfSync2Map(syncHeaderToMap),
		chain.OptOfInitHeight(mapprotocol.HeaderCountOfBsc), chain.OptOfMos(mosHandler))
}

func syncHeaderToMap(m *chain.Maintainer, latestBlock *big.Int) error {
	remainder := big.NewInt(0).Mod(new(big.Int).Sub(latestBlock, new(big.Int).SetInt64(mapprotocol.HeaderCountOfBsc-1)),
		big.NewInt(mapprotocol.EpochOfBsc))
	if remainder.Cmp(mapprotocol.Big0) != 0 {
		return nil
	}
	// synced height check
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
	m.Log.Info("Find sync block", "current height", latestBlock)
	headers := make([]types.Header, mapprotocol.HeaderCountOfBsc)
	for i := 0; i < mapprotocol.HeaderCountOfBsc; i++ {
		headerHeight := new(big.Int).Sub(latestBlock, new(big.Int).SetInt64(int64(i)))
		header, err := m.Conn.Client().HeaderByNumber(context.Background(), headerHeight)
		if err != nil {
			return err
		}
		headers[mapprotocol.HeaderCountOfBsc-i-1] = *header
	}

	params := make([]bsc.Header, 0, len(headers))
	for _, h := range headers {
		params = append(params, bsc.ConvertHeader(h))
	}
	input, err := mapprotocol.Bsc.Methods[mapprotocol.MethodOfGetHeadersBytes].Inputs.Pack(params)
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
	m.Log.Debug("Querying block for events", "block", latestBlock)
	for idx, addr := range m.Cfg.McsContract {
		query := m.BuildQuery(addr, m.Cfg.Events, latestBlock, latestBlock)
		// querying for logs
		logs, err := m.Conn.Client().FilterLogs(context.Background(), query)
		if err != nil {
			return 0, fmt.Errorf("unable to Filter Logs: %w", err)
		}

		m.Log.Debug("event", "latestBlock ", latestBlock, " logs ", len(logs))
		// read through the log events and handle their deposit event if handler is recognized
		for _, log := range logs {
			// evm event to msg
			var message msg.Message
			// getOrderId
			orderId := log.Data[:32]
			method := m.GetMethod(log.Topics[0])
			// when syncToMap we need to assemble a tx proof
			txsHash, err := tx.GetTxsHashByBlockNumber(m.Conn.Client(), latestBlock)
			if err != nil {
				return 0, fmt.Errorf("unable to get tx hashes Logs: %w", err)
			}
			//now := time.Now()
			receipts, err := tx.GetReceiptsByTxsHash(m.Conn.Client(), txsHash)
			if err != nil {
				return 0, fmt.Errorf("unable to get receipts hashes Logs: %w", err)
			}
			//fmt.Println("--------------------------- ", time.Now().Unix()-now.Unix(), "+++", log.TxIndex)

			//for i, receipt := range receipts {
			//	fmt.Println("i ", i, "receipt", receipt.TxHash)
			//}

			headers := make([]types.Header, mapprotocol.HeaderCountOfBsc)
			for i := 0; i < mapprotocol.HeaderCountOfBsc; i++ {
				headerHeight := new(big.Int).Add(latestBlock, new(big.Int).SetInt64(int64(i)))
				header, err := m.Conn.Client().HeaderByNumber(context.Background(), headerHeight)
				if err != nil {
					return 0, err
				}
				headers[i] = *header
			}

			params := make([]bsc.Header, 0, len(headers))
			for _, h := range headers {
				params = append(params, bsc.ConvertHeader(h))
			}

			payload, err := bsc.AssembleProof(params, log, receipts, method, m.Cfg.Id)
			if err != nil {
				return 0, fmt.Errorf("unable to Parse Log: %w", err)
			}

			msgPayload := []interface{}{payload, orderId, latestBlock.Uint64(), log.TxHash}
			message = msg.NewSwapWithProof(m.Cfg.Id, m.Cfg.MapChainID, msgPayload, m.MsgCh)
			message.Idx = idx

			m.Log.Info("Event found", "BlockNumber", log.BlockNumber, "txHash", log.TxHash, "logIdx", log.Index,
				"orderId", ethcommon.Bytes2Hex(orderId))
			err = m.Router.Send(message)
			if err != nil {
				m.Log.Error("Subscription error: failed to route message", "err", err)
			}
			count++
		}
	}

	return count, nil
}
