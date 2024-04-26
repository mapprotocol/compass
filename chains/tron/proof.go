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
	receipt, err := mapprotocol.GetTxReceipt(receipts[log.TxIndex])
	if err != nil {
		return nil, err
	}
	prf, err := proof.Get(types.Receipts(receipts), log.TxIndex)
	if err != nil {
		return nil, err
	}

	var key, ret []byte
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
		ret, err = proof.Pack(fId, method, mapprotocol.OracleAbi, pd)
	default:
		panic("not support")
	}

	if err != nil {
		return nil, err
	}
	return ret, nil
}
