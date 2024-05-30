package ethereum

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"strconv"
	"strings"

	"github.com/mapprotocol/compass/internal/constant"
	"github.com/mapprotocol/compass/internal/mapo"
	"github.com/mapprotocol/compass/internal/tx"
	"github.com/pkg/errors"

	"github.com/ChainSafe/log15"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
	connection "github.com/mapprotocol/compass/connections/ethereum"
	"github.com/mapprotocol/compass/core"
	"github.com/mapprotocol/compass/internal/chain"
	"github.com/mapprotocol/compass/mapprotocol"
	"github.com/mapprotocol/compass/msg"
)

func InitializeChain(chainCfg *core.ChainConfig, logger log15.Logger, sysErr chan<- error,
	role mapprotocol.Role) (core.Chain, error) {
	opts := make([]chain.SyncOpt, 0)

	opts = append(opts, chain.OptOfInitHeight(mapprotocol.HeaderOneCount))
	if strconv.FormatUint(uint64(chainCfg.Id), 10) == mapprotocol.MapId {
		opts = append(opts, chain.OptOfSync2Map(mapToOther))
		opts = append(opts, chain.OptOfInitHeight(mapprotocol.EpochOfMap))
	} else {
		opts = append(opts, chain.OptOfSync2Map(headerToMap))
	}
	opts = append(opts, chain.OptOfAssembleProof(assembleProof))
	opts = append(opts, chain.OptOfOracleHandler(chain.DefaultOracleHandler))
	return chain.New(chainCfg, logger, sysErr, role, connection.NewConnection, opts...)
}

func mapToOther(m *chain.Maintainer, latestBlock *big.Int) error {
	if latestBlock.Cmp(big.NewInt(0)) == 0 {
		return nil
	}
	remainder := big.NewInt(0).Mod(latestBlock, big.NewInt(mapprotocol.EpochOfMap))
	if remainder.Cmp(mapprotocol.Big0) != 0 {
		return nil
	}
	m.Log.Info("sync block ", "current", latestBlock)
	header, err := m.Conn.Client().MAPHeaderByNumber(context.Background(), latestBlock)
	if err != nil {
		return err
	}

	h := mapprotocol.ConvertHeader(header)
	aggPK, ist, aggPKBytes, err := mapprotocol.GetAggPK(m.Conn.Client(), new(big.Int).Sub(header.Number, big.NewInt(1)), header.Extra)
	if err != nil {
		return err
	}
	istanbulExtra := mapprotocol.ConvertIstanbulExtra(ist)
	input, err := mapprotocol.PackInput(mapprotocol.Map2Other, mapprotocol.MethodUpdateBlockHeader, h, istanbulExtra, aggPK)
	if err != nil {
		return err
	}
	m.Log.Debug("sync block ", "current", latestBlock, "data", common.Bytes2Hex(input))
	msgpayload := []interface{}{input}
	waitCount := len(m.Cfg.SyncChainIDList)
	for _, cid := range m.Cfg.SyncChainIDList {
		if v, ok := mapprotocol.SyncOtherMap[cid]; ok && latestBlock.Cmp(v) <= 0 {
			waitCount--
			m.Log.Info("map to other current less than synchronized headerHeight", "toChainId", cid, "synced height", v,
				"current height", latestBlock)
			continue
		}
		// Query the latest height for comparison
		if fn, ok := mapprotocol.Map2OtherHeight[cid]; ok {
			height, err := fn()
			if err != nil {
				return fmt.Errorf("get headerHeight failed, err is %v", err)
			}
			if latestBlock.Cmp(height) <= 0 {
				waitCount--
				m.Log.Info("currentBlock less than latest synchronized headerHeight", "toChainId", cid, "synced height", height,
					"current height", latestBlock)
				continue
			}
		}
		if name, ok := mapprotocol.OnlineChaId[cid]; ok && strings.ToLower(name) == "near" {
			param := map[string]interface{}{
				"header": mapprotocol.ConvertNearNeedHeader(header),
				"agg_pk": map[string]interface{}{
					"xr": "0x" + common.Bytes2Hex(aggPKBytes[32:64]),
					"xi": "0x" + common.Bytes2Hex(aggPKBytes[:32]),
					"yi": "0x" + common.Bytes2Hex(aggPKBytes[64:96]),
					"yr": "0x" + common.Bytes2Hex(aggPKBytes[96:128]),
				},
			}
			data, _ := json.Marshal(param)
			msgpayload = []interface{}{data}
		} else {
			msgpayload = []interface{}{input}
		}
		message := msg.NewSyncFromMap(m.Cfg.MapChainID, cid, msgpayload, m.MsgCh)
		err = m.Router.Send(message)
		if err != nil {
			m.Log.Error("subscription error: failed to route message", "err", err)
			return nil
		}
	}

	err = m.WaitUntilMsgHandled(waitCount)
	if err != nil {
		return err
	}
	return nil
}

func headerToMap(m *chain.Maintainer, latestBlock *big.Int) error {
	syncedHeight, err := mapprotocol.Get2MapHeight(m.Cfg.Id)
	if err != nil {
		m.Log.Error("Get synced Height failed", "err", err)
		return err
	}
	// If the current block is lower than the latest height, it will not be synchronized
	if latestBlock.Cmp(syncedHeight) <= 0 {
		m.Log.Info("currentBlock less than synchronized headerHeight", "synced height", syncedHeight,
			"current height", latestBlock)
		return nil
	}
	m.Log.Info("Sync Header to Map Chain", "current", latestBlock)
	header, err := m.Conn.Client().HeaderByNumber(context.Background(), latestBlock)
	if err != nil {
		return err
	}
	enc, err := rlpEthereumHeaders(m.Cfg.Id, m.Cfg.MapChainID, []types.Header{*header})
	if err != nil {
		m.Log.Error("failed to rlp ethereum headers", "err", err)
		return err
	}
	id := big.NewInt(0).SetUint64(uint64(m.Cfg.Id))
	data, err := mapprotocol.PackInput(mapprotocol.LightManger, mapprotocol.MethodUpdateBlockHeader, id, enc)
	if err != nil {
		m.Log.Error("block2Map Failed to pack abi data", "err", err)
		return err
	}
	msgpayload := []interface{}{id, data}
	message := msg.NewSyncToMap(m.Cfg.Id, m.Cfg.MapChainID, msgpayload, m.MsgCh)

	err = m.Router.Send(message)
	if err != nil {
		m.Log.Error("subscription error: failed to route message", "err", err)
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
	if log.Topics[0].Hex() == constant.TopicsOfSwapInVerified {
		data, err := mapprotocol.Mcs.Events[mapprotocol.EventOfSwapInVerified].Inputs.UnpackValues(log.Data)
		if err != nil {
			return nil, errors.Wrap(err, "swapIn unpackData failed")
		}

		input, _ := mapprotocol.Mcs.Pack(mapprotocol.MethodOfSwapInVerified, data[0].([]byte))
		msgPayload := []interface{}{input, orderId, log.BlockNumber, log.TxHash, mapprotocol.MethodOfSwapInVerified}
		message = msg.NewSwapWithMerlin(m.Cfg.MapChainID, m.Cfg.Id, msgPayload, m.MsgCh)
		return &message, nil
	}
	if m.Cfg.Id == m.Cfg.MapChainID {
		if log.Topics[0] == mapprotocol.HashOfMessageIn {
			method = mapprotocol.MethodOfTransferInWithIndex
		}
		if method == mapprotocol.MethodOfSwapIn {
			method = mapprotocol.MethodOfSwapInWithIndex
		}
		header, err := m.Conn.Client().MAPHeaderByNumber(context.Background(), bigNumber)
		if err != nil {
			return nil, fmt.Errorf("unable to query header Logs: %w", err)
		}
		txsHash, err := tx.GetTxsHashByBlockNumber(m.Conn.Client(), bigNumber)
		if err != nil {
			return nil, fmt.Errorf("idSame unable to get tx hashes Logs: %w", err)
		}
		receipts, err := tx.GetReceiptsByTxsHash(m.Conn.Client(), txsHash)
		if err != nil {
			return nil, fmt.Errorf("unable to get receipts hashes Logs: %w", err)
		}
		remainder := big.NewInt(0).Mod(bigNumber, big.NewInt(mapprotocol.EpochOfMap))
		if remainder.Cmp(mapprotocol.Big0) == 0 {
			lr, err := mapprotocol.GetLastReceipt(m.Conn.Client(), bigNumber)
			if err != nil {
				return nil, fmt.Errorf("unable to get last receipts in epoch last %w", err)
			}
			receipts = append(receipts, lr)
		}

		if toChainID == constant.MerlinChainId {
			method = mapprotocol.MethodOfVerifyAndStore
		}
		_, payload, err := mapo.AssembleMapProof(m.Conn.Client(), log, receipts, header, m.Cfg.MapChainID, method, m.Cfg.ApiUrl, proofType)
		if err != nil {
			return nil, fmt.Errorf("unable to Parse Log: %w", err)
		}

		msgPayload := []interface{}{payload, orderId, log.BlockNumber, log.TxHash, method}
		message = msg.NewSwapWithMapProof(m.Cfg.MapChainID, msg.ChainId(toChainID), msgPayload, m.MsgCh)
		if toChainID == constant.MerlinChainId {
			message = msg.NewSwapWithMerlin(m.Cfg.MapChainID, msg.ChainId(toChainID), msgPayload, m.MsgCh)
		}
	} else if m.Cfg.SyncToMap {
		txsHash, err := mapprotocol.GetMapTransactionsHashByBlockNumber(m.Conn.Client(), bigNumber)
		if err != nil {
			return nil, fmt.Errorf("unable to get tx hashes Logs: %w", err)
		}
		receipts, err := tx.GetReceiptsByTxsHash(m.Conn.Client(), txsHash)
		if err != nil {
			return nil, fmt.Errorf("unable to get receipts hashes Logs: %w", err)
		}
		payload, err := mapo.AssembleEthProof(m.Conn.Client(), log, receipts, method, m.Cfg.Id, proofType)
		if err != nil {
			return nil, fmt.Errorf("unable to Parse Log: %w", err)
		}

		msgPayload := []interface{}{payload, orderId, log.BlockNumber, log.TxHash}
		message = msg.NewSwapWithProof(m.Cfg.Id, m.Cfg.MapChainID, msgPayload, m.MsgCh)
	}
	return &message, nil
}

func rlpEthereumHeaders(source, destination msg.ChainId, headers []types.Header) ([]byte, error) {
	h, err := rlp.EncodeToBytes(&headers)
	if err != nil {
		return nil, fmt.Errorf("rpl encode ethereum headers error: %v", err)
	}

	params := struct {
		From    *big.Int
		To      *big.Int
		Headers []byte
	}{
		From:    big.NewInt(int64(source)),
		To:      big.NewInt(int64(destination)),
		Headers: h,
	}

	enc, err := rlp.EncodeToBytes(params)
	if err != nil {
		return nil, fmt.Errorf("rpl encode params error: %v", err)
	}
	return enc, nil
}
