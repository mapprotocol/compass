package eth2

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"strconv"
	"time"

	log "github.com/ChainSafe/log15"
	"github.com/ethereum/go-ethereum/common"
	"github.com/mapprotocol/compass/internal/constant"
	"github.com/mapprotocol/compass/internal/eth2"
	"github.com/mapprotocol/compass/msg"
	"github.com/mapprotocol/compass/pkg/util"

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
	if !m.Cfg.SyncToMap {
		time.Sleep(time.Hour * 2400)
		return nil
	}
	var currentBlock = m.Cfg.StartBlock
	m.Log.Info("Polling Blocks...", "block", currentBlock)

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

	for {
		select {
		case <-m.Stop:
			return errors.New("polling terminated")
		default:
			err := m.updateSyncHeight()
			if err != nil {
				m.Log.Error("UpdateSyncHeight failed", "err", err)
				time.Sleep(constant.BlockRetryInterval)
				continue
			}

			startNumber, endNumber, err := mapprotocol.GetEth22MapNumber(m.Cfg.Id)
			if err != nil {
				m.Log.Error("Get startNumber failed", "err", err)
				time.Sleep(constant.BlockRetryInterval)
				continue
			}

			log.Info("UpdateRange ", "startNumber", startNumber, "endNumber", endNumber)
			if startNumber.Int64() != 0 && endNumber.Int64() != 0 {
				// updateHeader 流程
				err = m.updateHeaders(startNumber, endNumber)
				if err != nil {
					m.Log.Error("updateHeaders failed", "err", err)
					time.Sleep(constant.QueryRetryInterval)
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

			if !m.isEnoughBlocksForLightClientUpdate(lastFinalizedSlotOnContract, lastFinalizedSlotOnEth) {
				time.Sleep(time.Second * 60)
				continue
			}

			latestBlock, err := m.Conn.LatestBlock()
			if err != nil {
				m.Log.Error("Unable to get latest block", "block", currentBlock, "err", err)
				time.Sleep(constant.BlockRetryInterval)
				continue
			}

			err = m.sendRegularLightClientUpdate(lastFinalizedSlotOnContract, lastFinalizedSlotOnEth)
			if err != nil {
				m.Log.Error("Failed to listen header for block", "block", currentBlock, "err", err)
				if !errors.Is(err, constant.ErrUnWantedSync) {
					util.Alarm(context.Background(), fmt.Sprintf("eth2 sync lightClient failed, err is %s", err.Error()))
				}
				time.Sleep(constant.BlockRetryInterval)
				continue
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
				time.Sleep(time.Second * 10)
			} else {
				time.Sleep(time.Millisecond * 20)
			}
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

func (m *Maintainer) isEnoughBlocksForLightClientUpdate(lastFinalizedSlotOnContract, lastFinalizedSlotOnEth *big.Int) bool {
	if (lastFinalizedSlotOnEth.Int64() - lastFinalizedSlotOnContract.Int64()) < (constant.SlotsPerEpoch * 1) {
		m.Log.Info("Light client update were send less then 1 epochs ago. Skipping sending light client update",
			"lastFinalizedSlotOnEth", lastFinalizedSlotOnEth, "lastFinalizedSlotOnContract", lastFinalizedSlotOnContract)
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
	m.Log.Info("Period check", "periodOnContract", lastEth2PeriodOnContract, "endPeriod", endPeriod,
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
		m.Log.Warn(fmt.Sprintf("not enought sync committe count %d", count))
		return nil, constant.ErrUnWantedSync
	}

	//signatureSlot, err := m.getSignatureSlot(resp.Data.AttestedHeader.Beacon.Slot, &resp.Data.SyncAggregate)
	//if err != nil {
	//	return nil, err
	//}
	signatureSlot, err := strconv.ParseUint(resp.Data.SignatureSlot, 10, 64)
	if err != nil {
		return nil, err
	}

	fhSlot, _ := big.NewInt(0).SetString(resp.Data.FinalizedHeader.Beacon.Slot, 10)
	fhProposerIndex, ok := big.NewInt(0).SetString(resp.Data.FinalizedHeader.Beacon.ProposerIndex, 10)
	if !ok {
		return nil, errors.New("FinalizedHeader Slot Not Number")
	}

	if fhSlot.Cmp(lastFinalizedSlotOnContract) <= 0 {
		m.Log.Warn("Finally slot less than slot on contract", "slot", fhSlot.Int64(), "contract.Int64()", lastFinalizedSlotOnContract.Int64())
		return nil, constant.ErrUnWantedSync
	}

	m.Log.Info("Slot compare", "fhSlot", resp.Data.FinalizedHeader.Beacon.Slot, "fsOnContract ", lastFinalizedSlotOnContract)
	slot, _ := big.NewInt(0).SetString(resp.Data.AttestedHeader.Beacon.Slot, 10)
	proposerIndex, ok := big.NewInt(0).SetString(resp.Data.AttestedHeader.Beacon.ProposerIndex, 10)
	if !ok {
		return nil, errors.New("AttestedHeader Slot Not Number")
	}
	finalityBranch := make([][32]byte, 0, len(resp.Data.FinalityBranch))
	for _, fb := range resp.Data.FinalityBranch {
		finalityBranch = append(finalityBranch, common.HexToHash(fb))
	}

	exeFinalityBranch := make([][32]byte, 0)
	execution := &eth2.ContractExecution{}
	//fmt.Println("resp.Version ", resp.Version)
	if resp.Version == "capella" {
		branches := make([]string, 0, len(resp.Data.FinalizedHeader.ExecutionBranch))
		branches = append(branches, resp.Data.FinalizedHeader.ExecutionBranch...)
		exeFinalityBranch = eth2.GenerateByApi(branches)
		execution, err = eth2.ConvertExecution(&resp.Data.FinalizedHeader.Execution)
		if err != nil {
			return nil, err
		}
	} else {
		block, err := m.eth2Client.GetBlocks(context.Background(), resp.Data.FinalizedHeader.Beacon.Slot)
		if err != nil {
			return nil, err
		}
		branches, txRoot, wdRoot, err := eth2.Generate(resp.Data.FinalizedHeader.Beacon.Slot, m.Cfg.Eth2Endpoint)
		if err != nil {
			return nil, err
		}
		exeFinalityBranch = branches
		block.Data.Message.Body.ExecutionPayload.TransactionsRoot = txRoot
		block.Data.Message.Body.ExecutionPayload.WithdrawalsRoot = wdRoot
		execution, err = eth2.ConvertExecution(&block.Data.Message.Body.ExecutionPayload)
		if err != nil {
			return nil, err
		}
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
			ParentRoot:    common.HexToHash(resp.Data.AttestedHeader.Beacon.ParentRoot),
			StateRoot:     common.HexToHash(resp.Data.AttestedHeader.Beacon.StateRoot),
			BodyRoot:      common.HexToHash(resp.Data.AttestedHeader.Beacon.BodyRoot),
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
			ParentRoot:    common.HexToHash(resp.Data.FinalizedHeader.Beacon.ParentRoot),
			StateRoot:     common.HexToHash(resp.Data.FinalizedHeader.Beacon.StateRoot),
			BodyRoot:      common.HexToHash(resp.Data.FinalizedHeader.Beacon.BodyRoot),
		},
		ExecutionBranch:    exeFinalityBranch,
		FinalizedExecution: execution,
	}, nil
}

func (m *Maintainer) getSignatureSlot(slot string, sa *eth2.SyncAggregate) (uint64, error) {
	var CheckSlotsForwardLimit uint64 = 10
	ahSlot, ok := big.NewInt(0).SetString(slot, 10)
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
	slot, _ := big.NewInt(0).SetString(resp.Data.AttestedHeader.Beacon.Slot, 10)
	proposerIndex, ok := big.NewInt(0).SetString(resp.Data.AttestedHeader.Beacon.ProposerIndex, 10)
	if !ok {
		return nil, errors.New("AttestedHeader Slot Not Number")
	}

	nextSyncCommitteeBranch := make([][32]byte, 0, len(resp.Data.NextSyncCommitteeBranch))
	for _, b := range resp.Data.NextSyncCommitteeBranch {
		nextSyncCommitteeBranch = append(nextSyncCommitteeBranch, common.HexToHash(b))
	}
	pubKeys := make([]byte, 0, len(resp.Data.NextSyncCommittee.Pubkeys)*48)
	for _, pk := range resp.Data.NextSyncCommittee.Pubkeys {
		pubKeys = append(pubKeys, util.FromHexString(pk)...)
	}
	finalityBranch := make([][32]byte, 0, len(resp.Data.FinalityBranch))
	for _, fb := range resp.Data.FinalityBranch {
		finalityBranch = append(finalityBranch, common.HexToHash(fb))
	}
	fhSlot, _ := big.NewInt(0).SetString(resp.Data.FinalizedHeader.Beacon.Slot, 10)
	fhProposerIndex, ok := big.NewInt(0).SetString(resp.Data.FinalizedHeader.Beacon.ProposerIndex, 10)
	if !ok {
		return nil, errors.New("FinalizedHeader  Slot Not Number")
	}

	exeFinalityBranch := make([][32]byte, 0)
	execution := &eth2.ContractExecution{}
	signatureSlot, err := strconv.ParseUint(resp.Data.SignatureSlot, 10, 64)
	if err != nil {
		return nil, err
	}
	//fmt.Println("resp.Version ", resp.Version)
	if resp.Version == "capella" {
		branches := make([]string, 0, len(resp.Data.FinalizedHeader.ExecutionBranch))
		branches = append(branches, resp.Data.FinalizedHeader.ExecutionBranch...)
		exeFinalityBranch = eth2.GenerateByApi(branches)
		execution, err = eth2.ConvertExecution(&resp.Data.FinalizedHeader.Execution)
		if err != nil {
			return nil, err
		}
	} else {
		block, err := m.eth2Client.GetBlocks(context.Background(), resp.Data.FinalizedHeader.Beacon.Slot)
		if err != nil {
			return nil, err
		}
		branches, txRoot, wdRoot, err := eth2.Generate(resp.Data.FinalizedHeader.Beacon.Slot, m.Cfg.Eth2Endpoint)
		if err != nil {
			return nil, err
		}
		exeFinalityBranch = branches
		block.Data.Message.Body.ExecutionPayload.TransactionsRoot = txRoot
		block.Data.Message.Body.ExecutionPayload.WithdrawalsRoot = wdRoot
		execution, err = eth2.ConvertExecution(&block.Data.Message.Body.ExecutionPayload)
		if err != nil {
			return nil, err
		}
	}
	return &eth2.LightClientUpdate{
		AttestedHeader: eth2.BeaconBlockHeader{
			Slot:          slot.Uint64(),
			ProposerIndex: proposerIndex.Uint64(),
			ParentRoot:    common.HexToHash(resp.Data.AttestedHeader.Beacon.ParentRoot),
			StateRoot:     common.HexToHash(resp.Data.AttestedHeader.Beacon.StateRoot),
			BodyRoot:      common.HexToHash(resp.Data.AttestedHeader.Beacon.BodyRoot),
		},
		SyncAggregate: eth2.ContractSyncAggregate{
			SyncCommitteeBits:      util.FromHexString(resp.Data.SyncAggregate.SyncCommitteeBits),
			SyncCommitteeSignature: util.FromHexString(resp.Data.SyncAggregate.SyncCommitteeSignature),
		},
		SignatureSlot:           signatureSlot,
		NextSyncCommitteeBranch: nextSyncCommitteeBranch,
		NextSyncCommittee: eth2.ContractSyncCommittee{
			Pubkeys:         pubKeys,
			AggregatePubkey: util.FromHexString(resp.Data.NextSyncCommittee.AggregatePubkey),
		},
		FinalityBranch: finalityBranch,
		FinalizedHeader: eth2.BeaconBlockHeader{
			Slot:          fhSlot.Uint64(),
			ProposerIndex: fhProposerIndex.Uint64(),
			ParentRoot:    common.HexToHash(resp.Data.FinalizedHeader.Beacon.ParentRoot),
			StateRoot:     common.HexToHash(resp.Data.FinalizedHeader.Beacon.StateRoot),
			BodyRoot:      common.HexToHash(resp.Data.FinalizedHeader.Beacon.BodyRoot),
		},
		ExecutionBranch:    exeFinalityBranch,
		FinalizedExecution: execution,
	}, nil
}

func (m *Maintainer) updateHeaders(startNumber, endNumber *big.Int) error {
	m.Log.Info("Sync Header", "startNumber", startNumber, "endNumber", endNumber)
	headers := make([]eth2.BlockHeader, mapprotocol.HeaderLengthOfEth2)
	idx := mapprotocol.HeaderLengthOfEth2 - 1
	for i := endNumber.Int64(); i >= startNumber.Int64(); i-- {
		header, err := m.Conn.Client().EthLatestHeaderByNumber(m.Cfg.Endpoint, new(big.Int).SetInt64(i))
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
		time.Sleep(time.Second * 2)
	}

	return nil
}
