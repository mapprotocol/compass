package proof

import (
	"bytes"
	"math/big"
	"sync"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/mapprotocol/compass/mapprotocol"
	"github.com/mapprotocol/compass/msg"
	"github.com/mapprotocol/compass/pkg/util"
	"github.com/pkg/errors"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethdb/memorydb"
	"github.com/ethereum/go-ethereum/light"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/trie"
)

var (
	CacheReceipt = make(map[string][]*types.Receipt) // key -> chainId_blockHeight
)

type ReceiptRLP struct {
	PostStateOrStatus []byte
	CumulativeGasUsed uint64
	Bloom             types.Bloom
	Logs              []*types.Log
}

type Data struct {
	BlockNum     *big.Int
	ReceiptProof ReceiptProof
}

type ReceiptProof struct {
	TxReceipt mapprotocol.TxReceipt
	KeyIndex  []byte
	Proof     [][]byte
}

type NewData struct {
	BlockNum     *big.Int
	ReceiptProof NewReceiptProof
}

type NewReceiptProof struct {
	TxReceipt   []byte
	ReceiptType *big.Int
	KeyIndex    []byte
	Proof       [][]byte
}

var encodeBufferPool = sync.Pool{
	New: func() interface{} { return new(bytes.Buffer) },
}

func Get(receipts DerivableList, txIndex uint) ([][]byte, error) {
	tr, err := trie.New(common.Hash{}, trie.NewDatabase(memorydb.New()))
	if err != nil {
		return nil, err
	}

	tr = DeriveTire(receipts, tr)
	ns := light.NewNodeSet()
	key, err := rlp.EncodeToBytes(txIndex)
	if err != nil {
		return nil, err
	}
	if err = tr.Prove(key, 0, ns); err != nil {
		return nil, err
	}

	proof := make([][]byte, 0, len(ns.NodeList()))
	for _, v := range ns.NodeList() {
		proof = append(proof, v)
	}

	return proof, nil
}

func DeriveTire(rs DerivableList, tr *trie.Trie) *trie.Trie {
	valueBuf := encodeBufferPool.Get().(*bytes.Buffer)
	defer encodeBufferPool.Put(valueBuf)

	var indexBuf []byte
	for i := 1; i < rs.Len() && i <= 0x7f; i++ {
		indexBuf = rlp.AppendUint64(indexBuf[:0], uint64(i))
		value := encodeForDerive(rs, i, valueBuf)
		tr.Update(indexBuf, value)
	}
	if rs.Len() > 0 {
		indexBuf = rlp.AppendUint64(indexBuf[:0], 0)
		value := encodeForDerive(rs, 0, valueBuf)
		tr.Update(indexBuf, value)
	}
	for i := 0x80; i < rs.Len(); i++ {
		indexBuf = rlp.AppendUint64(indexBuf[:0], uint64(i))
		value := encodeForDerive(rs, i, valueBuf)
		tr.Update(indexBuf, value)
	}
	return tr
}

type DerivableList interface {
	Len() int
	EncodeIndex(int, *bytes.Buffer)
}

func encodeForDerive(list DerivableList, i int, buf *bytes.Buffer) []byte {
	buf.Reset()
	list.EncodeIndex(i, buf)
	return common.CopyBytes(buf.Bytes())
}

func Pack(fId msg.ChainId, method string, abi abi.ABI, params ...interface{}) ([]byte, error) {
	input, err := abi.Methods[mapprotocol.MethodOfGetBytes].Inputs.Pack(params...)
	if err != nil {
		return nil, errors.Wrap(err, "pack getBytes failed")
	}

	ret, err := mapprotocol.PackInput(mapprotocol.Mcs, method, big.NewInt(0).SetUint64(uint64(fId)), input)
	if err != nil {
		return nil, errors.Wrap(err, "pack mcs input failed")
	}

	return ret, nil
}

func Oracle(blockNumber uint64, receipt *mapprotocol.TxReceipt, key []byte, prf [][]byte, fId msg.ChainId, method string, idx uint,
	abi abi.ABI) ([]byte, error) {
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

	pd := NewData{
		BlockNum: big.NewInt(int64(blockNumber)),
		ReceiptProof: NewReceiptProof{
			TxReceipt:   nrRlp,
			ReceiptType: receipt.ReceiptType,
			KeyIndex:    util.Key2Hex(key, len(prf)),
			Proof:       prf,
		},
	}

	input, err := abi.Methods[mapprotocol.MethodOfGetBytes].Inputs.Pack(pd)
	if err != nil {
		return nil, errors.Wrap(err, "pack getBytes failed")
	}

	if method == mapprotocol.MethodOfTransferInWithIndex {
		return mapprotocol.PackInput(mapprotocol.Mcs, method, big.NewInt(int64(fId)), big.NewInt(int64(idx)), input)
	}
	ret, err := mapprotocol.PackInput(mapprotocol.Mcs, method, big.NewInt(0).SetUint64(uint64(fId)), input)
	if err != nil {
		return nil, errors.Wrap(err, "pack mcs input failed")
	}

	return ret, nil
}

func V3Pack(fId msg.ChainId, method string, abi abi.ABI, idx uint, params ...interface{}) ([]byte, error) {
	input, err := abi.Methods[mapprotocol.MethodOfGetBytes].Inputs.Pack(params...)
	if err != nil {
		return nil, errors.Wrap(err, "pack getBytes failed")
	}

	if method == mapprotocol.MethodOfTransferInWithIndex || method == mapprotocol.MethodOfSwapInWithIndex {
		return mapprotocol.PackInput(mapprotocol.Mcs, method, big.NewInt(0).SetUint64(uint64(fId)), big.NewInt(int64(idx)), input)
	}
	ret, err := mapprotocol.PackInput(mapprotocol.Mcs, method, big.NewInt(0).SetUint64(uint64(fId)), input)
	if err != nil {
		return nil, errors.Wrap(err, "pack mcs input failed")
	}

	return ret, nil
}
