// Copyright 2021 Compass Systems
// SPDX-License-Identifier: LGPL-3.0-only

package msg

type ChainId uint64
type TransferType string

var (
	SyncToMap        TransferType = "SyncToMap"
	SwapWithProof    TransferType = "SwapWithProof"
	SyncFromMap      TransferType = "SyncFromMap"
	SwapWithMapProof TransferType = "SwapWithMapProof"
)

// Message is used as a generic format to communicate between chains
type Message struct {
	Idx         int
	Source      ChainId         // Source where message was initiated
	Destination ChainId         // Destination chain of message
	Type        TransferType    // type of bridge transfer
	Payload     []interface{}   // data associated with event sequence
	DoneCh      chan<- struct{} // notify message is handled
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

func NewSwapWithMapProof(fromChainID, toChainID ChainId, payloads []interface{}, ch chan<- struct{}) Message {
	return Message{
		Source:      fromChainID,
		Destination: toChainID,
		Type:        SwapWithMapProof,
		Payload:     payloads,
		DoneCh:      ch,
	}
}
