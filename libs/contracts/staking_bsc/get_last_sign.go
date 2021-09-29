package staking_bsc

import (
	"log"
	"math/big"
	"signmap/libs"
	"signmap/libs/contracts"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

func GetLastSign() (*big.Int, bool) {
	client := libs.GetClient()

	fromAddress := BindAddress()
	var abiStaking, _ = abi.JSON(strings.NewReader(curAbi))
	input := contracts.PackInput(abiStaking, "getLastSign", fromAddress)
	ret := contracts.CallContract(client, fromAddress, libs.DataContractAddress, input)
	var res = big.NewInt(0)
	if len(ret) == 0 {
		log.Println("getLastSign error.")
		return res, false
	}
	err := abiStaking.UnpackIntoInterface(&res, "getLastSign", ret)

	if err != nil {
		log.Println("abi error", err)
		return res, false
	}
	return res, true
}
