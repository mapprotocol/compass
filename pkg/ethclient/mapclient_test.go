package ethclient

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"testing"
)

var MAPEndpoint = "http://18.142.54.137:7445"

func TestClient_MAPBlockByNumber(t *testing.T) {
	cli, err := Dial(MAPEndpoint)
	if err != nil {
		t.Fatalf(err.Error())
	}

	header, err := cli.MAPBlockByNumber(context.Background(), big.NewInt(15960))
	if err != nil {
		t.Fatalf(err.Error())
	}
	fmt.Printf("block: %+v\n", header)

	h, err := json.Marshal(header)
	if err != nil {
		t.Fatalf(err.Error())
	}

	fmt.Println("block: ", string(h))
}

func TestClient_MAPHeaderByNumber(t *testing.T) {
	cli, err := Dial(MAPEndpoint)
	if err != nil {
		t.Fatalf(err.Error())
	}

	header, err := cli.MAPHeaderByNumber(context.Background(), big.NewInt(1))
	if err != nil {
		t.Fatalf(err.Error())
	}
	h, err := json.Marshal(header)
	if err != nil {
		t.Fatalf(err.Error())
	}

	fmt.Println("header: ", string(h))
}

func TestClient_GetSnapshot(t *testing.T) {
	cli, err := Dial(MAPEndpoint)
	if err != nil {
		t.Fatalf(err.Error())
	}

	snapshot, err := cli.GetSnapshot(context.Background(), big.NewInt(0))
	if err != nil {
		t.Fatalf(err.Error())
	}

	snap, err := json.Marshal(snapshot)
	if err != nil {
		t.Fatalf(err.Error())
	}

	fmt.Println("snapshot: ", string(snap))
}
