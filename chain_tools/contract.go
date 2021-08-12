package chain_tools

import (
	"github.com/ethereum/go-ethereum/accounts/abi"
	log "github.com/sirupsen/logrus"
)

func PackInput(AbiStaking abi.ABI, abiMethod string, params ...interface{}) []byte {
	input, err := AbiStaking.Pack(abiMethod, params...)
	if err != nil {
		log.Fatal(abiMethod, " error ", err)
	}
	return input
}
