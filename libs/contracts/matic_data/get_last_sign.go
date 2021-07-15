package matic_data

import (
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/crypto"
	"log"
	"math/big"
	"signmap/libs"
	"signmap/libs/contracts"
	"strings"
)

func GetLastSign() *big.Int {
	client := libs.GetClient()

	privateKey := libs.GetKey("")
	fromAddress := crypto.PubkeyToAddress(privateKey.PublicKey)
	var abiStaking, _ = abi.JSON(strings.NewReader(curAbi))
	input := contracts.PackInput(abiStaking, "getLastSign", fromAddress)
	ret := contracts.CallContract(client, fromAddress, libs.MaticDataContractAddress, input)
	var res = big.NewInt(0)
	if len(ret) == 0 {
		return res
	}
	err := abiStaking.UnpackIntoInterface(&res, "getLastSign", ret)

	if err != nil {
		log.Println("abi error", err)
		return res
	}
	return res
}
