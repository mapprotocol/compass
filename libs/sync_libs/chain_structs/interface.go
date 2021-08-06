package chain_structs

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
	SyncBlock(data *[]byte)
}

var ChainEnum2Instance = map[ChainEnum]ChainInterface{
	MapId: NewEthChain("map_chain_test", 1133, MapId, "http://119.8.165.158:7445", 10,
		"0x0", "0x000068656164657273746F726541646472657373"),
	EthId: NewEthChain("Ethereum test net", 1, EthId, "http://119.8.165.158:8545", 10,
		"", ""),
}
