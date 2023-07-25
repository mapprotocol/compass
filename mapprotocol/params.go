// Copyright 2021 Compass Systems
// SPDX-License-Identifier: LGPL-3.0-only

package mapprotocol

import (
	"math/big"
	"strings"

	"github.com/mapprotocol/compass/core"
	"github.com/mapprotocol/compass/msg"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
)

const (
	MethodVerifyProofData        = "verifyProofData"
	MethodUpdateBlockHeader      = "updateBlockHeader"
	MethodVerifiableHeaderRange  = "verifiableHeaderRange"
	MethodOfHeaderHeight         = "headerHeight"
	MethodOfTransferIn           = "transferIn"
	MethodOfDepositIn            = "depositIn"
	MethodOfSwapIn               = "swapIn"
	MethodOfRegister             = "register"
	MethodOfBindWorker           = "bind"
	MethodOfOrderList            = "orderList"
	MethodOfIsUsedEvent          = "is_used_event"
	MethodOfGetBytes             = "getBytes"
	MethodOfGetFinalBytes        = "getFinalBytes"
	MethodOfGetHeadersBytes      = "getHeadersBytes"
	MethodOfGetBlockHeadersBytes = "getBlockHeaderBytes"
	MethodOfGetUpdatesBytes      = "getUpdateBytes"
	MethodUpdateLightClient      = "updateLightClient"
	MethodClientState            = "clientState"
	MethodClientStateAnalysis    = "clientStateAnalysis"
	MethodOfHeaderState          = "state"
)

const (
	NearVerifyRange  = "get_verifiable_header_range"
	NearHeaderHeight = "get_header_height"
)

const (
	EpochOfMap           = 50000
	EpochOfBsc           = 200
	HeaderCountOfBsc     = 12
	HeaderCountOfMatic   = 16
	HeaderCountOfPlaton  = 430
	EpochOfKlaytn        = 3600
	HeaderCountOfKlaytn  = 1
	HeaderCountOfConflux = 1
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
	HashOfDataIn    = common.HexToHash("0x30f032e802558749ee4be1c2a9269937ff74045819e844f0f18970c84d891d79")
	NearOfDepositIn = "150bd848adaf4e3e699dcac82d75f111c078ce893375373593cc1b9208998377"
	NearOfSwapIn    = "ca1cf8cebf88499429cca8f87cbca15ab8dafd06702259a5344ddce89ef3f3a5"
)

var (
	Mcs, _         = abi.JSON(strings.NewReader(McsAbi))
	Bsc, _         = abi.JSON(strings.NewReader(BscAbiJson))
	Conflux, _     = abi.JSON(strings.NewReader(ConfluxAbiJson))
	Klaytn, _      = abi.JSON(strings.NewReader(KlaytnAbiJson))
	Near, _        = abi.JSON(strings.NewReader(NearAbiJson))
	LightManger, _ = abi.JSON(strings.NewReader(LightMangerAbi))
	Map2Other, _   = abi.JSON(strings.NewReader(Map2OtherAbi))
	ABIRelayer, _  = abi.JSON(strings.NewReader(RelayerABIJSON))
	Height, _      = abi.JSON(strings.NewReader(HeightAbiJson))
	Verify, _      = abi.JSON(strings.NewReader(VerifiableHeaderRangeAbiJson))
	Matic, _       = abi.JSON(strings.NewReader(MaticAbiJson))
	Eth2, _        = abi.JSON(strings.NewReader(Eth2AbiJson))
	Platon, _      = abi.JSON(strings.NewReader(PlatonAbiJson))
)

type Role string

var (
	RoleOfMaintainer Role = "maintainer"
	RoleOfMessenger  Role = "messenger"
	RoleOfMonitor    Role = "monitor"
)

var (
	OnlineChaId    = map[msg.ChainId]string{}
	OnlineChainCfg = map[msg.ChainId]*core.ChainConfig{}
	Event          = map[common.Hash]string{
		common.HexToHash("0x56877b1dbedc6754c111b951146b820fe6b723af0213fc415d44b05e1758dd85"): MethodOfTransferIn,
		common.HexToHash("0xf4397fd41454e34a9a4015d05a670124ecd71fe7f1d05578a62f8009b1a57f8a"): MethodOfTransferIn,
		common.HexToHash("0xca1cf8cebf88499429cca8f87cbca15ab8dafd06702259a5344ddce89ef3f3a5"): MethodOfSwapIn,
		common.HexToHash("0xb7100086a8e13ebae772a0f09b07046e389a6b036406d22b86f2d2e5b860a8d9"): MethodOfDepositIn,
		common.HexToHash("0x44ff77018688dad4b245e8ab97358ed57ed92269952ece7ffd321366ce078622"): MethodOfTransferIn,
	}
)

var (
	ConfirmsOfMatic       = big.NewInt(10)
	HeaderLengthOfEth2    = 20
	HeaderLengthOfConflux = 20
)
