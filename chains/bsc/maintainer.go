package bsc

import (
	"context"
	"math/big"
	"time"

	"github.com/mapprotocol/compass/internal/bsc"

	"github.com/mapprotocol/compass/internal/constant"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/mapprotocol/compass/mapprotocol"
	"github.com/mapprotocol/compass/msg"
	"github.com/pkg/errors"
)

type Maintainer struct {
	*CommonSync
	syncedHeight *big.Int
}

func NewMaintainer(cs *CommonSync) *Maintainer {
	return &Maintainer{
		CommonSync:   cs,
		syncedHeight: new(big.Int),
	}
}

func (m *Maintainer) Sync() error {
	m.log.Debug("Starting listener...")
	go func() {
		err := m.sync()
		if err != nil {
			m.log.Error("Polling blocks failed", "err", err)
		}
	}()

	return nil
}

// sync function of Maintainer will poll for the latest block and proceed to parse the associated events as it sees new blocks.
// Polling begins at the block defined in `m.cfg.startBlock`. Failed attempts to fetch the latest block or parse
// a block will be retried up to BlockRetryLimit times before continuing to the next block.
func (m Maintainer) sync() error {
	var currentBlock = m.cfg.StartBlock
	m.log.Info("Polling Blocks...", "block", currentBlock)

	if m.cfg.SyncToMap {
		// check whether needs quick listen
		//syncedHeight, err := mapprotocol.Get2MapByLight()
		syncedHeight, err := mapprotocol.Get2MapHeight(m.cfg.Id)
		if err != nil {
			m.log.Error("Get synced Height failed", "err", err)
			return err
		}

		m.log.Info("Check Sync Status...", "synced", syncedHeight)
		m.syncedHeight = syncedHeight

		if syncedHeight.Cmp(currentBlock) != 0 {
			m.log.Info("SyncedHeight is higher or lower than currentHeight, so let currentHeight = syncedHeight",
				"syncedHeight", syncedHeight, "currentBlock", currentBlock)
			currentBlock.Add(syncedHeight, new(big.Int).SetInt64(mapprotocol.HeaderCountOfBsc))
		}
	}

	var retry = constant.BlockRetryLimit
	for {
		select {
		case <-m.stop:
			return errors.New("polling terminated")
		default:
			// No more retries, goto next block
			if retry == 0 {
				m.log.Error("Polling failed, retries exceeded")
				m.sysErr <- constant.ErrFatalPolling
				return nil
			}

			latestBlock, err := m.conn.LatestBlock()
			if err != nil {
				m.log.Error("Unable to get latest block", "block", currentBlock, "err", err)
				retry--
				time.Sleep(constant.BlockRetryInterval)
				continue
			}

			if m.metrics != nil {
				m.metrics.LatestKnownBlock.Set(float64(latestBlock.Int64()))
			}

			if m.cfg.SyncToMap && currentBlock.Cmp(m.syncedHeight) == 1 {
				// Sync headers to Map
				err = m.syncHeaderToMap(currentBlock)
				if err != nil {
					m.log.Error("Failed to listen header for block", "block", currentBlock, "err", err)
					retry--
					time.Sleep(constant.BlockRetryInterval)
					continue
				}
			}

			// Write to block store. Not a critical operation, no need to retry
			err = m.blockStore.StoreBlock(currentBlock)
			if err != nil {
				m.log.Error("Failed to write latest block to blockstore", "block", currentBlock, "err", err)
			}

			if m.metrics != nil {
				m.metrics.BlocksProcessed.Inc()
				m.metrics.LatestProcessedBlock.Set(float64(latestBlock.Int64()))
			}

			m.latestBlock.Height = big.NewInt(0).Set(latestBlock)
			m.latestBlock.LastUpdated = time.Now()

			currentBlock.Add(currentBlock, big.NewInt(1))
			retry = constant.BlockRetryLimit
		}
	}
}

// syncHeaderToMap listen header from current chain to Map chain
func (m *Maintainer) syncHeaderToMap(latestBlock *big.Int) error {
	remainder := big.NewInt(0).Mod(new(big.Int).Sub(latestBlock, new(big.Int).SetInt64(mapprotocol.HeaderCountOfBsc-1)),
		big.NewInt(mapprotocol.EpochOfBsc))
	if remainder.Cmp(mapprotocol.Big0) != 0 {
		return nil
	}
	m.log.Info("find sync block", "current height", latestBlock)
	// It is checked whether the latest height is higher than the current height
	//syncedHeight, err := mapprotocol.Get2MapByLight()
	syncedHeight, err := mapprotocol.Get2MapHeight(m.cfg.Id)
	if err != nil {
		m.log.Error("Get current synced Height failed", "err", err)
		return err
	}
	// If the current block is lower than the latest height, it will not be synchronized
	if latestBlock.Cmp(syncedHeight) <= 0 {
		m.log.Info("CurrentBlock less than synchronized headerHeight", "synced height", syncedHeight,
			"current height", latestBlock)
		return nil
	}

	headers := make([]types.Header, mapprotocol.HeaderCountOfBsc)
	for i := 0; i < mapprotocol.HeaderCountOfBsc; i++ {
		headerHeight := new(big.Int).Sub(latestBlock, new(big.Int).SetInt64(int64(i)))
		header, err := m.conn.Client().HeaderByNumber(context.Background(), headerHeight)
		if err != nil {
			return err
		}
		headers[mapprotocol.HeaderCountOfBsc-i-1] = *header
		//m.log.Info("getHeader", "header", header.Number)
	}

	params := make([]bsc.Header, 0, len(headers))
	for _, h := range headers {
		params = append(params, bsc.ConvertHeader(h))
	}
	input, err := mapprotocol.Bsc.Methods[mapprotocol.MethodOfGetHeadersBytes].Inputs.Pack(params)
	if err != nil {
		m.log.Error("failed to abi pack", "err", err)
		return err
	}

	//fmt.Println("getHeadersBytes得到的信息", "0x"+common.Bytes2Hex(input))
	id := big.NewInt(0).SetUint64(uint64(m.cfg.Id))
	msgpayload := []interface{}{id, input}
	message := msg.NewSyncToMap(m.cfg.Id, m.cfg.MapChainID, msgpayload, m.msgCh)

	err = m.router.Send(message)
	if err != nil {
		m.log.Error("subscription error: failed to route message", "err", err)
		return err
	}

	err = m.waitUntilMsgHandled(1)
	if err != nil {
		return err
	}
	return nil
}
