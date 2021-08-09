package chain_structs

import "math/big"

type ChainEnum int

const (
	MapId ChainEnum = 1001
	EthId ChainEnum = 1000
)

type ChainInterface interface {
	GetName() string
	GetChainEnum() ChainEnum
	GetChainId() int
	GetBlockNumber() uint64
	GetRpcUrl() string
	GetBlockHeader(num uint64) []byte
	GetAddress() string
	SetTarget(keystoreStr string, password string)
	SyncBlock(from ChainEnum, Cdata *[]byte)
	ContractInterface
}
type ContractInterface interface {
	Register(value *big.Int) bool
	UnRegister(value *big.Int) bool
	GetRelayerBalance() GetRelayerBalanceResponse
	GetRelayer() GetRelayerResponse
	GetPeriodHeight() GetPeriodHeightResponse
}

var ChainEnum2Instance = map[ChainEnum]ChainInterface{
	MapId: NewEthChain("map_chain_test", 213, MapId, "http://119.8.165.158:7445", 10,
		"0x00000000000052656c6179657241646472657373", "0x000068656164657273746F726541646472657373"),
	EthId: NewEthChain("Ethereum test net", 1, EthId, "http://119.8.165.158:8545", 10,
		"", ""),
}
