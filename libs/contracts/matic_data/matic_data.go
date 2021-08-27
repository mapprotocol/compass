package matic_data

import (
	"fmt"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"log"
	"math/big"
	"signmap/libs"
	"signmap/libs/contracts"
	"strings"
	"time"
)

func GetData() *big.Int {
	client := libs.GetClient()

	libs.GetKey("")
	fromAddress := BindAddress()

	var abiStaking, _ = abi.JSON(strings.NewReader(curAbi))
	input := contracts.PackInput(abiStaking, "getUserInfo", fromAddress)
	userInfo := struct {
		Amount        *big.Int
		DayCount      *big.Int
		DaySign       *big.Int
		StakingStatus *big.Int
		SignTm        interface{}
	}{}
	var err error
	var ret []byte
	for i := 0; i < 3; i++ {
		ret = contracts.CallContract(client, fromAddress, libs.DataContractAddress, input)
		err = abiStaking.UnpackIntoInterface(&userInfo, "getUserInfo", ret)
		if err == nil {
			break
		} else {
			time.Sleep(5 * time.Second)
		}
	}
	if err != nil {
		log.Println("call getData error :", err)
		return nil
	}

	fmt.Printf("It has been signed in for %s/%s days.", userInfo.DaySign, userInfo.DayCount)
	println()
	fmt.Printf("%f was pledged, ", libs.WeiToEther(userInfo.Amount))

	var award *big.Int
	input = contracts.PackInput(abiStaking, "getAward", fromAddress)

	for i := 0; i < 3; i++ {
		ret = contracts.CallContract(client, fromAddress, libs.DataContractAddress, input)
		err = abiStaking.UnpackIntoInterface(&award, "getAward", ret)
		if err == nil {
			break
		} else {
			time.Sleep(5 * time.Second)
		}
	}

	if err != nil {
		log.Println("call getAward error :", err)
		return nil
	}
	fmt.Printf("%f of interest", libs.WeiToEther(award))
	println()
	return userInfo.Amount
}
