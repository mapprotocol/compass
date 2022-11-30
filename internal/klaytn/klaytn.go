package klaytn

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethdb/memorydb"
	"github.com/ethereum/go-ethereum/light"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/trie"
	"github.com/mapprotocol/compass/mapprotocol"
	"github.com/mapprotocol/compass/msg"
	utils "github.com/mapprotocol/compass/shared/ethereum"
	"math/big"
	"strings"
)

type Header struct {
	ParentHash       []byte         `json:"parentHash"`
	Reward           common.Address `json:"reward"`
	StateRoot        []byte         `json:"stateRoot"`
	TransactionsRoot []byte         `json:"transactionsRoot"`
	ReceiptsRoot     []byte         `json:"receiptsRoot"`
	LogsBloom        []byte         `json:"logsBloom"`
	BlockScore       *big.Int       `json:"blockScore"`
	Number           *big.Int       `json:"number"`
	GasUsed          *big.Int       `json:"gasUsed"`
	Timestamp        *big.Int       `json:"timestamp"`
	TimestampFoS     *big.Int       `json:"timestampFoS"`
	ExtraData        []byte         `json:"extraData"`
	GovernanceData   []byte         `json:"governanceData"`
	VoteData         []byte         `json:"voteData"`
	BaseFee          *big.Int       `json:"baseFee"`
}

const (
	// BloomByteLength represents the number of bytes used in a header log bloom.
	BloomByteLength = 256
	PrefixOfHex     = "0x"
)

type Bloom [BloomByteLength]byte

type RpcHeader struct {
	BaseFeePerGas    string         `json:"baseFeePerGas"`
	BlockScore       string         `json:"blockscore"`
	ExtraData        string         `json:"extraData"`
	GasUsed          string         `json:"gasUsed"`
	GovernanceData   string         `json:"governanceData"`
	Hash             common.Hash    `json:"hash"`
	LogsBloom        string         `json:"logsBloom"`
	Number           string         `json:"number"`
	ParentHash       common.Hash    `json:"parentHash"`
	ReceiptsRoot     common.Hash    `json:"receiptsRoot"`
	Reward           common.Address `json:"reward"`
	Size             string         `json:"size"`
	StateRoot        common.Hash    `json:"stateRoot"`
	Timestamp        string         `json:"timestamp"`
	TimestampFoS     string         `json:"timestampFoS"`
	TotalBlockScore  string         `json:"totalBlockScore"`
	TransactionsRoot common.Hash    `json:"transactionsRoot"`
	VoteData         string         `json:"voteData"`
}

func ConvertContractHeader(ethHeader *types.Header, rh *RpcHeader) Header {
	bloom := make([]byte, 0, len(ethHeader.Bloom))
	for _, b := range ethHeader.Bloom {
		bloom = append(bloom, b)
	}
	blockScore := new(big.Int)
	blockScore.SetString(strings.TrimPrefix(rh.BlockScore, PrefixOfHex), 16)
	baseFeePerGas := new(big.Int)
	baseFeePerGas.SetString(strings.TrimPrefix(rh.BaseFeePerGas, PrefixOfHex), 16)
	timestamp := new(big.Int)
	timestamp.SetString(strings.TrimPrefix(rh.Timestamp, PrefixOfHex), 16)
	timestampFos := new(big.Int)
	timestampFos.SetString(strings.TrimPrefix(rh.TimestampFoS, PrefixOfHex), 16)
	return Header{
		ParentHash:       hashToByte(ethHeader.ParentHash),
		Reward:           rh.Reward,
		StateRoot:        hashToByte(ethHeader.Root),
		TransactionsRoot: hashToByte(ethHeader.TxHash),
		ReceiptsRoot:     hashToByte(ethHeader.ReceiptHash),
		LogsBloom:        bloom,
		BlockScore:       blockScore,
		BaseFee:          baseFeePerGas,
		Number:           ethHeader.Number,
		GasUsed:          new(big.Int).SetUint64(ethHeader.GasUsed),
		Timestamp:        timestamp,
		TimestampFoS:     timestampFos,
		ExtraData:        common.Hex2Bytes(strings.TrimPrefix(rh.ExtraData, PrefixOfHex)),
		GovernanceData:   common.Hex2Bytes(strings.TrimPrefix(rh.GovernanceData, PrefixOfHex)),
		VoteData:         common.Hex2Bytes(strings.TrimPrefix(rh.VoteData, PrefixOfHex)),
	}
}

func hashToByte(h common.Hash) []byte {
	ret := make([]byte, 0, len(h))
	for _, b := range h {
		ret = append(ret, b)
	}
	return ret
}

type ReceiptProof struct {
	BlockHeader Header
	TxReceipt   mapprotocol.TxReceipt
	KeyIndex    []byte
	Proof       [][]byte
}

func AssembleProof(header Header, log types.Log, fId msg.ChainId, receipts []*types.Receipt, method string) ([]byte, error) {
	txIndex := log.TxIndex
	receipt, err := mapprotocol.GetTxReceipt(receipts[txIndex])
	if err != nil {
		return nil, err
	}

	proof, err := getProof(receipts, txIndex)
	if err != nil {
		return nil, err
	}

	var key []byte
	key = rlp.AppendUint64(key[:0], uint64(txIndex))
	ek := utils.Key2Hex(key, len(proof))

	pd := ReceiptProof{
		BlockHeader: header,
		TxReceipt:   *receipt,
		KeyIndex:    ek,
		Proof:       proof,
	}

	input, err := mapprotocol.Klaytn.Methods[mapprotocol.MethodOfGetBytes].Inputs.Pack(pd)
	if err != nil {
		return nil, err
	}
	pack, err := mapprotocol.PackInput(mapprotocol.Mcs, method, new(big.Int).SetUint64(uint64(fId)), input)
	// input, err := mapprotocol.LightManger.Pack(mapprotocol.MethodVerifyProofData, new(big.Int).SetUint64(uint64(fId)), all)
	if err != nil {
		return nil, err
	}

	return pack, nil
}

func getProof(receipts []*types.Receipt, txIndex uint) ([][]byte, error) {
	tr, err := trie.New(common.Hash{}, trie.NewDatabase(memorydb.New()))
	if err != nil {
		return nil, err
	}

	tr = utils.DeriveTire(receipts, tr)
	ns := light.NewNodeSet()
	key, err := rlp.EncodeToBytes(txIndex)
	if err != nil {
		return nil, err
	}
	if err = tr.Prove(key, 0, ns); err != nil {
		return nil, err
	}

	proof := make([][]byte, 0, len(ns.NodeList()))
	for _, v := range ns.NodeList() {
		proof = append(proof, v)
	}

	return proof, nil
}
