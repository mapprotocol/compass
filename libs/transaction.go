package libs

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"log"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
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

	signedTx, err := types.SignTx(tx, types.NewEIP2930Signer(ChainId), privateKey)
	if err != nil {
		log.Println(err)
	}

	err = client.SendTransaction(context.Background(), signedTx)
	if err != nil {
		log.Println(err)
	}

	fmt.Printf("tx sent: %s", signedTx.Hash().Hex())
}

func GetBalance(addr common.Address) *big.Int {
	client := GetClient()
	balance, err := client.BalanceAt(context.Background(), addr, nil)
	if err != nil {
		log.Fatal(err)
	}
	return balance
}
