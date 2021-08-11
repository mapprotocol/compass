package matic_data

import (
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/mapprotocol/compass/libs"
	"github.com/mapprotocol/compass/libs/contracts"
	log "github.com/sirupsen/logrus"
	"strings"
)

func BindAddress() common.Address {
	client := libs.GetClient()

	privateKey := libs.GetKey("")
	fromAddress := crypto.PubkeyToAddress(privateKey.PublicKey)
	var abiStaking, _ = abi.JSON(strings.NewReader(curAbi))
	input := contracts.PackInput(abiStaking, "bindAddress", fromAddress)
	ret := contracts.CallContract(client, fromAddress, libs.DataContractAddress, input)

	var res = common.Address{}
	if len(ret) == 0 {
		return res
	}
	err := abiStaking.UnpackIntoInterface(&res, "bindAddress", ret)

	if err != nil {
		log.Infoln("abi error", err)
		return common.Address{}
	}
	return res
}
