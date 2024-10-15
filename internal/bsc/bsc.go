package bsc

import (
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
	maptypes "github.com/mapprotocol/atlas/core/types"
	"github.com/mapprotocol/compass/internal/constant"
	"github.com/mapprotocol/compass/internal/mapo"
	"github.com/mapprotocol/compass/internal/op"
	"github.com/mapprotocol/compass/internal/proof"
	iproof "github.com/mapprotocol/compass/internal/proof"
	"github.com/mapprotocol/compass/mapprotocol"
	"github.com/mapprotocol/compass/msg"
	"github.com/mapprotocol/compass/pkg/ethclient"
	"math/big"
	"strings"
)

type Header struct {
	ParentHash            []byte         `json:"parentHash"`
	Sha3Uncles            []byte         `json:"sha3Uncles"`
	Miner                 common.Address `json:"miner"`
	StateRoot             []byte         `json:"stateRoot"`
	TransactionsRoot      []byte         `json:"transactionsRoot"`
	ReceiptsRoot          []byte         `json:"receiptsRoot"`
	LogsBloom             []byte         `json:"logsBloom"`
	Difficulty            *big.Int       `json:"difficulty"`
	Number                *big.Int       `json:"number"`
	GasLimit              *big.Int       `json:"gasLimit"`
	GasUsed               *big.Int       `json:"gasUsed"`
	Timestamp             *big.Int       `json:"timestamp"`
	ExtraData             []byte         `json:"extraData"`
	MixHash               []byte         `json:"mixHash"`
	Nonce                 []byte         `json:"nonce"`
	BaseFeePerGas         *big.Int       `json:"baseFeePerGas"`
	WithdrawalsRoot       []byte         `json:"withdrawalsRoot"`
	BlobGasUsed           *big.Int       `json:"blobGasUsed"`
	ExcessBlobGas         *big.Int       `json:"excessBlobGas"`
	ParentBeaconBlockRoot []byte         `json:"parentBeaconBlockRoot"`
}

func ConvertHeader(header *ethclient.BscHeader) Header {
	bloom := make([]byte, 0, len(header.Bloom))
	for _, b := range header.Bloom {
		bloom = append(bloom, b)
	}
	nonce := make([]byte, 0, len(header.Nonce))
	for _, b := range header.Nonce {
		nonce = append(nonce, b)
	}
	if header.BaseFee == nil {
		header.BaseFee = new(big.Int)
	}
	parentBeaconBlockRoot := common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000001")
	if header.ParentBeaconBlockRoot != "" && strings.TrimPrefix(header.ParentBeaconBlockRoot, "0x") != "" {
		fmt.Println(header.Number, " ---- header.ParentBeaconBlockRoot ---------------------------- ", header.ParentBeaconBlockRoot)
		parentBeaconBlockRoot = common.Hex2Bytes(strings.TrimPrefix(header.ParentBeaconBlockRoot, "0x"))
	}

	blobGasUsed, excessBlobGas := big.NewInt(0), big.NewInt(0)
	if header.BlobGasUsed != "" && strings.TrimPrefix(header.BlobGasUsed, "0x") != "" {
		blobGasUsed, _ = blobGasUsed.SetString(strings.TrimPrefix(header.BlobGasUsed, "0x"), 16)
	}
	if header.ExcessBlobGas != "" && strings.TrimPrefix(header.ExcessBlobGas, "0x") != "" {
		excessBlobGas, _ = excessBlobGas.SetString(strings.TrimPrefix(header.ExcessBlobGas, "0x"), 16)
	}

	return Header{
		ParentHash:            hashToByte(header.ParentHash),
		Sha3Uncles:            hashToByte(header.UncleHash),
		Miner:                 header.Coinbase,
		StateRoot:             hashToByte(header.Root),
		TransactionsRoot:      hashToByte(header.TxHash),
		ReceiptsRoot:          hashToByte(header.ReceiptHash),
		LogsBloom:             bloom,
		Difficulty:            header.Difficulty,
		Number:                header.Number,
		GasLimit:              new(big.Int).SetUint64(header.GasLimit),
		GasUsed:               new(big.Int).SetUint64(header.GasUsed),
		Timestamp:             new(big.Int).SetUint64(header.Time),
		ExtraData:             header.Extra,
		MixHash:               hashToByte(header.MixDigest),
		Nonce:                 nonce,
		BaseFeePerGas:         header.BaseFee,
		WithdrawalsRoot:       common.Hex2Bytes(strings.TrimPrefix(header.WithdrawalsRoot, "0x")),
		BlobGasUsed:           blobGasUsed,
		ExcessBlobGas:         excessBlobGas,
		ParentBeaconBlockRoot: parentBeaconBlockRoot,
	}
}

func hashToByte(h common.Hash) []byte {
	ret := make([]byte, 0, len(h))
	for _, b := range h {
		ret = append(ret, b)
	}
	return ret
}

type ProofData struct {
	Headers      []Header
	ReceiptProof ReceiptProof
}

type ReceiptProof struct {
	TxReceipt mapprotocol.TxReceipt
	KeyIndex  []byte
	Proof     [][]byte
}

func AssembleProof(header []Header, log *types.Log, receipts []*types.Receipt, method string,
	fId msg.ChainId, proofType int64, sign [][]byte, orderId [32]byte) ([]byte, error) {
	var ret []byte
	txIndex := log.TxIndex
	receipt, err := mapprotocol.GetTxReceipt(receipts[txIndex])
	if err != nil {
		return nil, err
	}

	pr := op.Receipts{}
	for _, r := range receipts {
		pr = append(pr, &op.Receipt{Receipt: r})
	}

	prf, err := iproof.Get(pr, txIndex)
	if err != nil {
		return nil, err
	}

	var key []byte
	key = rlp.AppendUint64(key[:0], uint64(txIndex))
	ek := mapo.Key2Hex(key, len(prf))

	idx := 0
	for i, ele := range receipts[txIndex].Logs {
		if ele.Index != log.Index {
			continue
		}
		idx = i
	}

	switch proofType {
	case constant.ProofTypeOfOrigin:
		pd := ProofData{
			Headers: header,
			ReceiptProof: ReceiptProof{
				TxReceipt: *receipt,
				KeyIndex:  ek,
				Proof:     prf,
			},
		}

		ret, err = iproof.V3Pack(fId, method, mapprotocol.Bsc, idx, orderId, false, pd)
		if err != nil {
			return nil, err
		}
	case constant.ProofTypeOfNewOracle:
		ret, err = proof.SignOracle(&maptypes.Header{
			ReceiptHash: common.BytesToHash(header[0].ReceiptsRoot),
		}, receipt, key, prf, fId, idx, method, sign, orderId, log, proofType)
	}

	return ret, nil
}
