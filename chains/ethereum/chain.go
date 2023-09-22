package ethereum

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"strconv"
	"strings"
	"time"

	utils "github.com/mapprotocol/compass/shared/ethereum"

	"github.com/ethereum/go-ethereum/common"
	ethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/mapprotocol/compass/msg"
	"github.com/pkg/errors"

	metrics "github.com/ChainSafe/chainbridge-utils/metrics/types"
	"github.com/ChainSafe/log15"
	"github.com/ethereum/go-ethereum/rlp"
	connection "github.com/mapprotocol/compass/connections/ethereum"
	"github.com/mapprotocol/compass/core"
	"github.com/mapprotocol/compass/internal/chain"
	"github.com/mapprotocol/compass/mapprotocol"
)

func InitializeChain(chainCfg *core.ChainConfig, logger log15.Logger, sysErr chan<- error, m *metrics.ChainMetrics,
	role mapprotocol.Role) (core.Chain, error) {
	opts := make([]chain.SyncOpt, 0)

	opts = append(opts, chain.OptOfInitHeight(mapprotocol.HeaderOneCount))
	if strconv.FormatUint(uint64(chainCfg.Id), 10) == mapprotocol.MapId {
		opts = append(opts, chain.OptOfSync2Map(mapToOther))
		opts = append(opts, chain.OptOfInitHeight(mapprotocol.EpochOfMap))
	} else {
		opts = append(opts, chain.OptOfSync2Map(headerToMap))
	}
	opts = append(opts, chain.OptOfMos(mosHandler))
	return chain.New(chainCfg, logger, sysErr, m, role, connection.NewConnection, opts...)
}

func mapToOther(m *chain.Maintainer, latestBlock *big.Int) error {
	if latestBlock.Cmp(big.NewInt(0)) == 0 {
		return nil
	}
	remainder := big.NewInt(0).Mod(latestBlock, big.NewInt(mapprotocol.EpochOfMap))
	if remainder.Cmp(mapprotocol.Big0) != 0 {
		// only listen last block of the epoch
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
	tmp := map[string]interface{}{
		"header":        h,
		"aggpk":         aggPK,
		"istanbulExtra": istanbulExtra,
	}
	tmpData, _ := json.Marshal(tmp)
	m.Log.Info("sync block ", "current", latestBlock, "data", string(tmpData))
	msgpayload := []interface{}{input}
	waitCount := len(m.Cfg.SyncChainIDList)
	for _, cid := range m.Cfg.SyncChainIDList {
		// Only when the latestblock is greater than the height of the synchronized block, the synchronization is performed
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
				return errors.Wrap(err, "get headerHeight failed")
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
	//syncedHeight, err := mapprotocol.Get2MapByLight()
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
	msgpayload := []interface{}{id, enc}
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

func mosHandler(m *chain.Messenger, latestBlock *big.Int) (int, error) {
	m.Log.Debug("Querying block for events", "block", latestBlock)
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
			if m.Cfg.SyncToMap {
				// when syncToMap we need to assemble a tx proof
				txsHash, err := mapprotocol.GetTransactionsHashByBlockNumber(m.Conn.Client(), latestBlock)
				if err != nil {
					return 0, fmt.Errorf("unable to get tx hashes Logs: %w", err)
				}
				receipts, err := mapprotocol.GetReceiptsByTxsHash(m.Conn.Client(), txsHash)
				if err != nil {
					return 0, fmt.Errorf("unable to get receipts hashes Logs: %w", err)
				}
				payload, err := utils.ParseEthLogIntoSwapWithProofArgs(log, addr, receipts, method, m.Cfg.Id, m.Cfg.MapChainID)
				if err != nil {
					return 0, fmt.Errorf("unable to Parse Log: %w", err)
				}

				msgPayload := []interface{}{payload, orderId, latestBlock.Uint64(), log.TxHash}
				message = msg.NewSwapWithProof(m.Cfg.Id, m.Cfg.MapChainID, msgPayload, m.MsgCh)
			} else if m.Cfg.Id == m.Cfg.MapChainID {
				// when listen from map we also need to assemble a tx prove in a different way
				header, err := m.Conn.Client().MAPHeaderByNumber(context.Background(), latestBlock)
				if err != nil {
					return 0, fmt.Errorf("unable to query header Logs: %w", err)
				}
				txsHash, err := mapprotocol.GetMapTransactionsHashByBlockNumber(m.Conn.Client(), latestBlock)
				if err != nil {
					return 0, fmt.Errorf("idSame unable to get tx hashes Logs: %w", err)
				}
				receipts, err := mapprotocol.GetReceiptsByTxsHash(m.Conn.Client(), txsHash)
				if err != nil {
					return 0, fmt.Errorf("unable to get receipts hashes Logs: %w", err)
				}
				//
				remainder := big.NewInt(0).Mod(latestBlock, big.NewInt(mapprotocol.EpochOfMap))
				if remainder.Cmp(mapprotocol.Big0) == 0 {
					lr, err := mapprotocol.GetLastReceipt(m.Conn.Client(), latestBlock)
					if err != nil {
						return 0, fmt.Errorf("unable to get last receipts in epoch last %w", err)
					}
					receipts = append(receipts, lr)
				}

				toChainID, payload, err := utils.AssembleMapProof(m.Conn.Client(), log, receipts, header, m.Cfg.MapChainID, method)
				if err != nil {
					return 0, fmt.Errorf("unable to Parse Log: %w", err)
				}

				if _, ok := mapprotocol.OnlineChaId[msg.ChainId(toChainID)]; !ok {
					m.Log.Info("Found a log that is not the current task ", "toChainID", toChainID)
					continue
				}

				if fn, ok := mapprotocol.Map2OtherVerifyRange[msg.ChainId(toChainID)]; ok {
					left, right, err := fn()
					if err != nil {
						m.Log.Warn("map chain Get2OtherVerifyRange failed", "err", err)
					}
					if left != nil && left.Uint64() != 0 && left.Cmp(latestBlock) == 1 {
						m.Log.Info("min verify range greater than currentBlock, skip ", "currentBlock", latestBlock, "minVerify", left)
						continue
					}
					if right != nil && right.Uint64() != 0 && right.Cmp(latestBlock) == -1 {
						m.Log.Info("currentBlock less than max verify range", "currentBlock", latestBlock, "maxVerify", right)
						time.Sleep(time.Minute * 3)
					}
				}

				msgPayload := []interface{}{payload, orderId, latestBlock.Uint64(), log.TxHash, method}
				message = msg.NewSwapWithMapProof(m.Cfg.MapChainID, msg.ChainId(toChainID), msgPayload, m.MsgCh)
			}

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
