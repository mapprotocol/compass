package matic_staking

import (
	"context"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"log"
	"signmap/libs"
	"signmap/libs/contracts"
	"strings"
)

func DO() bool {
	client := libs.GetClient()

	privateKey := libs.GetKey("")
	fromAddress := crypto.PubkeyToAddress(privateKey.PublicKey)
	var abiStaking, _ = abi.JSON(strings.NewReader(curAbi))
	input := contracts.PackInput(abiStaking, "sign")
	tx := contracts.SendContractTransaction(client, fromAddress, libs.MaticStakingContractAddress, nil, privateKey, input)
	if tx == nil {
		return false
	}
	for {
		receipt, err := client.TransactionReceipt(context.Background(), tx.Hash())
		if err != nil {
			log.Println(err)
			return false
		}
		switch receipt.Status {
		case types.ReceiptStatusSuccessful:
			println("Sign in successfully.")
			return true
		case types.ReceiptStatusFailed:
			log.Println("Transaction not completed，unconfirmed.")
			return false
		default:
			continue
		}
	}

}
