// Copyright 2021 Compass Systems
// SPDX-License-Identifier: LGPL-3.0-only

package mapprotocol

import (
	"github.com/mapprotocol/compass/msg"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
)

const (
	MethodVerifyProofData       = "verifyProofData"
	MethodUpdateBlockHeader     = "updateBlockHeader"
	MethodVerifiableHeaderRange = "verifiableHeaderRange"
	MethodOfHeaderHeight        = "headerHeight"
	MethodOfTransferIn          = "transferIn"
	MethodOfDepositIn           = "depositIn"
	MethodOfSwapIn              = "swapIn"
	MethodOfRegister            = "register"
	MethodOfBindWorker          = "bind"
	MethodOfOrderList           = "orderList"
	MethodOfIsUsedEvent         = "is_used_event"
	MethodOfGetBytes            = "getBytes"
	MethodOfGetHeadersBytes     = "getHeadersBytes"
	MethodOfGetConfirms         = "confirms"
	MethodOfGetUpdatesBytes     = "getUpdateBytes"
)

const (
	NearVerifyRange  = "get_verifiable_header_range"
	NearHeaderHeight = "get_header_height"
)

const (
	EpochOfMap          = 50000
	EpochOfBsc          = 200
	HeaderCountOfBsc    = 12
	HeaderCountOfMatic  = 16
	EpochOfKlaytn       = 3600
	HeaderCountOfKlaytn = 1
)

// common varible
var (
	Big0           = big.NewInt(0)
	Big1           = big.NewInt(1)
	RegisterAmount = int64(100) // for test purpose
)

var (
	ZeroAddress     = common.HexToAddress("0x0000000000000000000000000000000000000000")
	RelayerAddress  = common.HexToAddress("0x000068656164657273746F726541646472657373")
	HashOfDepositIn = common.HexToHash("0xb7100086a8e13ebae772a0f09b07046e389a6b036406d22b86f2d2e5b860a8d9")
	HashOfSwapIn    = common.HexToHash("0xca1cf8cebf88499429cca8f87cbca15ab8dafd06702259a5344ddce89ef3f3a5")
	NearOfDepositIn = "150bd848adaf4e3e699dcac82d75f111c078ce893375373593cc1b9208998377"
	NearOfSwapIn    = "ca1cf8cebf88499429cca8f87cbca15ab8dafd06702259a5344ddce89ef3f3a5"
)

var (
	Mcs, _         = abi.JSON(strings.NewReader(McsAbi))
	Bsc, _         = abi.JSON(strings.NewReader(BscAbiJson))
	Klaytn, _      = abi.JSON(strings.NewReader(KlaytnAbiJson))
	Near, _        = abi.JSON(strings.NewReader(NearAbiJson))
	LightManger, _ = abi.JSON(strings.NewReader(LightMangerAbi))
	Map2Other, _   = abi.JSON(strings.NewReader(Map2OtherAbi))
	ABIRelayer, _  = abi.JSON(strings.NewReader(RelayerABIJSON))
	Height, _      = abi.JSON(strings.NewReader(HeightAbiJson))
	Verify, _      = abi.JSON(strings.NewReader(VerifiableHeaderRangeAbiJson))
	Matic, _       = abi.JSON(strings.NewReader(MaticAbiJson))
	Eth2, _        = abi.JSON(strings.NewReader(Eth2AbiJson))
)

type Role string

var (
	RoleOfMaintainer Role = "maintainer"
	RoleOfMessenger  Role = "messenger"
	RoleOfMonitor    Role = "monitor"
)

var (
	OnlineChaId = map[msg.ChainId]string{}
)

var (
	ConfirmsOfMatic    = big.NewInt(10)
	InputOfConfirms, _ = PackInput(Matic, MethodOfGetConfirms)
)
