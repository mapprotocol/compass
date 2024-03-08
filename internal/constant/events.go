// Copyright 2021 Compass Systems
// SPDX-License-Identifier: LGPL-3.0-only

package constant

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

type EventSig string

func (es EventSig) GetTopic() common.Hash {
	return crypto.Keccak256Hash([]byte(es))
}
