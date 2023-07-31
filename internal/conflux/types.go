package conflux

import (
	"encoding/json"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/mapprotocol/compass/internal/conflux/types"
	"io"
)

type BlockHeader struct {
	Hash                  types.Hash     `json:"hash"`
	ParentHash            types.Hash     `json:"parentHash"`
	Height                *hexutil.Big   `json:"height"`
	Miner                 types.Address  `json:"miner"`
	DeferredStateRoot     types.Hash     `json:"deferredStateRoot"`
	DeferredReceiptsRoot  types.Hash     `json:"deferredReceiptsRoot"`
	DeferredLogsBloomHash types.Hash     `json:"deferredLogsBloomHash"`
	Blame                 hexutil.Uint64 `json:"blame"`
	TransactionsRoot      types.Hash     `json:"transactionsRoot"`
	EpochNumber           *hexutil.Big   `json:"epochNumber"`
	BlockNumber           *hexutil.Big   `json:"blockNumber"`
	GasLimit              *hexutil.Big   `json:"gasLimit"`
	GasUsed               *hexutil.Big   `json:"gasUsed"`
	Timestamp             *hexutil.Big   `json:"timestamp"`
	Difficulty            *hexutil.Big   `json:"difficulty"`
	PowQuality            *hexutil.Big   `json:"powQuality"`
	RefereeHashes         []types.Hash   `json:"refereeHashes"`
	Adaptive              bool           `json:"adaptive"`
	Nonce                 *hexutil.Big   `json:"nonce"`
	Size                  *hexutil.Big   `json:"size"`
	Custom                []Bytes        `json:"custom"`
	PosReference          *types.Hash    `json:"posReference"`
}

type Bytes []byte

func (b Bytes) MarshalText() ([]byte, error) {
	return hexutil.Bytes(b).MarshalText()
}

func (b *Bytes) UnmarshalJSON(data []byte) error {
	var hex hexutil.Bytes
	if err := json.Unmarshal(data, &hex); err == nil {
		*b = Bytes([]byte(hex))
		return nil
	}

	nums := make([]uint, 0)
	if err := json.Unmarshal(data, &nums); err != nil {
		return err
	}

	for _, v := range nums {
		*b = append(*b, byte(v))
	}
	return nil
}

func (b *Bytes) ToBytes() []byte {
	return []byte(*b)
}

func (b *Bytes) ToHexBytes() hexutil.Bytes {
	return hexutil.Bytes(*b)
}

type CfxBlock struct {
	BlockHeader
	Transactions []Transaction `json:"transactions"`
}

// rlpEncodableBlock block struct used for rlp encoding
type rlpEncodableBlock struct {
	BlockHeader  BlockHeader
	Transactions []Transaction
}

// EncodeRLP implements the rlp.Encoder interface.
func (block CfxBlock) EncodeRLP(w io.Writer) error {
	rblock := rlpEncodableBlock{
		block.BlockHeader, block.Transactions,
	}

	return rlp.Encode(w, rblock)
}

// DecodeRLP implements the rlp.Decoder interface.
func (block *CfxBlock) DecodeRLP(r *rlp.Stream) error {
	var rblock rlpEncodableBlock
	if err := r.Decode(&rblock); err != nil {
		return err
	}

	block.BlockHeader = rblock.BlockHeader
	block.Transactions = rblock.Transactions

	return nil
}
