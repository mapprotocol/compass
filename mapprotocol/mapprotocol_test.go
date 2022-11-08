package mapprotocol

import (
	"context"
	"log"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/mapprotocol/compass/pkg/ethclient"
)

func dialConn() *ethclient.Client {
	conn, err := ethclient.Dial("http://18.142.54.137:7445")
	if err != nil {
		log.Fatalf("Failed to connect to the atlas: %v", err)
	}
	return conn
}

func TestRegisterRelayerWithConn(t *testing.T) {
	from := common.HexToAddress("0xf03aDB732FBa8Fca38C00253B1A1aa72CCA026E6")
	to := common.HexToAddress("0xd8c65C51f32324d34Ab1d34cC4895702d64EE0ef")

	var useAbi = Height
	input, err := PackInput(useAbi, MethodOfHeaderHeight)
	if err != nil {
		t.Fatalf("PackLightNodeInput failed, err is %v", err.Error())
	}

	output, err := dialConn().CallContract(context.Background(),
		ethereum.CallMsg{
			From: from,
			To:   &to,
			Data: input,
		},
		nil,
	)
	if err != nil {
		t.Fatalf("CallContract failed, err is %v", err.Error())
	}

	t.Log("----------------", string(output))
	resp, err := useAbi.Methods[MethodOfHeaderHeight].Outputs.Unpack(output)
	if err != nil {
		t.Fatalf("Unpack failed, err is %v", err.Error())
	}
	var ret *big.Int

	err = useAbi.Methods[MethodOfHeaderHeight].Outputs.Copy(&ret, resp)
	if err != nil {
		t.Fatalf("Outputs Copy failed, err is %v", err.Error())
	}
	t.Logf("ret is %v", ret)
}
