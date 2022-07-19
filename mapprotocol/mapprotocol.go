// Copyright 2021 Compass Systems
// SPDX-License-Identifier: LGPL-3.0-only

package mapprotocol

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/mapprotocol/compass/msg"

	"github.com/ChainSafe/chainbridge-utils/crypto/secp256k1"
	"github.com/ChainSafe/log15"
	goeth "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/mapprotocol/compass/pkg/ethclient"
)

// GlobalMapConn global Map connection; assign at cmd/main
var GlobalMapConn *ethclient.Client

func packInput(commonAbi abi.ABI, abiMethod string, params ...interface{}) ([]byte, error) {
	input, err := commonAbi.Pack(abiMethod, params...)
	if err != nil {
		return nil, err
	}
	return input, nil
}

func PackLightNodeInput(method string, params ...interface{}) ([]byte, error) {
	return packInput(ABILightNode, method, params...)
}

func PackVerifyInput(method string, params ...interface{}) ([]byte, error) {
	return packInput(Verify, method, params...)
}

func Eth2MapTransferInPackInput(method string, params ...interface{}) ([]byte, error) {
	return packInput(Eth2MapTransferInAbi, method, params...)
}

func SaveHeaderTxData(params ...interface{}) ([]byte, error) {
	return packInput(ABIRelayer,
		UpdateBlockHeader,
		params...)
}

func SaveHeaderLiteTxData(marshal []byte) ([]byte, error) {
	return packInput(ABILiteNode, SaveHeader, marshal)
}

func GetCurrentNumberAbi(from common.Address, chainId msg.ChainId) (*big.Int, string, error) {
	if GlobalMapConn == nil {
		return Big0, "", errors.New(" Global Map Connection is not assigned!")
	}

	blockNum, err := GlobalMapConn.BlockNumber(context.Background())
	if err != nil {
		return Big0, "", err
	}
	input, _ := packInput(ABIRelayer, CurNbrAndHash, big.NewInt(int64(chainId)))

	msg := goeth.CallMsg{
		From: from,
		To:   &RelayerAddress,
		Data: input,
	}

	output, err := GlobalMapConn.CallContract(context.Background(), msg, big.NewInt(0).SetUint64(blockNum))
	if err != nil {
		return Big0, "", err
	}
	method, _ := ABIRelayer.Methods[CurNbrAndHash]
	ret, err := method.Outputs.Unpack(output)
	if err != nil {
		return Big0, "", err
	}
	height := ret[0].(*big.Int)
	hash := common.BytesToHash(ret[1].([]byte))

	return height, hash.String(), nil
}

type Connection interface {
	Keypair() *secp256k1.Keypair
	Client() *ethclient.Client
}

func RegisterRelayerWithConn(conn Connection, value int64, logger log15.Logger) error {
	amoutnOfwei := ethToWei(value)
	input, err := packInput(ABIRelayer, RegisterRelayer)
	if err != nil {
		return err
	}
	kp := conn.Keypair()
	err = sendContractTransaction(conn.Client(), kp.CommonAddress(),
		RelayerAddress, amoutnOfwei, kp.PrivateKey(), input, logger)
	if err != nil {
		return err
	}

	return nil
}

func BindWorkerWithConn(conn Connection, worker string, logger log15.Logger) error {
	workerAddr := common.HexToAddress(worker)
	input, err := packInput(ABIRelayer, BindWorker, workerAddr)
	if err != nil {
		return err
	}
	kp := conn.Keypair()
	err = sendContractTransaction(conn.Client(), kp.CommonAddress(),
		RelayerAddress, nil, kp.PrivateKey(), input, logger)
	if err != nil {
		return err
	}

	return nil
}

func ethToWei(registerValue int64) *big.Int {
	baseUnit := big.NewInt(0).Exp(big.NewInt(10), big.NewInt(18), nil)
	value := big.NewInt(0).Mul(big.NewInt(registerValue), baseUnit)
	return value
}

func sendContractTransaction(client *ethclient.Client, from, toAddress common.Address,
	value *big.Int, privateKey *ecdsa.PrivateKey, input []byte, logger log15.Logger) error {

	// Ensure a valid value field and resolve the account nonce
	nonce, err := client.PendingNonceAt(context.Background(), from)
	if err != nil {
		logger.Error("sendContractTransaction PendingNonceAt")
		return err
	}

	gasPrice, err := client.SuggestGasPrice(context.Background())
	if err != nil {
		logger.Error("sendContractTransaction SuggestGasPrice")
		return err
	}

	gasLimit := uint64(2100000) // in units
	// If the contract surely has code (or code is not needed), estimate the transaction
	// 如果合同确实有代码（或不需要代码），则估计交易
	msg := goeth.CallMsg{From: from, To: &toAddress, GasPrice: gasPrice, Value: value, Data: input}
	gasLimit, err = client.EstimateGas(context.Background(), msg)
	if err != nil {
		logger.Warn("client.EstimateGas failed!", "err", err)
	}
	//log.Info("EstimateGas gasLimit : ", gasLimit)
	if gasLimit < 1 {
		//gasLimit = 866328
		gasLimit = 2000000
		logger.Info("use specified gasLimit", "gasLimit", gasLimit)
	}

	// Create the transaction, sign it and schedule it for execution
	tx := types.NewTransaction(nonce, toAddress, value, gasLimit, gasPrice, input)
	logger.Info("NewTx", "gasLimit", gasLimit, "gasPrice", gasPrice)

	chainID, err := client.ChainID(context.Background())
	if err != nil {
		logger.Error("sendContractTransaction ChainID")
		return err
	}
	//log.Info("TX data nonce ", nonce, " transfer value ", value, " gasLimit ", gasLimit, " gasPrice ", gasPrice, " chainID ", chainID)
	signer := types.LatestSignerForChainID(chainID)
	signedTx, err := types.SignTx(tx, signer, privateKey)
	if err != nil {
		logger.Error("sendContractTransaction signedTx")
		return err
	}

	err = client.SendTransaction(context.Background(), signedTx)
	if err != nil {
		logger.Error("sendContractTransaction client.SendTransaction")
		return err
	}
	txHash := signedTx.Hash()
	logger.Info("transaction sent", "txHash", txHash)
	count := 0
	for {
		time.Sleep(time.Millisecond * 500)
		_, isPending, err := client.TransactionByHash(context.Background(), txHash)

		if err != nil {
			logger.Error("sendContractTransaction TransactionByHash")
			return err
		}
		count++
		if !isPending {
			break
		} else {
			logger.Info("transaction is pending, please wait...")
		}
	}
	receipt, err := client.TransactionReceipt(context.Background(), txHash)
	count1 := 0
	if err != nil {
		logger.Warn("TransactionReceipt failed, Retrying...", "err", err)
		for {
			time.Sleep(time.Second * 5)
			count1++
			receipt, err = client.TransactionReceipt(context.Background(), txHash)
			if err == nil {
				break
			} else {
				logger.Error("TransactionReceipt receipt finding...", "err", err)
			}
			if count1 > 10 {
				return fmt.Errorf("exceed MAX tryout")
			}
		}
	}
	if receipt.Status == types.ReceiptStatusSuccessful {
		logger.Info("Transaction Success", "block Number", receipt.BlockNumber.Uint64(), "blockhash", receipt.BlockHash.Hex())
		return nil
	} else if receipt.Status == types.ReceiptStatusFailed {
		logger.Warn("TX data  ", "nonce", nonce, " transfer value", value, " gasLimit", gasLimit, " gasPrice", gasPrice, " chainID", chainID)
		logger.Warn("Transaction Failed", "Block Number", receipt.BlockNumber.Uint64())
		return fmt.Errorf("ReceiptStatusFailed")
	}
	return fmt.Errorf("ReceiptStatus:%v", receipt.Status)
}
