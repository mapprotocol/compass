package xrp

import (
	"github.com/mapprotocol/compass/core"
	"github.com/mapprotocol/compass/internal/chain"
)

type Config struct {
	chain.Config
	Addr string
}

func parseCfg(chainCfg *core.ChainConfig) (*Config, error) {
	cfg, err := chain.ParseConfig(chainCfg)
	if err != nil {
		return nil, err
	}
	ret := Config{
		Config: *cfg,
	}

	if ele, ok := chainCfg.Opts[chain.Addr]; ok && ele != "" {
		ret.Addr = ele
	}
	return &ret, nil
}
