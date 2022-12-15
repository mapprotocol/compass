package eth2

import (
	"github.com/ethereum/go-ethereum/common"
	"math/big"
)

type LightClientUpdate struct {
	AttestedHeader          BeaconBlockHeader     `json:"attested_header"`
	SignatureSlot           uint64                `json:"signature_slot"` // todo 难点
	SyncAggregate           ContractSyncAggregate `json:"sync_aggregate"`
	NextSyncCommittee       ContractSyncCommittee `json:"next_sync_committee"`
	NextSyncCommitteeBranch [][32]byte            `json:"next_sync_committee_branch"`
	FinalizedHeader         BeaconBlockHeader
	FinalityBranch          [][32]byte
	ExeFinalityBranch       [][32]byte
	FinalizedExeHeader      BlockHeader
}

type BeaconBlockHeader struct {
	Slot          uint64   `json:"slot"`
	ProposerIndex uint64   `json:"proposer_index"`
	ParentRoot    [32]byte `json:"parent_root"` // bytes32
	StateRoot     [32]byte `json:"state_root"`  // bytes32
	BodyRoot      [32]byte `json:"body_root"`
}

type ContractSyncAggregate struct {
	SyncCommitteeBits      string `json:"sync_committee_bits"`
	SyncCommitteeSignature string `json:"sync_committee_signature"`
}

type ContractSyncCommittee struct {
	PubKeys         []byte // 48 * 512
	AggregatePubKey []byte // 48
}

type FinalizedHeaderUpdate struct {
	HeaderUpdate   HeaderUpdate `json:"header_update"`
	FinalityBranch []string     `json:"finality_branch"`
}

type HeaderUpdate struct {
	BeaconHeader        AttestedHeader `json:"beacon_header"`
	ExecutionBlockHash  string         `json:"execution_block_hash"`
	ExecutionHashBranch []string       `json:"execution_hash_branch"`
}

type SyncCommitteeUpdate struct {
	NextSyncCommittee       NextSyncCommittee `json:"next_sync_committee"`
	NextSyncCommitteeBranch []string          `json:"next_sync_committee_branch"`
}

type BlockHeader struct {
	ParentHash       []byte         `json:"parent_hash"`
	Sha3Uncles       []byte         `json:"sha_3_uncles"`
	Miner            common.Address `json:"miner"`
	StateRoot        []byte         `json:"stateRoot"`
	TransactionsRoot []byte         `json:"transactionsRoot"`
	ReceiptsRoot     []byte         `json:"receiptsRoot"`
	LogsBloom        []byte         `json:"logsBloom"`
	Difficulty       *big.Int       `json:"difficulty"`
	Number           *big.Int       `json:"number"`
	GasLimit         *big.Int       `json:"gasLimit"`
	GasUsed          *big.Int       `json:"gasUsed"`
	Timestamp        *big.Int       `json:"timestamp"`
	ExtraData        []byte         `json:"extraData"`
	MixHash          []byte         `json:"mixHash"`
	Nonce            []byte         `json:"nonce"`
	BaseFeePerGas    *big.Int       `json:"baseFeePerGas"`
}
