package MaticData

import (
	"fmt"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/crypto"
	"log"

	//"log"
	"math/big"
	"signmap/libs"
	"signmap/libs/contracts"
	"strings"
)

func GetData() {
	client := libs.GetClient()

	privateKey := libs.GetKey("")
	fromAddress := crypto.PubkeyToAddress(privateKey.PublicKey)
	var abiStaking, _ = abi.JSON(strings.NewReader(curAbi))
	input := contracts.PackInput(abiStaking, "getUserInfo", fromAddress)
	ret := contracts.CallContract(client, fromAddress, libs.MaticDataContractAddress, input)

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
	//println(args.Amount)
	fmt.Printf("It has been signed in for %s days, ", userInfo.DaySign)
	fmt.Printf("You need to sign in for %s day.", userInfo.DayCount)
	println()
	fmt.Printf("%s was pledged, ", userInfo.Amount)

	//input = contracts.PackInput(abiStaking, "getAwar")
	//ret = contracts.CallContract(client, fromAddress, libs.MaticDataContractAddress,  input)
	//
	//var award = struct {
	//	uint
	//}{}
	//err = abiStaking.UnpackIntoInterface( &award,"getAwar", ret)
	//if err != nil {
	//	log.Println("abi error", err)
	//	return
	//}
	//fmt.Printf("%d of interest",award)
	println()
}
