package eth2

import (
	"bytes"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/mapprotocol/compass/internal/constant"
	"github.com/mapprotocol/compass/internal/proof"
)

type Receipt struct {
	*types.Receipt
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
	case constant.AccessListTxType, constant.DynamicFeeTxType, constant.BlobTxType:
		rlp.Encode(w, data)
	default:
	}
}
