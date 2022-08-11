package near

import (
	"context"
	"errors"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"

	"github.com/mapprotocol/compass/internal/near"
	"github.com/mapprotocol/compass/mapprotocol"
	"github.com/mapprotocol/compass/msg"
	"github.com/mapprotocol/near-api-go/pkg/client/block"
)

var NearEpochSize = big.NewInt(43200)

type Maintainer struct {
	*CommonListen
	syncedHeight *big.Int
}

func NewMaintainer(cs *CommonListen) *Maintainer {
	return &Maintainer{
		CommonListen: cs,
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
	var currentBlock = m.cfg.startBlock
	m.log.Info("Polling Blocks...", "block", currentBlock)

	var retry = BlockRetryLimit
	for {
		select {
		case <-m.stop:
			return errors.New("polling terminated")
		default:
			// No more retries, goto next block
			if retry == 0 {
				m.log.Error("Polling failed, retries exceeded")
				m.sysErr <- ErrFatalPolling
				return nil
			}

			latestBlock, err := m.conn.LatestBlock()
			if err != nil {
				m.log.Error("Unable to get latest block", "block", currentBlock, "err", err)
				retry--
				time.Sleep(BlockRetryInterval)
				continue
			}

			if m.metrics != nil {
				m.metrics.LatestKnownBlock.Set(float64(latestBlock.Int64()))
			}

			// Sleep if the difference is less than BlockDelay; (latest - current) < BlockDelay
			if big.NewInt(0).Sub(latestBlock, currentBlock).Cmp(m.blockConfirmations) == -1 {
				m.log.Debug("Block not ready, will retry", "target", currentBlock, "latest", latestBlock)
				time.Sleep(BlockRetryInterval)
				continue
			}

			if m.cfg.syncToMap {
				// listen when catchup
				m.log.Info("Sync Header to Map Chain", "target", currentBlock)
				err = m.syncHeaderToMapChain(currentBlock)
				if err != nil {
					m.log.Error("Failed to listen header for block", "block", currentBlock, "err", err)
					retry--
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

			// Goto next block and reset retry counter
			currentBlock.Add(currentBlock, big.NewInt(1))
			retry = BlockRetryLimit
		}
	}
}

// syncHeaderToMapChain listen header from current chain to Map chain
func (m *Maintainer) syncHeaderToMapChain(latestBlock *big.Int) error {
	input, err := mapprotocol.PackHeaderHeightInput()
	if err != nil {
		m.log.Error("failed to pack update header height input", "err", err)
		return err
	}
	height, err := mapprotocol.HeaderHeight(mapprotocol.NearLightNodeContractOnMAP, input)
	if err != nil {
		m.log.Error("failed to get near header height on map", "err", err, "input", common.Bytes2Hex(input))
		return err
	}

	gap := new(big.Int).Sub(NearEpochSize, new(big.Int).Sub(latestBlock, height)).Int64()
	if gap > 0 {
		time.Sleep(time.Duration(gap/10) * time.Second)
		return nil
	}

	blockDetails, err := m.conn.Client().BlockDetails(context.Background(), block.BlockID(latestBlock.Uint64()))
	if err != nil {
		m.log.Error("failed to get block", "err", err, "number", latestBlock.Uint64())
		return err
	}
	lightBlock, err := m.conn.Client().NextLightClientBlock(context.Background(), blockDetails.Header.Hash)
	if err != nil {
		m.log.Error("failed to get next light client block", "err", err, "number", latestBlock.Uint64(), "hash", blockDetails.Header.Hash)
		return err
	}
	input, err = mapprotocol.PackUpdateBlockHeaderInput(near.Borshify(lightBlock))
	if err != nil {
		m.log.Error("failed to pack update block header input", "err", err, "number", latestBlock.Uint64(), "hash", blockDetails.Header.Hash)
		return err
	}

	message := msg.NewSyncToMap(m.cfg.id, m.cfg.mapChainID, []interface{}{input}, m.msgCh)
	err = m.router.Send(message)
	if err != nil {
		m.log.Error("subscription error: failed to route message", "err", err)
		return nil
	}
	err = m.waitUntilMsgHandled(1)
	if err != nil {
		return err
	}
	return nil
}
