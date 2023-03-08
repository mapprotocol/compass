// Copyright 2021 Compass Systems
// SPDX-License-Identifier: LGPL-3.0-only

package chains

import (
	metrics "github.com/ChainSafe/chainbridge-utils/metrics/types"
	"github.com/mapprotocol/compass/msg"
)

type Router interface {
	Send(message msg.Message) error
}

type Listener interface {
	Sync() error
	SetRouter(r Router)
	GetLatestBlock() metrics.LatestBlock
}

const (
	Map      = "map"
	Bsc      = "bsc"
	Matic    = "matic"
	Klaytn   = "klaytn"
	Eth2     = "eth2"
	Near     = "near"
	Ethereum = "ethereum"
)

var (
	NearChainId = map[msg.ChainId]struct{}{
		1313161556:          {},
		1313161555:          {},
		1313161554:          {},
		5566818579631833089: {},
	}
)

//type Writer interface {
//	ResolveMessage(message msg.Message) bool
//}
