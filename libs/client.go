package libs

import (
	"github.com/ethereum/go-ethereum/ethclient"
	"log"
)

var (
	client *ethclient.Client
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
