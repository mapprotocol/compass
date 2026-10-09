package tron

import (
	"fmt"

	"github.com/ChainSafe/log15"
	"github.com/lbtsm/gotron-sdk/pkg/proto/api"
	"github.com/lbtsm/gotron-sdk/pkg/proto/core"
)

// energyBuffer is the safety factor applied to the estimated energy of a tx.
const energyBuffer = 1.1

// resourceQuerier is the subset of the tron client used by the resource pre-check.
type resourceQuerier interface {
	GetAccountResource(addr string) (*api.AccountResourceMessage, error)
	GetAccount(addr string) (*core.Account, error)
}

// checkEnergySupply reports whether the account delegating energy still has
// enough left to cover a tx estimated at used energy.
//
// The returned error is used as the alarm text, so it carries no live numbers:
// util.Alarm dedupes by exact message, and a changing message would defeat it.
// The live numbers go to the log instead.
func checkEnergySupply(cli resourceQuerier, log log15.Logger, addr string, used int64) error {
	need := int64(float64(used) * energyBuffer)
	res, err := cli.GetAccountResource(addr)
	if err != nil {
		log.Error("Check energy supply, GetAccountResource failed", "account", addr, "err", err)
		return fmt.Errorf("tron energy supply account(%s) query resource failed, please check", addr)
	}

	available := res.EnergyLimit - res.EnergyUsed
	log.Info("Check energy supply", "account", addr, "available", available, "need", need)
	if available < need {
		return fmt.Errorf("tron energy supply account(%s) energy not enough, please recharge", addr)
	}
	return nil
}

// checkSelfResource reports whether the sending account has enough energy for a
// tx estimated at used energy, and at least minTrx of trx left.
func checkSelfResource(cli resourceQuerier, log log15.Logger, addr string, used int64, minTrx float64) error {
	need := int64(float64(used) * energyBuffer)
	res, err := cli.GetAccountResource(addr)
	if err != nil {
		log.Error("Check sender resource, GetAccountResource failed", "account", addr, "err", err)
		return fmt.Errorf("tron sender account(%s) query resource failed, please check", addr)
	}

	account, err := cli.GetAccount(addr)
	if err != nil {
		log.Error("Check sender resource, GetAccount failed", "account", addr, "err", err)
		return fmt.Errorf("tron sender account(%s) query account failed, please check", addr)
	}

	available := res.EnergyLimit - res.EnergyUsed
	balance := float64(account.Balance) / 1e6
	log.Info("Check sender resource", "account", addr, "available", available, "need", need,
		"trx", balance, "minTrx", minTrx)
	if available < need {
		return fmt.Errorf("tron sender account(%s) energy not enough, please recharge", addr)
	}
	if balance < minTrx {
		return fmt.Errorf("tron sender account(%s) trx balance not enough, please recharge", addr)
	}
	return nil
}
