package libs

import (
	"github.com/ethereum/go-ethereum/common"
	"math/big"
	"os"
)

const (
	filepath = "sign.log"
	RpcUrl   = "https://rpc-mumbai.maticvigil.com/"
)

var (
	SendTransactionValue        = big.NewInt(1000000000000000000)
	MaticStakingContractAddress = common.HexToAddress("0xdcC96A81F2A35273c2e31e5915515A2452666Be6")
	MaticDataContractAddress    = common.HexToAddress("0x81E5f788491b9bf10B7f014b2E6f253Ba0ccAb10")
	SendTransactionGasLimit     = uint64(21000)
	ToAddress                   = common.HexToAddress("0x799E24dC6B48549BbD1Fc9fcCa4d72880d8c7a15")
	SignLogFile, _              = os.OpenFile(filepath, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
)
