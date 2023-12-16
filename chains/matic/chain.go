package matic

import (
	"context"
	"fmt"
	metrics "github.com/ChainSafe/chainbridge-utils/metrics/types"
	"github.com/ChainSafe/log15"
	ethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	connection "github.com/mapprotocol/compass/connections/ethereum"
	"github.com/mapprotocol/compass/core"
	"github.com/mapprotocol/compass/internal/chain"
	"github.com/mapprotocol/compass/internal/matic"
	"github.com/mapprotocol/compass/internal/tx"
	"github.com/mapprotocol/compass/mapprotocol"
	"github.com/mapprotocol/compass/msg"
	"math/big"
)

var (
	cacheReceipt = make(map[uint64][]*types.Receipt) // key -> chainId_blockHeight
)

func InitializeChain(chainCfg *core.ChainConfig, logger log15.Logger, sysErr chan<- error, m *metrics.ChainMetrics,
	role mapprotocol.Role) (core.Chain, error) {
	return chain.New(chainCfg, logger, sysErr, m, role, connection.NewConnection,
		chain.OptOfSync2Map(syncHeaderToMap),
		chain.OptOfInitHeight(mapprotocol.HeaderOneCount),
		chain.OptOfMos(mosHandler),
	)
}

func syncHeaderToMap(m *chain.Maintainer, latestBlock *big.Int) error {
	remainder := big.NewInt(0).Mod(new(big.Int).Sub(latestBlock, mapprotocol.ConfirmsOfMatic), big.NewInt(mapprotocol.HeaderCountOfMatic))
	if remainder.Cmp(mapprotocol.Big0) != 0 {
		return nil
	}
	syncedHeight, err := mapprotocol.Get2MapHeight(m.Cfg.Id)
	//syncedHeight, err := mapprotocol.Get2MapByLight()
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
	startBlock := new(big.Int).Sub(latestBlock, new(big.Int).SetInt64(mapprotocol.ConfirmsOfMatic.Int64()+1))
	headers := make([]*types.Header, mapprotocol.ConfirmsOfMatic.Int64())
	for i := 0; i < int(mapprotocol.ConfirmsOfMatic.Int64()); i++ {
		headerHeight := new(big.Int).Add(startBlock, new(big.Int).SetInt64(int64(i)))
		header, err := m.Conn.Client().HeaderByNumber(context.Background(), headerHeight)
		if err != nil {
			return err
		}
		headers[i] = header
	}

	mHeaders := make([]matic.BlockHeader, 0, len(headers))
	for _, h := range headers {
		mHeaders = append(mHeaders, matic.ConvertHeader(h))
	}

	//d, _ := json.Marshal(mHeaders)
	//fmt.Println("matic getBmytes input ", string(d))
	input, err := mapprotocol.Matic.Methods[mapprotocol.MethodOfGetHeadersBytes].Inputs.Pack(mHeaders)
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
			var receipts []*types.Receipt
			if v, ok := cacheReceipt[latestBlock.Uint64()]; ok {
				receipts = v
				m.Log.Info("use cache receipt", "latestBlock ", latestBlock, "txHash", log.TxHash)
			} else {
				tmp, err := tx.GetMaticReceiptsByTxsHash(m.Conn.Client(), txsHash)
				if err != nil {
					return 0, fmt.Errorf("unable to get receipts hashes Logs: %w", err)
				}
				for _, t := range tmp {
					if t == nil {
						continue
					}
					receipts = append(receipts, t)
				}
				if len(logs) > 1 {
					cacheReceipt[latestBlock.Uint64()] = receipts
				}
			}

			headers := make([]*types.Header, mapprotocol.ConfirmsOfMatic.Int64())
			for i := 0; i < int(mapprotocol.ConfirmsOfMatic.Int64()); i++ {
				headerHeight := new(big.Int).Add(latestBlock, new(big.Int).SetInt64(int64(i)))
				tmp, err := m.Conn.Client().HeaderByNumber(context.Background(), headerHeight)
				if err != nil {
					return 0, fmt.Errorf("getHeader failed, err is %v", err)
				}
				headers[i] = tmp
			}

			mHeaders := make([]matic.BlockHeader, 0, len(headers))
			for _, h := range headers {
				mHeaders = append(mHeaders, matic.ConvertHeader(h))
			}

			payload, err := matic.AssembleProof(mHeaders, log, m.Cfg.Id, receipts, method)
			if err != nil {
				return 0, fmt.Errorf("unable to Parse Log: %w", err)
			}

			msgPayload := []interface{}{payload, orderId, latestBlock.Uint64(), log.TxHash}
			message = msg.NewSwapWithProof(m.Cfg.Id, m.Cfg.MapChainID, msgPayload, m.MsgCh)
			message.Idx = idx

			m.Log.Info("Event found", "BlockNumber", log.BlockNumber, "txHash", log.TxHash, "logIdx", log.Index, "orderId", ethcommon.Bytes2Hex(orderId))
			err = m.Router.Send(message)
			if err != nil {
				m.Log.Error("subscription error: failed to route message", "err", err)
			}
			count++
		}
	}

	return count, nil
}
