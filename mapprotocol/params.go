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

// common varible
var (
	Big0           = big.NewInt(0)
	Big1           = big.NewInt(1)
	RegisterAmount = int64(100) // for test purpose
)

var ZeroAddress = common.HexToAddress("0x0000000000000000000000000000000000000000")

const (
	MethodVerifyProofData   = "verifyProofData"
	MethodUpdateBlockHeader = "updateBlockHeader"
	MethodOfHeaderHeight    = "headerHeight"
	MethodOfTransferIn      = "transferIn"
	MethodOfCurNbrAndHash   = "currentNumberAndHash"
	MethodOfRegister        = "register"
	MethodOfBindWorker      = "bind"
	MethodOfOrderList       = "orderList"
)

var (
	RelayerAddress          = common.HexToAddress("0x000068656164657273746F726541646472657373")
	ABIRelayer, _           = abi.JSON(strings.NewReader(RelayerABIJSON))
	ABILightNode, _         = abi.JSON(strings.NewReader(LightNode))
	NearVerify, _           = abi.JSON(strings.NewReader(NearVerifyAbi))
	NearGetBytes, _         = abi.JSON(strings.NewReader(NearGetBytesAbi))
	Eth2MapTransferInAbi, _ = abi.JSON(strings.NewReader(Eth2MapTransferIn))
	ABIEncodeReceipt, _     = abi.JSON(strings.NewReader(EncodeReceiptABI))
	LightNodeInterface, _   = abi.JSON(strings.NewReader(LightNodeInterfaceABI))
	OrderList, _            = abi.JSON(strings.NewReader(OrderListAbi))
	LightManger, _          = abi.JSON(strings.NewReader(LightMangerAbi))
)

type Role string

var (
	RoleOfMaintainer Role = "maintainer"
	RoleOfMessenger  Role = "messenger"
)
