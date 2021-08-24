package chain_tools

import (
	"context"
	"crypto/ecdsa"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	log "github.com/sirupsen/logrus"
	"math/big"
)

func SendContractTransactionWithoutOutputUnlessError(client *ethclient.Client, from, toAddress common.Address, value *big.Int, privateKey *ecdsa.PrivateKey, input []byte) *types.Transaction {
	nonce, err := client.PendingNonceAt(context.Background(), from)
	if err != nil {
		log.Warnln(err)
		return nil
	}
	gasPrice, err := client.SuggestGasPrice(context.Background())
	if err != nil {
		log.Warnln(err)
		return nil
	}
	var gasLimit uint64
	//msg := ethereum.CallMsg{From: from, To: &toAddress, GasPrice: gasPrice, Value: value, Data: input}
	//gasLimit, err = client.EstimateGas(context.Background(), msg)
	//if err != nil {
	//	log.Infoln("EstimateGas error: ", err)
	//	return nil
	//}
	gasLimit = uint64(800000)
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
		log.Infoln("Get ChainID error:", err)
	}
	signedTx, err := types.SignTx(tx, types.NewEIP2930Signer(chainID), privateKey)
	if err != nil {
		log.Warnln(err)
		return nil
	}
	err = client.SendTransaction(context.Background(), signedTx)
	if err != nil {
		log.Warnln("SendTransaction error: ", err)
		return nil
	}
	return signedTx
}
