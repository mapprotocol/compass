package matic_staking

import (
	"context"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/mapprotocol/compass/chain_tools"
	"github.com/mapprotocol/compass/libs"
	"github.com/mapprotocol/compass/libs/contracts"
	"github.com/mapprotocol/compass/libs/contracts/matic_data"
	log "github.com/sirupsen/logrus"
	"strings"
	"time"
)

func DO() bool {
	client := libs.GetClient()

	privateKey := libs.GetKey("")
	fromAddress := crypto.PubkeyToAddress(privateKey.PublicKey)
	var abiStaking, _ = abi.JSON(strings.NewReader(curAbi))
	input := chain_tools.PackInput(abiStaking, "sign")
	tx := contracts.SendContractTransaction(client, fromAddress, libs.StakingContractAddress, nil, privateKey, input)
	if tx == nil {
		return false
	}
	i := -1
	tryTimes := 5
	sleepSecond := 5 * time.Second
	for {
		i += 1
		if i >= tryTimes {
			if libs.Unix2Time(*matic_data.GetLastSign()).Format("20060102") == time.Now().Format("20060102") {
				println("Sign in successfully.")
				return true
			}
			log.Infoln("Attempts to get the receipt ", tryTimes, " times，unsuccessful.")
			return false
		}
		receipt, err := client.TransactionReceipt(context.Background(), tx.Hash())
		if err != nil {
			log.Infoln("Get receipt error: ", err)
			time.Sleep(sleepSecond)
			continue
		}
		switch receipt.Status {
		case types.ReceiptStatusSuccessful:
			println("Sign in successfully.")
			return true
		case types.ReceiptStatusFailed:
			log.Infoln("Transaction not completed，unconfirmed.")
			return false
		default:
			//should unreachable
			log.Infoln("Unknown receipt status: ", receipt.Status)
			time.Sleep(sleepSecond / 2)
			continue
		}
	}

}
