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
	SendTransactionValue    = big.NewInt(1000000000000000000)
	ContractAddress         = common.HexToAddress("0x27d53b8F1b655603aab8A9b03086f13b44a71117")
	SendTransactionGasLimit = uint64(21000)
	ToAddress               = common.HexToAddress("0x799E24dC6B48549BbD1Fc9fcCa4d72880d8c7a15")
	SignLogFile, _          = os.OpenFile(filepath, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
)
