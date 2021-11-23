// Copyright 2021 Compass Systems
// SPDX-License-Identifier: LGPL-3.0-only

package mapprotocol

import (
	"context"
	"errors"
	"math/big"

	goeth "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
)

// global Map connection; assign at cmd/main
var GlobalMapConn *ethclient.Client

func packInput(abiHeaderStore abi.ABI, abiMethod string, params ...interface{}) ([]byte, error) {
	input, err := abiHeaderStore.Pack(abiMethod, params...)
	if err != nil {
		return nil, err
	}
	return input, nil
}

func SaveHeaderTxData(marshal []byte) ([]byte, error) {
	return packInput(ABIHeaderStore,
		SaveHeader,
		big.NewInt(int64(ChainTypeETH)),
		big.NewInt(int64(ChainTypeMAP)),
		marshal)
}

func GetCurrentNumberAbi(from common.Address) (*big.Int, string, error) {
	if GlobalMapConn == nil {
		return Big0, "", errors.New("Global Map Connection is not assigned!")
	}

	blockNum, err := GlobalMapConn.BlockNumber(context.Background())
	if err != nil {
		return Big0, "", err
	}
	input, _ := packInput(ABIHeaderStore, CurNbrAndHash, big.NewInt(int64(ChainTypeETH)))

	msg := goeth.CallMsg{
		From: from,
		To:   &HeaderStoreAddress,
		Data: input}

	output, err := GlobalMapConn.CallContract(context.Background(), msg, big.NewInt(0).SetUint64(blockNum))
	if err != nil {
		return Big0, "", err
	}
	method, _ := ABIHeaderStore.Methods[CurNbrAndHash]
	ret, err := method.Outputs.Unpack(output)
	if err != nil {
		return Big0, "", err
	}
	height := ret[0].(*big.Int)
	hash := common.BytesToHash(ret[1].([]byte))

	return height, hash.String(), nil
}
