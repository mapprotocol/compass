package ethclient

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/mapprotocol/atlas/consensus/istanbul/backend"
	"github.com/mapprotocol/atlas/core/types"
)

type rpcMAPBlock struct {
	Hash           common.Hash           `json:"hash"`
	Transactions   []rpcMAPTransaction   `json:"transactions"`
	Randomness     *types.Randomness     `json:"randomness"`
	EpochSnarkData *types.EpochSnarkData `json:"epochSnarkData"`
}

type rpcMAPTransaction struct {
	tx *types.Transaction
	txExtraInfo
}

func (tx *rpcMAPTransaction) UnmarshalJSON(msg []byte) error {
	if err := json.Unmarshal(msg, &tx.tx); err != nil {
		return err
	}
	return json.Unmarshal(msg, &tx.txExtraInfo)
}

func (ec *Client) getMAPBlock(ctx context.Context, method string, args ...interface{}) (*types.Block, error) {
	var raw json.RawMessage
	err := ec.c.CallContext(ctx, &raw, method, args...)
	if err != nil {
		return nil, err
	} else if len(raw) == 0 {
		return nil, ethereum.NotFound
	}
	// Decode header and transactions.
	var head *types.Header
	var body rpcMAPBlock
	if err := json.Unmarshal(raw, &head); err != nil {
		return nil, err
	}
	if err := json.Unmarshal(raw, &body); err != nil {
		return nil, err
	}
	if head.TxHash == types.EmptyRootHash && len(body.Transactions) > 0 {
		return nil, fmt.Errorf("server returned non-empty transaction list but block header indicates no transactions")
	}
	if head.TxHash != types.EmptyRootHash && len(body.Transactions) == 0 {
		return nil, fmt.Errorf("server returned empty transaction list but block header indicates transactions")
	}
	// Fill the sender cache of transactions in the block.
	txs := make([]*types.Transaction, len(body.Transactions))
	for i, tx := range body.Transactions {
		if tx.From != nil {
			setMAPSenderFromServer(tx.tx, *tx.From, body.Hash)
		}
		txs[i] = tx.tx
	}
	return types.NewBlockWithHeader(head).WithBody(txs, body.Randomness, body.EpochSnarkData), nil
}

// MAPBlockByHash returns the given full block.
//
// Note that loading full blocks requires two requests. Use HeaderByHash
// if you don't need all transactions or uncle headers.
func (ec *Client) MAPBlockByHash(ctx context.Context, hash common.Hash) (*types.Block, error) {
	return ec.getMAPBlock(ctx, "eth_getBlockByHash", hash, true)
}

// MAPBlockByNumber returns a block from the current canonical chain. If number is nil, the
// latest known block is returned.
//
// Note that loading full blocks requires two requests. Use HeaderByNumber
// if you don't need all transactions or uncle headers.
func (ec *Client) MAPBlockByNumber(ctx context.Context, number *big.Int) (*types.Block, error) {
	return ec.getMAPBlock(ctx, "eth_getBlockByNumber", toBlockNumArg(number), true)
}

// MAPHeaderByNumber returns a block header from the current canonical chain. If number is
// nil, the latest known header is returned.
func (ec *Client) MAPHeaderByNumber(ctx context.Context, number *big.Int) (*types.Header, error) {
	var head *types.Header
	err := ec.c.CallContext(ctx, &head, "eth_getBlockByNumber", toBlockNumArg(number), false)
	if err == nil && head == nil {
		err = ethereum.NotFound
	}
	return head, err
}

func (ec *Client) GetSnapshot(ctx context.Context, number *big.Int) (*backend.Snapshot, error) {
	var snap *backend.Snapshot
	err := ec.c.CallContext(ctx, &snap, "istanbul_getSnapshot", toBlockNumArg(number))
	if err != nil {
		return nil, err
	}
	return snap, err
}
