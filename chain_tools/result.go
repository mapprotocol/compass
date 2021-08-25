package chain_tools

import (
	"context"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	log "github.com/sirupsen/logrus"
	"time"
)

func WaitingForEndPending(conn *ethclient.Client, txHash common.Hash, waitingSeconds int) bool {
	count := 0
	time.Sleep(time.Millisecond * 200)
	for {
		_, isPending, err := conn.TransactionByHash(context.Background(), txHash)
		if err != nil {
			log.Warnln(err)
			return false
		}
		count++
		if !isPending {
			return true
		}
		if count >= waitingSeconds {
			log.Warnln("Not waiting for the result.", txHash.String())
			return false
		}
		time.Sleep(time.Second)
	}
}

// WaitForReceipt
// @waitingSeconds int  <=0 means forever，
func WaitForReceipt(conn *ethclient.Client, txHash common.Hash, waitingSeconds int) bool {
	onceTime := time.Second
	count := 0
	for {
		count += 1

		receipt, err := conn.TransactionReceipt(context.Background(), txHash)
		if err != nil {
			if count == waitingSeconds {
				log.Warnln("Not waiting for the Receipt.", txHash.String(), waitingSeconds, "times.")
				return false
			} else {
				time.Sleep(onceTime)
				continue
			}
		}
		switch receipt.Status {
		case types.ReceiptStatusSuccessful:
			return true
		case types.ReceiptStatusFailed:
			log.Warnln("Transaction not completed，unconfirmed.", txHash.String())
			return false
		default:
			//should unreachable
			log.Warnln("Unknown receipt status: ", txHash.String(), receipt.Status)
			time.Sleep(onceTime / 2)
			continue
		}
	}

}
