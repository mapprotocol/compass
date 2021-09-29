package staking_bsc

import (
	"log"
	"signmap/libs"
	"signmap/libs/contracts"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

func BindAddress() common.Address {
	client := libs.GetClient()

	privateKey := libs.GetKey("")
	fromAddress := crypto.PubkeyToAddress(privateKey.PublicKey)
	var abiStaking, _ = abi.JSON(strings.NewReader(curAbi))
	input := contracts.PackInput(abiStaking, "bindAddress", fromAddress)
	var res = common.Address{}
	var ret []byte
	var err error
	for i := 0; i < 3; i++ {
		ret = contracts.CallContract(client, fromAddress, libs.DataContractAddress, input)
		err = abiStaking.UnpackIntoInterface(&res, "bindAddress", ret)
		if err == nil {
			break
		} else {
			time.Sleep(5 * time.Second)
		}
	}

	if err != nil {
		log.Println("call bindAddress error:", err)
		return common.Address{}
	}
	return res
}
