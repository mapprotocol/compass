package bttc

import (
	"github.com/mapprotocol/compass/internal/constant"
	"github.com/mapprotocol/compass/internal/proof"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/klaytn/klaytn/rlp"
	"github.com/mapprotocol/compass/mapprotocol"
	"github.com/mapprotocol/compass/msg"
	"github.com/mapprotocol/compass/pkg/util"
)

type BlockHeader struct {
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

func convertHeader(header *types.Header) BlockHeader {
	bloom := make([]byte, 0, len(header.Bloom))
	for _, b := range header.Bloom {
		bloom = append(bloom, b)
	}
	nonce := make([]byte, 0, len(header.Nonce))
	for _, b := range header.Nonce {
		nonce = append(nonce, b)
	}
	return BlockHeader{
		ParentHash:       util.HashToByte(header.ParentHash),
		Sha3Uncles:       util.HashToByte(header.UncleHash),
		Miner:            constant.ZeroAddress,
		StateRoot:        util.HashToByte(header.Root),
		TransactionsRoot: util.HashToByte(header.TxHash),
		ReceiptsRoot:     util.HashToByte(header.ReceiptHash),
		LogsBloom:        bloom,
		Difficulty:       header.Difficulty,
		Number:           header.Number,
		GasLimit:         new(big.Int).SetUint64(header.GasLimit),
		GasUsed:          new(big.Int).SetUint64(header.GasUsed),
		Timestamp:        new(big.Int).SetUint64(header.Time),
		ExtraData:        header.Extra,
		MixHash:          util.HashToByte(header.MixDigest),
		Nonce:            nonce,
	}
}

type ProofData struct {
	Headers      []BlockHeader
	ReceiptProof ReceiptProof
}

type ReceiptProof struct {
	TxReceipt mapprotocol.TxReceipt
	KeyIndex  []byte
	Proof     [][]byte
}

func AssembleProof(headers []BlockHeader, txIndex uint, fId msg.ChainId, allR []*types.Receipt, cullSys []*types.Receipt, method string) ([]byte, error) {
	receipt, err := mapprotocol.GetTxReceipt(allR[txIndex])
	if err != nil {
		return nil, err
	}

	prf, err := proof.Get(types.Receipts(cullSys), txIndex)
	if err != nil {
		return nil, err
	}

	var key []byte
	key = rlp.AppendUint64(key[:0], uint64(txIndex))
	ek := util.Key2Hex(key, len(prf))

	pd := ProofData{
		Headers: headers,
		ReceiptProof: ReceiptProof{
			TxReceipt: *receipt,
			KeyIndex:  ek,
			Proof:     prf,
		},
	}

	input, err := mapprotocol.Bttc.Methods[mapprotocol.MethodOfGetBytes].Inputs.Pack(pd)
	if err != nil {
		return nil, err
	}
	pack, err := mapprotocol.PackInput(mapprotocol.Mcs, method, new(big.Int).SetUint64(uint64(fId)), input)
	if err != nil {
		return nil, err
	}

	return pack, nil
}
