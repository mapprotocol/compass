package cmd_runtime

import (
	"github.com/mapprotocol/compass/chains"
	"github.com/mapprotocol/compass/chains/ethereum"
)

const (
	MapId chains.ChainEnum = 1000
	EthId chains.ChainEnum = 1003
)

var ChainEnum2Instance = map[chains.ChainEnum]chains.ChainInterface{
	MapId: ethereum.NewEthChain("map_chain_test", 213, MapId, 6, "http://119.8.165.158:7445", 10,
		"0x00000000000052656c6179657241646472657373", "0x000068656164657273746F726541646472657373"),
	EthId: ethereum.NewEthChain("Ethereum test net", 1, EthId, 15, "http://119.8.165.158:8545", 10,
		"", ""),
}
