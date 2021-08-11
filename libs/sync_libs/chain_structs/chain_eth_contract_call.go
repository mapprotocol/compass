package chain_structs

import (
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/mapprotocol/compass/libs"
	"github.com/mapprotocol/compass/libs/contracts"
	contracts2 "github.com/mapprotocol/compass/libs/sync_libs/contracts"
	"log"
	"math/big"
	"strings"
)

func (t *TypeEther) Register(value *big.Int) bool {

	var abiStaking, _ = abi.JSON(strings.NewReader(contracts2.RelayerContractAbi))

	input := contracts.PackInput(abiStaking, "register", value)
	//_,ok := contracts.CallContractReturnBool(t.client, t.address, t.relayerContractAddress, input)
	tx := contracts.SendContractTransactionWithoutOutputUnlessError(t.client, t.address, t.relayerContractAddress, nil, t.PrivateKey, input)
	if tx == nil {
		return false
	}
	libs.GetResult(t.client, tx.Hash())
	return true
}

func (t *TypeEther) UnRegister(value *big.Int) bool {
	var abiStaking, _ = abi.JSON(strings.NewReader(contracts2.RelayerContractAbi))
	input := contracts.PackInput(abiStaking, "unregister", &value)
	tx := contracts.SendContractTransactionWithoutOutputUnlessError(t.client, t.address, t.relayerContractAddress, nil, t.PrivateKey, input)
	if tx == nil {
		return false
	}
	libs.GetResult(t.client, tx.Hash())
	return true
}
func (t *TypeEther) Withdraw(value *big.Int) bool {
	var abiStaking, _ = abi.JSON(strings.NewReader(contracts2.RelayerContractAbi))
	input := contracts.PackInput(abiStaking, "withdraw", &value)
	tx := contracts.SendContractTransactionWithoutOutputUnlessError(t.client, t.address, t.relayerContractAddress, nil, t.PrivateKey, input)
	if tx == nil {
		return false
	}
	libs.GetResult(t.client, tx.Hash())
	return true
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
func (t *TypeEther) GetRelayer() GetRelayerResponse {
	var abiStaking, _ = abi.JSON(strings.NewReader(contracts2.RelayerContractAbi))
	input := contracts.PackInput(abiStaking, "getRelayer", t.address)
	ret, _ := contracts.CallContractReturnBool(t.client, t.address, t.relayerContractAddress, input)

	var res GetRelayerResponse
	err := abiStaking.UnpackIntoInterface(&res, "getRelayer", ret)
	if err != nil {
		log.Println("abi error", err)
		return res
	}
	return res
}
func (t *TypeEther) GetPeriodHeight() GetPeriodHeightResponse {
	var abiStaking, _ = abi.JSON(strings.NewReader(contracts2.RelayerContractAbi))
	input := contracts.PackInput(abiStaking, "getPeriodHeight", t.address)
	ret, _ := contracts.CallContractReturnBool(t.client, t.address, t.relayerContractAddress, input)

	var res GetPeriodHeightResponse

	err := abiStaking.UnpackIntoInterface(&res, "getPeriodHeight", ret)
	if err != nil {
		log.Println("abi error", err)
		return res
	}
	return res
}
