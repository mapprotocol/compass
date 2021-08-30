package libs

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"log"
	"math/big"
)

func SendTransaction() {
	client := GetClient()
	privateKey := GetKey("")

	publicKey := privateKey.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		log.Println("error casting public key to ECDSA")
	}

	fromAddress := crypto.PubkeyToAddress(*publicKeyECDSA)
	nonce, err := client.PendingNonceAt(context.Background(), fromAddress)
	if err != nil {
		log.Println(err)
	}

	gasPrice, err := client.SuggestGasPrice(context.Background())
	if err != nil {
		log.Println(err)
	}

	var data []byte

	tx := types.NewTx(&types.LegacyTx{
		Nonce:    nonce,
		Value:    SendTransactionValue,
		To:       &ToAddress,
		Gas:      SendTransactionGasLimit,
		GasPrice: gasPrice,
		Data:     data,
	})

	if err != nil {
		log.Println(err)
	}

	signedTx, err := types.SignTx(tx, types.NewEIP2930Signer(big.NewInt(137)), privateKey)
	if err != nil {
		log.Println(err)
	}

	err = client.SendTransaction(context.Background(), signedTx)
	if err != nil {
		log.Println(err)
	}

	fmt.Printf("tx sent: %s", signedTx.Hash().Hex())
}

func GetBalance() *big.Int {
	client := GetClient()
	privateKey := GetKey("")
	fromAddress := crypto.PubkeyToAddress(privateKey.PublicKey)
	balance, err := client.BalanceAt(context.Background(), fromAddress, nil)
	if err != nil {
		log.Fatal(err)
	}
	return balance
}
