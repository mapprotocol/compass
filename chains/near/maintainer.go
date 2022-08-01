package near

import (
	"context"
	"errors"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"

	"github.com/mapprotocol/near-api-go/pkg/client/block"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/mapprotocol/compass/msg"
)

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
	characteristic := block.BlockID(latestBlock.Uint64())
	block, err := m.conn.Client().BlockDetails(context.Background(), characteristic)
	if err != nil {
		return err
	}

	//var chains = []types.Header{{
	//	ParentHash:     types.NewHash([]byte(block.Header.PrevHash.String())),
	//	Number:         types.BlockNumber(block.Header.Height),
	//	StateRoot:      types.Hash{},
	//	ExtrinsicsRoot: types.Hash{},
	//	Digest:         nil,
	//}}
	var chains = []types.Header{{
		ParentHash:  common.HexToHash(block.Header.PrevHash.String()),
		UncleHash:   common.Hash{},
		Coinbase:    common.Address{},
		Root:        common.Hash{},
		TxHash:      common.HexToHash(block.Header.Hash.String()),
		ReceiptHash: common.Hash{},
		Bloom:       types.Bloom{},
		Difficulty:  nil,
		Number:      nil,
		GasLimit:    0,
		GasUsed:     0,
		Time:        0,
		Extra:       nil,
		MixDigest:   common.Hash{},
		Nonce:       types.BlockNonce{},
		BaseFee:     nil,
	}}
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
