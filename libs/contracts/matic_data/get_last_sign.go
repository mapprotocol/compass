package matic_data

import (
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/mapprotocol/compass/chain_tools"
	"github.com/mapprotocol/compass/libs"
	"github.com/mapprotocol/compass/libs/contracts"
	log "github.com/sirupsen/logrus"
	"math/big"
	"strings"
)

func GetLastSign() *big.Int {
	client := libs.GetClient()

	privateKey := libs.GetKey("")
	fromAddress := crypto.PubkeyToAddress(privateKey.PublicKey)
	var abiStaking, _ = abi.JSON(strings.NewReader(curAbi))
	input := chain_tools.PackInput(abiStaking, "getLastSign", fromAddress)
	ret := contracts.CallContract(client, fromAddress, libs.DataContractAddress, input)
	var res = big.NewInt(0)
	if len(ret) == 0 {
		return res
	}
	err := abiStaking.UnpackIntoInterface(&res, "getLastSign", ret)

	if err != nil {
		log.Infoln("abi error", err)
		return res
	}
	return res
}
