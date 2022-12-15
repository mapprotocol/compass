package eth2

import (
	"context"
	"errors"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/mapprotocol/compass/internal/constant"
	"github.com/mapprotocol/compass/internal/eth2"
	"math/big"
	"strconv"
	"time"

	"github.com/mapprotocol/compass/internal/chain"
	"github.com/mapprotocol/compass/mapprotocol"
)

type Maintainer struct {
	*chain.CommonSync
	syncedHeight *big.Int
	eth2Client   *eth2.Client
}

func NewMaintainer(cs *chain.CommonSync, eth2Client *eth2.Client) *Maintainer {
	return &Maintainer{
		CommonSync:   cs,
		eth2Client:   eth2Client,
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
			currentBlock.Add(syncedHeight, new(big.Int).SetInt64(1))
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

			resp, err := m.eth2Client.BeaconHeaders(context.Background(), constant.FinalBlockIdOfEth2)
			if err != nil {
				m.Log.Error("Unable to get latest block", "block", currentBlock, "err", err)
				time.Sleep(constant.BlockRetryInterval)
				continue
			}

			lastFinalizedSlotOnContract := m.syncedHeight
			lastFinalizedSlotOnEth, ok := new(big.Int).SetString(resp.Data.Header.Message.Slot, 10)
			if !ok {
				time.Sleep(constant.BlockRetryInterval)
				continue
			}

			if !m.isEnoughBlocksForLightClientUpdate(lastFinalizedSlotOnContract, lastFinalizedSlotOnEth) {
				time.Sleep(constant.BlockRetryInterval)
				continue
			}

			latestBlock, err := m.Conn.LatestBlock()
			if err != nil {
				m.Log.Error("Unable to get latest block", "block", currentBlock, "err", err)
				time.Sleep(constant.BlockRetryInterval)
				continue
			}
			//
			//if m.Metrics != nil {
			//	m.Metrics.LatestKnownBlock.Set(float64(latestBlock.Int64()))
			//}
			//
			//// Sleep if the difference is less than BlockDelay; (latest - current) < BlockDelay
			//if big.NewInt(0).Sub(latestBlock, currentBlock).Cmp(m.BlockConfirmations) == -1 {
			//	m.Log.Debug("Block not ready, will retry", "current", currentBlock, "latest", latestBlock)
			//	time.Sleep(constant.BlockRetryInterval)
			//	continue
			//}

			if m.Cfg.SyncToMap && currentBlock.Cmp(m.syncedHeight) == 1 {
				err = m.sendRegularLightClientUpdate(currentBlock, lastFinalizedSlotOnContract, lastFinalizedSlotOnEth)
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

func (m *Maintainer) isEnoughBlocksForLightClientUpdate(lastFinalizedSlotOnContract, lastFinalizedSlotOnEth *big.Int) bool {
	// todo 第一个参数
	if (new(big.Int).Int64() - lastFinalizedSlotOnContract.Int64()) < (constant.SlotsPerEpoch * 1) {
		m.Log.Info("Light client update were send less then 1 epochs ago. Skipping sending light client update")
		return false
	}
	if lastFinalizedSlotOnEth.Uint64() <= lastFinalizedSlotOnContract.Uint64() {
		m.Log.Info("Last finalized slot on Eth equal to last finalized slot on NEAR. Skipping sending light client update.")
		return false
	}

	return true
}

func (m *Maintainer) getPeriodForSlot(slot uint64) uint64 {
	return slot / uint64(constant.SlotsPerEpoch*constant.EpochsPerPeriod)
}

// sendRegularLightClientUpdate listen header from current chain to Map chain
func (m *Maintainer) sendRegularLightClientUpdate(latestBlock, lastFinalizedSlotOnContract, lastFinalizedSlotOnEth *big.Int) error {
	lastEth2PeriodOnContract := m.getPeriodForSlot(lastFinalizedSlotOnContract.Uint64())
	endPeriod := m.getPeriodForSlot(lastFinalizedSlotOnEth.Uint64())

	var (
		err             error
		lightUpdateData = &eth2.LightClientUpdate{}
	)
	if lastEth2PeriodOnContract == endPeriod {
		lightUpdateData, err = m.getFinalityLightClientUpdate()
	} else {

	}
	if err != nil {
		return err
	}
	fmt.Println(lightUpdateData)

	return nil
}

func (m *Maintainer) getFinalityLightClientUpdate() (*eth2.LightClientUpdate, error) {
	resp, err := m.eth2Client.FinallyUpdate(context.Background())
	if err != nil {
		return nil, err
	}
	signatureSlot, err := m.getSignatureSlot(&resp.Data.AttestedHeader, &resp.Data.SyncAggregate)
	if err != nil {
		return nil, err
	}

	slot, _ := big.NewInt(0).SetString(resp.Data.AttestedHeader.Slot, 10)
	proposerIndex, ok := big.NewInt(0).SetString(resp.Data.AttestedHeader.ProposerIndex, 10)
	if !ok {
		return nil, errors.New("") // todo
	}

	return &eth2.LightClientUpdate{
		SignatureSlot: signatureSlot,
		SyncAggregate: eth2.ContractSyncAggregate{
			SyncCommitteeBits:      resp.Data.SyncAggregate.SyncCommitteeBits,
			SyncCommitteeSignature: resp.Data.SyncAggregate.SyncCommitteeSignature,
		},
		AttestedHeader: eth2.BeaconBlockHeader{
			Slot:          slot.Uint64(),
			ProposerIndex: proposerIndex.Uint64(),
			ParentRoot:    common.HexToHash(resp.Data.AttestedHeader.ParentRoot),
			StateRoot:     common.HexToHash(resp.Data.AttestedHeader.StateRoot),
			BodyRoot:      common.HexToHash(resp.Data.AttestedHeader.BodyRoot),
		},
		NextSyncCommittee:       eth2.ContractSyncCommittee{},
		NextSyncCommitteeBranch: nil,

		FinalizedHeader:    eth2.BeaconBlockHeader{},
		FinalityBranch:     nil,
		ExeFinalityBranch:  nil,
		FinalizedExeHeader: eth2.BlockHeader{},
	}, nil
}

func (m *Maintainer) getSignatureSlot(ah *eth2.AttestedHeader, sa *eth2.SyncAggregate) (uint64, error) {
	var CheckSlotsForwardLimit uint64 = 10
	ahSlot, ok := big.NewInt(0).SetString(ah.Slot, 10)
	if !ok {
		return 0, errors.New("ahSlot not number")
	}
	var signatureSlot = ahSlot.Uint64() + 1
	for {
		blocks, err := m.eth2Client.GetBlocks(context.Background(), strconv.FormatUint(signatureSlot, 10))
		if err != nil {
			m.Log.Info("GetSignatureSlot GetBlocks failed", "blockId", signatureSlot, "err", err)
		}

		if blocks != nil && blocks.Data.Message.Body.SyncAggregate.SyncCommitteeSignature == sa.SyncCommitteeSignature {
			break
		}

		signatureSlot += 1
		if signatureSlot-ahSlot.Uint64() > CheckSlotsForwardLimit {
			return 0, errors.New("signature slot not found")
		}
	}

	return signatureSlot, nil
}
