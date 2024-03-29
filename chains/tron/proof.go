package tron

import (
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/mapprotocol/compass/internal/constant"
	"github.com/mapprotocol/compass/internal/proof"
	"github.com/mapprotocol/compass/mapprotocol"
	"github.com/mapprotocol/compass/msg"
	"github.com/mapprotocol/compass/pkg/util"
	"math/big"
)

func assembleProof(log *types.Log, receipts []*types.Receipt, method string, fId msg.ChainId, proofType int64) ([]byte, error) {
	var (
		input []byte
	)
	receipt, err := mapprotocol.GetTxReceipt(receipts[log.TxIndex])
	if err != nil {
		return nil, err
	}
	prf, err := proof.Get(types.Receipts(receipts), log.TxIndex)
	if err != nil {
		return nil, err
	}

	var key []byte
	key = rlp.AppendUint64(key[:0], uint64(log.TxIndex))

	switch proofType {
	case constant.ProofTypeOfOracle:
		pd := proof.Data{
			BlockNum: big.NewInt(int64(log.BlockNumber)),
			ReceiptProof: proof.ReceiptProof{
				TxReceipt: *receipt,
				KeyIndex:  util.Key2Hex(key, len(prf)),
				Proof:     prf,
			},
		}
		input, err = mapprotocol.OracleAbi.Methods[mapprotocol.MethodOfGetBytes].Inputs.Pack(pd)
		if err != nil {
			return nil, err
		}
	default:
		panic("not support")
	}

	ret, err := mapprotocol.PackInput(mapprotocol.Mcs, method, new(big.Int).SetUint64(uint64(fId)), input)
	if err != nil {
		return nil, err
	}
	return ret, nil
}
