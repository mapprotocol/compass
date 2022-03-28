// Copyright 2021 Compass Systems
// SPDX-License-Identifier: LGPL-3.0-only

package msg

import (
	"math/big"
)

type ChainId uint64
type TransferType string

// type ResourceId [32]byte

// func (r ResourceId) Hex() string {
// 	return fmt.Sprintf("%x", r)
// }

type Nonce uint64

func (n Nonce) Big() *big.Int {
	return big.NewInt(int64(n))
}

var SwapTransfer TransferType = "SwapTransfer"
var SyncToMap TransferType = "SyncToMap"
var SwapWithProof TransferType = "SwapWithProof"
var SyncFromMap TransferType = "SyncFromMap"

// Message is used as a generic format to communicate between chains
type Message struct {
	Source       ChainId      // Source where message was initiated
	Destination  ChainId      // Destination chain of message
	Type         TransferType // type of bridge transfer
	DepositNonce Nonce        // Nonce for the deposit
	// ResourceId   ResourceId
	Payload []interface{}   // data associated with event sequence
	DoneCh  chan<- struct{} // notify message is handled
}

func NewSwapTransfer(fromChainID, toChainID ChainId, payloads []interface{}, ch chan<- struct{}) Message {
	return Message{
		Source:      fromChainID,
		Destination: toChainID,
		Type:        SwapTransfer,
		Payload:     payloads,
		DoneCh:      ch,
	}
}

func NewSyncToMap(fromChainID, toChainID ChainId, payloads []interface{}, ch chan<- struct{}) Message {
	return Message{
		Source:      fromChainID,
		Destination: toChainID,
		Type:        SyncToMap,
		Payload:     payloads,
		DoneCh:      ch,
	}
}

func NewSwapWithProof(fromChainID, toChainID ChainId, payloads []interface{}, ch chan<- struct{}) Message {
	return Message{
		Source:      fromChainID,
		Destination: toChainID,
		Type:        SwapWithProof,
		Payload:     payloads,
		DoneCh:      ch,
	}
}

func NewSyncFromMap(mapChainID, toChainID ChainId, payloads []interface{}, ch chan<- struct{}) Message {
	return Message{
		Source:      mapChainID,
		Destination: toChainID,
		Type:        SyncFromMap,
		Payload:     payloads,
		DoneCh:      ch,
	}
}

// func ResourceIdFromSlice(in []byte) ResourceId {
// 	var res ResourceId
// 	copy(res[:], in)
// 	return res
// }
