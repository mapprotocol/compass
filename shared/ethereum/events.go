// Copyright 2021 Compass Systems
// SPDX-License-Identifier: LGPL-3.0-only

package utils

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

type EventSig string

func (es EventSig) GetTopic() common.Hash {
	return crypto.Keccak256Hash([]byte(es))
}

const (
	//event LogSwapOut(bytes32 hash, address indexed token, address indexed from, address indexed to, uint amount, uint fromChainID, uint toChainID);
	SwapOut EventSig = "LogSwapOut(bytes32,address,address,address,uint256,uint256,uint256)"
)

type ProposalStatus int

const (
	Inactive ProposalStatus = iota
	Active
	Passed
	Executed
	Cancelled
)

func IsActive(status uint8) bool {
	return ProposalStatus(status) == Active
}

func IsFinalized(status uint8) bool {
	return ProposalStatus(status) == Passed
}

func IsExecuted(status uint8) bool {
	return ProposalStatus(status) == Executed
}