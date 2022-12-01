package klaytn

import (
	"context"
	"github.com/ethereum/go-ethereum/common"
	"github.com/mapprotocol/compass/internal/klaytn"
	"github.com/mapprotocol/compass/mapprotocol"
	"github.com/mapprotocol/compass/msg"
	"log"
	"math/big"
	"time"

	"github.com/mapprotocol/compass/internal/chain"
	"github.com/mapprotocol/compass/internal/constant"
	"github.com/pkg/errors"
)

type Maintainer struct {
	*chain.CommonSync
	syncedHeight *big.Int
	kClient      *klaytn.Client
}

func NewMaintainer(cs *chain.CommonSync, kc *klaytn.Client) *Maintainer {
	return &Maintainer{
		CommonSync:   cs,
		syncedHeight: new(big.Int),
		kClient:      kc,
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
// Polling begins at the block defined in `m.Cfg.startBlock`. Failed attempts to fetch the latest block or parse
// a block will be retried up to BlockRetryLimit times before continuing to the next block.
func (m Maintainer) sync() error {
	var currentBlock = m.Cfg.StartBlock
	m.Log.Info("Polling Blocks...", "block", currentBlock)

	if m.Cfg.SyncToMap {
		// check whether needs quick listen
		syncedHeight, err := mapprotocol.Get2MapByLight()
		//syncedHeight, err := mapprotocol.Get2MapHeight(m.Cfg.Id)
		if err != nil {
			m.Log.Error("Get synced Height failed", "err", err)
			return err
		}

		m.Log.Info("Check Sync Status...", "synced", syncedHeight)
		m.syncedHeight = syncedHeight

		if syncedHeight.Cmp(currentBlock) != 0 {
			m.Log.Info("SyncedHeight is higher or lower than currentHeight, so let currentHeight = syncedHeight",
				"syncedHeight", syncedHeight, "currentBlock", currentBlock)
			currentBlock.Add(syncedHeight, new(big.Int)) // .SetInt64(mapprotocol.EpochOfKlaytn)
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
				time.Sleep(constant.RetryLongInterval)
				continue
			}

			if m.Metrics != nil {
				m.Metrics.LatestKnownBlock.Set(float64(latestBlock.Int64()))
			}

			// latestBlock must less than blockNumber of chain online，otherwise time.sleep
			difference := new(big.Int).Sub(currentBlock, latestBlock)
			if difference.Int64() > 0 {
				m.Log.Info("Chain online blockNumber less than local block height, waiting...", "onlineHeight", latestBlock,
					"localHeight", currentBlock, "waiting", difference.Int64())
				time.Sleep(constant.BlockRetryInterval * time.Duration(difference.Int64()))
			}

			// Sleep if the difference is less than BlockDelay; (latest - current) < BlockDelay
			if big.NewInt(0).Sub(latestBlock, currentBlock).Cmp(m.BlockConfirmations) == -1 {
				m.Log.Debug("Block not ready, will retry", "current", currentBlock, "latest", latestBlock)
				time.Sleep(constant.BlockRetryInterval)
				continue
			}

			if m.Cfg.SyncToMap && currentBlock.Cmp(m.syncedHeight) == 1 {
				// Sync headers to Map
				err = m.syncHeaderToMap(currentBlock)
				if err != nil {
					m.Log.Error("Failed to listen header for block", "block", currentBlock, "err", err)
					retry--
					time.Sleep(constant.BlockRetryInterval)
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

			currentBlock.Add(currentBlock, big.NewInt(3600))
			retry = constant.BlockRetryLimit
		}
	}
}

// syncHeaderToMap listen header from current chain to Map chain
func (m *Maintainer) syncHeaderToMap(latestBlock *big.Int) error {
	remainder := big.NewInt(0).Mod(new(big.Int).Sub(latestBlock, new(big.Int).SetInt64(mapprotocol.HeaderCountOfKlaytn-1)),
		big.NewInt(mapprotocol.EpochOfKlaytn))
	if remainder.Cmp(mapprotocol.Big0) != 0 {
		return nil
	}

	m.Log.Info("Find sync block", "current height", latestBlock)
	syncedHeight, err := mapprotocol.Get2MapByLight()
	//syncedHeight, err := mapprotocol.Get2MapHeight(m.Cfg.Id)
	if err != nil {
		m.Log.Error("Get current synced Height failed", "err", err)
		return err
	}
	if latestBlock.Cmp(syncedHeight) <= 0 {
		m.Log.Info("CurrentBlock less than synchronized headerHeight", "synced height", syncedHeight,
			"current height", latestBlock)
		return nil
	}

	headers := make([]klaytn.Header, mapprotocol.HeaderCountOfKlaytn)
	for i := 0; i < mapprotocol.HeaderCountOfKlaytn; i++ {
		headerHeight := new(big.Int).Sub(latestBlock, new(big.Int).SetInt64(int64(i)))
		header, err := m.Conn.Client().HeaderByNumber(context.Background(), headerHeight)
		if err != nil {
			return err
		}
		hKheader, err := m.kClient.BlockByNumber(context.Background(), headerHeight)
		if err != nil {
			return err
		}

		headers[mapprotocol.HeaderCountOfKlaytn-i-1] = klaytn.ConvertContractHeader(header, hKheader)
	}

	input, err := mapprotocol.Klaytn.Methods[mapprotocol.MethodOfGetHeadersBytes].Inputs.Pack(headers)
	if err != nil {
		m.Log.Error("Failed to abi pack", "err", err)
		return err
	}

	log.Println("hex --------- ", "0x"+common.Bytes2Hex(input))

	id := big.NewInt(0).SetUint64(uint64(m.Cfg.Id))
	msgpayload := []interface{}{id, input}
	message := msg.NewSyncToMap(m.Cfg.Id, m.Cfg.MapChainID, msgpayload, m.MsgCh)

	err = m.Router.Send(message)
	if err != nil {
		m.Log.Error("Subscription error: failed to route message", "err", err)
		return err
	}

	err = m.WaitUntilMsgHandled(1)
	if err != nil {
		return err
	}
	return nil
}
