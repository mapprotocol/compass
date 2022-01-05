package libs

import (
	"encoding/json"
	"math/big"
	"os"
	filepath2 "path/filepath"

	"github.com/ethereum/go-ethereum/common"
	"github.com/peterbourgon/diskv"
)

var (
	SendTransactionValue    = big.NewInt(1000000000000000000)
	RpcUrl                  = GetBlockChainMap()[ReadConfigWithCondition("selected_chain", "1", keyInBlockChainMap)].RpcUrl
	EthRpcUrl               = "https://mainnet.infura.io/v3/8a12910e30bb4e4ab04cd8d761a0552d"
	StakingContractAddress  = common.HexToAddress(GetBlockChainMap()[ReadConfigWithCondition("selected_chain", "1", keyInBlockChainMap)].StakingContractAddress)
	DataContractAddress     = common.HexToAddress(GetBlockChainMap()[ReadConfigWithCondition("selected_chain", "1", keyInBlockChainMap)].DataContractAddress)
	ChainId                 = big.NewInt(137)
	SendTransactionGasLimit = uint64(21000)
	ToAddress               = common.HexToAddress("0x799E24dC6B48549BbD1Fc9fcCa4d72880d8c7a15")
	SignLogFile, _          = os.OpenFile(LogFile, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0700)
	RuntimeDirectory        = filepath2.Join(filepath2.Dir(os.Args[0]), "runtime")
	ConfigDirectory         = filepath2.Join(RuntimeDirectory, "config")
	LogFile                 = filepath2.Join(RuntimeDirectory, "sign.log")
	DiskCache               = diskv.New(diskv.Options{
		BasePath:     ConfigDirectory,
		CacheSizeMax: 1024 * 1024,
	})
	blockChainMap         map[string]Chain
	ExternalBlockChainMap map[string]Chain
	internalBlockChainMap = map[string]Chain{"1": {
		"https://polygon-rpc.com/",
		"0x567F5d1Bb4c095E7fD0AC939be1aeb1c661413a8",
		"0x611A4593f4d4BA964E789a2E4681FA97364F73Cf",
	}}
	ExternalBlockChainKey = "externalBlockChain"
)

func GetBlockChainMap() map[string]Chain {
	if blockChainMap != nil {
		return blockChainMap
	}
	blockChainMap = make(map[string]Chain)
	for k, v := range internalBlockChainMap {
		blockChainMap[k] = v
	}

	for k, v := range GetExternalBlockChainMap() {
		blockChainMap[k] = v
	}

	return blockChainMap
}
func GetExternalBlockChainMap() map[string]Chain {
	if ExternalBlockChainMap != nil {
		return ExternalBlockChainMap
	}
	ExternalBlockChainMap = make(map[string]Chain)
	err := json.Unmarshal([]byte(ReadConfig(ExternalBlockChainKey, "[]")), &ExternalBlockChainMap)
	if err != nil {
		return nil
	}
	return ExternalBlockChainMap
}
func keyInBlockChainMap(key string) bool {
	_, ok := GetBlockChainMap()[key]
	return ok
}

type Chain struct {
	RpcUrl                 string
	StakingContractAddress string
	DataContractAddress    string
}
