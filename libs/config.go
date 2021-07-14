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
	MaticStakingContractAddress = common.HexToAddress("0xD9397A23c06418E5733ECBb434A3008e3CbB7647")
	MaticDataContractAddress    = common.HexToAddress("0xb4D6d8f5E7f971B8B89d15d813C6d3278A66BcbD")
	SendTransactionGasLimit     = uint64(21000)
	ToAddress                   = common.HexToAddress("0x799E24dC6B48549BbD1Fc9fcCa4d72880d8c7a15")
	SignLogFile, _              = os.OpenFile(filepath, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
)
