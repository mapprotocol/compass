package contracts

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"log"
	"math/big"
)

func PackInput(AbiStaking abi.ABI, abiMethod string, params ...interface{}) []byte {
	input, err := AbiStaking.Pack(abiMethod, params...)
	if err != nil {
		log.Fatal(abiMethod, " error ", err)
	}
	return input
}
func SendContractTransaction(client *ethclient.Client, from, toAddress common.Address, value *big.Int, privateKey *ecdsa.PrivateKey, input []byte) *types.Transaction {

	nonce, err := client.PendingNonceAt(context.Background(), from)
	if err != nil {
		log.Println(err)
		return nil
	}

	gasPrice, err := client.SuggestGasPrice(context.Background())
	if err != nil {
		log.Println(err)
		return nil
	}

	var gasLimit uint64
	msg := ethereum.CallMsg{From: from, To: &toAddress, GasPrice: gasPrice, Value: value, Data: input}
	gasLimit, err = client.EstimateGas(context.Background(), msg)
	if err != nil {
		log.Println("Contract exec failed", err)
		return nil
	}
	tx := types.NewTx(&types.LegacyTx{
		Nonce:    nonce,
		Value:    value,
		To:       &toAddress,
		Gas:      gasLimit,
		GasPrice: gasPrice,
		Data:     input,
	})
	chainID, err := client.ChainID(context.Background())
	if err != nil {
		log.Println(err)
	}
	fmt.Println("TX data nonce ", nonce, " transfer value ", value, " gasLimit ", gasLimit, " gasPrice ", gasPrice, " chainID ", chainID)

	signedTx, err := types.SignTx(tx, types.NewEIP155Signer(chainID), privateKey)
	if err != nil {
		log.Println(err)
		return nil
	}

	err = client.SendTransaction(context.Background(), signedTx)
	if err != nil {
		log.Println(err)
		return nil
	}

	log.Println(signedTx.Hash())
	return signedTx
}
