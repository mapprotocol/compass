package chain_structs

import (
	"github.com/ethereum/go-ethereum/accounts/abi"
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
