package matic_data

import (
	"fmt"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"log"
	"math/big"
	"signmap/libs"
	"signmap/libs/contracts"
	"strings"
)

func GetData() {
	client := libs.GetClient()

	libs.GetKey("")
	fromAddress := BindAddress()

	var abiStaking, _ = abi.JSON(strings.NewReader(curAbi))
	input := contracts.PackInput(abiStaking, "getUserInfo", fromAddress)
	ret := contracts.CallContract(client, fromAddress, libs.DataContractAddress, input)

	userInfo := struct {
		Amount        *big.Int
		DayCount      *big.Int
		DaySign       *big.Int
		StakingStatus *big.Int
		SignTm        interface{}
	}{}
	err := abiStaking.UnpackIntoInterface(&userInfo, "getUserInfo", ret)

	if err != nil {
		log.Println("abi error", err)
		return
	}
	fmt.Printf("It has been signed in for %s/%s days.", userInfo.DaySign, userInfo.DayCount)
	println()
	fmt.Printf("%f was pledged, ", libs.WeiToEther(userInfo.Amount))

	input = contracts.PackInput(abiStaking, "getAward", fromAddress)
	ret = contracts.CallContract(client, fromAddress, libs.DataContractAddress, input)
	if len(ret) == 0 {
		return
	}
	var award *big.Int
	err = abiStaking.UnpackIntoInterface(&award, "getAward", ret)
	if err != nil {
		log.Println("abi error", err)
		return
	}
	fmt.Printf("%f of interest", libs.WeiToEther(award))
	println()
}
