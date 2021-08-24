package chain_tools

import (
	"context"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	log "github.com/sirupsen/logrus"
)

func PackInput(AbiStaking abi.ABI, abiMethod string, params ...interface{}) []byte {
	input, err := AbiStaking.Pack(abiMethod, params...)
	if err != nil {
		log.Fatal(abiMethod, " error ", err)
	}
	return input
}

func CallContractReturnBool(client *ethclient.Client, from, toAddress common.Address, input []byte) ([]byte, bool) {
	var ret []byte

	gasPrice, err := client.SuggestGasPrice(context.Background())
	if err != nil {
		log.Warnln("Get SuggestGasPrice  error: ", err)
		return ret, false
	}
	msg := ethereum.CallMsg{From: from, To: &toAddress, GasPrice: gasPrice, Data: input}

	header, err := client.HeaderByNumber(context.Background(), nil)
	if err != nil {
		log.Warnln("Get blockNumber error: ", err)
		return ret, false
	}
	ret, err = client.CallContract(context.Background(), msg, header.Number)
	if err != nil {
		log.Warnln("method CallContract error: ", err)
		return ret, false
	}
	return ret, true
}
