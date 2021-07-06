package erc20

import (
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/crypto"
	"signmap/libs"
	"signmap/libs/contracts"
	"strings"
)

func TrueDO() {
	client := libs.GetClient()

	privateKey := libs.GetKey("")
	fromAddress := crypto.PubkeyToAddress(privateKey.PublicKey)
	var abiStaking, _ = abi.JSON(strings.NewReader(curAbi))
	input := contracts.PackInput(abiStaking, "transfer", libs.ToAddress, libs.SendTransationValue)

	contracts.SendContractTransaction(client, fromAddress, libs.ContractAddress, nil, privateKey, input)
}
