package matic_staking

import (
	"context"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"log"
	"signmap/libs"
	"signmap/libs/contracts"
	"signmap/libs/contracts/matic_data"
	"strings"
	"time"
)

func DO() bool {
	client := libs.GetClient()

	privateKey := libs.GetKey("")
	fromAddress := crypto.PubkeyToAddress(privateKey.PublicKey)
	var abiStaking, _ = abi.JSON(strings.NewReader(curAbi))
	input := contracts.PackInput(abiStaking, "sign")
	var tx *types.Transaction
	for i := 0; i < 5; i++ {
		tx = contracts.SendContractTransaction(client, fromAddress, libs.StakingContractAddress, nil, privateKey, input)
		if tx == nil {
			time.Sleep(10 * time.Second)
		} else {
			break
		}
	}
	if tx == nil {
		log.Println("Attempts to sign 5 times，unsuccessful. The network impassability。")
		return false
	}
	i := -1
	tryTimes := 10
	sleepSecond := 5 * time.Second
	for {
		i += 1
		if i >= tryTimes {
			if libs.Unix2Time(*matic_data.GetLastSign()).Format("20060102") == time.Now().Format("20060102") {
				println("Sign in successfully.")
				return true
			}
			log.Println("Attempts to get the receipt ", tryTimes, " times，unsuccessful.")
			return false
		}
		receipt, err := client.TransactionReceipt(context.Background(), tx.Hash())
		if err != nil {
			time.Sleep(sleepSecond)
			continue
		}
		switch receipt.Status {
		case types.ReceiptStatusSuccessful:
			println("Sign in successfully.")
			return true
		case types.ReceiptStatusFailed:
			log.Println("Transaction not completed，unconfirmed.")
			return false
		default:
			//should unreachable
			log.Println("Unknown receipt status: ", receipt.Status)
			time.Sleep(sleepSecond / 2)
			continue
		}
	}

}
