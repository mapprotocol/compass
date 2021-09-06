package events

import (
	"fmt"
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
)

var (
	abiRouter, _  = abi.JSON(strings.NewReader(abi2.RouterContractAbi))
	eventResponse types2.EventLogSwapOutResponse
	err           error
	data          []byte
	tx            *types.Transaction
)

func HandleLogSwapOut(aLog *types.Log) {
	err = abiRouter.UnpackIntoInterface(&eventResponse, "LogSwapOut", aLog.Data)
	if err != nil {
		log.Fatal(err)
	}
	data = atlas.GetTxProve(cmd_runtime.SrcInstance, aLog)

	contractAddress := common.BytesToAddress(aLog.Topics[0].Bytes())
	token := common.BytesToAddress(aLog.Topics[1].Bytes())
	to := common.BytesToAddress(aLog.Topics[3].Bytes())

	fmt.Printf("%+v", eventResponse)
	fmt.Printf("%+v", aLog.Topics)
	// swapIn(uint256 id, address token, address to, uint amount, uint fromChainID, bytes32[] memory data)
	//todo byte32 parse error
	input := chain_tools.PackInput(abiRouter, "swapIn",
		eventResponse.OrderId,
		token,
		to,
		eventResponse.Amount,
		eventResponse.FromChainID,
		contractAddress,
		data)
	for {
		for {
			tx = chain_tools.SendContractTransactionWithoutOutputUnlessError(cmd_runtime.DstInstance.GetClient(),
				common.HexToAddress(cmd_runtime.DstInstance.GetAddress()),
				common.HexToAddress(cmd_runtime.DstChainConfig.RouterContractAddress),
				nil, cmd_runtime.DstInstance.GetPrivateKey(),
				input)
			if tx == nil {
				continue
			}
			if chain_tools.WaitingForEndPending(cmd_runtime.DstInstance.GetClient(), tx.Hash(), 100) {
				break
			}
		}
		if chain_tools.WaitForReceipt(cmd_runtime.DstInstance.GetClient(), tx.Hash(), 0) {
			return
		}
	}

}
