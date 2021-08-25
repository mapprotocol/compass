package cmd_runtime

import (
	"github.com/mapprotocol/compass/chains"
	"github.com/pelletier/go-toml"
	log "github.com/sirupsen/logrus"
	"os"
	"path/filepath"
)

type GlobalConfig struct {
	Keystore                string `toml:"keystore"`
	Password                string `toml:"password"`
	BlockNumberByEstimation bool   `toml:"block_number_by_estimation"`
}
type ChainConfig struct {
	Name                       string         `toml:"name"`
	ChainId                    chains.ChainId `toml:"chain_id"`
	BlockCreatingTime          int            `toml:"block_creating_seconds"`
	RpcUrl                     string         `toml:"rpc_url"`
	StableBlock                uint64         `toml:"stable_block"`
	RelayerContractAddress     string         `toml:"relayer_contract_address"`
	HeaderStoreContractAddress string         `toml:"header_store_contract_address"`
}

func ReadTomlConfig() (globalConfig GlobalConfig, srcChainConfig ChainConfig, dstChainConfig ChainConfig) {
	rootTree, err := toml.LoadFile(filepath.Join(filepath.Dir(os.Args[0]), "config.toml"))
	rootTree, err = toml.LoadFile("/Users/yangdianqing/code/go/compass/config.toml") // for dev
	if err != nil {
		log.Fatalln(err)
	}
	_ = parseKey("global", rootTree, func() {
		log.Fatal("Config.toml does not contain global block")
	}).(*toml.Tree).Unmarshal(&globalConfig)

	_ = parseKey("src_chain", rootTree, func() {
		log.Fatal("Config.toml does not contain src_chain block")
	}).(*toml.Tree).Unmarshal(&srcChainConfig)

	_ = parseKey("dst_chain", rootTree, func() {
		log.Fatal("Config.toml does not contain dst_chain block")
	}).(*toml.Tree).Unmarshal(&dstChainConfig)

	if srcChainConfig.ChainId <= 0 || dstChainConfig.ChainId <= 0 {
		log.Fatal("chain_id is required, it has to be a natural number.")
	}
	if srcChainConfig.BlockCreatingTime <= 0 || dstChainConfig.BlockCreatingTime <= 0 {
		log.Fatal("block_creating_seconds is required, it has to be a natural number.")
	}
	if srcChainConfig.StableBlock <= 0 || dstChainConfig.StableBlock <= 0 {
		log.Fatal("stable_block is required, it has to be a natural number.")
	}
	return
}
func parseKey(key string, tree *toml.Tree, f func()) interface{} {
	v := tree.Get(key)
	if v == nil {
		f()
	}
	return v
}
