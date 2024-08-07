package tron

import (
	"github.com/ethereum/go-ethereum/common"
	"strings"

	"github.com/mapprotocol/compass/core"
	"github.com/mapprotocol/compass/internal/chain"
)

type Config struct {
	chain.Config
	LightNode, RentNode, FeeKey, FeeType string
	EthFrom                              common.Address
	McsContract                          []string
}

func parseCfg(chainCfg *core.ChainConfig) (*Config, error) {
	cfg, err := chain.ParseConfig(chainCfg)
	if err != nil {
		return nil, err
	}
	ret := Config{
		Config:      *cfg,
		LightNode:   "",
		McsContract: nil,
	}

	if ele, ok := chainCfg.Opts[chain.LightNode]; ok && ele != "" {
		ret.LightNode = ele
	}
	if ele, ok := chainCfg.Opts[chain.McsOpt]; ok && ele != "" {
		for _, addr := range strings.Split(ele, ",") {
			ret.McsContract = append(ret.McsContract, addr)
		}
	}

	if ele, ok := chainCfg.Opts[chain.RentNode]; ok && ele != "" {
		ret.RentNode = ele
	}
	if ele, ok := chainCfg.Opts[chain.EthFrom]; ok && ele != "" {
		ret.EthFrom = common.HexToAddress(ele)
	}
	if ele, ok := chainCfg.Opts[chain.FeeKey]; ok && ele != "" {
		ret.FeeKey = ele
	}
	if ele, ok := chainCfg.Opts[chain.FeeType]; ok && ele != "" {
		ret.FeeType = ele
	}
	return &ret, nil
}
