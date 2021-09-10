package events

import (
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	abi2 "github.com/mapprotocol/compass/abi"
	"github.com/mapprotocol/compass/atlas"
	"github.com/mapprotocol/compass/chain_tools"
	"github.com/mapprotocol/compass/cmd/cmd_runtime"
	types2 "github.com/mapprotocol/compass/types"
	log "github.com/sirupsen/logrus"
	"strings"
	"time"
)

var (
	abiRouter, _  = abi.JSON(strings.NewReader(abi2.RouterContractAbi))
	eventResponse types2.EventLogSwapOutResponse
	err           error
	txProve       []byte
	tx            *types.Transaction
)

func HandleLogSwapOut(aLog *types.Log) {
	err = abiRouter.UnpackIntoInterface(&eventResponse, "LogSwapOut", aLog.Data)
	if err != nil {
		log.Fatal(err)
	}
	txProve = atlas.GetTxProve(cmd_runtime.SrcInstance, aLog, &eventResponse)

	token := common.BytesToAddress(aLog.Topics[1].Bytes())
	//to := common.BytesToAddress(aLog.Topics[3].Bytes())

	// swapIn(uint256 id, address token, address to, uint amount, uint fromChainID, bytes32[] memory data)
	//input := chain_tools.PackInput(abiRouter, "swapIn",
	//	eventResponse.OrderId,
	//	token,
	//	to,
	//	eventResponse.Amount,
	//	eventResponse.FromChainID,
	//	aLog.Address,
	//	txProve)
	//fmt.Printf("%+v", eventResponse)
	input := chain_tools.PackInput(abiRouter, "txVerify",
		aLog.Address,
		token,
		eventResponse.FromChainID,
		eventResponse.ToChainID,
		txProve)

	for {
		for {
			tx = chain_tools.SendContractTransactionWithoutOutputUnlessError(cmd_runtime.DstInstance.GetClient(),
				common.HexToAddress(cmd_runtime.DstInstance.GetAddress()),
				common.HexToAddress(cmd_runtime.DstChainConfig.RouterContractAddress),
				nil, cmd_runtime.DstInstance.GetPrivateKey(),
				input)
			if tx == nil {
				time.Sleep(5 * time.Second)
				continue
			}
			if chain_tools.WaitingForEndPending(cmd_runtime.DstInstance.GetClient(), tx.Hash(), 100) {
				break
			}
		}
		if chain_tools.WaitForReceipt(cmd_runtime.DstInstance.GetClient(), tx.Hash(), 1000) {
			return
		} else {
			time.Sleep(5 * time.Second)
		}
	}

}
