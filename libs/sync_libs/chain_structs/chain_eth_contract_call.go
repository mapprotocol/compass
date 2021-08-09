package chain_structs

import (
	"github.com/ethereum/go-ethereum/accounts/abi"
	"log"
	"math/big"
	"signmap/libs"
	"signmap/libs/contracts"
	contracts2 "signmap/libs/sync_libs/contracts"
	"strings"
)

func (t *TypeEther) Register(value *big.Int) bool {

	var abiStaking, _ = abi.JSON(strings.NewReader(contracts2.RelayerContractAbi))

	input := contracts.PackInput(abiStaking, "register", value)
	//_,ok := contracts.CallContractReturnBool(t.client, t.address, t.relayerContractAddress, input)
	tx := contracts.SendContractTransactionWithoutOutputUnlessError(t.client, t.address, t.relayerContractAddress, nil, t.PrivateKey, input)
	libs.GetResult(t.client, tx.Hash())
	return tx != nil
}

func (t *TypeEther) UnRegister(value *big.Int) bool {
	var abiStaking, _ = abi.JSON(strings.NewReader(contracts2.RelayerContractAbi))
	input := contracts.PackInput(abiStaking, "unregister", &value)
	tx := contracts.SendContractTransactionWithoutOutputUnlessError(t.client, t.address, t.relayerContractAddress, nil, t.PrivateKey, input)
	libs.GetResult(t.client, tx.Hash())
	return tx != nil
}

func (t *TypeEther) GetRelayerBalance() GetRelayerBalanceResponse {
	var abiStaking, _ = abi.JSON(strings.NewReader(contracts2.RelayerContractAbi))
	input := contracts.PackInput(abiStaking, "getRelayerBalance", t.address)
	ret, _ := contracts.CallContractReturnBool(t.client, t.address, t.relayerContractAddress, input)

	var res GetRelayerBalanceResponse
	err := abiStaking.UnpackIntoInterface(&res, "getRelayerBalance", ret)
	if err != nil {
		log.Println("abi error", err)
		return res
	}
	return res
}
