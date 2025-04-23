package contract

import (
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/mapprotocol/compass/internal/mapprotocol"
	"github.com/mapprotocol/compass/pkg/contract"
	"math/big"
)

type Validator interface {
	Validate(param *SwapDataValidator) (bool, error)
}

type validator struct {
	c *contract.Call
}

func NewValidator(c *contract.Call) Validator {
	return &validator{c: c}
}

var defaultValidator Validator

func InitDefaultValidator(c *contract.Call) {
	defaultValidator = &validator{c: c}
}

type SwapDataValidator struct {
	Relay        bool
	DstChain     *big.Int
	DstToken     []byte
	DstReceiver  []byte
	DstMinAmount *big.Int
	SwapData     []byte
}

func (v *validator) Validate(param *SwapDataValidator) (bool, error) {
	var ret bool
	err := v.c.Call(mapprotocol.MethodOfValidate, &ret, 0, param)
	if err != nil {
		return false, err
	}
	return ret, nil
}

func Validate(relay bool, dstChain, dstMinAmount *big.Int, dstToken, dstReceiver, swapData []byte) (bool, error) {
	fmt.Println("relay ", relay)
	fmt.Println("dstChain ", dstChain)
	fmt.Println("dstMinAmount ", dstMinAmount)
	fmt.Println("dstToken ", common.Bytes2Hex(dstToken))
	fmt.Println("dstReceiver ", common.Bytes2Hex(dstReceiver))
	fmt.Println("swapData ", common.Bytes2Hex(swapData))
	return defaultValidator.Validate(&SwapDataValidator{
		Relay:        relay,
		DstChain:     dstChain,
		DstToken:     dstToken,
		DstReceiver:  dstReceiver,
		DstMinAmount: dstMinAmount,
		SwapData:     swapData,
	})
}
