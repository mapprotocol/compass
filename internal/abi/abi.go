package abi

import (
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
)

type BridgeParam struct {
	Relay      bool           `json:"relay"`
	Referrer   common.Address `json:"referrer"`
	TransferId [32]byte       `json:"transferId"`
	GasLimit   *big.Int       `json:"gasLimit"`
	SwapData   []byte         `json:"swapData"`
}

func DecodeBridgeParam(_bridgeData []byte) (*BridgeParam, error) {
	// 定义ABI参数类型
	nAbi, err := abi.JSON(strings.NewReader(`[{"inputs":[{"components":[{"name":"relay","type":"bool"},{"name":"referrer","type":"address"},{"name":"transferId","type":"bytes32"},{"name":"gasLimit","type":"uint256"},{"name":"swapData","type":"bytes"}],"internalType":"","name":"","type":"tuple"}],"name":"bridgeParse","outputs":[{"internalType":"bool","name":"","type":"bool"}],"stateMutability":"view","type":"function"}]`))
	if err != nil {
		return nil, err
	}
	decodedValues, err := nAbi.Methods["bridgeParse"].Inputs.UnpackValues(_bridgeData)
	if err != nil {
		return nil, err
	}

	c, ok := decodedValues[0].(struct {
		Relay      bool           `json:"relay"`
		Referrer   common.Address `json:"referrer"`
		TransferId [32]byte       `json:"transferId"`
		GasLimit   *big.Int       `json:"gasLimit"`
		SwapData   []byte         `json:"swapData"`
	})
	if !ok {
		return nil, fmt.Errorf("abi: cannot unpack into bridge param")
	}

	return &BridgeParam{
		Relay:      c.Relay,
		Referrer:   c.Referrer,
		TransferId: c.TransferId,
		GasLimit:   c.GasLimit,
		SwapData:   c.SwapData,
	}, nil
}
