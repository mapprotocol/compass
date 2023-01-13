// Copyright 2021 Compass Systems
// SPDX-License-Identifier: LGPL-3.0-only

package ethereum

import (
	"context"
	eth "github.com/ethereum/go-ethereum"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	maptypes "github.com/mapprotocol/atlas/core/types"
	"github.com/mapprotocol/compass/pkg/ethclient"
)

/*
	some common client operations
*/

func getTransactionsHashByBlockNumber(conn *ethclient.Client, number *big.Int) ([]common.Hash, error) {
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

func getMapTransactionsHashByBlockNumber(conn *ethclient.Client, number *big.Int) ([]common.Hash, error) {
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

func getReceiptsByTxsHash(conn *ethclient.Client, txsHash []common.Hash) ([]*types.Receipt, error) {
	rs := make([]*types.Receipt, 0, len(txsHash))
	for _, h := range txsHash {
		r, err := conn.TransactionReceipt(context.Background(), h)
		if err != nil {
			return nil, err
		}
		rs = append(rs, r)
	}
	return rs, nil
}

func getLastReceipt(conn *ethclient.Client, latestBlock *big.Int) (*types.Receipt, error) {
	query := eth.FilterQuery{
		FromBlock: latestBlock,
		ToBlock:   latestBlock,
	}
	lastLog, err := conn.FilterLogs(context.Background(), query)
	if err != nil {
		return nil, err
	}
	receipt := maptypes.NewReceipt(nil, false, 0)
	rl := make([]*maptypes.Log, 0, len(lastLog))
	el := make([]*types.Log, 0, len(lastLog))
	for idx, ll := range lastLog {
		if idx == 0 {
			continue
		}
		if ll.TxHash != ll.BlockHash {
			continue
		}
		rl = append(rl, &maptypes.Log{
			Address:     ll.Address,
			Topics:      ll.Topics,
			Data:        ll.Data,
			BlockNumber: ll.BlockNumber,
			TxHash:      ll.TxHash,
			TxIndex:     ll.TxIndex,
			BlockHash:   ll.BlockHash,
			Index:       ll.Index,
			Removed:     ll.Removed,
		})
		tl := ll
		el = append(el, &tl)
	}
	receipt.Logs = rl
	receipt.Bloom = maptypes.CreateBloom(maptypes.Receipts{receipt})
	return &types.Receipt{
		Type:              receipt.Type,
		PostState:         receipt.PostState,
		Status:            receipt.Status,
		CumulativeGasUsed: receipt.CumulativeGasUsed,
		Bloom:             types.BytesToBloom(receipt.Bloom.Bytes()),
		Logs:              el,
		TxHash:            receipt.TxHash,
		ContractAddress:   receipt.ContractAddress,
		GasUsed:           receipt.GasUsed,
		BlockHash:         receipt.BlockHash,
		BlockNumber:       receipt.BlockNumber,
		TransactionIndex:  receipt.TransactionIndex,
	}, nil
}
