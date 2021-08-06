package chain_structs

import (
	"github.com/ethereum/go-ethereum/accounts/abi"
	"log"
	"math/big"
	"signmap/libs/contracts"
	contracts2 "signmap/libs/sync_libs/contracts"
	"strings"
)

func (t *TypeEther) Register(value big.Int) bool {
	var abiStaking, _ = abi.JSON(strings.NewReader(contracts2.RelayerContractAbi))
	input := contracts.PackInput(abiStaking, "register", &value)
	ret := contracts.CallContract(t.client, t.address, t.relayerContractAddress, input)
	if len(ret) == 0 {
		return false
	}
	return true
}

func (t *TypeEther) UnRegister(value big.Int) bool {
	var abiStaking, _ = abi.JSON(strings.NewReader(contracts2.RelayerContractAbi))
	input := contracts.PackInput(abiStaking, "unregister", &value)
	_, ok := contracts.CallContractReturnBool(t.client, t.address, t.relayerContractAddress, input)

	return ok
}

func (t *TypeEther) GetRelayerBalance() GetRelayerBalanceResponse {
	var abiStaking, _ = abi.JSON(strings.NewReader(contracts2.RelayerContractAbi))
	input := contracts.PackInput(abiStaking, "getRelayerBalance", t.address)
	ret, ok := contracts.CallContractReturnBool(t.client, t.address, t.relayerContractAddress, input)
	println(ok)
	println(string(ret))

	var res GetRelayerBalanceResponse
	err := abiStaking.UnpackIntoInterface(&res, "getRelayerBalance", ret)
	if err != nil {
		log.Println("abi error", err)
		return res
	}
	return res
}
