package eth2

import (
	"context"
	"errors"
	"fmt"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/mapprotocol/compass/internal/constant"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/mapprotocol/compass/internal/chain"
	"github.com/mapprotocol/compass/mapprotocol"
	"github.com/mapprotocol/compass/msg"
)

type Maintainer struct {
	*chain.CommonSync
	syncedHeight *big.Int
	confirms     *big.Int
}

func NewMaintainer(cs *chain.CommonSync, confirms *big.Int) *Maintainer {
	return &Maintainer{
		CommonSync:   cs,
		confirms:     confirms,
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
	var currentBlock = m.Cfg.StartBlock
	m.Log.Info("Polling Blocks...", "block", currentBlock)

	if m.Cfg.SyncToMap {
		// check whether needs quick listen
		//syncedHeight, err := mapprotocol.Get2MapHeight(m.Cfg.Id)
		syncedHeight, err := mapprotocol.Get2MapByLight()
		if err != nil {
			m.Log.Error("Get synced Height failed", "err", err)
			return err
		}

		m.Log.Info("Check Sync Status...", "synced", syncedHeight)
		m.syncedHeight = syncedHeight

		if syncedHeight.Cmp(currentBlock) != 0 {
			currentBlock.Add(syncedHeight, new(big.Int).SetInt64(m.confirms.Int64()+2))
			m.Log.Info("SyncedHeight is higher or lower than currentHeight, so let currentHeight = syncedHeight",
				"syncedHeight", syncedHeight, "currentBlock", currentBlock)
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

			if m.Cfg.SyncToMap && currentBlock.Cmp(m.syncedHeight) == 1 {
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
	//syncedHeight, err := mapprotocol.Get2MapHeight(m.Cfg.Id)
	syncedHeight, err := mapprotocol.Get2MapByLight()
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
