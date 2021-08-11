package libs

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	log "github.com/sirupsen/logrus"
)

func SendTransaction() {
	client := GetClient()
	privateKey := GetKey("")

	publicKey := privateKey.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		log.Infoln("error casting public key to ECDSA")
	}

	fromAddress := crypto.PubkeyToAddress(*publicKeyECDSA)
	nonce, err := client.PendingNonceAt(context.Background(), fromAddress)
	if err != nil {
		log.Infoln(err)
	}

	gasPrice, err := client.SuggestGasPrice(context.Background())
	if err != nil {
		log.Infoln(err)
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

	chainID, err := client.NetworkID(context.Background())

	if err != nil {
		log.Infoln(err)
	}

	signedTx, err := types.SignTx(tx, types.NewEIP2930Signer(chainID), privateKey)
	if err != nil {
		log.Infoln(err)
	}

	err = client.SendTransaction(context.Background(), signedTx)
	if err != nil {
		log.Infoln(err)
	}

	fmt.Printf("tx sent: %s", signedTx.Hash().Hex())
}
