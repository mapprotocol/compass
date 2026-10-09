package chain

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"math/big"
	"strings"

	"github.com/ChainSafe/log15"
	"github.com/ethereum/go-ethereum/core/types"
)

const (
	replacementFeeBumpPercent = int64(20)
	maxReplacementAttempts    = 3
)

type sendTransactionFunc func(context.Context, *types.Transaction) error

type transactionPublisher struct {
	send      sendTransactionFunc
	key       *ecdsa.PrivateKey
	maxFeeCap *big.Int
	log       log15.Logger
}

func newTransactionPublisher(send sendTransactionFunc, key *ecdsa.PrivateKey,
	maxFeeCap *big.Int, logger log15.Logger) *transactionPublisher {
	return &transactionPublisher{
		send:      send,
		key:       key,
		maxFeeCap: maxFeeCap,
		log:       logger,
	}
}

func (p *transactionPublisher) Publish(ctx context.Context, tx *types.Transaction,
	allowReplacement bool) (*types.Transaction, error) {
	replacementAttempts := 0
	for {
		err := p.send(ctx, tx)
		if err == nil {
			return tx, nil
		}
		if !allowReplacement || !isReplacementUnderpriced(err) {
			return tx, err
		}
		if replacementAttempts >= maxReplacementAttempts {
			return tx, fmt.Errorf("transaction still underpriced after %d replacement attempts: %w",
				maxReplacementAttempts, err)
		}

		replacement, bumpErr := p.bumpDynamicFees(tx)
		if bumpErr != nil {
			return tx, bumpErr
		}
		replacementAttempts++
		if p.log != nil {
			p.log.Warn("Retry replacement transaction with bumped fees",
				"attempt", replacementAttempts,
				"nonce", replacement.Nonce(),
				"txHash", replacement.Hash(),
				"gasTipCap", replacement.GasTipCap(),
				"gasFeeCap", replacement.GasFeeCap())
		}
		tx = replacement
	}
}

func (p *transactionPublisher) bumpDynamicFees(tx *types.Transaction) (*types.Transaction, error) {
	if tx.Type() != types.DynamicFeeTxType {
		return nil, fmt.Errorf("replacement fee bump requires EIP-1559 transaction, got type %d", tx.Type())
	}

	oldTipCap := tx.GasTipCap()
	oldFeeCap := tx.GasFeeCap()
	tipCap := bumpFee(oldTipCap)
	feeCap := bumpFee(oldFeeCap)
	if p.maxFeeCap != nil {
		if tipCap.Cmp(p.maxFeeCap) > 0 {
			tipCap.Set(p.maxFeeCap)
		}
		if feeCap.Cmp(p.maxFeeCap) > 0 {
			feeCap.Set(p.maxFeeCap)
		}
	}
	if tipCap.Cmp(oldTipCap) <= 0 || feeCap.Cmp(oldFeeCap) <= 0 {
		return nil, fmt.Errorf("maxGasPrice %s prevents replacement fee bump from tipCap=%s feeCap=%s",
			p.maxFeeCap, oldTipCap, oldFeeCap)
	}

	replacement, err := types.SignNewTx(p.key, types.NewLondonSigner(tx.ChainId()), &types.DynamicFeeTx{
		ChainID:    new(big.Int).Set(tx.ChainId()),
		Nonce:      tx.Nonce(),
		GasTipCap:  tipCap,
		GasFeeCap:  feeCap,
		Gas:        tx.Gas(),
		To:         tx.To(),
		Value:      new(big.Int).Set(tx.Value()),
		Data:       append([]byte(nil), tx.Data()...),
		AccessList: append(types.AccessList(nil), tx.AccessList()...),
	})
	if err != nil {
		return nil, fmt.Errorf("sign replacement transaction: %w", err)
	}
	return replacement, nil
}

func bumpFee(value *big.Int) *big.Int {
	bumped := new(big.Int).Mul(value, big.NewInt(100+replacementFeeBumpPercent))
	bumped.Div(bumped, big.NewInt(100))
	if bumped.Cmp(value) <= 0 {
		bumped.Add(value, big.NewInt(1))
	}
	return bumped
}

func isReplacementUnderpriced(err error) bool {
	return err != nil && strings.Contains(strings.ToLower(err.Error()), "replacement transaction underpriced")
}
