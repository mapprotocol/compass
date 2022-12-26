package chain

import (
	"github.com/mapprotocol/compass/internal/constant"
	"github.com/mapprotocol/compass/mapprotocol"
	"github.com/pkg/errors"

	"math/big"
	"time"
)

type BaseMaintainer struct {
	*CommonSync
	syncedHeight    *big.Int
	confirms        *big.Int
	syncMap2Other   SyncMap2Other
	syncHeaderToMap SyncHeaderToMap
}

type SyncMap2Other func(*BaseMaintainer, *big.Int) error
type SyncHeaderToMap func(*BaseMaintainer, *big.Int) error

func NewBaseMaintainer(cs *CommonSync, confirms *big.Int, syncMap2Other SyncMap2Other, syncHeaderToMap SyncHeaderToMap) *BaseMaintainer {
	return &BaseMaintainer{
		CommonSync:      cs,
		confirms:        confirms,
		syncedHeight:    new(big.Int),
		syncMap2Other:   syncMap2Other,
		syncHeaderToMap: syncHeaderToMap,
	}
}

func (bm *BaseMaintainer) Sync() error {
	bm.Log.Debug("Starting listener...")
	go func() {
		err := bm.sync()
		if err != nil {
			bm.Log.Error("Polling blocks failed", "err", err)
		}
	}()

	return nil
}

// sync function of BaseMaintainer will poll for the latest block and proceed to parse the associated events as it sees new blocks.
// Polling begins at the block defined in `bm.Cfg.StartBlock`. Failed attempts to fetch the latest block or parse
// a block will be retried up to BlockRetryLimit times before continuing to the next block.
func (bm *BaseMaintainer) sync() error {
	var currentBlock = bm.Cfg.StartBlock
	bm.Log.Info("Polling Blocks...", "block", currentBlock)

	if bm.Cfg.SyncToMap {
		// check whether needs quick listen
		syncedHeight, err := mapprotocol.Get2MapHeight(bm.Cfg.Id)
		//syncedHeight, err := mapprotocol.Get2MapByLight()
		if err != nil {
			bm.Log.Error("Get synced Height failed", "err", err)
			return err
		}

		bm.Log.Info("Check Sync Status...", "synced", syncedHeight)
		bm.syncedHeight = syncedHeight

		if syncedHeight.Cmp(currentBlock) != 0 {
			currentBlock.Add(syncedHeight, new(big.Int).SetInt64(bm.confirms.Int64()+2))
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

	var retry = constant.BlockRetryLimit
	for {
		select {
		case <-bm.Stop:
			return errors.New("polling terminated")
		default:
			// No more retries, goto next block
			if retry == 0 {
				bm.Log.Error("Polling failed, retries exceeded")
				bm.SysErr <- constant.ErrFatalPolling
				return nil
			}

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
				time.Sleep(constant.BlockRetryInterval)
				continue
			}

			if bm.Cfg.Id == bm.Cfg.MapChainID && len(bm.Cfg.SyncChainIDList) > 0 {
				err = bm.syncMap2Other(bm, currentBlock)
				if err != nil {
					bm.Log.Error("Failed to listen header for block", "block", currentBlock, "err", err)
					retry--
					continue
				}
			} else if bm.Cfg.SyncToMap && currentBlock.Cmp(bm.syncedHeight) == 1 {
				err = bm.syncHeaderToMap(bm, currentBlock)
				if err != nil {
					bm.Log.Error("Failed to listen header for block", "block", currentBlock, "err", err)
					retry--
					continue
				}
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
			retry = constant.BlockRetryLimit
		}
	}
}
