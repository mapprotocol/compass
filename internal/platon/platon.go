package platon

import (
	"math/big"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
	iproof "github.com/mapprotocol/compass/internal/proof"
	"github.com/mapprotocol/compass/mapprotocol"
	"github.com/mapprotocol/compass/msg"
	utils "github.com/mapprotocol/compass/shared/ethereum"
)

type ReceiptProof struct {
	TxReceipt mapprotocol.TxReceipt
	KeyIndex  []byte
	Proof     [][]byte
}

func AssembleProof(log types.Log, receipts []*types.Receipt, method string, fId msg.ChainId) ([]byte, error) {
	txIndex := log.TxIndex
	receipt, err := mapprotocol.GetTxReceipt(receipts[txIndex])
	if err != nil {
		return nil, err
	}

	proof, err := iproof.Get(receipts, txIndex)
	if err != nil {
		return nil, err
	}

	var key []byte
	key = rlp.AppendUint64(key[:0], uint64(txIndex))
	ek := utils.Key2Hex(key, len(proof))

	// todo 缺少 block
	pd := ReceiptProof{
		TxReceipt: *receipt,
		KeyIndex:  ek,
		Proof:     proof,
	}

	input, err := mapprotocol.Platon.Methods[mapprotocol.MethodOfGetBytes].Inputs.Pack(pd)
	if err != nil {
		return nil, err
	}

	pack, err := mapprotocol.PackInput(mapprotocol.Mcs, method, new(big.Int).SetUint64(uint64(fId)), input)
	if err != nil {
		return nil, err
	}
	return pack, nil
}
