package cmd_runtime

import (
	"github.com/pelletier/go-toml"
	log "github.com/sirupsen/logrus"
	"os"
	"path/filepath"
)

func ReadTomlConfig() {
	rootTree, err := toml.LoadFile(filepath.Join(filepath.Dir(os.Args[0]), "config.toml"))
	rootTree, err = toml.LoadFile("/Users/yangdianqing/code/go/compass/config.toml") // for dev
	if err != nil {
		log.Fatalln(err)
	}
	_ = parseKey("global", rootTree, func() {
		log.Fatal("Config.toml does not contain global block")
	}).(*toml.Tree).Unmarshal(&GlobalConfigV)
	if GlobalConfigV.StartWithBlock <= 0 {
		GlobalConfigV.StartWithBlock = 1
	}

	if GlobalConfigV.BlockNumberLimitOnce == 0 {
		GlobalConfigV.BlockNumberLimitOnce = 1
	}
	if GlobalConfigV.BlockNumberLimitOnce > 20 {
		GlobalConfigV.BlockNumberLimitOnce = 20
	}
	_ = parseKey("src_chain", rootTree, func() {
		log.Fatal("Config.toml does not contain src_chain block")
	}).(*toml.Tree).Unmarshal(&SrcChainConfig)

	_ = parseKey("dst_chain", rootTree, func() {
		log.Fatal("Config.toml does not contain dst_chain block")
	}).(*toml.Tree).Unmarshal(&DstChainConfig)

	if SrcChainConfig.ChainId <= 0 || DstChainConfig.ChainId <= 0 {
		log.Fatal("chain_id is required, it has to be a natural number.")
	}
	if SrcChainConfig.BlockCreatingTime <= 0 || DstChainConfig.BlockCreatingTime <= 0 {
		log.Fatal("block_creating_seconds is required, it has to be a natural number.")
	}
	if SrcChainConfig.StableBlock <= 0 || DstChainConfig.StableBlock <= 0 {
		log.Fatal("stable_block is required, it has to be a natural number.")
	}
	if SrcChainConfig.RouterContractAddress == "" || DstChainConfig.RouterContractAddress == "" {
		log.Fatal("router_contract_address is required.")
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
