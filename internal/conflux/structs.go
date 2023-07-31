package conflux

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/mapprotocol/compass/internal/conflux/types"
	"github.com/pkg/errors"
	"io"
	"math/big"
	"regexp"
)

type Status struct {
	LatestCommitted hexutil.Uint64  `json:"latestCommitted"`
	Epoch           hexutil.Uint64  `json:"epoch"`
	PivotDecision   Decision        `json:"pivotDecision"`
	LatestVoted     *hexutil.Uint64 `json:"latestVoted"`
	LatestTxNumber  hexutil.Uint64  `json:"latestTxNumber"`
}

type Decision struct {
	BlockHash common.Hash    `json:"blockHash"`
	Height    hexutil.Uint64 `json:"height"`
}

var (
	BlockEarliest        = &BlockNumber{"earliest", 0}
	BlockLatestCommitted = &BlockNumber{"latest_committed", 0}
	BlockLatestVoted     = &BlockNumber{"latest_voted", 0}
)

type BlockNumber struct {
	name   string
	number hexutil.Uint64
}

// String implements the fmt.Stringer interface
func (e *BlockNumber) String() string {
	if e.name == "" {
		return e.number.String()
	}

	return e.name
}

// MarshalText implements the encoding.TextMarshaler interface.
func (e BlockNumber) MarshalText() ([]byte, error) {
	// fmt.Println("marshal text for epoch")
	return []byte(e.String()), nil
}

// UnmarshalJSON implements the json.Unmarshaler interface.
func (e *BlockNumber) UnmarshalJSON(data []byte) error {
	var input string

	if err := json.Unmarshal(data, &input); err != nil {
		return err
	}

	hexU64Pattern := `(?i)^0x[a-f0-9]*$`
	if ok, _ := regexp.Match(hexU64Pattern, []byte(input)); ok {
		numU64, err := hexutil.DecodeUint64(input)
		if err != nil {
			return errors.WithStack(err)
		}
		e.number = hexutil.Uint64(numU64)
		return nil
	}

	switch input {
	case BlockEarliest.name, BlockLatestCommitted.name, BlockLatestVoted.name:
		e.name = input
		return nil
	}

	return fmt.Errorf(`unsupported pos block number tag %v`, input)
}

type Address = common.Hash

type Signature struct {
	Account Address        `json:"account"`
	Votes   hexutil.Uint64 `json:"votes"`
}

type Block struct {
	Hash          common.Hash    `json:"hash"`
	Height        hexutil.Uint64 `json:"height"`
	Epoch         hexutil.Uint64 `json:"epoch"`
	Round         hexutil.Uint64 `json:"round"`
	LastTxNumber  hexutil.Uint64 `json:"lastTxNumber"`
	Miner         *Address       `json:"miner"`
	ParentHash    common.Hash    `json:"parentHash"`
	Timestamp     hexutil.Uint64 `json:"timestamp"`
	PivotDecision *Decision      `json:"pivotDecision"`
	Signatures    []Signature    `json:"signatures"`
}

func NewBlockNumber(number hexutil.Uint64) BlockNumber {
	return BlockNumber{"", number}
}

type LedgerInfo struct {
	CommitInfo BlockInfo `json:"commitInfo"`

	/// Hash of consensus specific data that is opaque to all parts of the
	/// system other than consensus.
	ConsensusDataHash hexutil.Bytes `json:"consensusDataHash"`
}

type BlockInfo struct {
	Epoch           hexutil.Uint64      `json:"epoch"`
	Round           hexutil.Uint64      `json:"round"`
	Id              hexutil.Bytes       `json:"id"`
	ExecutedStateId hexutil.Bytes       `json:"executedStateId"`
	Version         hexutil.Uint64      `json:"version"`
	TimestampUsecs  hexutil.Uint64      `json:"timestampUsecs"`
	NextEpochState  *EpochState         `json:"nextEpochState"`
	Pivot           *PivotBlockDecision `json:"pivot"`
}

type PivotBlockDecision struct {
	Height    hexutil.Uint64 `json:"height"`
	BlockHash H256           `json:"blockHash"`
}

type H256 string

func (h H256) ToHash() common.Hash {
	return common.HexToHash(string(h))
}

func (h H256) String() string {
	return string(h)
}

type EpochState struct {
	Epoch    hexutil.Uint64    `json:"epoch"`
	Verifier ValidatorVerifier `json:"verifier"`
	VrfSeed  hexutil.Bytes     `json:"vrfSeed"`
}

type ValidatorVerifier struct {
	// An ordered map of each validator's on-chain account address to its
	// pubkeys and voting power.
	AddressToValidatorInfo map[common.Hash]ValidatorConsensusInfo `json:"addressToValidatorInfo"`
	// The minimum voting power required to achieve a quorum
	QuorumVotingPower hexutil.Uint64 `json:"quorumVotingPower"`
	// Total voting power of all validators (cached from
	// address_to_validator_info)
	TotalVotingPower hexutil.Uint64 `json:"totalVotingPower"`
}

type ValidatorConsensusInfo struct {
	PublicKey    hexutil.Bytes  `json:"publicKey"`              // compressed BLS public key
	VrfPublicKey *hexutil.Bytes `json:"vrfPublicKey,omitempty"` // nil if VRF not needed
	VotingPower  hexutil.Uint64 `json:"votingPower"`
}

type LedgerInfoWithSignatures struct {
	LedgerInfo LedgerInfo `json:"ledgerInfo"`
	// The validator is identified by its account address: in order to verify
	// a signature one needs to retrieve the public key of the validator
	// for the given epoch.
	//
	// BLS signature in uncompressed format
	Signatures map[common.Hash]hexutil.Bytes `json:"signatures"`
	// Validators with uncompressed BLS public key (in 96 bytes) if next epoch
	// state available. Generally, this is used to verify BLS signatures at client side.
	NextEpochValidators map[common.Hash]hexutil.Bytes `json:"nextEpochValidators"`
}

// ILightNodeState is an auto generated low-level Go binding around an user-defined struct.
type ILightNodeState struct {
	Epoch                *big.Int
	Round                *big.Int
	EarliestBlockNumber  *big.Int
	FinalizedBlockNumber *big.Int
	Blocks               *big.Int
	MaxBlocks            *big.Int
}

// LedgerInfoLibLedgerInfoWithSignatures is an auto generated low-level Go binding around an user-defined struct.
type LedgerInfoLibLedgerInfoWithSignatures struct {
	Epoch             uint64
	Round             uint64
	Id                [32]byte
	ExecutedStateId   [32]byte
	Version           uint64
	TimestampUsecs    uint64
	NextEpochState    LedgerInfoLibEpochState
	Pivot             LedgerInfoLibDecision
	ConsensusDataHash [32]byte
	//Accounts            [][32]byte
	//AggregatedSignature []byte
	Signatures []LedgerInfoLibAccountSignature
}

/*
type LedgerInfoLibLedgerInfoWithSignatures struct {
	Epoch               uint64
	Round               uint64
	Id                  [32]byte
	ExecutedStateId     [32]byte
	Version             uint64
	TimestampUsecs      uint64
	NextEpochState      LedgerInfoLibEpochState
	Pivot               LedgerInfoLibDecision
	ConsensusDataHash   [32]byte
	Accounts            [][32]byte
	AggregatedSignature []byte
}
*/

type LedgerInfoLibEpochState struct {
	Epoch             uint64
	Validators        []LedgerInfoLibValidatorInfo
	QuorumVotingPower uint64
	TotalVotingPower  uint64
	VrfSeed           []byte
}

type LedgerInfoLibDecision struct {
	BlockHash [32]byte
	Height    uint64
}

type LedgerInfoLibAccountSignature struct {
	Account            [32]byte
	ConsensusSignature []byte
}

type LedgerInfoLibValidatorInfo struct {
	Account               [32]byte
	CompressedPublicKey   []byte
	UncompressedPublicKey []byte
	VrfPublicKey          []byte
	VotingPower           uint64
}

type sortableAccountSignatures []LedgerInfoLibAccountSignature

func (s sortableAccountSignatures) Len() int { return len(s) }
func (s sortableAccountSignatures) Less(i, j int) bool {
	return bytes.Compare(s[i].Account[:], s[j].Account[:]) < 0
}
func (s sortableAccountSignatures) Swap(i, j int) { s[i], s[j] = s[j], s[i] }

type sortableValidators []LedgerInfoLibValidatorInfo

func (s sortableValidators) Len() int { return len(s) }
func (s sortableValidators) Less(i, j int) bool {
	return bytes.Compare(s[i].Account[:], s[j].Account[:]) < 0
}
func (s sortableValidators) Swap(i, j int) { s[i], s[j] = s[j], s[i] }

// Epoch represents an epoch in Conflux.
type Epoch struct {
	name   string
	number *hexutil.Big
}

// Const epoch definitions
var (
	EpochEarliest         *Epoch = &Epoch{"earliest", nil}
	EpochLatestCheckpoint *Epoch = &Epoch{"latest_checkpoint", nil}
	EpochLatestConfirmed  *Epoch = &Epoch{"latest_confirmed", nil}
	EpochLatestState      *Epoch = &Epoch{"latest_state", nil}
	EpochLatestMined      *Epoch = &Epoch{"latest_mined", nil}
	EpochLatestFinalized  *Epoch = &Epoch{"latest_finalized", nil}
)

// NewEpochNumber creates an instance of Epoch with specified number.
func NewEpochNumber(number *hexutil.Big) *Epoch {
	return &Epoch{"", number}
}

func (e *Epoch) String() string {
	if e.number != nil {
		return e.number.String()
	}

	return e.name
}

func (e *Epoch) ToInt() (result *big.Int, isSuccess bool) {
	if e.number != nil {
		return e.number.ToInt(), true
	}

	if e.name == EpochEarliest.name {
		return common.Big0, true
	}

	return nil, false
}

func (e *Epoch) Equals(target *Epoch) bool {
	if e == nil {
		panic("input could not be nil")
	}

	if target == nil {
		return false
	}

	if e == target {
		return true
	}

	if len(e.name) > 0 || len(target.name) > 0 {
		return e.name == target.name
	}

	if e.number == nil || target.number == nil {
		return e.number == target.number
	}

	return e.number.ToInt().Cmp(target.number.ToInt()) == 0
}

func (e Epoch) MarshalText() ([]byte, error) {
	return []byte(e.String()), nil
}

func (e *Epoch) UnmarshalJSON(data []byte) error {
	var input string
	if err := json.Unmarshal(data, &input); err != nil {
		return err
	}

	switch input {
	case EpochEarliest.name,
		EpochLatestCheckpoint.name,
		EpochLatestConfirmed.name,
		EpochLatestState.name,
		EpochLatestMined.name,
		EpochLatestFinalized.name:
		e.name = input
		return nil
	default:
		if len(input) == 66 {
			e.name = input
			return nil
		}

		epochNumber, err := hexutil.DecodeBig(input)
		if err != nil {
			return err
		}

		e.number = NewBigIntByRaw(epochNumber)
		return nil
	}
}

func NewBigIntByRaw(x *big.Int) *hexutil.Big {
	if x == nil {
		return nil
	}
	v := hexutil.Big(*x)
	return &v
}

// BlockSummary includes block header and a list of transaction hashes
type BlockSummary struct {
	BlockHeader
	//Transactions []Hash `json:"transactions"`
}

// rlpEncodableBlockSummary block summary struct used for rlp encoding
type rlpEncodableBlockSummary struct {
	BlockHeader BlockHeader
	//Transactions []Hash
}

// EncodeRLP implements the rlp.Encoder interface.
func (bs BlockSummary) EncodeRLP(w io.Writer) error {
	rbs := rlpEncodableBlockSummary{
		bs.BlockHeader,
		//bs.Transactions,
	}

	return rlp.Encode(w, rbs)
}

// DecodeRLP implements the rlp.Decoder interface.
func (bs *BlockSummary) DecodeRLP(r *rlp.Stream) error {
	var rbs rlpEncodableBlockSummary
	if err := r.Decode(&rbs); err != nil {
		return err
	}

	bs.BlockHeader = rbs.BlockHeader
	//bs.Transactions = rbs.Transactions

	return nil
}

type rlpNilableBigInt struct {
	Val *big.Int
}

type rlpEncodableBlockHeader struct {
	Hash                  types.Hash
	ParentHash            types.Hash
	Height                *big.Int
	Miner                 types.Address
	DeferredStateRoot     types.Hash
	DeferredReceiptsRoot  types.Hash
	DeferredLogsBloomHash types.Hash
	Blame                 hexutil.Uint64
	TransactionsRoot      types.Hash
	EpochNumber           *big.Int
	BlockNumber           *rlpNilableBigInt `rlp:"nil"`
	GasLimit              *big.Int
	GasUsed               *rlpNilableBigInt `rlp:"nil"`
	Timestamp             *big.Int
	Difficulty            *big.Int
	PowQuality            *big.Int
	RefereeHashes         []types.Hash
	Adaptive              bool
	Nonce                 *big.Int
	Size                  *big.Int
	Custom                []Bytes
	PosReference          *types.Hash `rlp:"nil"`
}

// EncodeRLP implements the rlp.Encoder interface.
func (bh BlockHeader) EncodeRLP(w io.Writer) error {
	rbh := rlpEncodableBlockHeader{
		Hash:                  bh.Hash,
		ParentHash:            bh.ParentHash,
		Height:                bh.Height.ToInt(),
		Miner:                 bh.Miner,
		DeferredStateRoot:     bh.DeferredStateRoot,
		DeferredReceiptsRoot:  bh.DeferredReceiptsRoot,
		DeferredLogsBloomHash: bh.DeferredLogsBloomHash,
		Blame:                 bh.Blame,
		TransactionsRoot:      bh.TransactionsRoot,
		EpochNumber:           bh.EpochNumber.ToInt(),
		GasLimit:              bh.GasLimit.ToInt(),
		Timestamp:             bh.Timestamp.ToInt(),
		Difficulty:            bh.Difficulty.ToInt(),
		PowQuality:            bh.PowQuality.ToInt(),
		RefereeHashes:         bh.RefereeHashes,
		Adaptive:              bh.Adaptive,
		Nonce:                 bh.Nonce.ToInt(),
		Size:                  bh.Size.ToInt(),
		Custom:                bh.Custom,
		PosReference:          bh.PosReference,
	}

	if bh.BlockNumber != nil {
		rbh.BlockNumber = &rlpNilableBigInt{bh.BlockNumber.ToInt()}
	}

	if bh.GasUsed != nil {
		rbh.GasUsed = &rlpNilableBigInt{bh.GasUsed.ToInt()}
	}

	return rlp.Encode(w, rbh)
}

// DecodeRLP implements the rlp.Decoder interface.
func (bh *BlockHeader) DecodeRLP(r *rlp.Stream) error {
	var rbh rlpEncodableBlockHeader
	if err := r.Decode(&rbh); err != nil {
		return err
	}

	bh.Hash, bh.ParentHash, bh.Height = rbh.Hash, rbh.ParentHash, (*hexutil.Big)(rbh.Height)
	bh.Miner, bh.DeferredStateRoot = rbh.Miner, rbh.DeferredStateRoot
	bh.DeferredReceiptsRoot, bh.DeferredLogsBloomHash = rbh.DeferredReceiptsRoot, rbh.DeferredLogsBloomHash
	bh.Blame, bh.TransactionsRoot = rbh.Blame, rbh.TransactionsRoot
	bh.EpochNumber = (*hexutil.Big)(rbh.EpochNumber)
	bh.GasLimit = (*hexutil.Big)(rbh.GasLimit)
	bh.Timestamp = (*hexutil.Big)(rbh.Timestamp)
	bh.Difficulty, bh.PowQuality = (*hexutil.Big)(rbh.Difficulty), (*hexutil.Big)(rbh.PowQuality)
	bh.RefereeHashes, bh.Adaptive = rbh.RefereeHashes, rbh.Adaptive
	bh.Nonce, bh.Size, bh.Custom = (*hexutil.Big)(rbh.Nonce), (*hexutil.Big)(rbh.Size), rbh.Custom
	bh.PosReference = rbh.PosReference

	if rbh.BlockNumber != nil {
		bh.BlockNumber = (*hexutil.Big)(rbh.BlockNumber.Val)
	}

	if rbh.GasUsed != nil {
		bh.GasUsed = (*hexutil.Big)(rbh.GasUsed.Val)
	}

	return nil
}
