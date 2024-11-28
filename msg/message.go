// Copyright 2021 Compass Systems
// SPDX-License-Identifier: LGPL-3.0-only

package msg

type ChainId uint64
type TransferType string

var (
	SyncToMap        TransferType = "SyncToMap"
	SyncFromMap      TransferType = "SyncFromMap"
	SwapWithProof    TransferType = "SwapWithProof"
	SwapWithMapProof TransferType = "SwapWithMapProof"
	SwapWithMerlin   TransferType = "SwapWithMerlin"
	Proposal         TransferType = "Proposal"
	SwapSolProof     TransferType = "SwapSolProof"
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

func NewSolProof(fromChainID, toChainID ChainId, payloads []interface{}, ch chan<- struct{}) Message {
	return Message{
		Source:      fromChainID,
		Destination: toChainID,
		Type:        SwapSolProof,
		Payload:     payloads,
		DoneCh:      ch,
	}
}

func NewSwapWithMerlin(fromChainID, toChainID ChainId, payloads []interface{}, ch chan<- struct{}) Message {
	return Message{
		Source:      fromChainID,
		Destination: toChainID,
		Type:        SwapWithMerlin,
		Payload:     payloads,
		DoneCh:      ch,
	}
}

func NewProposal(fromChainID, toChainID ChainId, payloads []interface{}, ch chan<- struct{}) Message {
	return Message{
		Source:      fromChainID,
		Destination: toChainID,
		Type:        Proposal,
		Payload:     payloads,
		DoneCh:      ch,
	}
}
