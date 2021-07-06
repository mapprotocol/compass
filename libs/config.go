package libs

import (
	"github.com/ethereum/go-ethereum/common"
	"math/big"
	"os"
)

const (
	filepath = "sign.log"
	RpcUrl   = "https://rpc.pist.me"
)

var (
	SendTransationValue    = big.NewInt(1000000000000000000)
	ContractAddress        = common.HexToAddress("0x64c1855C45CD9d20f024DaC89bc48371CeE8737F")
	SendTransationGasLimit = uint64(21000)
	ToAddress              = common.HexToAddress("0x799E24dC6B48549BbD1Fc9fcCa4d72880d8c7a15")
	SignLogFile, _         = os.OpenFile(filepath, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
)
