package staking_bsc

import (
	"fmt"
	"log"
	"math/big"
	"os"
	"signmap/libs"
	"signmap/libs/contracts"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

func GetData() *big.Int {
	client := libs.GetClient()

	libs.GetKey("")
	fromAddress := BindAddress()

	var abiStaking, _ = abi.JSON(strings.NewReader(curAbi))
	input := contracts.PackInput(abiStaking, "userInfos", fromAddress)
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
		err = abiStaking.UnpackIntoInterface(&userInfo, "userInfos", ret)
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
	defer func(DaySign, DayCount *big.Int) {
		if DayCount != nil && DayCount != big.NewInt(0) && DaySign.Cmp(DayCount) >= 0 {
			log.Println("All sign-in completed")
			os.Exit(0)
		}
	}(userInfo.DaySign, userInfo.DayCount)

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
