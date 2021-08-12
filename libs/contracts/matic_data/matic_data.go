package matic_data

import (
	"fmt"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/mapprotocol/compass/chain_tools"
	"github.com/mapprotocol/compass/libs"
	"github.com/mapprotocol/compass/libs/contracts"
	log "github.com/sirupsen/logrus"
	"math/big"
	"strings"
)

func GetData() {
	client := libs.GetClient()

	privateKey := libs.GetKey("")
	fromAddress := crypto.PubkeyToAddress(privateKey.PublicKey)
	var abiStaking, _ = abi.JSON(strings.NewReader(curAbi))
	input := chain_tools.PackInput(abiStaking, "getUserInfo", fromAddress)
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
		log.Infoln("abi error", err)
		return
	}
	fmt.Printf("It has been signed in for %s/%s days.", userInfo.DaySign, userInfo.DayCount)
	println()
	fmt.Printf("%f was pledged, ", libs.WeiToEther(userInfo.Amount))

	input = chain_tools.PackInput(abiStaking, "getAward", fromAddress)
	ret = contracts.CallContract(client, fromAddress, libs.DataContractAddress, input)
	if len(ret) == 0 {
		return
	}
	var award *big.Int
	err = abiStaking.UnpackIntoInterface(&award, "getAward", ret)
	if err != nil {
		log.Infoln("abi error", err)
		return
	}
	fmt.Printf("%f of interest", libs.WeiToEther(award))
	println()
}
