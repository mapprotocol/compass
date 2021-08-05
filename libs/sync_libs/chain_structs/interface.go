package chain_structs

type ChainId int

const (
	MapId ChainId = 1
	EthId ChainId = 2
)

type mapChain interface {
	GetName() string
	GetChainId() ChainId
	GetBlockNumber() uint64
	GetRpcUrl() string
	GetBlockHeader(num uint64) []byte
}

var ChainId2Instance = map[ChainId]mapChain{
	MapId: NewEthChain("map chain", MapId, "http://127.0.0.1:7545", 10,
		"0x0", "0x0"),
	EthId: NewEthChain("Ethereum main net", MapId, "https://mainnet.infura.io/v3/9aa3d95b3bc440fa88ea12eaa4456161", 10,
		"0x0", "0x0"),
}
