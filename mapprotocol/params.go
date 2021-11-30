// Copyright 2021 Compass Systems
// SPDX-License-Identifier: LGPL-3.0-only

package mapprotocol

import (
	"math/big"
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

// common varible
var (
	Big0           = big.NewInt(0)
	Big1           = big.NewInt(1)
	RegisterAmount = int64(100000)
)

var (
	HeaderStoreAddress = common.BytesToAddress([]byte("headerstoreAddress"))
	SaveHeader         = "save"
	CurNbrAndHash      = "currentNumberAndHash"

	RelayerAddress  = common.BytesToAddress([]byte("RelayerAddress"))
	RegisterRelayer = "register"

	ABIHeaderStore, _ = abi.JSON(strings.NewReader(HeaderStoreABIJSON))
	ABIRelayer, _     = abi.JSON(strings.NewReader(RelayerABIJSON))

	ChainTypeETH ChainType = ChainType(RopstenCHainID) // todo change to eth when get online
	ChainTypeMAP ChainType = ChainType(TestNetChainID) // todo may change?
)
