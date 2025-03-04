package contract

import (
	"github.com/mapprotocol/compass/internal/mapprotocol"
	"github.com/mapprotocol/compass/pkg/contract"
	"math/big"
)

type Validator interface {
	Validate(relay bool, dstChain, dstMinAmount *big.Int, dstToken, dstReceiver, swapData []byte) (bool, error)
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

func (v *validator) Validate(relay bool, dstChain, dstMinAmount *big.Int, dstToken, dstReceiver, swapData []byte) (bool, error) {
	var ret bool
	err := v.c.Call(mapprotocol.MethodOfValidate, &ret, 0, relay, dstChain, dstToken, dstReceiver, dstMinAmount, swapData)
	if err != nil {
		return false, err
	}
	return ret, nil
}

func Validate(relay bool, dstChain, dstMinAmount *big.Int, dstToken, dstReceiver, swapData []byte) (bool, error) {
	return defaultValidator.Validate(relay, dstChain, dstMinAmount, dstToken, dstReceiver, swapData)
}
