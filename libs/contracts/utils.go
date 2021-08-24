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
		log.Println("EstimateGas error: ", err)
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
		log.Println("Get ChainID error:", err)
	}
	fmt.Println("TX data nonce ", nonce, " transfer value ", value, " gasLimit ", gasLimit, " gasPrice ", gasPrice, " chainID ", chainID)

	signedTx, err := types.SignTx(tx, types.NewEIP2930Signer(chainID), privateKey)
	if err != nil {
		log.Println(err)
		return nil
	}

	err = client.SendTransaction(context.Background(), signedTx)
	if err != nil {
		log.Println("SendTransaction error: ", err)
		return nil
	}

	log.Println("Transaction Hash: ", signedTx.Hash())
	return signedTx
}
func CallContract(client *ethclient.Client, from, toAddress common.Address, input []byte) []byte {
	var ret []byte

	msg := ethereum.CallMsg{From: from, To: &toAddress, Data: input}

	header, err := client.HeaderByNumber(context.Background(), nil)
	if err != nil {
		log.Println("Get blockNumber error: ", err)
	}

	ret, err = client.CallContract(context.Background(), msg, header.Number)
	if err != nil {
		log.Println("method CallContract error: ", err)
	}
	return ret
}
