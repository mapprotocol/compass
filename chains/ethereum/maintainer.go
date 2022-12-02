package ethereum

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/mapprotocol/compass/internal/chain"
	"github.com/mapprotocol/compass/internal/constant"
	"math/big"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/mapprotocol/compass/mapprotocol"
	"github.com/mapprotocol/compass/msg"
	"github.com/pkg/errors"
	"github.com/pkg/math"
)

type Maintainer struct {
	*chain.CommonSync
	syncedHeight *big.Int
}

func NewMaintainer(cs *chain.CommonSync) *Maintainer {
	return &Maintainer{
		CommonSync:   cs,
		syncedHeight: new(big.Int),
	}
}

func (m *Maintainer) Sync() error {
	m.Log.Debug("Starting listener...")
	go func() {
		err := m.sync()
		if err != nil {
			m.Log.Error("Polling blocks failed", "err", err)
		}
	}()

	return nil
}

// sync function of Maintainer will poll for the latest block and proceed to parse the associated events as it sees new blocks.
// Polling begins at the block defined in `m.Cfg.StartBlock`. Failed attempts to fetch the latest block or parse
// a block will be retried up to BlockRetryLimit times before continuing to the next block.
func (m Maintainer) sync() error {
	var currentBlock = m.Cfg.StartBlock
	m.Log.Info("Polling Blocks...", "block", currentBlock)

	if m.Cfg.SyncToMap {
		// check whether needs quick listen
		syncedHeight, err := mapprotocol.Get2MapHeight(m.Cfg.Id)
		if err != nil {
			m.Log.Error("Get synced Height failed", "err", err)
			return err
		}

		m.Log.Info("Check Sync Status...", "synced", syncedHeight)
		m.syncedHeight = syncedHeight

		// when listen to map there must be a 20 block confirmation at least
		big20 := big.NewInt(20)
		if m.BlockConfirmations.Cmp(big20) == -1 {
			m.BlockConfirmations = big20
		}
		// fix the currentBlock Number
		currentBlock = big.NewInt(0).Sub(currentBlock, m.BlockConfirmations)

		if currentBlock.Cmp(m.syncedHeight) == 1 {
			//listen and start block differs too much perform a fast synced
			m.Log.Info("Perform fast listen to catch up...")
			err = m.batchSyncHeadersTo(big.NewInt(0).Sub(currentBlock, mapprotocol.Big1))
			if err != nil {
				m.Log.Error("Fast batch listen failed")
				return err
			}
		}
	} else if m.Cfg.Id == m.Cfg.MapChainID {
		minHeight := big.NewInt(0)
		for cId, height := range mapprotocol.SyncOtherMap {
			if minHeight.Uint64() == 0 || minHeight.Cmp(height) == 1 {
				m.Log.Info("map to other chain find min sync height ", "chainId", cId,
					"syncedHeight", minHeight, "currentHeight", height)
				minHeight = height
			}
		}
		if m.Cfg.StartBlock.Cmp(minHeight) != 0 { // When the synchronized height is less than or more than the local starting height, use height
			currentBlock = big.NewInt(minHeight.Int64() + 1)
			m.Log.Info("map2other chain", "initial height", currentBlock)
		}
	}

	var retry = constant.BlockRetryLimit
	for {
		select {
		case <-m.Stop:
			return errors.New("polling terminated")
		default:
			// No more retries, goto next block
			if retry == 0 {
				m.Log.Error("Polling failed, retries exceeded")
				m.SysErr <- constant.ErrFatalPolling
				return nil
			}

			latestBlock, err := m.Conn.LatestBlock()
			if err != nil {
				m.Log.Error("Unable to get latest block", "block", currentBlock, "err", err)
				retry--
				time.Sleep(constant.BlockRetryInterval)
				continue
			}

			if m.Metrics != nil {
				m.Metrics.LatestKnownBlock.Set(float64(latestBlock.Int64()))
			}

			// Sleep if the difference is less than BlockDelay; (latest - current) < BlockDelay
			if big.NewInt(0).Sub(latestBlock, currentBlock).Cmp(m.BlockConfirmations) == -1 {
				m.Log.Debug("Block not ready, will retry", "current", currentBlock, "latest", latestBlock)
				time.Sleep(constant.BlockRetryInterval)
				continue
			}

			if m.Cfg.Id == m.Cfg.MapChainID && len(m.Cfg.SyncChainIDList) > 0 {
				// mapchain
				err = m.syncMapHeader(currentBlock)
				if err != nil {
					m.Log.Error("Failed to listen header for block", "block", currentBlock, "err", err)
					retry--
					continue
				}
			} else if m.Cfg.SyncToMap && currentBlock.Cmp(m.syncedHeight) == 1 {
				// Sync headers to Map
				err = m.syncHeaderToMap(currentBlock)
				if err != nil {
					m.Log.Error("Failed to listen header for block", "block", currentBlock, "err", err)
					retry--
					continue
				}
			}

			// Write to block store. Not a critical operation, no need to retry
			err = m.BlockStore.StoreBlock(currentBlock)
			if err != nil {
				m.Log.Error("Failed to write latest block to blockstore", "block", currentBlock, "err", err)
			}

			if m.Metrics != nil {
				m.Metrics.BlocksProcessed.Inc()
				m.Metrics.LatestProcessedBlock.Set(float64(latestBlock.Int64()))
			}

			m.LatestBlock.Height = big.NewInt(0).Set(latestBlock)
			m.LatestBlock.LastUpdated = time.Now()

			currentBlock.Add(currentBlock, big.NewInt(1))
			retry = constant.BlockRetryLimit
		}
	}
}

// syncHeaderToMap listen header from current chain to Map chain
func (m *Maintainer) syncHeaderToMap(latestBlock *big.Int) error {
	// It is checked whether the latest height is higher than the current height
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

// batchSyncHeadersTo
func (m *Maintainer) batchSyncHeadersTo(height *big.Int) error {
	// batch
	var batch = big.NewInt(20)
	headers := make([]types.Header, 0, 20)
	var heightDiff = big.NewInt(0)
	id := big.NewInt(0).SetUint64(uint64(m.Cfg.Id))
	for m.syncedHeight.Cmp(height) == -1 {
		headers = headers[:0]
		heightDiff.Sub(height, m.syncedHeight)
		loop := math.MinBigInt(batch, heightDiff)
		for i := int64(1); i <= loop.Int64(); i++ {
			calcHeight := big.NewInt(0).Add(m.syncedHeight, big.NewInt(i))

			header, err := m.Conn.Client().HeaderByNumber(context.Background(), calcHeight)
			if err != nil {
				return err
			}
			headers = append(headers, *header)
		}

		enc, err := rlpEthereumHeaders(m.Cfg.Id, m.Cfg.MapChainID, headers)
		if err != nil {
			m.Log.Error("failed to rlp ethereum headers", "err", err)
			return err
		}
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

		m.syncedHeight = m.syncedHeight.Add(m.syncedHeight, loop)
		m.Log.Info("Headers synced...", "height", m.syncedHeight)
		time.Sleep(time.Second * 1)
	}

	m.Log.Info("Batch listen finished", "height", height, "syncHeight", m.syncedHeight)
	return nil
}

// syncMapHeader listen map header to every chains registered
func (m *Maintainer) syncMapHeader(latestBlock *big.Int) error {
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
	aggPK, err := mapprotocol.GetAggPK(m.Conn.Client(), new(big.Int).Sub(header.Number, big.NewInt(1)), header.Extra)
	if err != nil {
		return err
	}
	input, err := mapprotocol.PackInput(mapprotocol.Map2Other, mapprotocol.MethodUpdateBlockHeader, h, aggPK)
	if err != nil {
		return err
	}
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
					"xr": "0x" + common.Bytes2Hex(aggPK.Xr.Bytes()),
					"xi": "0x" + common.Bytes2Hex(aggPK.Xi.Bytes()),
					"yi": "0x" + common.Bytes2Hex(aggPK.Yi.Bytes()),
					"yr": "0x" + common.Bytes2Hex(aggPK.Yr.Bytes()),
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
