package MaticStaking

import (
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/crypto"
	"signmap/libs"
	"signmap/libs/contracts"
	"strings"
)

func DO() {
	client := libs.GetClient()

	privateKey := libs.GetKey("")
	fromAddress := crypto.PubkeyToAddress(privateKey.PublicKey)
	var abiStaking, _ = abi.JSON(strings.NewReader(curAbi))
	input := contracts.PackInput(abiStaking, "sign")
	contracts.SendContractTransaction(client, fromAddress, libs.MaticStakingContractAddress, nil, privateKey, input)
}
