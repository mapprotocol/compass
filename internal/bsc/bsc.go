package bsc

import (
	"math/big"

	"github.com/ethereum/go-ethereum/core/types"

	"github.com/ethereum/go-ethereum/common"
)

type Header struct {
	ParentHash       []byte         `json:"parentHash"`
	Sha3Uncles       []byte         `json:"sha3Uncles"`
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
}

func ConvertHeader(header types.Header) Header {
	bloom := make([]byte, 0, len(header.Bloom))
	for _, b := range header.Bloom {
		bloom = append(bloom, b)
	}
	nonce := make([]byte, 0, len(header.Nonce))
	for _, b := range header.Nonce {
		nonce = append(nonce, b)
	}
	return Header{
		ParentHash:       hashToByte(header.ParentHash),
		Sha3Uncles:       hashToByte(header.UncleHash),
		Miner:            header.Coinbase,
		StateRoot:        hashToByte(header.Root),
		TransactionsRoot: hashToByte(header.TxHash),
		ReceiptsRoot:     hashToByte(header.ReceiptHash),
		LogsBloom:        bloom,
		Difficulty:       header.Difficulty,
		Number:           header.Number,
		GasLimit:         new(big.Int).SetUint64(header.GasLimit),
		GasUsed:          new(big.Int).SetUint64(header.GasUsed),
		Timestamp:        new(big.Int).SetUint64(header.Time),
		ExtraData:        header.Extra,
		MixHash:          hashToByte(header.MixDigest),
		Nonce:            nonce,
	}
}

func hashToByte(h common.Hash) []byte {
	ret := make([]byte, 0, len(h))
	for _, b := range h {
		ret = append(ret, b)
	}
	return ret
}
