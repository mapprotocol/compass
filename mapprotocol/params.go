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
	RegisterAmount = int64(100) // for test purpose
)

const (
	MethodVerifyProofData   = "verifyProofData"
	MethodUpdateBlockHeader = "updateBlockHeader"
	MethodOfHeaderHeight    = "headerHeight"
)

var (
	// todo using
	RelayerAddress    = common.HexToAddress("0x000068656164657273746F726541646472657373")
	SaveHeader        = "save"
	UpdateBlockHeader = "updateBlockHeader"
	CurNbrAndHash     = "currentNumberAndHash"
	RegisterRelayer   = "register"
	BindWorker        = "bind"

	ABIRelayer, _   = abi.JSON(strings.NewReader(RelayerABIJSON))
	ABILiteNode, _  = abi.JSON(strings.NewReader(LiteABIJSON))
	ABILightNode, _ = abi.JSON(strings.NewReader(LightNode))

	ChainTypeETH = ChainType(RopstenCHainID) // todo change to eth when get online
	ChainTypeMAP = ChainType(TestNetChainID) // todo may change?
)

type Role string

var (
	RoleOfMaintainer Role = "maintainer"
	RoleOfMessenger  Role = "messenger"
)
