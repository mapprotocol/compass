package tron

import (
	"math/big"
	"testing"

	"google.golang.org/grpc"

	"github.com/mapprotocol/compass/mapprotocol"

	"github.com/lbtsm/gotron-sdk/pkg/client"
)

func Test_RPC(t *testing.T) {
	//// import keystore
	//key, err := account.ImportFromPrivateKey("123456", "", "123456")
	//if err != nil {
	//	t.Fatalf("importFromPrivateKey err is %v", err)
	//}
	//t.Logf("importFromPrivateKey key is %s", key)
	//
	////account.ImportKeyStore()
	//addr, err := address.Base58ToAddress("TUa7EVjjFnG2vM5x9RFbtRpcvSbMxNS9g6")
	//if err != nil {
	//	t.Fatalf("base58ToAddress err is %v", err)
	//}
	//t.Logf("tron convert eth address is %s", addr.Hex())

	cli := client.NewGrpcClient("grpc.nile.trongrid.io:50051")
	err := cli.Start(grpc.WithInsecure())
	if err != nil {
		t.Fatalf("cli.Start failed err is %v", err)
	}

	call, err := cli.TriggerConstantContract("",
		"TTZ687EZZYDUWYNEB5dbu2zxVAHUPrE7fr", "headerHeight()", "")
	if err != nil {
		t.Fatalf("cli.TriggerConstantContract failed err is %v", err)
	}

	i, _ := big.NewInt(0).SetString("522740", 16)
	t.Logf("i ------------- %v", i)
	t.Log(mapprotocol.UnpackHeaderHeightOutput(call.ConstantResult[0]))
	//t.Log(mapprotocol.UnpackHeaderHeightOutput(call))
}
