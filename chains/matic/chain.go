package matic

import (
	"context"
	"fmt"
	"github.com/ChainSafe/log15"
	"github.com/ethereum/go-ethereum/core/types"
	connection "github.com/mapprotocol/compass/connections/ethereum"
	"github.com/mapprotocol/compass/core"
	"github.com/mapprotocol/compass/internal/chain"
	"github.com/mapprotocol/compass/internal/matic"
	"github.com/mapprotocol/compass/internal/proof"
	"github.com/mapprotocol/compass/internal/tx"
	"github.com/mapprotocol/compass/mapprotocol"
	"github.com/mapprotocol/compass/msg"
	"math/big"
	"strconv"
)

func InitializeChain(chainCfg *core.ChainConfig, logger log15.Logger, sysErr chan<- error, role mapprotocol.Role) (core.Chain, error) {
	return chain.New(chainCfg, logger, sysErr, role, connection.NewConnection,
		chain.OptOfSync2Map(syncHeaderToMap),
		chain.OptOfInitHeight(12),
		chain.OptOfOracleHandler(chain.DefaultOracleHandler),
		chain.OptOfAssembleProof(assembleProof),
	)
}

func syncHeaderToMap(m *chain.Maintainer, latestBlock *big.Int) error {
	remainder := big.NewInt(0).Mod(new(big.Int).Sub(latestBlock, mapprotocol.ConfirmsOfMatic), big.NewInt(mapprotocol.HeaderCountOfMatic))
	if remainder.Cmp(mapprotocol.Big0) != 0 {
		return nil
	}
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

func assembleProof(m *chain.Messenger, log *types.Log, proofType int64, toChainID uint64) (*msg.Message, error) {
	var (
		message   msg.Message
		orderId   = log.Data[:32]
		method    = m.GetMethod(log.Topics[0])
		bigNumber = big.NewInt(int64(log.BlockNumber))
	)
	txsHash, err := tx.GetTxsHashByBlockNumber(m.Conn.Client(), bigNumber)
	if err != nil {
		return nil, fmt.Errorf("unable to get tx hashes Logs: %w", err)
	}
	var receipts []*types.Receipt
	key := strconv.FormatUint(uint64(m.Cfg.Id), 10) + "_" + bigNumber.String()
	if v, ok := proof.CacheReceipt[key]; ok {
		receipts = v
		m.Log.Info("use cache receipt", "bigNumber ", bigNumber, "txHash", log.TxHash)
	} else {
		tmp, err := tx.GetMaticReceiptsByTxsHash(m.Conn.Client(), txsHash)
		if err != nil {
			return nil, fmt.Errorf("unable to get receipts hashes Logs: %w", err)
		}
		for _, t := range tmp {
			if t == nil {
				continue
			}
			receipts = append(receipts, t)
		}
		proof.CacheReceipt[key] = receipts
	}

	headers := make([]*types.Header, mapprotocol.ConfirmsOfMatic.Int64())
	for i := 0; i < int(mapprotocol.ConfirmsOfMatic.Int64()); i++ {
		headerHeight := new(big.Int).Add(bigNumber, new(big.Int).SetInt64(int64(i)))
		tmp, err := m.Conn.Client().HeaderByNumber(context.Background(), headerHeight)
		if err != nil {
			return nil, fmt.Errorf("getHeader failed, err is %v", err)
		}
		headers[i] = tmp
	}

	mHeaders := make([]matic.BlockHeader, 0, len(headers))
	for _, h := range headers {
		mHeaders = append(mHeaders, matic.ConvertHeader(h))
	}

	payload, err := matic.AssembleProof(mHeaders, log, m.Cfg.Id, receipts, method, proofType)
	if err != nil {
		return nil, fmt.Errorf("unable to Parse Log: %w", err)
	}

	msgPayload := []interface{}{payload, orderId, log.BlockNumber, log.TxHash}
	message = msg.NewSwapWithProof(m.Cfg.Id, m.Cfg.MapChainID, msgPayload, m.MsgCh)
	return &message, nil
}
