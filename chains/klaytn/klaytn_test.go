package klaytn

import (
	"bytes"
	"context"
	"fmt"
	"math/big"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/trie"
	ksha3 "github.com/klaytn/klaytn/crypto/sha3"
	"github.com/mapprotocol/compass/internal/klaytn"
	"github.com/mapprotocol/compass/internal/tx"
	"github.com/mapprotocol/compass/pkg/ethclient"
	"golang.org/x/crypto/sha3"
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

func Test_VoteData(t *testing.T) {
	data := common.Hex2Bytes("f84294c0cbe1c770fbce1eb7786bfba1ac2115d5c0a45697676f7665726e616e63652e61646476616c696461746f72948d53a7dd56464ec4ba900cef1e7eab041ba61fc1")
	gVote := new(klaytn.GovernanceVote)
	err := rlp.DecodeBytes(data, gVote)
	if err != nil {
		t.Fatal("Failed to decode a vote", "key", gVote.Key, "value", gVote.Value, "validator", gVote.Validator, "err", err)
	}
	t.Log(gVote)
}

func Test01(t *testing.T) {
	//url := "https://public-node-api.klaytnapi.com/v1/cypress"
	url := "https://klaytn-baobab.blockpi.network/v1/rpc/053093fb6e5f618afa4e50921f5c605088b27175"
	c, e := klaytn.DialHttp(url, true)
	if e != nil {
		t.Error(e)
	}
	//num := big.NewInt(0x7b18371) // 主网任意
	num := big.NewInt(0x7b6541b) // 测试网有map tx的
	//num := big.NewInt(0x7b75b2b) // 测试网任意block，里面无交易，生成的root一致
	//num := big.NewInt(0x7b75b2d) // 测试网任意block，里面两笔交易，生成的root一致
	//num := big.NewInt(0x7b75b30) // 测试网任意block，里面两笔交易，生成的root一致
	//num := big.NewInt(0x7b75b31) // 测试网任意block，里面一笔交易，生成的root一致
	//num := big.NewInt(0x7b775f3) // 最新发送的交易
	block, e := c.BlockByNumber(context.Background(), num)
	if e != nil {
		t.Error(e)
	}
	bn, _ := big.NewInt(0).SetString(strings.TrimPrefix(block.Number, "0x"), 16)
	fmt.Println("vlock:", bn)
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
		fmt.Println("tx ---- ", txh.String())
	}
	receipts, e := tx.GetReceiptsByTxsHash(c2, txs_hash)
	root2 := DeriveSha2(t, receipts)
	root3 := DeriveSha0(t, receipts)
	//root4 := DeriveSha1(t, receipts)

	fmt.Println("root1:", root1.String())
	fmt.Println("contract:", root2.String())
	fmt.Println("origin:", root3.String())
	//fmt.Println("simple:", root4.String())

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

	trie.Reset()
	var buf []byte
	for i := 1; i < len(list) && i <= 0x7f; i++ {
		buf = rlp.AppendUint64(buf[:0], uint64(i))

		d, e := rlp.EncodeToBytes(toReceiptRLP(list[i]))
		if e != nil {
			t.Error(e)
		}
		trie.Update(buf, d)
	}
	if len(list) > 0 {
		buf = rlp.AppendUint64(buf[:0], 0)
		d, e := rlp.EncodeToBytes(toReceiptRLP(list[0]))
		if e != nil {
			t.Error(e)
		}
		trie.Update(buf, d)
	}
	for i := 0x80; i < len(list); i++ {
		buf = rlp.AppendUint64(buf[:0], uint64(i))
		d, e := rlp.EncodeToBytes(toReceiptRLP(list[i]))
		if e != nil {
			t.Error(e)
		}
		trie.Update(buf, d)
	}
	return trie.Hash()
}

func DeriveSha1(t *testing.T, list []*types.Receipt) common.Hash {
	//receipts := make([]*ktypes.Receipt, 0, len(list))
	//for _, l := range list {
	//	logs := make([]*ktypes.Log, 0, len(l.Logs))
	//	for _, ll := range l.Logs {
	//		topics := make([]kcommon.Hash, 0, len(ll.Topics))
	//		for _, tt := range ll.Topics {
	//			topics = append(topics, kcommon.HexToHash(tt.Hex()))
	//		}
	//		logs = append(logs, &ktypes.Log{
	//			Address:     kcommon.HexToAddress(ll.Address.Hex()),
	//			Topics:      topics,
	//			Data:        ll.Data,
	//			BlockNumber: ll.BlockNumber,
	//			TxHash:      kcommon.HexToHash(ll.TxHash.Hex()),
	//			TxIndex:     ll.TxIndex,
	//			BlockHash:   kcommon.HexToHash(ll.BlockHash.Hex()),
	//			Index:       ll.Index,
	//			Removed:     ll.Removed,
	//		})
	//	}
	//	receipts = append(receipts, &ktypes.Receipt{
	//		Status:          uint(l.Status),
	//		Bloom:           ktypes.BytesToBloom(l.Bloom.Bytes()),
	//		Logs:            logs,
	//		TxHash:          kcommon.HexToHash(l.TxHash.Hex()),
	//		ContractAddress: kcommon.HexToAddress(l.ContractAddress.Hex()),
	//		GasUsed:         l.GasUsed,
	//	})
	//}
	//
	//derivesha.DeriveShaConcat{}.DeriveSha(ktypes.Receipts(receipts))

	hasher := ksha3.NewKeccak256()

	encoded := make([][]byte, 0, len(list))
	for i := 0; i < len(list); i++ {
		d, e := rlp.EncodeToBytes(toReceiptRLP(list[i]))
		if e != nil {
			t.Error(e)
		}
		hasher.Write(d)
		encoded = append(encoded, hasher.Sum(nil))
	}

	for len(encoded) > 1 {
		// make even numbers
		if len(encoded)%2 == 1 {
			encoded = append(encoded, encoded[len(encoded)-1])
		}

		for i := 0; i < len(encoded)/2; i++ {
			hasher.Reset()
			hasher.Write(encoded[2*i])
			hasher.Write(encoded[2*i+1])

			encoded[i] = hasher.Sum(nil)
		}

		encoded = encoded[0 : len(encoded)/2]
	}

	if len(encoded) == 0 {
		hasher.Reset()
		hasher.Write(nil)
		return common.BytesToHash(hasher.Sum(nil))
	}

	return common.BytesToHash(encoded[0])
}
