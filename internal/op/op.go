package op

import (
	"bytes"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/mapprotocol/compass/internal/constant"
	"github.com/mapprotocol/compass/internal/proof"
)

const (
	LegacyTxType     = 0x00
	AccessListTxType = 0x01
	DynamicFeeTxType = 0x02
	BlobTxType       = 0x03
)

const DepositTxType = 0x7E

type Receipt struct {
	*types.Receipt
	DepositNonce          *uint64 `json:"depositNonce,omitempty"`
	DepositReceiptVersion *uint64 `json:"depositReceiptVersion,omitempty"`
}

func (r *Receipt) statusEncoding() []byte {
	if len(r.PostState) == 0 {
		if r.Status == constant.ReceiptStatusFailed {
			return constant.ReceiptStatusFailedRLP
		}
		return constant.ReceiptStatusSuccessfulRLP
	}
	return r.PostState
}

type depositReceiptRLP struct {
	PostStateOrStatus     []byte
	CumulativeGasUsed     uint64
	Bloom                 types.Bloom
	Logs                  []*types.Log
	DepositNonce          *uint64 `rlp:"optional"`
	DepositReceiptVersion *uint64 `rlp:"optional"`
}

type Receipts []*Receipt

func (rs Receipts) Len() int { return len(rs) }

func (rs Receipts) EncodeIndex(i int, w *bytes.Buffer) {
	r := rs[i]
	data := &proof.ReceiptRLP{PostStateOrStatus: r.statusEncoding(), CumulativeGasUsed: r.CumulativeGasUsed, Bloom: r.Bloom, Logs: r.Logs}
	if r.Type == constant.LegacyTxType {
		rlp.Encode(w, data)
		return
	}
	w.WriteByte(r.Type)
	switch r.Type {
	case AccessListTxType, DynamicFeeTxType, BlobTxType:
		rlp.Encode(w, data)
	case DepositTxType:
		if r.DepositReceiptVersion != nil {
			// post-canyon receipt hash computation update
			depositData := &depositReceiptRLP{data.PostStateOrStatus, data.CumulativeGasUsed, r.Bloom, r.Logs, r.DepositNonce, r.DepositReceiptVersion}
			rlp.Encode(w, depositData)
		} else {
			rlp.Encode(w, data)
		}
	default:
	}
}
