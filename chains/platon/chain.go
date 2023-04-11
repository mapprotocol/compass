package platon

import (
	"context"
	"fmt"
	"math/big"

	"github.com/mapprotocol/compass/internal/platon"

	ethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/mapprotocol/compass/internal/tx"

	metrics "github.com/ChainSafe/chainbridge-utils/metrics/types"
	"github.com/ChainSafe/log15"
	"github.com/mapprotocol/compass/core"
	"github.com/mapprotocol/compass/internal/chain"
	"github.com/mapprotocol/compass/mapprotocol"
	"github.com/mapprotocol/compass/msg"
)

func InitializeChain(chainCfg *core.ChainConfig, logger log15.Logger, sysErr chan<- error, m *metrics.ChainMetrics,
	role mapprotocol.Role) (core.Chain, error) {
	return chain.New(chainCfg, logger, sysErr, m, role, platon.NewConn, chain.OptOfSync2Map(syncHeaderToMap), chain.OptOfMos(mos))
}

func syncHeaderToMap(m *chain.Maintainer, latestBlock *big.Int) error {
	remainder := big.NewInt(0).Mod(latestBlock, big.NewInt(mapprotocol.HeaderCountOfPlaton))
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
	m.Log.Info("find sync block", "current height", latestBlock)
	headers := make([]*platon.BlockHeader, 1)
	header, err := m.Conn.Client().HeaderByNumber(context.Background(), latestBlock)
	if err != nil {
		return err
	}
	headers = append(headers, platon.ConvertHeader(header))

	block, err := platon.GetHeaderParam(m.Conn.Client(), latestBlock)
	if err != nil {
		return err
	}

	validators := make([]platon.Validator, 0, len(block.Validators))
	for _, v := range block.Validators {
		validators = append(validators, platon.Validator{
			Address:   ethcommon.HexToAddress(v.Address),
			NodeId:    ethcommon.Hex2Bytes(v.NodeId),
			BlsPubKey: ethcommon.Hex2Bytes(v.BlsPubKey),
		})
	}
	input, err := mapprotocol.Platon.Methods[mapprotocol.MethodOfGetHeadersBytes].Inputs.Pack(block.Header, block.Cert, validators)
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

func mos(m *chain.Messenger, latestBlock *big.Int) (int, error) {
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
	m.Log.Info("event", "latestBlock ", latestBlock, " logs ", len(logs))
	if len(logs) == 0 {
		return 0, nil
	}
	headerParam, err := platon.GetHeaderParam(m.Conn.Client(), latestBlock)
	if err != nil {
		return 0, err
	}
	count := 0
	// read through the log events and handle their deposit event if handler is recognized
	for _, log := range logs {
		// evm event to msg
		var message msg.Message
		// getOrderId
		orderId := log.Data[:32]
		method := m.GetMethod(log.Topics[0])
		txsHash, err := tx.GetTxsHashByBlockNumber(m.Conn.Client(), latestBlock)
		if err != nil {
			return 0, fmt.Errorf("unable to get tx hashes Logs: %w", err)
		}
		receipts, err := tx.GetReceiptsByTxsHash(m.Conn.Client(), txsHash)
		if err != nil {
			return 0, fmt.Errorf("unable to get receipts hashes Logs: %w", err)
		}

		payload, err := platon.AssembleProof(headerParam, log, receipts, method, m.Cfg.Id)
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
