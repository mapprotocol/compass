package conflux

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rlp"
	"io"
	"sort"
)

func ConvertLedger(ledger *LedgerInfoWithSignatures) LedgerInfoLibLedgerInfoWithSignatures {
	committee, _ := ConvertCommittee(ledger)

	result := LedgerInfoLibLedgerInfoWithSignatures{
		Epoch:             uint64(ledger.LedgerInfo.CommitInfo.Epoch),
		Round:             uint64(ledger.LedgerInfo.CommitInfo.Round),
		Id:                common.BytesToHash(ledger.LedgerInfo.CommitInfo.Id),
		ExecutedStateId:   common.BytesToHash(ledger.LedgerInfo.CommitInfo.ExecutedStateId),
		Version:           uint64(ledger.LedgerInfo.CommitInfo.Version),
		TimestampUsecs:    uint64(ledger.LedgerInfo.CommitInfo.TimestampUsecs),
		NextEpochState:    committee,
		ConsensusDataHash: common.BytesToHash(ledger.LedgerInfo.ConsensusDataHash),
	}

	if pivot := ledger.LedgerInfo.CommitInfo.Pivot; pivot != nil {
		result.Pivot.Height = uint64(pivot.Height)
		result.Pivot.BlockHash = pivot.BlockHash.ToHash()
	}

	var signatures sortableAccountSignatures
	for k, v := range ledger.Signatures {
		signatures = append(signatures, LedgerInfoLibAccountSignature{
			Account:            k,
			ConsensusSignature: v,
		})
	}
	sort.Sort(signatures)
	result.Signatures = signatures

	return result
}

func ConvertCommittee(ledger *LedgerInfoWithSignatures) (LedgerInfoLibEpochState, bool) {
	if ledger == nil {
		return LedgerInfoLibEpochState{}, false
	}

	state := ledger.LedgerInfo.CommitInfo.NextEpochState
	if state == nil {
		return LedgerInfoLibEpochState{}, false
	}

	var validators sortableValidators
	for k, v := range state.Verifier.AddressToValidatorInfo {
		validator := LedgerInfoLibValidatorInfo{
			Account:               k,
			CompressedPublicKey:   v.PublicKey,
			UncompressedPublicKey: ledger.NextEpochValidators[k],
			VotingPower:           uint64(v.VotingPower),
		}

		if len(validator.UncompressedPublicKey) == 0 {
			return LedgerInfoLibEpochState{}, false
		}

		if v.VrfPublicKey != nil {
			validator.VrfPublicKey = *v.VrfPublicKey
		}

		validators = append(validators, validator)
	}
	sort.Sort(validators)

	return LedgerInfoLibEpochState{
		Epoch:             uint64(state.Epoch),
		Validators:        validators,
		QuorumVotingPower: uint64(state.Verifier.QuorumVotingPower),
		TotalVotingPower:  uint64(state.Verifier.TotalVotingPower),
		VrfSeed:           state.VrfSeed,
	}, true
}

func MustRLPEncodeBlock(block *CfxBlock) []byte {
	val := ConvertBlock(block)
	encoded, err := rlp.EncodeToBytes(val)
	if err != nil {
		panic(err)
	}
	return encoded
}

func ConvertBlock(block *CfxBlock) BlockRlp {
	return BlockRlp{block}
}

type BlockRlp struct {
	raw *CfxBlock
}

func (header BlockRlp) EncodeRLP(w io.Writer) error {
	var adaptive uint64
	if header.raw.Adaptive {
		adaptive = 1
	}

	var referees []common.Hash
	for _, v := range header.raw.RefereeHashes {
		referees = append(referees, *v.ToCommonHash())
	}

	list := []interface{}{
		header.raw.ParentHash.ToCommonHash(),
		header.raw.Height.ToInt(),
		header.raw.Timestamp.ToInt(),
		header.raw.Miner.MustGetCommonAddress(),
		header.raw.TransactionsRoot.ToCommonHash(),
		header.raw.DeferredStateRoot.ToCommonHash(),
		header.raw.DeferredReceiptsRoot.ToCommonHash(),
		header.raw.DeferredLogsBloomHash.ToCommonHash(),
		header.raw.Blame,
		header.raw.Difficulty.ToInt(),
		adaptive,
		header.raw.GasLimit.ToInt(),
		referees,
		header.raw.Nonce.ToInt(),
	}

	if header.raw.PosReference != nil {
		// simulate RLP encoding for rust Option type
		item := []common.Hash{*header.raw.PosReference.ToCommonHash()}
		list = append(list, item)
	}

	for _, v := range header.raw.Custom {
		list = append(list, rlp.RawValue(v))
	}

	return rlp.Encode(w, list)
}
