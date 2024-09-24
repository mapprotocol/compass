package bsc

import (
	"fmt"
	"math/big"
	"strings"

	"github.com/mapprotocol/compass/internal/mapo"
	"github.com/mapprotocol/compass/internal/op"
	"github.com/mapprotocol/compass/pkg/ethclient"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
	iproof "github.com/mapprotocol/compass/internal/proof"
	"github.com/mapprotocol/compass/mapprotocol"
	"github.com/mapprotocol/compass/msg"
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
	//fmt.Println("idx -------------- ", idx)

	pd := ProofData{
		Headers: header,
		ReceiptProof: ReceiptProof{
			TxReceipt: *receipt,
			KeyIndex:  ek,
			Proof:     prf,
		},
	}

	//input, err := mapprotocol.Bsc.Methods[mapprotocol.MethodOfGetBytes].Inputs.Pack(pd)
	//if err != nil {
	//	return nil, errors.Wrap(err, "pack getBytes failed")
	//}
	//for _, h := range pd.Headers {
	//	fmt.Println("ParentHash", "0x"+common.Bytes2Hex(h.ParentHash))
	//	fmt.Println("Sha3Uncles", "0x"+common.Bytes2Hex(h.Sha3Uncles))
	//	fmt.Println("StateRoot", "0x"+common.Bytes2Hex(h.StateRoot))
	//	fmt.Println("TransactionsRoot", "0x"+common.Bytes2Hex(h.TransactionsRoot))
	//	fmt.Println("ReceiptsRoot", "0x"+common.Bytes2Hex(h.ReceiptsRoot))
	//	fmt.Println("LogsBloom", "0x"+common.Bytes2Hex(h.LogsBloom))
	//	fmt.Println("ExtraData", "0x"+common.Bytes2Hex(h.ExtraData))
	//	fmt.Println("MixHash", "0x"+common.Bytes2Hex(h.MixHash))
	//	fmt.Println("Nonce", "0x"+common.Bytes2Hex(h.Nonce))
	//	fmt.Println("WithdrawalsRoot", "0x"+common.Bytes2Hex(h.WithdrawalsRoot))
	//	fmt.Println("ParentBeaconBlockRoot", "0x"+common.Bytes2Hex(h.ParentBeaconBlockRoot))
	//	fmt.Println("Miner", h.Miner.String())
	//	fmt.Println("Difficulty", h.Difficulty.String())
	//	fmt.Println("Number", h.Number.String())
	//	fmt.Println("GasLimit", h.GasLimit.String())
	//	fmt.Println("GasUsed", h.GasUsed.String())
	//	fmt.Println("Timestamp", h.Timestamp.String())
	//	fmt.Println("BaseFeePerGas", h.BaseFeePerGas.String())
	//	fmt.Println("BlobGasUsed", h.BlobGasUsed.String())
	//	fmt.Println("ExcessBlobGas", h.ExcessBlobGas.String())
	//}
	//
	//fmt.Println("KeyIndex ", "0x"+common.Bytes2Hex(pd.ReceiptProof.KeyIndex))
	//for _, r := range pd.ReceiptProof.Proof {
	//	fmt.Println("proof ", "0x"+common.Bytes2Hex(r))
	//}
	//pack, err := mapprotocol.LightManger.Pack(mapprotocol.MethodVerifyProofData, new(big.Int).SetUint64(uint64(fId)), input)

	pack, err := iproof.V3Pack(fId, method, mapprotocol.Bsc, idx, orderId, false, pd)

	if err != nil {
		return nil, err
	}
	return pack, nil
}
