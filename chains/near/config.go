package near

import (
	"fmt"
	"strings"

	"github.com/mapprotocol/compass/internal/chain"

	"github.com/mapprotocol/compass/core"
)

type Config struct {
	chain.Config
	lightNode, redisUrl string
	events, mcsContract []string
}

// parseChainConfig uses a core.ChainConfig to construct a corresponding Config
func parseChainConfig(chainCfg *core.ChainConfig) (*Config, error) {
	cfg, err := chain.ParseConfig(chainCfg)
	if err != nil {
		return nil, err
	}
	ret := Config{
		Config:      *cfg,
		lightNode:   "",
		mcsContract: nil,
	}

	if v, ok := chainCfg.Opts[chain.Event]; ok && v != "" {
		vs := strings.Split(v, "|")
		for _, s := range vs {
			ret.events = append(ret.events, s)
		}
	}

	if contract, ok := chainCfg.Opts[chain.McsOpt]; ok && contract != "" {
		for _, addr := range strings.Split(contract, ",") {
			ret.mcsContract = append(ret.mcsContract, addr)
		}
	} else {
		return nil, fmt.Errorf("must provide opts.mcs field for ethereum config")
	}

	if v, ok := chainCfg.Opts[chain.RedisOpt]; ok && v != "" {
		ret.redisUrl = v
		delete(chainCfg.Opts, chain.RedisOpt)
	}

	return &ret, nil
}
