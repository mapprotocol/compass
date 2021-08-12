package chain_tools

import (
	"context"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	log "github.com/sirupsen/logrus"
	"time"
)

func WaitingForEndPending(conn *ethclient.Client, txHash common.Hash) bool {
	count := 0
	for {
		time.Sleep(time.Millisecond * 200)
		_, isPending, err := conn.TransactionByHash(context.Background(), txHash)
		if err != nil {
			log.Infoln(err)
			return false
		}
		count++
		if !isPending {
			break
		}
		if count >= 100 {
			log.Warnln("Not waiting for the result.")
			return false
		}
	}
	return true
}
