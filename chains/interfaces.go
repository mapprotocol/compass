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

//type Writer interface {
//	ResolveMessage(message msg.Message) bool
//}
