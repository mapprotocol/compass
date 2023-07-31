package conflux

import (
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/mapprotocol/compass/internal/conflux/types"
	"io"
	"math/big"
)

type Transaction struct {
	Hash             types.Hash      `json:"hash"`
	Nonce            *hexutil.Big    `json:"nonce"`
	BlockHash        *types.Hash     `json:"blockHash"`
	TransactionIndex *hexutil.Uint64 `json:"transactionIndex"`
	From             Address         `json:"from"`
	To               *Address        `json:"to"`
	Value            *hexutil.Big    `json:"value"`
	GasPrice         *hexutil.Big    `json:"gasPrice"`
	Gas              *hexutil.Big    `json:"gas"`
	ContractCreated  *Address        `json:"contractCreated"`
	Data             string          `json:"data"`
	StorageLimit     *hexutil.Big    `json:"storageLimit"`
	EpochHeight      *hexutil.Big    `json:"epochHeight"`
	ChainID          *hexutil.Big    `json:"chainId"`
	Status           *hexutil.Uint64 `json:"status"`
	V                *hexutil.Big    `json:"v"`
	R                *hexutil.Big    `json:"r"`
	S                *hexutil.Big    `json:"s"`
}

// rlpEncodableTransaction transaction struct used for rlp encoding
type rlpEncodableTransaction struct {
	Hash             types.Hash
	Nonce            *big.Int
	BlockHash        *types.Hash
	TransactionIndex *hexutil.Uint64
	From             Address
	To               *Address `rlp:"nil"`
	Value            *big.Int
	GasPrice         *big.Int
	Gas              *big.Int
	ContractCreated  *Address `rlp:"nil"` // nil means contract creation
	Data             string
	StorageLimit     *big.Int
	EpochHeight      *big.Int
	ChainID          *big.Int
	Status           *hexutil.Uint64

	//signature
	V *big.Int
	R *big.Int
	S *big.Int
}

// EncodeRLP implements the rlp.Encoder interface.
func (tx Transaction) EncodeRLP(w io.Writer) error {
	rtx := rlpEncodableTransaction{
		tx.Hash, tx.Nonce.ToInt(), tx.BlockHash, tx.TransactionIndex, tx.From, tx.To,
		tx.Value.ToInt(), tx.GasPrice.ToInt(), tx.Gas.ToInt(), tx.ContractCreated, tx.Data,
		tx.StorageLimit.ToInt(), tx.EpochHeight.ToInt(), tx.ChainID.ToInt(), tx.Status,
		tx.V.ToInt(), tx.R.ToInt(), tx.S.ToInt(),
	}

	return rlp.Encode(w, rtx)
}

// DecodeRLP implements the rlp.Decoder interface.
func (tx *Transaction) DecodeRLP(r *rlp.Stream) error {
	var rtx rlpEncodableTransaction
	if err := r.Decode(&rtx); err != nil {
		return err
	}

	tx.Hash, tx.Nonce, tx.BlockHash = rtx.Hash, (*hexutil.Big)(rtx.Nonce), rtx.BlockHash
	tx.TransactionIndex, tx.From, tx.To = rtx.TransactionIndex, rtx.From, rtx.To
	tx.Value, tx.GasPrice = (*hexutil.Big)(rtx.Value), (*hexutil.Big)(rtx.GasPrice)
	tx.Gas, tx.ContractCreated, tx.Data = (*hexutil.Big)(rtx.Gas), rtx.ContractCreated, rtx.Data
	tx.StorageLimit, tx.EpochHeight = (*hexutil.Big)(rtx.StorageLimit), (*hexutil.Big)(rtx.EpochHeight)
	tx.ChainID, tx.Status, tx.V = (*hexutil.Big)(rtx.ChainID), rtx.Status, (*hexutil.Big)(rtx.V)
	tx.R, tx.S = (*hexutil.Big)(rtx.R), (*hexutil.Big)(rtx.S)

	return nil
}
