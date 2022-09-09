package tests

import (
	"context"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum"
	"github.com/mapprotocol/compass/internal/near"

	"github.com/ethereum/go-ethereum/common"
	"github.com/mapprotocol/compass/mapprotocol"
	"github.com/mapprotocol/near-api-go/pkg/client"
	"github.com/mapprotocol/near-api-go/pkg/client/block"
)

var (
	from         = common.HexToAddress("0xec3e016916ba9f10762e33e03e8556409d096fb4")
	contractAddr = common.HexToAddress("0x3CE319B86ad4CC0623F7039C48978c1A2c6cF8eB")
)

var (
	ksPath     = "/Users/xm/Desktop/WL/code/atlas/node-1/keystore/UTC--2022-06-20T13-03-52.445629000Z--ec3e016916ba9f10762e33e03e8556409d096fb4"
	ksPassword = ""
)

func TestUpdateNearHeaderToMAP(t *testing.T) {
	h := headerHeight(t)
	t.Log("height: ", h)
	cli, err := client.NewClient("https://archival-rpc.testnet.near.org")
	if err != nil {
		t.Fatal(err)
	}
	//
	blockDetails, err := cli.BlockDetails(context.Background(), block.BlockID(h.Uint64()))
	if err != nil {
		t.Fatal(err)
	}
	lightBlock, err := cli.NextLightClientBlock(context.Background(), blockDetails.Header.Hash)
	if err != nil {
		t.Fatal(err)
	}
	t.Log("next height: ", lightBlock.InnerLite.Height)

	updateNearHeader(near.Borshify(lightBlock), t)
}

func updateNearHeader(data []byte, t *testing.T) {
	cli := dialMapConn()
	input, err := mapprotocol.PackInput(mapprotocol.LightManger, mapprotocol.MethodUpdateBlockHeader, data)
	if err != nil {
		t.Fatal(err)
	}

	from, private := LoadPrivate(ksPath, ksPassword)
	if err := SendContractTransaction(cli, from, contractAddr, nil, private, input); err != nil {
		t.Fatal(err)
	}
}

func headerHeight(t *testing.T) *big.Int {
	input, err := mapprotocol.PackInput(mapprotocol.LightManger, mapprotocol.MethodOfHeaderHeight)
	if err != nil {
		t.Fatal(err)
	}

	output, err := dialMapConn().CallContract(context.Background(), ethereum.CallMsg{From: from, To: &contractAddr, Data: input}, nil)
	if err != nil {
		t.Fatal(err)
	}
	height, err := mapprotocol.UnpackHeaderHeightOutput(output)
	if err != nil {
		t.Fatal(err)
	}
	return height
}
