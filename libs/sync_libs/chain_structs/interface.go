package chain_structs

type ChainId int

const (
	MapId ChainId = 1
)

type mapChain interface {
	GetChainId() ChainId
	GetBlockNumber() uint64
	GetRpcUrl() string
	GetBlockHeader(num uint64) []byte
}

var ChainId2Instance = map[ChainId]mapChain{
	MapId: NewEthChain(MapId, "http://127.0.0.1:7545", 10,
		"0x0", "0x0"),
}
