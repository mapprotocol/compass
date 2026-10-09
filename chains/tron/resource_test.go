package tron

import (
	"strings"
	"testing"

	"github.com/ChainSafe/log15"
	"github.com/lbtsm/gotron-sdk/pkg/proto/api"
	"github.com/lbtsm/gotron-sdk/pkg/proto/core"
	"github.com/pkg/errors"
)

type fakeQuerier struct {
	resource    *api.AccountResourceMessage
	resourceErr error
	account     *core.Account
	accountErr  error
}

func (f *fakeQuerier) GetAccountResource(addr string) (*api.AccountResourceMessage, error) {
	return f.resource, f.resourceErr
}

func (f *fakeQuerier) GetAccount(addr string) (*core.Account, error) {
	return f.account, f.accountErr
}

func testLogger() log15.Logger {
	logger := log15.New()
	logger.SetHandler(log15.DiscardHandler())
	return logger
}

func TestCheckEnergySupply(t *testing.T) {
	tests := []struct {
		name    string
		cli     *fakeQuerier
		used    int64
		wantErr string
	}{
		{
			name: "enough energy",
			cli:  &fakeQuerier{resource: &api.AccountResourceMessage{EnergyLimit: 200000, EnergyUsed: 50000}},
			used: 100000,
		},
		{
			name:    "available energy below estimate with buffer",
			cli:     &fakeQuerier{resource: &api.AccountResourceMessage{EnergyLimit: 200000, EnergyUsed: 95000}},
			used:    100000, // need 110000, only 105000 left
			wantErr: "energy supply account(TSupply) energy not enough",
		},
		{
			name:    "query failed",
			cli:     &fakeQuerier{resourceErr: errors.New("grpc timeout")},
			used:    100000,
			wantErr: "energy supply account(TSupply) query resource failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := checkEnergySupply(tt.cli, testLogger(), "TSupply", tt.used)
			assertErr(t, err, tt.wantErr)
		})
	}
}

func TestCheckSelfResource(t *testing.T) {
	tests := []struct {
		name    string
		cli     *fakeQuerier
		used    int64
		minTrx  float64
		wantErr string
	}{
		{
			name: "enough energy and trx",
			cli: &fakeQuerier{
				resource: &api.AccountResourceMessage{EnergyLimit: 200000, EnergyUsed: 0},
				account:  &core.Account{Balance: 200 * 1e6},
			},
			used:   100000,
			minTrx: 100,
		},
		{
			name: "energy not enough",
			cli: &fakeQuerier{
				resource: &api.AccountResourceMessage{EnergyLimit: 109999, EnergyUsed: 0},
				account:  &core.Account{Balance: 200 * 1e6},
			},
			used:    100000,
			minTrx:  100,
			wantErr: "sender account(TFrom) energy not enough",
		},
		{
			name: "trx below threshold",
			cli: &fakeQuerier{
				resource: &api.AccountResourceMessage{EnergyLimit: 200000, EnergyUsed: 0},
				account:  &core.Account{Balance: 99 * 1e6},
			},
			used:    100000,
			minTrx:  100,
			wantErr: "sender account(TFrom) trx balance not enough",
		},
		{
			name: "resource query failed",
			cli: &fakeQuerier{
				resourceErr: errors.New("grpc timeout"),
				account:     &core.Account{Balance: 200 * 1e6},
			},
			used:    100000,
			minTrx:  100,
			wantErr: "sender account(TFrom) query resource failed",
		},
		{
			name: "account query failed",
			cli: &fakeQuerier{
				resource:   &api.AccountResourceMessage{EnergyLimit: 200000, EnergyUsed: 0},
				accountErr: errors.New("grpc timeout"),
			},
			used:    100000,
			minTrx:  100,
			wantErr: "sender account(TFrom) query account failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := checkSelfResource(tt.cli, testLogger(), "TFrom", tt.used, tt.minTrx)
			assertErr(t, err, tt.wantErr)
		})
	}
}

func assertErr(t *testing.T, err error, want string) {
	t.Helper()
	if want == "" {
		if err != nil {
			t.Fatalf("expect no error, got %v", err)
		}
		return
	}
	if err == nil {
		t.Fatalf("expect error contains %q, got nil", want)
	}
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("expect error contains %q, got %q", want, err.Error())
	}
}
