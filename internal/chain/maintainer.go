package chain

import (
	"github.com/mapprotocol/compass/internal/constant"
	"github.com/mapprotocol/compass/mapprotocol"
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

func (bm *Maintainer) Sync() error {
	bm.Log.Debug("Starting listener...")
	go func() {
		err := bm.sync()
		if err != nil {
			bm.Log.Error("Polling blocks failed", "err", err)
		}
	}()

	return nil
}

// sync function of Maintainer will poll for the latest block and proceed to parse the associated events as it sees new blocks.
// Polling begins at the block defined in `bm.Cfg.StartBlock`. Failed attempts to fetch the latest block or parse
// a block will be retried up to BlockRetryLimit times before continuing to the next block.
func (bm *Maintainer) sync() error {
	var currentBlock = bm.Cfg.StartBlock
	bm.Log.Info("Polling Blocks...", "block", currentBlock)

	if bm.Cfg.SyncToMap {
		syncedHeight, err := mapprotocol.Get2MapHeight(bm.Cfg.Id)
		//syncedHeight, err := mapprotocol.Get2MapByLight()
		if err != nil {
			bm.Log.Error("Get synced Height failed", "err", err)
			return err
		}

		bm.Log.Info("Check Sync Status...", "synced", syncedHeight)
		bm.syncedHeight = syncedHeight

		if syncedHeight.Cmp(currentBlock) != 0 {
			currentBlock.Add(syncedHeight, new(big.Int).SetInt64(bm.height))
			bm.Log.Info("SyncedHeight is higher or lower than currentHeight, so let currentHeight = syncedHeight",
				"syncedHeight", syncedHeight, "currentBlock", currentBlock)
		}
	} else if bm.Cfg.Id == bm.Cfg.MapChainID {
		minHeight := big.NewInt(0)
		for cId, height := range mapprotocol.SyncOtherMap {
			if minHeight.Uint64() == 0 || minHeight.Cmp(height) == 1 {
				bm.Log.Info("map to other chain find min sync height ", "chainId", cId,
					"syncedHeight", minHeight, "currentHeight", height)
				minHeight = height
			}
		}
		if bm.Cfg.StartBlock.Cmp(minHeight) != 0 { // When the synchronized height is less than or more than the local starting height, use height
			currentBlock = big.NewInt(minHeight.Int64() + 1)
			bm.Log.Info("map2other chain", "initial height", currentBlock)
		}
	}

	for {
		select {
		case <-bm.Stop:
			return errors.New("polling terminated")
		default:
			latestBlock, err := bm.Conn.LatestBlock()
			if err != nil {
				bm.Log.Error("Unable to get latest block", "block", currentBlock, "err", err)
				time.Sleep(constant.BlockRetryInterval)
				continue
			}
			if bm.Metrics != nil {
				bm.Metrics.LatestKnownBlock.Set(float64(latestBlock.Int64()))
			}

			// Sleep if the difference is less than BlockDelay; (latest - current) < BlockDelay
			if big.NewInt(0).Sub(latestBlock, currentBlock).Cmp(bm.BlockConfirmations) == -1 {
				bm.Log.Debug("Block not ready, will retry", "current", currentBlock, "latest", latestBlock)
				time.Sleep(constant.QueryRetryInterval)
				continue
			}
			// latestBlock must less than blockNumber of chain online，otherwise time.sleep
			difference := new(big.Int).Sub(currentBlock, latestBlock)
			if difference.Int64() > 0 {
				bm.Log.Info("chain online blockNumber less than local latestBlock, waiting...", "chainBlcNum", latestBlock,
					"localBlock", currentBlock, "waiting", difference.Int64())
				time.Sleep(constant.BlockRetryInterval * time.Duration(difference.Int64()))
			}

			if bm.Cfg.Id == bm.Cfg.MapChainID && len(bm.Cfg.SyncChainIDList) > 0 {
				err = bm.syncMap2Other(bm, currentBlock)
				if err != nil {
					bm.Log.Error("Failed to listen header for block", "block", currentBlock, "err", err)
					continue
				}
			} else if bm.Cfg.SyncToMap && currentBlock.Cmp(bm.syncedHeight) == 1 {
				err = bm.syncHeaderToMap(bm, currentBlock)
				if err != nil {
					bm.Log.Error("Failed to listen header for block", "block", currentBlock, "err", err)
					continue
				}
			} else {
				time.Sleep(time.Hour)
			}

			// Write to block store. Not a critical operation, no need to retry
			err = bm.BlockStore.StoreBlock(currentBlock)
			if err != nil {
				bm.Log.Error("Failed to write latest block to blockstore", "block", currentBlock, "err", err)
			}

			if bm.Metrics != nil {
				bm.Metrics.BlocksProcessed.Inc()
				bm.Metrics.LatestProcessedBlock.Set(float64(latestBlock.Int64()))
			}

			bm.LatestBlock.Height = big.NewInt(0).Set(latestBlock)
			bm.LatestBlock.LastUpdated = time.Now()

			currentBlock.Add(currentBlock, big.NewInt(1))
			time.Sleep(constant.MaintainerInterval)
		}
	}
}
