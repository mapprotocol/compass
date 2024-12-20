package ton

import (
	"github.com/mapprotocol/compass/core"
	"github.com/mapprotocol/compass/internal/chain"
	"strings"
)

type Config struct {
	chain.Config
	Words       string
	McsContract []string
	Event       []string
}

func parseConfig(chainCfg *core.ChainConfig) (*Config, error) {
	cfg, err := chain.ParseConfig(chainCfg)
	if err != nil {
		return nil, err
	}
	ret := Config{
		Config: *cfg,
	}

	if words, ok := chainCfg.Opts[chain.Words]; ok && words != "" {
		ret.Words = words
	}

	if ele, ok := chainCfg.Opts[chain.McsOpt]; ok && ele != "" {
		for _, addr := range strings.Split(ele, ",") {
			ret.McsContract = append(ret.McsContract, addr)
		}
	}

	if ele, ok := chainCfg.Opts[chain.Event]; ok && ele != "" {
		for _, event := range strings.Split(ele, "|") {
			ret.Event = append(ret.Event, event)
		}
	}

	return &ret, nil
}
