// Copyright 2021 Compass Systems
// SPDX-License-Identifier: LGPL-3.0-only

package chains

import (
	"github.com/mapprotocol/compass/msg"
)

type Router interface {
	Send(message msg.Message) error
}

type Listener interface {
	Sync() error
	SetRouter(r Router)
}

const (
	Bsc      = "bsc"
	Matic    = "matic"
	Klaytn   = "klaytn"
	Eth2     = "eth2"
	Near     = "near"
	Ethereum = "ethereum"
	Conflux  = "conflux"
	Tron     = "tron"
	Solana   = "sol"
	Ton      = "ton"
)
