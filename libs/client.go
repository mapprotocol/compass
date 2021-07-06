package libs

import (
	"github.com/ethereum/go-ethereum/ethclient"
	"log"
)

func GetClient() *ethclient.Client {
	client, err := ethclient.Dial(RpcUrl)
	if err != nil {
		log.Fatal(err)
	}
	return client
}
