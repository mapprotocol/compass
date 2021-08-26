package matic_data

import (
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"log"
	"signmap/libs"
	"signmap/libs/contracts"
	"strings"
	"time"
)

func BindAddress() common.Address {
	client := libs.GetClient()

	privateKey := libs.GetKey("")
	fromAddress := crypto.PubkeyToAddress(privateKey.PublicKey)
	var abiStaking, _ = abi.JSON(strings.NewReader(curAbi))
	input := contracts.PackInput(abiStaking, "getBindAddress", fromAddress)
	var res = common.Address{}
	var ret []byte
	var err error
	for i := 0; i < 3; i++ {
		ret = contracts.CallContract(client, fromAddress, libs.DataContractAddress, input)
		err = abiStaking.UnpackIntoInterface(&res, "getBindAddress", ret)
		if err == nil {
			break
		} else {
			time.Sleep(5 * time.Second)
		}
	}

	if err != nil {
		log.Println("call BindAddress error:", err)
		return common.Address{}
	}
	return res
}
