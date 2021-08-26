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

	msg := ethereum.CallMsg{From: from, To: &toAddress, Data: input}

	ret, err := client.CallContract(context.Background(), msg, nil)
	if err != nil {
		log.Warnln("method CallContract error: ", err)
		return ret, false
	}
	return ret, true
}
