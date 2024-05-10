package scroll

import (
	"bytes"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/mapprotocol/compass/internal/constant"
	"github.com/mapprotocol/compass/internal/proof"
)

type Receipts []*Receipt

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

// Len returns the number of receipts in this list.
func (rs Receipts) Len() int { return len(rs) }

// EncodeIndex encodes the i'th receipt to w.
func (rs Receipts) EncodeIndex(i int, w *bytes.Buffer) {
	r := rs[i]
	data := &proof.ReceiptRLP{PostStateOrStatus: r.statusEncoding(), CumulativeGasUsed: r.CumulativeGasUsed, Bloom: r.Bloom, Logs: r.Logs}
	switch r.Type {
	case constant.LegacyTxType:
		rlp.Encode(w, data)
	default:
		w.WriteByte(r.Type)
		rlp.Encode(w, data)
	}
}
