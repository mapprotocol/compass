package tx

import (
	"context"
	"errors"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/mapprotocol/compass/pkg/ethclient"
	"math/big"
	"time"
)

func GetTxsHashByBlockNumber(conn *ethclient.Client, number *big.Int) ([]common.Hash, error) {
	block, err := conn.BlockByNumber(context.Background(), number)
	if err != nil {
		return nil, err
	}

	txs := make([]common.Hash, 0, len(block.Transactions()))
	for _, tx := range block.Transactions() {
		txs = append(txs, tx.Hash())
	}
	return txs, nil
}

func GetMapTxsHashByBlockNumber(conn *ethclient.Client, number *big.Int) ([]common.Hash, error) {
	block, err := conn.MAPBlockByNumber(context.Background(), number)
	if err != nil {
		return nil, err
	}

	txs := make([]common.Hash, 0, len(block.Transactions()))
	for _, tx := range block.Transactions() {
		txs = append(txs, tx.Hash())
	}
	return txs, nil
}

func GetReceiptsByTxsHash(conn *ethclient.Client, txsHash []common.Hash) ([]*types.Receipt, error) {
	type ele struct {
		r   *types.Receipt
		idx int
	}
	var (
		count      = len(txsHash)
		errReceive = make(chan error)
		receive    = make(chan *ele, len(txsHash))
		rs         = make([]*types.Receipt, len(txsHash))
	)
	go func() {
		for idx, h := range txsHash {
			tmpIdx := idx
			tmpHash := h
			go func(i int, tx common.Hash) {
				for {
					r, err := conn.TransactionReceipt(context.Background(), tx)
					if err != nil {
						if err.Error() == "not found" {
							continue
						}
						//fmt.Println("err ------------------- ", err)
						errReceive <- err
						return
					}
					//fmt.Println("i ", i, "tx ", tx)
					receive <- &ele{
						r:   r,
						idx: i,
					}
					break
				}
			}(tmpIdx, tmpHash)

			if idx%30 == 0 {
				time.Sleep(time.Millisecond * 500)
			}
		}
	}()

	for {
		select {
		case v, ok := <-receive:
			if !ok {
				return nil, errors.New("receive chan is closed")
			}
			if v != nil {
				rs[v.idx] = v.r
			}
			count--
			if count == 0 {
				return rs, nil
			}
		case err := <-errReceive:
			return nil, err
		}
	}
}
