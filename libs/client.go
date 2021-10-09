package libs

import (
	"log"

	"github.com/ethereum/go-ethereum/ethclient"
)

var (
	client    *ethclient.Client
	ethClient *ethclient.Client
)

func GetClient() *ethclient.Client {
	if client == nil {
		var err error
		client, err = ethclient.Dial(RpcUrl)
		if err != nil {
			log.Fatal(err)
		}
	}
	return client
}

func GetEthClient() *ethclient.Client {
	if ethClient == nil {
		var err error
		ethClient, err = ethclient.Dial(EthRpcUrl)
		if err != nil {
			log.Fatal(err)
		}
	}
	return ethClient
}
