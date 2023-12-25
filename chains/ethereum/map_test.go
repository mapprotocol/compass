package ethereum

import (
	"context"
	"math/big"
	"strings"
	"testing"

	"github.com/mapprotocol/compass/internal/tx"

	"github.com/mapprotocol/compass/pkg/ethclient"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/mapprotocol/compass/internal/chain"
	"github.com/mapprotocol/compass/mapprotocol"
	utils "github.com/mapprotocol/compass/shared/ethereum"
)

func Test_Other(t *testing.T) {
	var (
		addr        = common.HexToAddress("0x8c3ccc219721b206da4a2070fd96e4911a48cb4f")
		events      = make([]utils.EventSig, 0)
		latestBlock = new(big.Int).SetUint64(5103580)
	)
	vs := strings.Split("mapTransferOut(uint256,uint256,bytes32,bytes,bytes,bytes,uint256,bytes)|mapDepositOut(uint256,uint256,bytes32,address,bytes,address,uint256)|mapSwapOut(uint256,uint256,bytes32,bytes,bytes,bytes,uint256,bytes)|mapMessageOut(uint256,uint256,bytes32,bytes,bytes)", "|")
	for _, s := range vs {
		events = append(events, utils.EventSig(s))
	}

	rpcClient, err := rpc.DialHTTP("https://testnet-rpc.maplabs.io")
	client := ethclient.NewClient(rpcClient)

	cs := chain.NewCommonSync(nil, &chain.Config{}, nil, nil, nil, nil, nil)
	m := chain.NewMaintainer(cs)
	query := m.BuildQuery(addr, events, latestBlock, latestBlock)
	// querying for logs
	logs, err := client.FilterLogs(context.Background(), query)
	if err != nil {
		t.Fatalf("unable to Filter Logs: %s", err)
	}

	// read through the log events and handle their deposit event if handler is recognized
	for _, log := range logs {
		// getOrderId
		method := m.GetMethod(log.Topics[0])

		// when listen from map we also need to assemble a tx prove in a different way
		header, err := client.MAPHeaderByNumber(context.Background(), latestBlock)
		if err != nil {
			t.Fatalf("unable to query header Logs: %s", err)
		}
		txsHash, _, err := mapprotocol.GetMapTransactionsHashByBlockNumber(client, latestBlock, log.TxHash)
		if err != nil {
			t.Fatalf("idSame unable to get tx hashes Logs: %s", err)
		}
		receipts, err := tx.GetReceiptsByTxsHash(client, txsHash)
		if err != nil {
			t.Fatalf("unable to get receipts hashes Logs: %s", err)
		}

		toChainID, payload, err := utils.AssembleMapProof(client, log, receipts, header, 212, method, "")
		if err != nil {
			t.Fatalf("unable to Parse Log: %s", err)
		}

		t.Logf("toChainID %d, input %s", toChainID, common.Bytes2Hex(payload))
	}
}
