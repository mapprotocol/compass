package conflux

import (
	"context"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/mapprotocol/compass/internal/conflux/mpt"
	primitives "github.com/mapprotocol/compass/internal/conflux/primipives"
	"github.com/mapprotocol/compass/internal/conflux/types"
	"github.com/mapprotocol/compass/mapprotocol"
	"github.com/mapprotocol/compass/msg"
	"github.com/pkg/errors"
	"io"
	"math/big"
)

const DeferredExecutionEpochs uint64 = 5

var ErrTransactionExecutionFailed = errors.New("transaction execution failed")

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

	result.AggregatedSignature = ABIEncodeSignature(ledger.AggregatedSignature)
	for _, v := range ledger.ValidatorsSorted() {
		result.Accounts = append(result.Accounts, v)
	}

	return result
}

func ABIEncodeSignature(signature []byte) []byte {
	if len(signature) != 192 {
		return signature
	}

	encoded := make([]byte, 256)

	copy(encoded[16:64], signature[:48])
	copy(encoded[80:128], signature[48:96])
	copy(encoded[144:192], signature[96:144])
	copy(encoded[208:256], signature[144:192])

	return encoded
}

func ABIEncodePublicKey(publicKey []byte) []byte {
	if len(publicKey) != 96 {
		return publicKey
	}

	encoded := make([]byte, 128)

	copy(encoded[16:64], publicKey[:48])
	copy(encoded[80:128], publicKey[48:])

	return encoded
}

func ConvertCommittee(ledger *LedgerInfoWithSignatures) (LedgerInfoLibEpochState, bool) {
	if ledger == nil {
		return LedgerInfoLibEpochState{}, false
	}

	state := ledger.LedgerInfo.CommitInfo.NextEpochState
	if state == nil {
		return LedgerInfoLibEpochState{}, false
	}

	var validators []LedgerInfoLibValidatorInfo
	for _, v := range ledger.NextEpochValidatorsSorted() {
		info := state.Verifier.AddressToValidatorInfo[v]

		uncompressedPubKey, ok := ledger.NextEpochValidators[v]
		if !ok {
			return LedgerInfoLibEpochState{}, false
		}

		validator := LedgerInfoLibValidatorInfo{
			Account:               v,
			UncompressedPublicKey: ABIEncodePublicKey(uncompressedPubKey),
			VotingPower:           uint64(info.VotingPower),
		}

		if info.VrfPublicKey != nil {
			validator.VrfPublicKey = *info.VrfPublicKey
		}

		validators = append(validators, validator)
	}

	return LedgerInfoLibEpochState{
		Epoch:             uint64(state.Epoch),
		Validators:        validators,
		QuorumVotingPower: uint64(state.Verifier.QuorumVotingPower),
		TotalVotingPower:  uint64(state.Verifier.TotalVotingPower),
		VrfSeed:           state.VrfSeed,
	}, true
}

func MustRLPEncodeBlock(block *BlockSummary) []byte {
	val := ConvertBlock(block)
	encoded, err := rlp.EncodeToBytes(val)
	if err != nil {
		panic(err)
	}
	return encoded
}

func ConvertBlock(block *BlockSummary) BlockRlp {
	return BlockRlp{block}
}

type BlockRlp struct {
	raw *BlockSummary
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

func AssembleProof(client *Client, txHash common.Hash, epochNumber, pivot uint64, method string, fId msg.ChainId) ([]byte, error) {
	if epochNumber+DeferredExecutionEpochs > pivot {
		return nil, errors.New("Pivot less than current block")
	}

	epoch := types.NewEpochNumberUint64(epochNumber)
	epochOrHash := types.NewEpochOrBlockHashWithEpoch(epoch)
	epochReceipts, err := client.GetEpochReceipts(context.Background(), *epochOrHash, true)
	if err != nil {
		return nil, errors.WithMessagef(err, "Failed to get receipts by epoch number %v", epochNumber)
	}

	blockIndex, receipt := matchReceipt(epochReceipts, txHash.Hex())
	if receipt == nil {
		return nil, nil
	}

	if receipt.MustGetOutcomeType() != types.TRANSACTION_OUTCOME_SUCCESS {
		return nil, ErrTransactionExecutionFailed
	}

	subtrees, root := CreateReceiptsMPT(epochReceipts)

	blockIndexKey := mpt.IndexToKey(blockIndex, len(subtrees))
	blockProof, ok := root.Proof(blockIndexKey)
	if !ok {
		return nil, errors.New("Failed to generate block proof")
	}

	receiptsRoot := subtrees[blockIndex].Hash()
	receiptKey := mpt.IndexToKey(int(receipt.Index), len(epochReceipts[blockIndex]))
	receiptProof, ok := subtrees[blockIndex].Proof(receiptKey)
	if !ok {
		return nil, errors.New("Failed to generate receipt proof")
	}

	var headers [][]byte
	// 195 - 200
	for i := epochNumber + DeferredExecutionEpochs; i <= pivot; i++ {
		block, err := client.GetBlockByEpochNumber(context.Background(), hexutil.Uint64(i))
		if err != nil {
			return nil, errors.WithMessagef(err, "Failed to get block summary by epoch %v", i)
		}

		headers = append(headers, MustRLPEncodeBlock(block))
	}

	proof := &mpt.TypesReceiptProof{
		Headers:      headers,
		BlockIndex:   blockIndexKey,
		BlockProof:   mpt.ConvertProofNode(blockProof),
		ReceiptsRoot: receiptsRoot,
		Index:        receiptKey,
		Receipt:      primitives.MustRLPEncodeReceipt(receipt),
		ReceiptProof: mpt.ConvertProofNode(receiptProof),
	}
	input, err := mapprotocol.Conflux.Methods[mapprotocol.MethodOfVerifyReceiptProof].Inputs.Pack(proof)
	if err != nil {
		return nil, err
	}
	//fmt.Println("proof --------------------- ", common.Bytes2Hex(input))
	d, _ := rlp.EncodeToBytes(primitives.MustRLPEncodeReceipt(receipt))
	fmt.Println("-------------", "0x"+common.Bytes2Hex(d))

	//pack, err := mapprotocol.PackInput(mapprotocol.Mcs, method, new(big.Int).SetUint64(uint64(fId)), input)
	pack, err := mapprotocol.PackInput(mapprotocol.LightManger, mapprotocol.MethodVerifyProofData, new(big.Int).SetUint64(uint64(fId)), input)
	if err != nil {
		return nil, err
	}

	return pack, nil
}

func matchReceipt(epochReceipts [][]types.TransactionReceipt, txHash string) (blockIndex int, receipt *types.TransactionReceipt) {
	for i, blockReceipts := range epochReceipts {
		for _, v := range blockReceipts {
			if v.MustGetOutcomeType() == types.TRANSACTION_OUTCOME_SKIPPED {
				continue
			}

			if v.TransactionHash.String() != txHash {
				continue
			}

			return i, &v
		}
	}

	return 0, nil
}

func CreateReceiptsMPT(epochReceipts [][]types.TransactionReceipt) ([]*mpt.Node, *mpt.Node) {
	var subtrees []*mpt.Node

	for _, blockReceipts := range epochReceipts {
		var root mpt.Node

		keyLen := mpt.MinReprBytes(len(blockReceipts))

		for i, v := range blockReceipts {
			key := mpt.IndexToKey(i, keyLen)
			value := primitives.MustRLPEncodeReceipt(&v)
			root.Insert(key, value)
		}

		subtrees = append(subtrees, &root)
	}

	var root mpt.Node
	keyLen := mpt.MinReprBytes(len(subtrees))
	for i, v := range subtrees {
		key := mpt.IndexToKey(i, keyLen)
		value := v.Hash().Bytes()
		root.Insert(key, value)
	}

	return subtrees, &root
}
