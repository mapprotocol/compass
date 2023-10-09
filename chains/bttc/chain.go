package bttc

import (
	"context"
	"fmt"
	"math/big"
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
	for idx, addr := range m.Cfg.McsContract {
		query := m.BuildQuery(addr, m.Cfg.Events, latestBlock, latestBlock)
		logs, err := m.Conn.Client().FilterLogs(context.Background(), query)
		if err != nil {
			return 0, fmt.Errorf("unable to Filter Logs: %w", err)
		}

		m.Log.Debug("event", "latestBlock ", latestBlock, " logs ", len(logs))
		for _, log := range logs {
			var message msg.Message
			orderId := log.Data[:32]
			method := m.GetMethod(log.Topics[0])
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

			payload, err := AssembleProof(mHeaders, log, m.Cfg.Id, allR, cullSys, method)
			if err != nil {
				return 0, fmt.Errorf("unable to Parse Log: %w", err)
			}

			msgPayload := []interface{}{payload, orderId, latestBlock.Uint64(), log.TxHash}
			message = msg.NewSwapWithProof(m.Cfg.Id, m.Cfg.MapChainID, msgPayload, m.MsgCh)
			message.Idx = idx

			m.Log.Info("Event found", "BlockNumber", log.BlockNumber, "txHash", log.TxHash, "logIdx", log.Index,
				"orderId", common.Bytes2Hex(orderId))
			err = m.Router.Send(message)
			if err != nil {
				m.Log.Error("subscription error: failed to route message", "err", err)
			}
			count++
		}
	}

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
