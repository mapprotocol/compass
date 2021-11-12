// Copyright 2021 Compass Systems
// SPDX-License-Identifier: LGPL-3.0-only

package utils

import (
	"encoding/binary"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
)

var (
	ZeroAddress = common.HexToAddress("0x0000000000000000000000000000000000000000")
	// function swapIn(bytes32 hash, address token, address from, address to, uint amount, uint fromChainID,uint toChainID)
	SwapIn = "swapIn(bytes32,address,address,address,uint256,uint256,uint256)"

	//bytesTy, _ = abi.NewType("bytes", "", nil)
	bytes32Ty, _ = abi.NewType("bytes32", "", nil)
	uint256Ty, _ = abi.NewType("uint256", "", nil)
	addressTy, _ = abi.NewType("address", "", nil)

	SwapInArgs = abi.Arguments{
		{
			Name: "_hash",
			Type: bytes32Ty,
		},
		{
			Name: "_token",
			Type: addressTy,
		},
		{
			Name: "_from",
			Type: addressTy,
		},
		{
			Name: "_to",
			Type: addressTy,
		},
		{
			Name: "_amount",
			Type: uint256Ty,
		},
		{
			Name: "_fromChainID",
			Type: uint256Ty,
		},
		{
			Name: "_toChainID",
			Type: uint256Ty,
		},
	}
)

func ComposeMsgPayloadWithSignature(sig string, msgPayload []interface{}) []byte {
	// signature
	sigbytes := crypto.Keccak256Hash([]byte(sig))

	var data []byte
	data = append(data, sigbytes[:4]...)
	data = append(data, msgPayload[0].([]byte)...)
	return data
}

func ParseEthLog(log types.Log, bridge common.Address) (uint64, uint64, []byte, error) {
	token := log.Topics[1].Bytes()
	from := log.Topics[2].Bytes()
	to := log.Topics[3].Bytes()
	// every 32 bytes forms a value
	var orderHash [32]byte
	copy(orderHash[:], log.Data[:32])
	amount := log.Data[32:64]

	fromChainID := log.Data[64:96]
	toChainID := log.Data[96:128]
	uFromChainID := binary.BigEndian.Uint64(fromChainID[len(fromChainID)-8:])
	uToChainID := binary.BigEndian.Uint64(toChainID[len(toChainID)-8:])

	payloads, err := SwapInArgs.Pack(
		orderHash,
		common.BytesToAddress(token),
		common.BytesToAddress(from),
		common.BytesToAddress(to),
		big.NewInt(0).SetBytes(amount),
		big.NewInt(0).SetBytes(fromChainID),
		big.NewInt(0).SetBytes(toChainID),
		// bridge,
		// []byte("123456"),
	)
	if err != nil {
		return 0, 0, nil, err
	}
	return uFromChainID, uToChainID, payloads, nil
}
