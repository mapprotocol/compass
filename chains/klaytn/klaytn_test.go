package klaytn

import (
	"bytes"
	"context"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/trie"
	"github.com/mapprotocol/compass/internal/klaytn"
	"github.com/mapprotocol/compass/internal/tx"
	"github.com/mapprotocol/compass/pkg/ethclient"
	"golang.org/x/crypto/sha3"
	"math/big"
	"testing"
)

type receiptRLP struct {
	Status  uint
	GasUsed uint64
	Bloom   types.Bloom
	Logs    []*types.Log
}

func toReceiptRLP(r *types.Receipt) *receiptRLP {
	return &receiptRLP{
		Status:  uint(r.Status),
		GasUsed: r.GasUsed,
		Bloom:   r.Bloom,
		Logs:    r.Logs,
	}
}
func Test01(t *testing.T) {
	url := "https://public-node-api.klaytnapi.com/v1/cypress"
	c, e := klaytn.DialHttp(url, true)
	if e != nil {
		t.Error(e)
	}
	num := big.NewInt(0x67503f8)
	block, e := c.BlockByNumber(context.Background(), num)
	if e != nil {
		t.Error(e)
	}
	fmt.Println("ReceiptsRoot:", block.ReceiptsRoot.String())
	fmt.Println("txsroot:", block.TransactionsRoot.String())
	fmt.Println("parent:", block.ParentHash.String())
	root1 := block.ReceiptsRoot
	c2, e := ethclient.Dial(url)
	if e != nil {
		t.Error(e)
	}
	txs_hash, e := klaytn.GetTxsHashByBlockNumber(c, num)
	if e != nil {
		t.Error(e)
	}
	for _, txh := range txs_hash {
		fmt.Println(txh.String())
	}
	receipts, e := tx.GetReceiptsByTxsHash(c2, txs_hash)
	root2 := DeriveSha2(t, receipts)

	fmt.Println("root1:", root1.String())
	fmt.Println("root2:", root2.String())

	if bytes.Equal(root1[:], root2[:]) {
		fmt.Println("equal")
	} else {
		fmt.Println("fault")
	}
}
func DeriveSha2(t *testing.T, list []*types.Receipt) (hash common.Hash) {
	//hasher0 := sha3.NewKeccak256()
	hasher := sha3.NewLegacyKeccak256()
	fmt.Println("============")
	for i := 0; i < len(list); i++ {
		d, e := rlp.EncodeToBytes(toReceiptRLP(list[i]))
		if e != nil {
			t.Error(e)
		}
		hasher.Write(d)
	}
	hasher.Sum(hash[:0])

	return hash
}
func DeriveSha0(t *testing.T, list []*types.Receipt) common.Hash {
	trie := trie.NewStackTrie(rawdb.NewMemoryDatabase())
	//trie := statedb.NewStackTrie(nil)

	trie.Reset()
	var buf []byte

	// StackTrie requires values to be inserted in increasing
	// hash order, which is not the order that `list` provides
	// hashes in. This insertion sequence ensures that the
	// order is correct.
	for i := 1; i < len(list) && i <= 0x7f; i++ {
		buf = rlp.AppendUint64(buf[:0], uint64(i))
		d, e := rlp.EncodeToBytes(list[i])
		if e != nil {
			t.Error(e)
		}
		trie.Update(buf, d)
	}
	if len(list) > 0 {
		buf = rlp.AppendUint64(buf[:0], 0)
		d, e := rlp.EncodeToBytes(list[0])
		if e != nil {
			t.Error(e)
		}
		trie.Update(buf, d)
	}
	for i := 0x80; i < len(list); i++ {
		buf = rlp.AppendUint64(buf[:0], uint64(i))
		d, e := rlp.EncodeToBytes(list[i])
		if e != nil {
			t.Error(e)
		}
		trie.Update(buf, d)
	}
	return trie.Hash()
}
