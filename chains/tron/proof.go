package tron

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethdb/memorydb"
	"github.com/ethereum/go-ethereum/trie"
	maptypes "github.com/mapprotocol/atlas/core/types"
	"math/big"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/mapprotocol/compass/internal/constant"
	"github.com/mapprotocol/compass/internal/proof"
	"github.com/mapprotocol/compass/mapprotocol"
	"github.com/mapprotocol/compass/msg"
	"github.com/mapprotocol/compass/pkg/util"
)

func assembleProof(log *types.Log, receipts []*types.Receipt, method string, fId, toChainId msg.ChainId,
	proofType int64, orderId [32]byte) ([]byte, error) {
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
	idx := 0
	for i, ele := range receipts[log.TxIndex].Logs {
		if ele.Index != log.Index {
			continue
		}
		idx = i
	}

	switch proofType {
	case constant.ProofTypeOfOracle:
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
		pd := proof.NewData{
			BlockNum: big.NewInt(int64(log.BlockNumber)),
			ReceiptProof: proof.NewReceiptProof{
				ReceiptType: receipt.ReceiptType,
				TxReceipt:   nrRlp,
				KeyIndex:    util.Key2Hex(key, len(prf)),
				Proof:       prf,
			},
		}
		ret, err = proof.Pack(fId, method, mapprotocol.ProofAbi, pd)
	case constant.ProofTypeOfNewOracle:
		tr, _ := trie.New(common.Hash{}, trie.NewDatabase(memorydb.New()))
		tr = proof.DeriveTire(types.Receipts(receipts), tr)
		signerRet, err := getSigner(log, tr.Hash(), uint64(fId), uint64(toChainId))
		if err != nil {
			return nil, err
		}
		ret, err = proof.SignOracle(&maptypes.Header{
			ReceiptHash: tr.Hash(),
			Number:      big.NewInt(int64(log.BlockNumber)),
		}, receipt, key, prf, fId, idx, method, signerRet.Signatures, orderId, log, proofType)
	default:
		panic("not support")
	}

	if err != nil {
		return nil, err
	}
	return ret, nil
}
