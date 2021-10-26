// Copyright 2020 ChainSafe Systems
// SPDX-License-Identifier: LGPL-3.0-only

package msg

import (
	"fmt"
	"math/big"
)

type ChainId uint64
type TransferType string
type ResourceId [32]byte

func (r ResourceId) Hex() string {
	return fmt.Sprintf("%x", r)
}

type Nonce uint64

func (n Nonce) Big() *big.Int {
	return big.NewInt(int64(n))
}

var SwapTransfer TransferType = "SwapTransfer"

// Message is used as a generic format to communicate between chains
type Message struct {
	Source       ChainId      // Source where message was initiated
	Destination  ChainId      // Destination chain of message
	Type         TransferType // type of bridge transfer
	DepositNonce Nonce        // Nonce for the deposit
	ResourceId   ResourceId
	Payload      []interface{} // data associated with event sequence
}

func NewSwapTransfer(fromChainID, toChainID ChainId, payloads []interface{}) Message {
	return Message{
		Source:      fromChainID,
		Destination: toChainID,
		Type:        SwapTransfer,
		//DepositNonce: nonce,
		//ResourceId: resourceId,
		Payload: payloads,
	}
}

func ResourceIdFromSlice(in []byte) ResourceId {
	var res ResourceId
	copy(res[:], in)
	return res
}
