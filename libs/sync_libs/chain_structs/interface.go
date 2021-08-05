package chain_structs

type ChainEnum int

const (
	MapId ChainEnum = 1
	EthId ChainEnum = 2
)

type MapChain interface {
	GetName() string
	GetChainEnum() ChainEnum
	GetChainId() int
	GetBlockNumber() uint64
	GetRpcUrl() string
	GetBlockHeader(num uint64) []byte
	GetAddress() string
	SetTarget(keystoreStr string, password string)
}

var ChainEnum2Instance = map[ChainEnum]MapChain{
	MapId: NewEthChain("map chain", 1133, MapId, "http://127.0.0.1:7545", 10,
		"0x0", "0x0"),
	EthId: NewEthChain("Ethereum main net", 1, MapId, "https://mainnet.infura.io/v3/9aa3d95b3bc440fa88ea12eaa4456161", 10,
		"0x0", "0x0"),
}
