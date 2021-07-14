package matic_data

import (
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"log"
	"signmap/libs"
	"signmap/libs/contracts"
	"strings"
)

func BindAddress() common.Address {
	client := libs.GetClient()

	privateKey := libs.GetKey("")
	fromAddress := crypto.PubkeyToAddress(privateKey.PublicKey)
	var abiStaking, _ = abi.JSON(strings.NewReader(curAbi))
	input := contracts.PackInput(abiStaking, "bindAddress", fromAddress)
	ret := contracts.CallContract(client, fromAddress, libs.MaticDataContractAddress, input)

	var res common.Address

	err := abiStaking.UnpackIntoInterface(&res, "bindAddress", ret)

	if err != nil {
		log.Println("abi error", err)
		return common.Address{}
	}
	return res
}
