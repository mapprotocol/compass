package ton

import (
	"github.com/mapprotocol/compass/core"
	"github.com/mapprotocol/compass/internal/chain"
)

type Config struct {
	chain.Config
	LightNode   string
	McsContract []string // EQCW_VWZ1LLI8nFoNk0dI62sNP0AGAy3ENSAaRb6hh2A1Pim
}

func parseCfg(chainCfg *core.ChainConfig) (*Config, error) { // todo ton 自己解析配置(主要关注自己的地址)
	cfg, err := chain.ParseConfig(chainCfg)
	if err != nil {
		return nil, err
	}
	ret := Config{
		Config:      *cfg,
		LightNode:   "",
		McsContract: nil,
	}

	return &ret, nil
}
