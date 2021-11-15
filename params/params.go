// Copyright 2021 Compass Systems
// SPDX-License-Identifier: LGPL-3.0-only

package params

import (
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
)

type ChainType uint64

// Map Chain ID
const (
	MainNetChainID    uint64 = 177
	TestNetChainID    uint64 = 212
	DevNetChainID     uint64 = 213
	SingleNodeChainID uint64 = 214
)

// ETH Chain ID
const (
	ETHChainID     uint64 = 1
	RopstenCHainID uint64 = 3
)

var (
	HeaderStoreAddress           = common.BytesToAddress([]byte("headerstoreAddress"))
	HeaderStoreFuncSig           = "save"
	ABIHeaderStore, _            = abi.JSON(strings.NewReader(HeaderStoreABIJSON))
	ChainTypeETH       ChainType = ChainType(RopstenCHainID)
	ChainTypeMAP       ChainType = ChainType(MainNetChainID)
)

func PackInput(abiHeaderStore abi.ABI, abiMethod string, params ...interface{}) ([]byte, error) {
	input, err := abiHeaderStore.Pack(abiMethod, params...)
	if err != nil {
		return nil, err
	}
	return input, nil
}
