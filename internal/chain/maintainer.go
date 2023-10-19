package chain

import (
	"context"
	"fmt"

	"github.com/mapprotocol/compass/internal/constant"
	"github.com/mapprotocol/compass/mapprotocol"
	"github.com/mapprotocol/compass/pkg/util"
	"github.com/pkg/errors"

	"math/big"
	"time"
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
func (m *Maintainer) sync() error {
	if m.Cfg.Id != m.Cfg.MapChainID && !m.Cfg.SyncToMap {
		time.Sleep(time.Hour * 2400)
		return nil
	}
	var currentBlock = m.Cfg.StartBlock
	m.Log.Info("Polling Blocks...", "block", currentBlock)

	if m.Cfg.SyncToMap {
		syncedHeight, err := mapprotocol.Get2MapHeight(m.Cfg.Id)
		//syncedHeight, err := mapprotocol.Get2MapByLight()
		if err != nil {
			m.Log.Error("Get synced Height failed", "err", err)
			return err
		}

		m.Log.Info("Check Block Sync Status...", "synced", syncedHeight, "currentBlock", currentBlock)
		m.syncedHeight = syncedHeight

		if syncedHeight.Cmp(currentBlock) != 0 {
			currentBlock.Add(syncedHeight, new(big.Int).SetInt64(m.height))
			m.Log.Info("SyncedHeight is higher or lower than currentHeight, so let currentHeight = syncedHeight",
				"syncedHeight", syncedHeight, "currentBlock", currentBlock)
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
			currentBlock = big.NewInt(minHeight.Int64() + m.height)
			m.Log.Info("Map2other chain", "initial height", currentBlock)
		}
	}

	for {
		select {
		case <-m.Stop:
			return errors.New("polling terminated")
		default:
			latestBlock, err := m.Conn.LatestBlock()
			if err != nil {
				m.Log.Error("Unable to get latest block", "block", currentBlock, "err", err)
				time.Sleep(constant.BlockRetryInterval)
				continue
			}
			if m.Metrics != nil {
				m.Metrics.LatestKnownBlock.Set(float64(latestBlock.Int64()))
			}

			// Sleep if the difference is less than BlockDelay; (latest - current) < BlockDelay
			if big.NewInt(0).Sub(latestBlock, currentBlock).Cmp(m.BlockConfirmations) == -1 {
				m.Log.Debug("Block not ready, will retry", "current", currentBlock, "latest", latestBlock)
				time.Sleep(constant.QueryRetryInterval)
				continue
			}
			// latestBlock must less than blockNumber of chain online，otherwise time.sleep
			difference := new(big.Int).Sub(currentBlock, latestBlock)
			if difference.Int64() > 0 {
				m.Log.Info("chain online blockNumber less than local latestBlock, waiting...", "chainBlcNum", latestBlock,
					"localBlock", currentBlock, "waiting", difference.Int64())
				time.Sleep(constant.BlockRetryInterval * time.Duration(difference.Int64()))
			}

			if m.Cfg.Id == m.Cfg.MapChainID && len(m.Cfg.SyncChainIDList) > 0 {
				err = m.syncHeaderToMap(m, currentBlock)
				if err != nil {
					m.Log.Error("Failed to listen header for block", "block", currentBlock, "err", err)
					time.Sleep(constant.QueryRetryInterval)
					util.Alarm(context.Background(), fmt.Sprintf("map sync header to other failed, err is %s", err.Error()))
					continue
				}
			} else if currentBlock.Cmp(m.syncedHeight) == 1 {
				err = m.syncHeaderToMap(m, currentBlock)
				if err != nil {
					m.Log.Error("Failed to listen header for block", "block", currentBlock, "err", err)
					time.Sleep(constant.QueryRetryInterval)
					if err.Error() != "not found" {
						util.Alarm(context.Background(), fmt.Sprintf("%s sync header failed, err is %s", m.Cfg.Name, err.Error()))
					}
					continue
				}
			} else {
				time.Sleep(time.Hour)
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
			if latestBlock.Int64()-currentBlock.Int64() <= m.Cfg.BlockConfirmations.Int64() {
				time.Sleep(constant.MaintainerInterval)
			}
		}
	}
}
