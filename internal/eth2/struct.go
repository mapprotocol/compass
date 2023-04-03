package eth2

import (
	"fmt"
	"math/big"
	"strings"

	"github.com/mapprotocol/compass/pkg/ethclient"

	"github.com/pkg/errors"

	"github.com/ethereum/go-ethereum/common"
)

type LightClientUpdate struct {
	AttestedHeader          BeaconBlockHeader     `json:"attested_header"`
	SignatureSlot           uint64                `json:"signature_slot"`
	SyncAggregate           ContractSyncAggregate `json:"sync_aggregate"`
	NextSyncCommittee       ContractSyncCommittee `json:"nextSyncCommittee"`
	NextSyncCommitteeBranch [][32]byte            `json:"nextSyncCommitteeBranch"`
	FinalizedHeader         BeaconBlockHeader
	FinalityBranch          [][32]byte
	ExecutionBranch         [][32]byte
	FinalizedExecution      *ContractExecution
}

type BeaconBlockHeader struct {
	Slot          uint64   `json:"slot"`
	ProposerIndex uint64   `json:"proposer_index"`
	ParentRoot    [32]byte `json:"parent_root"` // bytes32
	StateRoot     [32]byte `json:"state_root"`  // bytes32
	BodyRoot      [32]byte `json:"body_root"`
}

type ContractSyncAggregate struct {
	SyncCommitteeBits      []byte `json:"sync_committee_bits"`
	SyncCommitteeSignature []byte `json:"sync_committee_signature"`
}

type ContractSyncCommittee struct {
	Pubkeys         []byte // 48 * 512
	AggregatePubkey []byte // 48
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
	WithdrawalsRoot  [32]byte       `json:"withdrawalsRoot"`
}

func ConvertHeader(header *ethclient.Header) *BlockHeader {
	withdrawalsRoot := common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000000")
	if header.WithdrawalsHash != "" {
		withdrawalsRoot = common.HexToHash(header.WithdrawalsHash)
	}
	fmt.Println("withdrawalsRoot ----------- ", withdrawalsRoot)
	return &BlockHeader{
		ParentHash:       header.ParentHash.Bytes(),
		Sha3Uncles:       header.UncleHash.Bytes(),
		Miner:            header.Coinbase,
		StateRoot:        header.Root.Bytes(),
		TransactionsRoot: header.TxHash.Bytes(),
		ReceiptsRoot:     header.ReceiptHash.Bytes(),
		LogsBloom:        header.Bloom.Bytes(),
		Difficulty:       header.Difficulty,
		Number:           header.Number,
		GasLimit:         new(big.Int).SetUint64(header.GasLimit),
		GasUsed:          new(big.Int).SetUint64(header.GasUsed),
		Timestamp:        new(big.Int).SetUint64(header.Time),
		ExtraData:        header.Extra,
		MixHash:          header.MixDigest.Bytes(),
		Nonce:            header.Nonce[:],
		BaseFeePerGas:    header.BaseFee,
		WithdrawalsRoot:  withdrawalsRoot,
	}
}

type ContractExecution struct {
	ParentHash       [32]byte       `json:"parent_hash"`
	FeeRecipient     common.Address `json:"fee_recipient"`
	StateRoot        [32]byte       `json:"state_root"`
	ReceiptsRoot     [32]byte       `json:"receipts_root"`
	LogsBloom        []byte         `json:"logs_bloom"`
	PrevRandao       [32]byte       `json:"prev_randao"`
	BlockNumber      *big.Int       `json:"block_number"`
	GasLimit         *big.Int       `json:"gas_limit"`
	GasUsed          *big.Int       `json:"gas_used"`
	Timestamp        *big.Int       `json:"timestamp"`
	ExtraData        []byte         `json:"extra_data"`
	BaseFeePerGas    *big.Int       `json:"base_fee_per_gas"`
	BlockHash        [32]byte       `json:"block_hash"`
	TransactionsRoot [32]byte       `json:"transactions_root"`
	WithdrawalsRoot  [32]byte       `json:"withdrawals_root"`
}

func ConvertExecution(execution *Execution) (*ContractExecution, error) {
	blockNumber, ok := big.NewInt(0).SetString(execution.BlockNumber, 10)
	if !ok {
		return nil, errors.New("execution blockNumber error")
	}
	gasLimit, ok := big.NewInt(0).SetString(execution.GasLimit, 10)
	if !ok {
		return nil, errors.New("execution gasLimit error")
	}
	gasUsed, ok := big.NewInt(0).SetString(execution.GasUsed, 10)
	if !ok {
		return nil, errors.New("execution gasUsed error")
	}
	timestamp, ok := big.NewInt(0).SetString(execution.Timestamp, 10)
	if !ok {
		return nil, errors.New("execution timestamp error")
	}
	baseFeePerGas, ok := big.NewInt(0).SetString(execution.BaseFeePerGas, 10)
	if !ok {
		return nil, errors.New("execution baseFeePerGas error")
	}
	return &ContractExecution{
		ParentHash:       common.HexToHash(execution.ParentHash),
		FeeRecipient:     common.HexToAddress(execution.FeeRecipient),
		StateRoot:        common.HexToHash(execution.StateRoot),
		ReceiptsRoot:     common.HexToHash(execution.ReceiptsRoot),
		LogsBloom:        common.Hex2Bytes(strings.TrimPrefix(execution.LogsBloom, "0x")),
		PrevRandao:       common.HexToHash(execution.PrevRandao),
		BlockNumber:      blockNumber,
		GasLimit:         gasLimit,
		GasUsed:          gasUsed,
		Timestamp:        timestamp,
		ExtraData:        common.Hex2Bytes(strings.TrimPrefix(execution.ExtraData, "0x")),
		BaseFeePerGas:    baseFeePerGas,
		BlockHash:        common.HexToHash(execution.BlockHash),
		TransactionsRoot: common.HexToHash(execution.TransactionsRoot),
		WithdrawalsRoot:  common.HexToHash(execution.WithdrawalsRoot),
	}, nil
}
