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

const (
	MethodVerifyProofData   = "verifyProofData"
	MethodUpdateBlockHeader = "updateBlockHeader"
	MethodOfHeaderHeight    = "headerHeight"
	MethodOfTransferIn      = "transferIn"
)

const (
	NearMethodOfReceiptProof = ""
)

var (
	RelayerAddress    = common.HexToAddress("0x000068656164657273746F726541646472657373")
	Eth2MapTmpAddress = common.HexToAddress("0x0000000000747856657269667941646472657373")
	SaveHeader        = "save"
	UpdateBlockHeader = "updateBlockHeader"
	CurNbrAndHash     = "currentNumberAndHash"
	RegisterRelayer   = "register"
	BindWorker        = "bind"

	ABIRelayer, _           = abi.JSON(strings.NewReader(RelayerABIJSON))
	ABILiteNode, _          = abi.JSON(strings.NewReader(LiteABIJSON))
	ABILightNode, _         = abi.JSON(strings.NewReader(LightNode))
	Verify, _               = abi.JSON(strings.NewReader(VerifyAbi))
	NearVerify, _           = abi.JSON(strings.NewReader(NearVerifyAbi))
	NearGetBytes, _         = abi.JSON(strings.NewReader(NearGetBytesAbi))
	Eth2MapTransferInAbi, _ = abi.JSON(strings.NewReader(Eth2MapTransferIn))
	ABIEncodeReceipt, _     = abi.JSON(strings.NewReader(EncodeReceiptABI))
)

type Role string

var (
	RoleOfMaintainer Role = "maintainer"
	RoleOfMessenger  Role = "messenger"
)
