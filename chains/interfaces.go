// Copyright 2021 Compass Systems
// SPDX-License-Identifier: LGPL-3.0-only

package chains

import (
	"github.com/ChainSafe/log15"
	"github.com/mapprotocol/compass/chains/bsc"
	"github.com/mapprotocol/compass/chains/conflux"
	"github.com/mapprotocol/compass/chains/eth2"
	"github.com/mapprotocol/compass/chains/ethereum"
	"github.com/mapprotocol/compass/chains/klaytn"
	"github.com/mapprotocol/compass/chains/matic"
	"github.com/mapprotocol/compass/chains/near"
	"github.com/mapprotocol/compass/chains/sol"
	"github.com/mapprotocol/compass/chains/ton"
	"github.com/mapprotocol/compass/chains/tron"
	"github.com/mapprotocol/compass/core"
	"github.com/mapprotocol/compass/mapprotocol"
)

const (
	Bsc      = "bsc"
	Conflux  = "conflux"
	Eth2     = "eth2"
	Ethereum = "ethereum"
	Klaytn   = "klaytn"
	Matic    = "matic"
	Near     = "near"
	Solana   = "sol"
	Ton      = "ton"
	Tron     = "tron"
)

var (
	chainMap = map[string]Chainer{
		Bsc:      bsc.New(),
		Matic:    matic.New(),
		Conflux:  conflux.New(),
		Eth2:     eth2.New(),
		Ethereum: ethereum.New(),
		Klaytn:   klaytn.New(),
		Near:     near.New(),
		Solana:   sol.New(),
		Ton:      ton.New(),
		Tron:     tron.New(),
	}
)

func Create(_type string) (Chainer, bool) {
	if chain, ok := chainMap[_type]; ok {
		return chain, true
	}
	return nil, false
}

type Chainer interface {
	New(*core.ChainConfig, log15.Logger, chan<- error, mapprotocol.Role) (core.Chain, error)
}
