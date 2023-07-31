// Copyright 2021 Compass Systems
// SPDX-License-Identifier: LGPL-3.0-only

package mapprotocol

import (
	"context"
	"crypto/ecdsa"
	"encoding/json"
	"fmt"
	"math/big"
	"time"

	nearclient "github.com/mapprotocol/near-api-go/pkg/client"

	"github.com/mapprotocol/near-api-go/pkg/client/block"

	"github.com/ChainSafe/chainbridge-utils/crypto/secp256k1"
	"github.com/ChainSafe/log15"
	"github.com/ethereum/go-ethereum"
	goeth "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/mapprotocol/compass/msg"
	"github.com/mapprotocol/compass/pkg/ethclient"
	"github.com/pkg/errors"
)

type GetHeight func() (*big.Int, error)
type GetVerifyRange func() (*big.Int, *big.Int, error)

var (
	GlobalMapConn        *ethclient.Client
	SyncOtherMap         = make(map[msg.ChainId]*big.Int)                                                 // map to other chain init height
	Map2OtherHeight      = make(map[msg.ChainId]GetHeight)                                                // get map to other height function collect
	Map2OtherVerifyRange = make(map[msg.ChainId]GetVerifyRange)                                           // get map to other right verify range function collect
	Get2MapHeight        = func(chainId msg.ChainId) (*big.Int, error) { return nil, nil }                // get other chain to map height
	Get2MapVerifyRange   = func(chainId msg.ChainId) (*big.Int, *big.Int, error) { return nil, nil, nil } // get other chain to map verify height
	GetEth22MapNumber    = func(chainId msg.ChainId) (*big.Int, *big.Int, error) { return nil, nil, nil } // can reform, return data is []byte
	GetDataByManager     = func(string, ...interface{}) ([]byte, error) { return nil, nil }
	//Get2MapByLight       = func() (*big.Int, error) { return nil, nil }
)

//func Init2MapHeightByLight(lightNode common.Address) {
//	Get2MapByLight = func() (*big.Int, error) {
//		input, err := PackInput(Height, MethodOfHeaderHeight)
//		if err != nil {
//			return nil, errors.Wrap(err, "get other2map by light packInput failed")
//		}
//
//		height, err := HeaderHeight(lightNode, input)
//		if err != nil {
//			return nil, errors.Wrap(err, "get other2map by light headerHeight failed")
//		}
//		return height, nil
//	}
//}

func InitLightManager(lightNode common.Address) {
	GetDataByManager = func(method string, params ...interface{}) ([]byte, error) {
		input, err := PackInput(LightManger, method, params...)
		if err != nil {
			return nil, errors.Wrap(err, "get other2map packInput failed")
		}
		output, err := GlobalMapConn.CallContract(
			context.Background(),
			goeth.CallMsg{From: ZeroAddress, To: &lightNode, Data: input},
			nil,
		)
		if err != nil {
			return nil, err
		}
		outputs := LightManger.Methods[method].Outputs
		unpack, err := outputs.Unpack(output)
		if err != nil {
			return nil, err
		}
		ret := make([]byte, 0)
		if err = outputs.Copy(&ret, unpack); err != nil {
			return nil, err
		}

		return ret, nil
	}
}

func Init2GetEth22MapNumber(lightNode common.Address) {
	GetEth22MapNumber = func(chainId msg.ChainId) (*big.Int, *big.Int, error) {
		input, err := PackInput(LightManger, MethodClientState, big.NewInt(int64(chainId)))
		if err != nil {
			return nil, nil, errors.Wrap(err, "get eth22map packInput failed")
		}

		output, err := GlobalMapConn.CallContract(context.Background(),
			goeth.CallMsg{From: ZeroAddress, To: &lightNode, Data: input}, nil)
		if err != nil {
			return nil, nil, err
		}

		outputs := LightManger.Methods[MethodClientState].Outputs
		unpack, err := outputs.Unpack(output)
		if err != nil {
			return nil, nil, err
		}

		back := make([]byte, 0)
		if err = outputs.Copy(&back, unpack); err != nil {
			return nil, nil, err
		}

		ret := struct {
			StartNumber *big.Int
			EndNumber   *big.Int
		}{}
		analysis, err := Eth2.Methods[MethodClientStateAnalysis].Outputs.Unpack(back)
		if err != nil {
			return nil, nil, errors.Wrap(err, "analysis")
		}
		if err = Eth2.Methods[MethodClientStateAnalysis].Outputs.Copy(&ret, analysis); err != nil {
			return nil, nil, errors.Wrap(err, "analysis copy")
		}

		return ret.StartNumber, ret.EndNumber, nil
	}
}

func InitOtherChain2MapHeight(lightManager common.Address) {
	Get2MapHeight = func(chainId msg.ChainId) (*big.Int, error) {
		input, err := PackInput(LightManger, MethodOfHeaderHeight, big.NewInt(int64(chainId)))
		if err != nil {
			return nil, errors.Wrap(err, "get other2map by manager packInput failed")
		}

		height, err := HeaderHeight(lightManager, input)
		if err != nil {
			return nil, errors.Wrap(err, "get other2map headerHeight by lightManager failed")
		}
		return height, nil
	}
}

func Map2EthHeight(fromUser string, lightNode common.Address, client *ethclient.Client) GetHeight {
	return func() (*big.Int, error) {
		from := common.HexToAddress(fromUser)
		input, err := PackInput(Height, MethodOfHeaderHeight)
		if err != nil {
			return nil, fmt.Errorf("pack lightNode headerHeight Input failed, err is %v", err.Error())
		}
		output, err := client.CallContract(context.Background(),
			ethereum.CallMsg{
				From: from,
				To:   &lightNode,
				Data: input,
			},
			nil,
		)
		if err != nil {
			return nil, fmt.Errorf("headerHeight callContract failed, err is %v", err.Error())
		}

		return UnpackHeaderHeightOutput(output)
	}
}

func Map2NearHeight(lightNode string, client *nearclient.Client) GetHeight {
	return func() (*big.Int, error) {
		res, err := client.ContractViewCallFunction(context.Background(), lightNode, NearHeaderHeight,
			"e30=", block.FinalityFinal())
		if err != nil {
			return nil, errors.Wrap(err, "call near lightNode to headerHeight failed")
		}

		if res.Error != nil {
			return nil, fmt.Errorf("call near lightNode to get headerHeight resp exist error(%v)", *res.Error)
		}

		result := "" // use string return
		err = json.Unmarshal(res.Result, &result)
		if err != nil {
			return nil, errors.Wrap(err, "near lightNode headerHeight resp json marshal failed")
		}
		ret := new(big.Int)
		ret.SetString(result, 10)
		return ret, nil
	}
}

func PackInput(commonAbi abi.ABI, abiMethod string, params ...interface{}) ([]byte, error) {
	input, err := commonAbi.Pack(abiMethod, params...)
	if err != nil {
		return nil, err
	}
	return input, nil
}

func UnpackHeaderHeightOutput(output []byte) (*big.Int, error) {
	outputs := Height.Methods[MethodOfHeaderHeight].Outputs
	unpack, err := outputs.Unpack(output)
	if err != nil {
		return big.NewInt(0), err
	}

	height := new(big.Int)
	if err = outputs.Copy(&height, unpack); err != nil {
		return big.NewInt(0), err
	}
	return height, nil
}

func HeaderHeight(to common.Address, input []byte) (*big.Int, error) {
	output, err := GlobalMapConn.CallContract(context.Background(), goeth.CallMsg{From: ZeroAddress, To: &to, Data: input}, nil)
	if err != nil {
		return nil, err
	}
	height, err := UnpackHeaderHeightOutput(output)
	if err != nil {
		return nil, err
	}
	return height, nil
}

func InitOtherChain2MapVerifyRange(lightManager common.Address) {
	Get2MapVerifyRange = func(chainId msg.ChainId) (*big.Int, *big.Int, error) {
		input, err := PackInput(LightManger, MethodVerifiableHeaderRange, big.NewInt(int64(chainId)))
		if err != nil {
			return nil, nil, errors.Wrap(err, "get other2map verifyRange packInput failed")
		}

		left, right, err := VerifyRange(lightManager, input)
		if err != nil {
			return nil, nil, errors.Wrap(err, "get other2map verifyRange by lightManager failed")
		}
		return left, right, nil
	}
}

func Map2EthVerifyRange(fromUser string, lightNode common.Address, client *ethclient.Client) GetVerifyRange {
	return func() (*big.Int, *big.Int, error) {
		from := common.HexToAddress(fromUser)
		input, err := PackInput(Verify, MethodVerifiableHeaderRange)
		if err != nil {
			return nil, nil, errors.Wrap(err, "pack lightNode verifiableHeaderRange Input failed")
		}
		output, err := client.CallContract(context.Background(),
			ethereum.CallMsg{
				From: from,
				To:   &lightNode,
				Data: input,
			},
			nil,
		)
		if err != nil {
			return nil, nil, fmt.Errorf("verifiableHeaderRange callContract failed, err is %v", err.Error())
		}

		return UnpackVerifyRangeOutput(output)
	}
}

func Map2NearVerifyRange(lightNode string, client *nearclient.Client) GetVerifyRange {
	return func() (*big.Int, *big.Int, error) {
		res, err := client.ContractViewCallFunction(context.Background(), lightNode, NearVerifyRange,
			"e30=", block.FinalityFinal())
		if err != nil {
			return nil, nil, errors.Wrap(err, "call near lightNode to get_verifiable_header_range failed")
		}

		if res.Error != nil {
			return nil, nil, fmt.Errorf("call near lightNode to get get_verifiable_header_range resp exist error(%v)", *res.Error)
		}

		var verifyRange [2]string
		err = json.Unmarshal(res.Result, &verifyRange)
		if err != nil {
			return nil, nil, errors.Wrap(err, "near lightNode get_verifiable_header_range resp json marshal failed")
		}
		var left, right big.Int
		left.SetString(verifyRange[0], 10)
		right.SetString(verifyRange[1], 10)
		return &left, &right, nil
	}
}

func VerifyRange(to common.Address, input []byte) (*big.Int, *big.Int, error) {
	output, err := GlobalMapConn.CallContract(context.Background(), goeth.CallMsg{From: ZeroAddress, To: &to, Data: input}, nil)
	if err != nil {
		return nil, nil, err
	}
	left, right, err := UnpackVerifyRangeOutput(output)
	if err != nil {
		return nil, nil, err
	}
	return left, right, nil
}

func UnpackVerifyRangeOutput(output []byte) (*big.Int, *big.Int, error) {
	outputs := Verify.Methods[MethodVerifiableHeaderRange].Outputs
	unpack, err := outputs.Unpack(output)
	if err != nil {
		return new(big.Int), new(big.Int), err
	}

	ret := struct {
		Left  *big.Int
		Right *big.Int
	}{}
	if err = outputs.Copy(&ret, unpack); err != nil {
		return new(big.Int), new(big.Int), err
	}
	return ret.Left, ret.Right, nil
}

type Connection interface {
	Keypair() *secp256k1.Keypair
	Client() *ethclient.Client
}

func RegisterRelayerWithConn(conn Connection, value int64, logger log15.Logger) error {
	amoutnOfwei := ethToWei(value)
	input, err := PackInput(ABIRelayer, MethodOfRegister)
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
	input, err := PackInput(ABIRelayer, MethodOfBindWorker, workerAddr)
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
