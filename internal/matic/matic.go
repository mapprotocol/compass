package matic

import (
	"bytes"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethdb/memorydb"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/trie"
	"github.com/mapprotocol/compass/internal/constant"
	"github.com/mapprotocol/compass/internal/mapprotocol"
	"github.com/mapprotocol/compass/internal/proof"
	"github.com/mapprotocol/compass/pkg/msg"
	"github.com/mapprotocol/compass/pkg/util"
	"math/big"
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
	BaseFeePerGas    *big.Int       `json:"baseFeePerGas"`
}

func ConvertHeader(header *types.Header) BlockHeader {
	bloom := make([]byte, 0, len(header.Bloom))
	for _, b := range header.Bloom {
		bloom = append(bloom, b)
	}
	nonce := make([]byte, 0, len(header.Nonce))
	for _, b := range header.Nonce {
		nonce = append(nonce, b)
	}
	return BlockHeader{
		ParentHash:       hashToByte(header.ParentHash),
		Sha3Uncles:       hashToByte(header.UncleHash),
		Miner:            constant.ZeroAddress,
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
		BaseFeePerGas:    header.BaseFee,
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
	Headers      []BlockHeader
	ReceiptProof proof.NewReceiptProof
}

func AssembleProof(headers []BlockHeader, log *types.Log, fId msg.ChainId, receipts []*types.Receipt,
	method string, proofType int64, orderId [32]byte) ([]byte, error) {
	txIndex := log.TxIndex
	receipt, err := mapprotocol.GetTxReceipt(receipts[txIndex])
	if err != nil {
		return nil, err
	}

	prf, err := proof.Get(Receipts(receipts), txIndex)
	if err != nil {
		return nil, err
	}

	tr, _ := trie.New(common.Hash{}, trie.NewDatabase(memorydb.New()))
	tr = proof.DeriveTire(Receipts(receipts), tr)
	ret := tr.Hash()
	if ret != common.BytesToHash(headers[0].ReceiptsRoot) {
		fmt.Println("Matic generate", ret, "oracle", common.BytesToHash(headers[0].ReceiptsRoot), " not same")
	}

	var key []byte
	key = rlp.AppendUint64(key[:0], uint64(txIndex))
	//ek := mapo.Key2Hex(key, len(prf))

	idx := 0
	for i, ele := range receipts[txIndex].Logs {
		if ele.Index != log.Index {
			continue
		}
		idx = i
	}

	nr := mapprotocol.MapTxReceipt{
		PostStateOrStatus: receipt.PostStateOrStatus,
		CumulativeGasUsed: receipt.CumulativeGasUsed,
		Bloom:             receipt.Bloom,
		Logs:              receipt.Logs,
	}
	nrRlp, err := rlp.EncodeToBytes(nr)
	if err != nil {
		return nil, err
	}

	if receipt.ReceiptType.Int64() != 0 {
		n := make([]byte, 0)
		n = append(n, receipt.ReceiptType.Bytes()...)
		n = append(n, nrRlp...)
		nrRlp = n
	}

	var pack []byte
	switch proofType {
	case constant.ProofTypeOfOrigin:
		pd := ProofData{
			Headers: headers,
			ReceiptProof: proof.NewReceiptProof{
				TxReceipt:   nrRlp,
				ReceiptType: receipt.ReceiptType,
				KeyIndex:    util.Key2Hex(key, len(prf)),
				Proof:       prf,
			},
		}

		pack, err = proof.V3Pack(fId, method, mapprotocol.Matic, idx, orderId, false, pd)
	case constant.ProofTypeOfZk:
	case constant.ProofTypeOfOracle:
		pd := proof.Data{
			BlockNum: big.NewInt(int64(log.BlockNumber)),
			ReceiptProof: proof.ReceiptProof{
				TxReceipt: *receipt,
				KeyIndex:  util.Key2Hex(key, len(prf)),
				Proof:     prf,
			},
		}

		pack, err = proof.Pack(fId, method, mapprotocol.OracleAbi, pd)
	}

	if err != nil {
		return nil, err
	}

	return pack, nil
}

type receiptRLP struct {
	PostStateOrStatus []byte
	CumulativeGasUsed uint64
	Bloom             types.Bloom
	Logs              []*types.Log
}

type Receipts []*types.Receipt

// Len returns the number of receipts in this list.
func (rs Receipts) Len() int { return len(rs) }

// EncodeIndex encodes the i'th receipt to w.
func (rs Receipts) EncodeIndex(i int, w *bytes.Buffer) {
	r := rs[i]
	data := &receiptRLP{statusEncoding(r), r.CumulativeGasUsed, r.Bloom, r.Logs}
	switch r.Type {
	case constant.LegacyTxType:
		rlp.Encode(w, data)
	case constant.AccessListTxType, constant.BlobTxType, constant.SetCodeTxType, constant.DynamicFeeTxType:
		w.WriteByte(r.Type)
		rlp.Encode(w, data)
	default:
	}
}

func statusEncoding(r *types.Receipt) []byte {
	if len(r.PostState) == 0 {
		if r.Status == constant.ReceiptStatusFailed {
			return constant.ReceiptStatusFailedRLP
		}
		return constant.ReceiptStatusSuccessfulRLP
	}
	return r.PostState
}
