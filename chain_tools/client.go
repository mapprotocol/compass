package chain_tools

import (
	"github.com/ethereum/go-ethereum/ethclient"
	log "github.com/sirupsen/logrus"
)

func GetClientByUrl(rpcUrl string) *ethclient.Client {
	client, err := ethclient.Dial(rpcUrl)
	if err != nil {
		log.Fatal(err)
	}
	return client
}
