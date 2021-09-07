package atlas

import (
	"context"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/ethdb/memorydb"
	"github.com/ethereum/go-ethereum/light"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/trie"
	"github.com/mapprotocol/compass/chains"
	types2 "github.com/mapprotocol/compass/types"
	"math/big"
)

var (
	tr              *trie.Trie
	receipts        []*types.Receipt
	lastBlockNumber uint64
)

func GetTxProve(src chains.ChainInterface, aLog *types.Log, eventResponse *types2.EventLogSwapOutResponse) []byte {

	// 调用以太坊接口获取 receipts
	blockNumber := aLog.BlockNumber
	transactionIndex := aLog.TxIndex
	if blockNumber != lastBlockNumber {
		queryNewReceiptsAndTr(blockNumber, src.GetClient())
	}

	proof := light.NewNodeSet()
	key, err := rlp.EncodeToBytes(transactionIndex)
	if err != nil {
		panic(err)
	}
	if err = tr.Prove(key, 0, proof); err != nil {
		panic(err)
	}

	txProve := TxProve{
		Tx: &TxParams{
			From:  aLog.Topics[2].Bytes(),
			To:    aLog.Topics[3].Bytes(),
			Value: eventResponse.Amount,
		},
		Receipt:          receipts[transactionIndex],
		Prove:            proof.NodeList(),
		TransactionIndex: transactionIndex,
	}

	input, err := rlp.EncodeToBytes(txProve)
	if err != nil {
		panic(err)
	}
	return input
}
func queryNewReceiptsAndTr(blockNumber uint64, conn *ethclient.Client) {
	txsHash := getTransactionsHashByBlockNumber(conn, big.NewInt(int64(blockNumber)))
	receipts = getReceiptsByTxsHash(conn, txsHash)

	// 根据 receipts 生成 trie
	var err error
	tr, err = trie.New(common.Hash{}, trie.NewDatabase(memorydb.New()))
	if err != nil {
		panic(err)
	}
	for i, r := range receipts {
		key, err := rlp.EncodeToBytes(uint(i))
		if err != nil {
			panic(err)
		}
		value, err := rlp.EncodeToBytes(r)
		if err != nil {
			panic(err)
		}

		tr.Update(key, value)
	}
}
func getTransactionsHashByBlockNumber(conn *ethclient.Client, number *big.Int) []common.Hash {
	block, err := conn.BlockByNumber(context.Background(), number)
	if err != nil {
		panic(err)
	}
	if block == nil {
		panic("failed to connect to the eth node, please check the network")
	}

	txs := make([]common.Hash, 0, len(block.Transactions()))
	for _, tx := range block.Transactions() {
		txs = append(txs, tx.Hash())
	}
	return txs
}

func getReceiptsByTxsHash(conn *ethclient.Client, txsHash []common.Hash) []*types.Receipt {
	rs := make([]*types.Receipt, 0, len(txsHash))
	for _, h := range txsHash {
		r, err := conn.TransactionReceipt(context.Background(), h)
		if err != nil {
			panic(err)
		}
		if r == nil {
			panic("failed to connect to the eth node, please check the network")
		}
		rs = append(rs, r)
	}
	return rs

}
