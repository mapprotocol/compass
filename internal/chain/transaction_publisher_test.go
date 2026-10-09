package chain

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"math/big"
	"strings"
	"testing"

	"github.com/ChainSafe/log15"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
)

func newTestDynamicTransaction(t *testing.T) (*ecdsa.PrivateKey, *types.Transaction) {
	t.Helper()
	privateKey, err := crypto.GenerateKey()
	if err != nil {
		t.Fatalf("generate private key: %v", err)
	}
	initial, err := types.SignNewTx(privateKey, types.NewLondonSigner(big.NewInt(196)), &types.DynamicFeeTx{
		ChainID:   big.NewInt(196),
		Nonce:     17,
		GasTipCap: big.NewInt(1_000_000),
		GasFeeCap: big.NewInt(42_000_000),
		Gas:       5_000_000,
		Data:      []byte{1, 2, 3, 4},
	})
	if err != nil {
		t.Fatalf("sign initial transaction: %v", err)
	}
	return privateKey, initial
}

func TestTransactionPublisherBumpsBothDynamicFees(t *testing.T) {
	privateKey, initial := newTestDynamicTransaction(t)

	var sent []*types.Transaction
	send := func(_ context.Context, tx *types.Transaction) error {
		sent = append(sent, tx)
		if len(sent) == 1 {
			return errors.New("replacement transaction underpriced")
		}
		return nil
	}
	publisher := newTransactionPublisher(send, privateKey, big.NewInt(100_000_000), log15.New())

	got, err := publisher.Publish(context.Background(), initial, true)
	if err != nil {
		t.Fatalf("Publish returned error: %v", err)
	}
	if len(sent) != 2 {
		t.Fatalf("send called %d times, want 2", len(sent))
	}
	if got.GasTipCap().Cmp(big.NewInt(1_200_000)) != 0 {
		t.Fatalf("gas tip cap = %s, want 1200000", got.GasTipCap())
	}
	if got.GasFeeCap().Cmp(big.NewInt(50_400_000)) != 0 {
		t.Fatalf("gas fee cap = %s, want 50400000", got.GasFeeCap())
	}
	if got.Nonce() != initial.Nonce() || string(got.Data()) != string(initial.Data()) {
		t.Fatal("replacement changed transaction identity fields")
	}
}

func TestTransactionPublisherStopsAtMaxGasPrice(t *testing.T) {
	privateKey, initial := newTestDynamicTransaction(t)
	calls := 0
	send := func(context.Context, *types.Transaction) error {
		calls++
		return errors.New("replacement transaction underpriced")
	}
	publisher := newTransactionPublisher(send, privateKey, big.NewInt(42_000_000), log15.New())

	_, err := publisher.Publish(context.Background(), initial, true)
	if err == nil || !strings.Contains(err.Error(), "maxGasPrice") {
		t.Fatalf("Publish error = %v, want maxGasPrice error", err)
	}
	if calls != 1 {
		t.Fatalf("send called %d times, want 1", calls)
	}
}

func TestTransactionPublisherLimitsReplacementAttempts(t *testing.T) {
	privateKey, initial := newTestDynamicTransaction(t)
	calls := 0
	send := func(context.Context, *types.Transaction) error {
		calls++
		return errors.New("replacement transaction underpriced")
	}
	publisher := newTransactionPublisher(send, privateKey, big.NewInt(1_000_000_000), log15.New())

	_, err := publisher.Publish(context.Background(), initial, true)
	if err == nil || !strings.Contains(err.Error(), "after 3 replacement attempts") {
		t.Fatalf("Publish error = %v, want retry limit error", err)
	}
	if calls != 4 {
		t.Fatalf("send called %d times, want initial send plus 3 replacements", calls)
	}
}

func TestTransactionPublisherDoesNotReplaceInitialSend(t *testing.T) {
	privateKey, initial := newTestDynamicTransaction(t)
	calls := 0
	wantErr := errors.New("replacement transaction underpriced")
	send := func(context.Context, *types.Transaction) error {
		calls++
		return wantErr
	}
	publisher := newTransactionPublisher(send, privateKey, big.NewInt(1_000_000_000), log15.New())

	_, err := publisher.Publish(context.Background(), initial, false)
	if !errors.Is(err, wantErr) {
		t.Fatalf("Publish error = %v, want %v", err, wantErr)
	}
	if calls != 1 {
		t.Fatalf("send called %d times, want 1", calls)
	}
}
