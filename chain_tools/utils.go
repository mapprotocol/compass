package chain_tools

import (
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/mapprotocol/compass/atlas"
)

func ConvertHeader(eh *types.Header) *atlas.Header {
	return &atlas.Header{
		ParentHash:  eh.ParentHash,
		UncleHash:   eh.UncleHash,
		Coinbase:    eh.Coinbase,
		Root:        eh.Root,
		TxHash:      eh.TxHash,
		ReceiptHash: eh.ReceiptHash,
		Bloom:       eh.Bloom,
		Difficulty:  eh.Difficulty,
		Number:      eh.Number,
		GasLimit:    eh.GasLimit,
		GasUsed:     eh.GasUsed,
		Time:        eh.Time,
		Extra:       eh.Extra,
		MixDigest:   eh.MixDigest,
		Nonce:       eh.Nonce,
	}
}
