package contract

import (
	"context"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/mapprotocol/compass/internal/constant"
	"github.com/mapprotocol/compass/pkg/abi"
	"github.com/mapprotocol/compass/pkg/ethclient"
)

type Connection interface {
	Client() *ethclient.Client
}

type Call struct {
	abi  *abi.Abi
	toC  []common.Address
	conn Connection
}

func New(conn Connection, addr []common.Address, abi *abi.Abi) *Call {
	return &Call{
		conn: conn,
		toC:  addr,
		abi:  abi,
	}
}

func (c *Call) Call(method string, ret interface{}, idx int, params ...interface{}) error {
	input, err := c.abi.PackInput(method, params...)
	if err != nil {
		return err
	}

	outPut, err := c.conn.Client().CallContract(context.Background(),
		ethereum.CallMsg{
			From: constant.ZeroAddress,
			To:   &c.toC[idx],
			Data: input,
		},
		nil,
	)
	if err != nil {
		return err
	}

	return c.abi.UnpackOutput(method, ret, outPut)
}
