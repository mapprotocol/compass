package abi

import (
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/pkg/errors"
	"strings"
)

type Abi struct {
	bridgeAbi abi.ABI
}

func New(abiStr string) (*Abi, error) {
	a, err := abi.JSON(strings.NewReader(abiStr))
	if err != nil {
		return nil, err
	}

	return &Abi{bridgeAbi: a}, nil
}

func (a *Abi) PackInput(abiMethod string, params ...interface{}) ([]byte, error) {
	input, err := a.bridgeAbi.Pack(abiMethod, params...)
	if err != nil {
		return nil, err
	}
	return input, nil
}

func (a *Abi) UnpackValues(method string, data []byte) ([]interface{}, error) {
	return a.bridgeAbi.Events[method].Inputs.UnpackValues(data)
}

func (a *Abi) UnpackOutput(method string, ret interface{}, output []byte) error {
	outputs := a.bridgeAbi.Methods[method].Outputs
	unpack, err := outputs.Unpack(output)
	if err != nil {
		return errors.Wrap(err, "unpack output")
	}

	if err = outputs.Copy(ret, unpack); err != nil {
		return errors.Wrap(err, "copy output")
	}
	return nil
}
