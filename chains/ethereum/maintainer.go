package ethereum

import (
	"context"
	"errors"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/rlp"
	"github.com/mapprotocol/compass/msg"
	"github.com/pkg/math"

	ethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/mapprotocol/compass/blockstore"
	"github.com/mapprotocol/compass/mapprotocol"
)

type Maintainer struct {
	*CommonSync
	blockStore   blockstore.Blockstorer
	syncedHeight *big.Int
}

func NewMaintainer(cs *CommonSync, bs blockstore.Blockstorer) *Maintainer {
	return &Maintainer{
		CommonSync: cs,
		blockStore: bs,
	}
}

// Sync function of Maintainer will poll for the latest block and proceed to parse the associated events as it sees new blocks.
// Polling begins at the block defined in `m.cfg.startBlock`. Failed attempts to fetch the latest block or parse
// a block will be retried up to BlockRetryLimit times before continuing to the next block.
func (m *Maintainer) Sync() error {
	var currentBlock = m.cfg.startBlock
	m.log.Info("Polling Blocks...", "block", currentBlock)

	if m.cfg.syncToMap {
		// check whether needs quick listen
		syncedHeight, _, err := mapprotocol.GetCurrentNumberAbi(ethcommon.HexToAddress(m.cfg.from))
		if err != nil {
			m.log.Error("Get synced Height failed")
			return err
		}

		m.log.Info("Check Sync Status...", "synced", syncedHeight)
		m.syncedHeight = syncedHeight

		// when listen to map there must be a 20 block confirmation at least
		big20 := big.NewInt(20)
		if m.blockConfirmations.Cmp(big20) == -1 {
			m.blockConfirmations = big20
		}
		// fix the currentBlock Number
		currentBlock = big.NewInt(0).Sub(currentBlock, m.blockConfirmations)

		if currentBlock.Cmp(m.syncedHeight) == 1 {
			//listen and start block differs too much perform a fast synced
			m.log.Info("Perform fast listen to catch up...")
			err = m.batchSyncHeadersTo(big.NewInt(0).Sub(currentBlock, mapprotocol.Big1))
			if err != nil {
				m.log.Error("Fast batch listen failed")
				return err
			}
		}
	}

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

			if m.cfg.id == m.cfg.mapChainID && len(m.cfg.syncChainIDList) > 0 {
				// mapchain
				err = m.syncMapHeader(currentBlock)
				if err != nil {
					m.log.Error("Failed to listen header for block", "block", currentBlock, "err", err)
					retry--
					continue
				}
			} else if m.cfg.syncToMap {
				// Sync headers to Map
				if currentBlock.Cmp(m.syncedHeight) == 1 {
					// listen when catchup
					m.log.Info("Sync Header to Map Chain", "target", currentBlock)
					err = m.syncHeaderToMapChain(currentBlock)
					if err != nil {
						m.log.Error("Failed to listen header for block", "block", currentBlock, "err", err)
						retry--
						continue
					}
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
	header, err := m.conn.Client().HeaderByNumber(context.Background(), latestBlock)
	if err != nil {
		return err
	}
	chains := []types.Header{*header}
	marshal, _ := rlp.EncodeToBytes(chains)

	msgpayload := []interface{}{marshal}
	message := msg.NewSyncToMap(m.cfg.id, m.cfg.mapChainID, msgpayload, m.msgCh)

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

// batchSyncHeadersTo
func (m *Maintainer) batchSyncHeadersTo(height *big.Int) error {
	// batch
	var batch = big.NewInt(20)
	chains := make([]types.Header, batch.Int64())
	var heightDiff = big.NewInt(0)
	for m.syncedHeight.Cmp(height) == -1 {
		chains = chains[:0]
		heightDiff.Sub(height, m.syncedHeight)
		loop := math.MinBigInt(batch, heightDiff)
		for i := int64(1); i <= loop.Int64(); i++ {
			calcHeight := big.NewInt(0).Add(m.syncedHeight, big.NewInt(i))

			header, err := m.conn.Client().HeaderByNumber(context.Background(), calcHeight)
			if err != nil {
				return err
			}
			chains = append(chains, *header)
		}

		marshal, _ := rlp.EncodeToBytes(chains)
		msgpayload := []interface{}{marshal}
		message := msg.NewSyncToMap(m.cfg.id, m.cfg.mapChainID, msgpayload, m.msgCh)
		err := m.router.Send(message)
		if err != nil {
			m.log.Error("subscription error: failed to route message", "err", err)
			return err
		}

		err = m.waitUntilMsgHandled(1)
		if err != nil {
			return err
		}

		m.syncedHeight = m.syncedHeight.Add(m.syncedHeight, loop)
		m.log.Info("Headers synced...", "height", m.syncedHeight)
	}

	m.log.Info("Batch listen finished", "height", height, "syncHeight", m.syncedHeight)
	return nil
}

// syncMapHeader listen map header to every chains registered
func (m *Maintainer) syncMapHeader(latestBlock *big.Int) error {
	// todo 通过 rpc 查询 epoch size
	remainder := latestBlock.Mod(latestBlock, big.NewInt(30000))
	if remainder.Cmp(mapprotocol.Big0) != 0 {
		// only listen last block of the epoch
		return nil
	}
	header, err := m.conn.Client().MAPHeaderByNumber(context.Background(), latestBlock)
	if err != nil {
		return err
	}

	h := mapprotocol.ConvertHeader(header)
	aggPK, err := mapprotocol.GetAggPK(m.conn.Client(), header.Number, header.Extra)
	if err != nil {
		return err
	}
	input, err := mapprotocol.PackLightNodeInput(mapprotocol.MethodUpdateBlockHeader, h, aggPK)
	if err != nil {
		return err
	}
	msgpayload := []interface{}{input}
	for _, cid := range m.cfg.syncChainIDList {
		message := msg.NewSyncFromMap(m.cfg.mapChainID, cid, msgpayload, m.msgCh)
		err = m.router.Send(message)
		if err != nil {
			m.log.Error("subscription error: failed to route message", "err", err)
			return nil
		}
	}

	err = m.waitUntilMsgHandled(len(m.cfg.syncChainIDList))
	if err != nil {
		return err
	}
	return nil
}
