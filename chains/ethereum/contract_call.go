package ethereum

import (
	"github.com/ethereum/go-ethereum/accounts/abi"
	abi2 "github.com/mapprotocol/compass/abi"
	"github.com/mapprotocol/compass/chain_tools"
	"github.com/mapprotocol/compass/libs"
	"github.com/mapprotocol/compass/types"
	log "github.com/sirupsen/logrus"
	"math/big"
	"strings"
)

func (t *TypeEther) Register(value *big.Int) bool {
	var abiStaking, _ = abi.JSON(strings.NewReader(abi2.RelayerContractAbi))
	input := chain_tools.PackInput(abiStaking, "register", value)
	tx := chain_tools.SendContractTransactionWithoutOutputUnlessError(t.client, t.address, t.relayerContractAddress, nil, t.PrivateKey, input)
	if tx == nil {
		return false
	}
	return libs.WaitingForEndPending(t.client, tx.Hash())
}

func (t *TypeEther) UnRegister(value *big.Int) bool {
	var abiStaking, _ = abi.JSON(strings.NewReader(abi2.RelayerContractAbi))
	input := chain_tools.PackInput(abiStaking, "unregister", &value)
	tx := chain_tools.SendContractTransactionWithoutOutputUnlessError(t.client, t.address, t.relayerContractAddress, nil, t.PrivateKey, input)
	if tx == nil {
		return false
	}
	return libs.WaitingForEndPending(t.client, tx.Hash())
}

func (t *TypeEther) Withdraw(value *big.Int) bool {
	var abiStaking, _ = abi.JSON(strings.NewReader(abi2.RelayerContractAbi))
	input := chain_tools.PackInput(abiStaking, "withdraw", &value)
	tx := chain_tools.SendContractTransactionWithoutOutputUnlessError(t.client, t.address, t.relayerContractAddress, nil, t.PrivateKey, input)
	if tx == nil {
		return false
	}
	return libs.WaitingForEndPending(t.client, tx.Hash())
}

func (t *TypeEther) GetRelayerBalance() types.GetRelayerBalanceResponse {
	var abiStaking, _ = abi.JSON(strings.NewReader(abi2.RelayerContractAbi))
	input := chain_tools.PackInput(abiStaking, "getRelayerBalance", t.address)
	ret, _ := chain_tools.CallContractReturnBool(t.client, t.address, t.relayerContractAddress, input)
	var res types.GetRelayerBalanceResponse
	if len(ret) == 0 {
		return res
	}
	err := abiStaking.UnpackIntoInterface(&res, "getRelayerBalance", ret)
	if err != nil {
		log.Warnln("abi error", err)
		return res
	}
	return res
}
func (t *TypeEther) GetRelayer() types.GetRelayerResponse {
	var abiStaking, _ = abi.JSON(strings.NewReader(abi2.RelayerContractAbi))
	input := chain_tools.PackInput(abiStaking, "getRelayer", t.address)
	ret, _ := chain_tools.CallContractReturnBool(t.client, t.address, t.relayerContractAddress, input)
	var res types.GetRelayerResponse
	if len(ret) == 0 {
		return res
	}
	err := abiStaking.UnpackIntoInterface(&res, "getRelayer", ret)
	if err != nil {
		log.Warnln("abi error", err)
		return res
	}
	return res
}
func (t *TypeEther) GetPeriodHeight() types.GetPeriodHeightResponse {
	var abiStaking, _ = abi.JSON(strings.NewReader(abi2.RelayerContractAbi))
	input := chain_tools.PackInput(abiStaking, "getPeriodHeight", t.address)
	ret, _ := chain_tools.CallContractReturnBool(t.client, t.address, t.relayerContractAddress, input)

	var res types.GetPeriodHeightResponse
	if len(ret) == 0 {
		return res
	}
	err := abiStaking.UnpackIntoInterface(&res, "getPeriodHeight", ret)
	if err != nil {
		log.Warnln("abi error", err)
		return res
	}
	return res
}
