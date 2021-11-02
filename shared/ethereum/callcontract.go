// Copyright 2020 ChainSafe Systems
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
	// swapIn( uint256 id, address token, address to, uint amount, uint fromChainID, address sourceRouter, bytes memory data)
	// swapIn(uint id, address token, address to, uint amount, uint fromChainID,uint toChainID)
	//SwapIn = "swapIn(uint256,address,address,uint256,uint256,address,bytes)"
	SwapIn = "swapIn(uint256,address,address,uint256,uint256,uint256)"

	uint256Ty, _ = abi.NewType("uint256", "", nil)
	addressTy, _ = abi.NewType("address", "", nil)
	bytesTy, _   = abi.NewType("bytes", "", nil)

	SwapInArgs = abi.Arguments{
		{
			Name: "_id",
			Type: uint256Ty,
		},
		{
			Name: "_token",
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
	to := log.Topics[3].Bytes()
	// every 32 bytes forms a value
	orderID := log.Data[:32]
	amount := log.Data[32:64]

	fromChainID := log.Data[64:96]
	toChainID := log.Data[96:128]
	uFromChainID := binary.BigEndian.Uint64(fromChainID[len(fromChainID)-8:])
	uToChainID := binary.BigEndian.Uint64(toChainID[len(toChainID)-8:])

	payloads, err := SwapInArgs.Pack(
		big.NewInt(0).SetBytes(orderID),
		common.BytesToAddress(token),
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
