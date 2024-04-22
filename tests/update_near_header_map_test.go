package tests

import (
	"github.com/lbtsm/gotron-sdk/pkg/client"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"testing"
)

func Test_grpc(t *testing.T) {
	conn := client.NewGrpcClient("grpc.trongrid.io:50051")
	err := conn.Start(grpc.WithInsecure())
	require.Nil(t, err)

	// `[{"bytes32":"1eee75d90926c3470877c4da9c21d52c6e762225fec1386f7a526bf3f1ce440e"}]`
	tx, err := conn.TriggerConstantContract("TNoZuAuL83PSh8TG4W92AvLqkA4E2dKSNm",
		"TYMpgB8Q9vSoGtkyE3hXsvUrpte3KCDGj6",
		"orderList(bytes32)", `[{"bytes32":"[30 238 117 217 9 38 195 71 8 119 196 218 156 33 213 44 110 118 34 37 254 193 56 111 122 82 107 243 241 206 68 14]"}]`)
	t.Log("err", err)
	t.Log("tx", tx.ConstantResult[0])
}
