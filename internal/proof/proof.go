package proof

import (
	"bytes"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/mapprotocol/compass/mapprotocol"
	"math/big"
	"sync"

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

var encodeBufferPool = sync.Pool{
	New: func() interface{} { return new(bytes.Buffer) },
}

func Get(receipts DerivableList, txIndex uint) ([][]byte, error) {
	tr, err := trie.New(common.Hash{}, trie.NewDatabase(memorydb.New()))
	if err != nil {
		return nil, err
	}

	tr = deriveTire(receipts, tr)
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

func deriveTire(rs DerivableList, tr *trie.Trie) *trie.Trie {
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
