package eth2

import (
	"context"
	"errors"
	"fmt"
	log "github.com/ChainSafe/log15"
	"github.com/ethereum/go-ethereum/common"
	"github.com/mapprotocol/compass/internal/constant"
	"github.com/mapprotocol/compass/internal/eth2"
	"github.com/mapprotocol/compass/msg"
	"github.com/mapprotocol/compass/pkg/util"
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
		err := m.updateSyncHeight()
		if err != nil {
			m.Log.Error("Get synced Height failed", "err", err)
			return err
		}

		if m.syncedHeight.Cmp(currentBlock) != 0 {
			currentBlock.Add(m.syncedHeight, new(big.Int).SetInt64(1))
			m.Log.Info("SyncedHeight is higher or lower than currentHeight, so let currentHeight = syncedHeight",
				"syncedHeight", m.syncedHeight, "currentBlock", currentBlock)
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

			if m.Cfg.SyncToMap {
				err := m.updateSyncHeight()
				if err != nil {
					m.Log.Error("UpdateSyncHeight failed", "err", err)
					time.Sleep(constant.BlockRetryInterval)
					continue
				}
			}

			startNumber, endNumber, err := mapprotocol.GetEth22MapNumber(m.Cfg.Id)
			if err != nil {
				m.Log.Error("Get startNumber failed", "err", err)
				time.Sleep(constant.BlockRetryInterval)
				continue
			}

			log.Info("updateRange ", "startNumber", startNumber, "endNumber", endNumber)
			if startNumber.Int64() != 0 && endNumber.Int64() != 0 {
				// updateHeader 流程
				err = m.updateHeaders(startNumber, endNumber)
				if err != nil {
					m.Log.Error("updateHeaders failed", "err", err)
					time.Sleep(constant.BlockRetryInterval)
					util.Alarm(context.Background(), fmt.Sprintf("eth2 sync header failed, err is %s", err.Error()))
					continue
				}
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

			if !m.isEnoughBlocksForLightClientUpdate(currentBlock, lastFinalizedSlotOnContract, lastFinalizedSlotOnEth) {
				time.Sleep(constant.BlockRetryInterval)
				continue
			}

			latestBlock, err := m.Conn.LatestBlock()
			if err != nil {
				m.Log.Error("Unable to get latest block", "block", currentBlock, "err", err)
				time.Sleep(constant.BlockRetryInterval)
				continue
			}

			if m.Cfg.SyncToMap {
				err = m.sendRegularLightClientUpdate(lastFinalizedSlotOnContract, lastFinalizedSlotOnEth)
				if err != nil {
					m.Log.Error("Failed to listen header for block", "block", currentBlock, "err", err)
					time.Sleep(constant.BlockRetryInterval)
					util.Alarm(context.Background(), fmt.Sprintf("eth2 sync lightClient failed, err is %s", err.Error()))
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

func (m *Maintainer) updateSyncHeight() error {
	syncedHeight, err := mapprotocol.Get2MapHeight(m.Cfg.Id)
	//syncedHeight, err := mapprotocol.Get2MapByLight()
	if err != nil {
		m.Log.Error("Get synced Height failed", "err", err)
		return err
	}

	m.Log.Info("Check Sync Status...", "synced", syncedHeight)
	m.syncedHeight = syncedHeight
	return nil
}

func (m *Maintainer) isEnoughBlocksForLightClientUpdate(currentBlock, lastFinalizedSlotOnContract, lastFinalizedSlotOnEth *big.Int) bool {
	if (lastFinalizedSlotOnEth.Int64() - lastFinalizedSlotOnContract.Int64()) < (constant.SlotsPerEpoch * 1) {
		m.Log.Info("Light client update were send less then 1 epochs ago. Skipping sending light client update",
			"currentBlock", currentBlock, "lastFinalizedSlotOnContract", lastFinalizedSlotOnContract)
		return false
	}
	if lastFinalizedSlotOnEth.Uint64() <= lastFinalizedSlotOnContract.Uint64() {
		m.Log.Info("Last finalized slot on Eth equal to last finalized slot on Contract. Skipping sending light client update.",
			"lastFinalizedSlotOnEth", lastFinalizedSlotOnEth, "lastFinalizedSlotOnContract", lastFinalizedSlotOnContract)
		return false
	}

	return true
}

func (m *Maintainer) getPeriodForSlot(slot uint64) uint64 {
	return slot / uint64(constant.SlotsPerEpoch*constant.EpochsPerPeriod)
}

// sendRegularLightClientUpdate listen header from current chain to Map chain
func (m *Maintainer) sendRegularLightClientUpdate(lastFinalizedSlotOnContract, lastFinalizedSlotOnEth *big.Int) error {
	lastEth2PeriodOnContract := m.getPeriodForSlot(lastFinalizedSlotOnContract.Uint64())
	endPeriod := m.getPeriodForSlot(lastFinalizedSlotOnEth.Uint64())

	var (
		err             error
		lightUpdateData = &eth2.LightClientUpdate{}
	)
	m.Log.Info("period check", "periodOnContract", lastEth2PeriodOnContract, "endPeriod", endPeriod,
		"slotOnEth", lastFinalizedSlotOnEth, "slotOnContract", lastFinalizedSlotOnContract)
	if lastEth2PeriodOnContract == endPeriod {
		lightUpdateData, err = m.getFinalityLightClientUpdate(lastFinalizedSlotOnContract)
	} else {
		lightUpdateData, err = m.getLightClientUpdateForLastPeriod(lastEth2PeriodOnContract)
	}
	if err != nil {
		return err
	}
	lightClientInput, err := mapprotocol.Eth2.Methods[mapprotocol.MethodOfGetUpdatesBytes].Inputs.Pack(lightUpdateData)
	if err != nil {
		m.Log.Error("Failed to abi pack", "err", err)
		return err
	}

	id := big.NewInt(0).SetUint64(uint64(m.Cfg.Id))
	msgpayload := []interface{}{id, lightClientInput, true}
	message := msg.NewSyncToMap(m.Cfg.Id, m.Cfg.MapChainID, msgpayload, m.MsgCh)
	err = m.Router.Send(message)
	if err != nil {
		m.Log.Error("Subscription error: failed to route message", "err", err)
		return nil
	}
	err = m.WaitUntilMsgHandled(1)
	if err != nil {
		return err
	}
	return nil
}

func (m *Maintainer) getFinalityLightClientUpdate(lastFinalizedSlotOnContract *big.Int) (*eth2.LightClientUpdate, error) {
	resp, err := m.eth2Client.FinallyUpdate(context.Background())
	if err != nil {
		return nil, err
	}

	bitvector512 := util.NewBitvector512(util.FromHexString(resp.Data.SyncAggregate.SyncCommitteeBits))
	count := bitvector512.Count()

	m.Log.Info("521 check", "len", len(util.FromHexString(resp.Data.SyncAggregate.SyncCommitteeBits)),
		"count", count, "512Len", bitvector512.Len())

	if count*3 < bitvector512.Len()*2 {
		return nil, fmt.Errorf("not enought sync committe count %d", count)
	}

	signatureSlot, err := m.getSignatureSlot(&resp.Data.AttestedHeader, &resp.Data.SyncAggregate)
	if err != nil {
		return nil, err
	}

	fhSlot, _ := big.NewInt(0).SetString(resp.Data.FinalizedHeader.Slot, 10)
	fhProposerIndex, ok := big.NewInt(0).SetString(resp.Data.FinalizedHeader.ProposerIndex, 10)
	if !ok {
		return nil, errors.New("FinalizedHeader Slot Not Number")
	}

	if fhSlot.Cmp(lastFinalizedSlotOnContract) <= 0 {
		return nil, fmt.Errorf("finaliy slot(%d) less than slot on contract(%d)", fhSlot.Int64(), lastFinalizedSlotOnContract.Int64())
	}

	m.Log.Info("Slot compare", "fhSlot", resp.Data.FinalizedHeader.Slot, "fsOnContract ", lastFinalizedSlotOnContract)
	slot, _ := big.NewInt(0).SetString(resp.Data.AttestedHeader.Slot, 10)
	proposerIndex, ok := big.NewInt(0).SetString(resp.Data.AttestedHeader.ProposerIndex, 10)
	if !ok {
		return nil, errors.New("AttestedHeader Slot Not Number")
	}
	finalityBranch := make([][32]byte, 0, len(resp.Data.FinalityBranch))
	for _, fb := range resp.Data.FinalityBranch {
		finalityBranch = append(finalityBranch, common.HexToHash(fb))
	}

	exeFinalityBranch, err := eth2.Generate(strconv.FormatUint(fhSlot.Uint64(), 10), m.Cfg.Eth2Endpoint)
	if err != nil {
		return nil, err
	}

	block, err := m.eth2Client.GetBlocks(context.Background(), resp.Data.FinalizedHeader.Slot)
	if err != nil {
		return nil, err
	}

	blockNumber, ok := new(big.Int).SetString(block.Data.Message.Body.ExecutionPayload.BlockNumber, 10)
	if !ok {
		return nil, errors.New("block executionPayload blockNumber Not Number")
	}
	header, err := m.Conn.Client().HeaderByNumber(context.Background(), blockNumber)
	if err != nil {
		return nil, err
	}
	return &eth2.LightClientUpdate{
		SignatureSlot: signatureSlot,
		SyncAggregate: eth2.ContractSyncAggregate{
			SyncCommitteeBits:      util.FromHexString(resp.Data.SyncAggregate.SyncCommitteeBits),
			SyncCommitteeSignature: util.FromHexString(resp.Data.SyncAggregate.SyncCommitteeSignature),
		},
		AttestedHeader: eth2.BeaconBlockHeader{
			Slot:          slot.Uint64(),
			ProposerIndex: proposerIndex.Uint64(),
			ParentRoot:    common.HexToHash(resp.Data.AttestedHeader.ParentRoot),
			StateRoot:     common.HexToHash(resp.Data.AttestedHeader.StateRoot),
			BodyRoot:      common.HexToHash(resp.Data.AttestedHeader.BodyRoot),
		},
		NextSyncCommittee: eth2.ContractSyncCommittee{
			Pubkeys:         make([]byte, 0),
			AggregatePubkey: make([]byte, 0),
		},
		NextSyncCommitteeBranch: nil,
		FinalityBranch:          finalityBranch,
		FinalizedHeader: eth2.BeaconBlockHeader{
			Slot:          fhSlot.Uint64(),
			ProposerIndex: fhProposerIndex.Uint64(),
			ParentRoot:    common.HexToHash(resp.Data.FinalizedHeader.ParentRoot),
			StateRoot:     common.HexToHash(resp.Data.FinalizedHeader.StateRoot),
			BodyRoot:      common.HexToHash(resp.Data.FinalizedHeader.BodyRoot),
		},
		ExeFinalityBranch:  exeFinalityBranch,
		FinalizedExeHeader: *eth2.ConvertHeader(header),
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

func (m *Maintainer) getLightClientUpdateForLastPeriod(lastEth2PeriodOnContract uint64) (*eth2.LightClientUpdate, error) {
	headers, err := m.eth2Client.BeaconHeaders(context.Background(), constant.HeadBlockIdOfEth2)
	if err != nil {
		return nil, err
	}

	headerSlot, ok := big.NewInt(0).SetString(headers.Data.Header.Message.Slot, 10)
	if !ok {
		return nil, errors.New("BeaconHeaders Slot Not Number")
	}

	lastPeriod := m.getPeriodForSlot(headerSlot.Uint64())
	if lastPeriod-lastEth2PeriodOnContract != 1 { // More than one intervals
		lastPeriod = lastEth2PeriodOnContract + 1
	}
	resp, err := m.eth2Client.LightClientUpdate(context.Background(), lastPeriod)
	if err != nil {
		return nil, err
	}

	slot, _ := big.NewInt(0).SetString(resp.Data[0].AttestedHeader.Slot, 10)
	proposerIndex, ok := big.NewInt(0).SetString(resp.Data[0].AttestedHeader.ProposerIndex, 10)
	if !ok {
		return nil, errors.New("AttestedHeader Slot Not Number")
	}

	signatureSlot, err := m.getSignatureSlot(&resp.Data[0].AttestedHeader, &resp.Data[0].SyncAggregate)
	if err != nil {
		return nil, err
	}
	nextSyncCommitteeBranch := make([][32]byte, 0, len(resp.Data[0].NextSyncCommitteeBranch))
	for _, b := range resp.Data[0].NextSyncCommitteeBranch {
		nextSyncCommitteeBranch = append(nextSyncCommitteeBranch, common.HexToHash(b))
	}
	pubKeys := make([]byte, 0, len(resp.Data[0].NextSyncCommittee.Pubkeys)*48)
	for _, pk := range resp.Data[0].NextSyncCommittee.Pubkeys {
		pubKeys = append(pubKeys, util.FromHexString(pk)...)
	}
	finalityBranch := make([][32]byte, 0, len(resp.Data[0].FinalityBranch))
	for _, fb := range resp.Data[0].FinalityBranch {
		finalityBranch = append(finalityBranch, common.HexToHash(fb))
	}
	fhSlot, _ := big.NewInt(0).SetString(resp.Data[0].FinalizedHeader.Slot, 10)
	fhProposerIndex, ok := big.NewInt(0).SetString(resp.Data[0].FinalizedHeader.ProposerIndex, 10)
	if !ok {
		return nil, errors.New("FinalizedHeader  Slot Not Number")
	}
	exeFinalityBranch, err := eth2.Generate(strconv.FormatUint(fhSlot.Uint64(), 10), m.Cfg.Eth2Endpoint)
	if err != nil {
		return nil, err
	}

	block, err := m.eth2Client.GetBlocks(context.Background(), resp.Data[0].FinalizedHeader.Slot)
	if err != nil {
		return nil, err
	}

	blockNumber, ok := new(big.Int).SetString(block.Data.Message.Body.ExecutionPayload.BlockNumber, 10)
	if !ok {
		return nil, errors.New("block executionPayload blockNumber Not Number")
	}
	header, err := m.Conn.Client().HeaderByNumber(context.Background(), blockNumber)
	if err != nil {
		return nil, err
	}

	return &eth2.LightClientUpdate{
		AttestedHeader: eth2.BeaconBlockHeader{
			Slot:          slot.Uint64(),
			ProposerIndex: proposerIndex.Uint64(),
			ParentRoot:    common.HexToHash(resp.Data[0].AttestedHeader.ParentRoot),
			StateRoot:     common.HexToHash(resp.Data[0].AttestedHeader.StateRoot),
			BodyRoot:      common.HexToHash(resp.Data[0].AttestedHeader.BodyRoot),
		},
		SyncAggregate: eth2.ContractSyncAggregate{
			SyncCommitteeBits:      util.FromHexString(resp.Data[0].SyncAggregate.SyncCommitteeBits),
			SyncCommitteeSignature: util.FromHexString(resp.Data[0].SyncAggregate.SyncCommitteeSignature),
		},
		SignatureSlot:           signatureSlot,
		NextSyncCommitteeBranch: nextSyncCommitteeBranch,
		NextSyncCommittee: eth2.ContractSyncCommittee{
			Pubkeys:         pubKeys,
			AggregatePubkey: util.FromHexString(resp.Data[0].NextSyncCommittee.AggregatePubkey),
		},
		FinalityBranch: finalityBranch,
		FinalizedHeader: eth2.BeaconBlockHeader{
			Slot:          fhSlot.Uint64(),
			ProposerIndex: fhProposerIndex.Uint64(),
			ParentRoot:    common.HexToHash(resp.Data[0].FinalizedHeader.ParentRoot),
			StateRoot:     common.HexToHash(resp.Data[0].FinalizedHeader.StateRoot),
			BodyRoot:      common.HexToHash(resp.Data[0].FinalizedHeader.BodyRoot),
		},
		ExeFinalityBranch:  exeFinalityBranch,
		FinalizedExeHeader: *eth2.ConvertHeader(header),
	}, nil
}

func (m *Maintainer) updateHeaders(startNumber, endNumber *big.Int) error {
	m.Log.Info("Sync Header", "startNumber", startNumber, "endNumber", endNumber)
	headers := make([]eth2.BlockHeader, mapprotocol.HeaderLengthOfEth2)
	idx := mapprotocol.HeaderLengthOfEth2 - 1
	for i := endNumber.Int64(); i >= startNumber.Int64(); i-- {
		header, err := m.Conn.Client().HeaderByNumber(context.Background(), new(big.Int).SetInt64(i))
		if err != nil {
			return err
		}

		headers[idx] = *eth2.ConvertHeader(header)
		idx--
		if idx != -1 && i != startNumber.Int64() {
			continue
		}
		if i == startNumber.Int64() {
			headers = headers[idx+1:]
		}
		input, err := mapprotocol.Eth2.Methods[mapprotocol.MethodOfGetHeadersBytes].Inputs.Pack(headers)
		if err != nil {
			m.Log.Error("Failed to header abi pack", "err", err)
			return err
		}

		id := big.NewInt(0).SetUint64(uint64(m.Cfg.Id))
		msgPayload := []interface{}{id, input}
		message := msg.NewSyncToMap(m.Cfg.Id, m.Cfg.MapChainID, msgPayload, m.MsgCh)
		err = m.Router.Send(message)
		if err != nil {
			m.Log.Error("Subscription header error: failed to route message", "err", err)
			return nil
		}
		err = m.WaitUntilMsgHandled(1)
		if err != nil {
			return err
		}
		idx = mapprotocol.HeaderLengthOfEth2 - 1
	}

	return nil
}
