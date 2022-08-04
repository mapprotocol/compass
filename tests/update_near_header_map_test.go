package tests

import (
	"context"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/mapprotocol/compass/internal/near"
	"github.com/mapprotocol/compass/mapprotocol"
	"github.com/mapprotocol/near-api-go/pkg/client"
	"github.com/mapprotocol/near-api-go/pkg/client/block"
)

var (
	from         = common.HexToAddress("0xd9b31120b910c7d239a03062ab1d9403f30fb7d5")
	contractAddr = common.HexToAddress("0xeA9066b735dA0ad462B269711be8e39fe7156d15")
)

var (
	ksPath     = "/Users/t/data/atlas-1/keystore/UTC--2022-07-11T07-25-34.126829000Z--d9b31120b910c7d239a03062ab1d9403f30fb7d5"
	ksPassword = ""
)

func TestUpdateNearHeaderToMAP(t *testing.T) {
	h := headerHeight(t)
	t.Log("height: ", h)
	cli, err := client.NewClient("https://archival-rpc.mainnet.near.org")
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
	input, err := mapprotocol.PackUpdateBlockHeaderInput(data)
	if err != nil {
		t.Fatal(err)
	}

	from, private := LoadPrivate(ksPath, ksPassword)
	if err := SendContractTransaction(cli, from, contractAddr, nil, private, input); err != nil {
		t.Fatal(err)
	}
}

func headerHeight(t *testing.T) *big.Int {
	input, err := mapprotocol.PackHeaderHeightInput()
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