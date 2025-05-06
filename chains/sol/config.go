package sol

import (
	"github.com/mapprotocol/compass/core"
	"github.com/mapprotocol/compass/internal/chain"
	"strings"
)

type Config struct {
	chain.Config
	Pri         string
	MessageIn   string
	LightNode   string
	McsContract []string
	SolEvent    []string
	UsdcAda     string
	WsolAda     string
	ButterHost  string
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
		ButterHost:  chainCfg.ButterHost,
	}

	if ele, ok := chainCfg.Opts[chain.LightNode]; ok && ele != "" {
		ret.LightNode = ele
	}
	if ele, ok := chainCfg.Opts[chain.Private]; ok && ele != "" {
		ret.Pri = ele
	}
	if ele, ok := chainCfg.Opts["messageIn"]; ok && ele != "" {
		ret.MessageIn = ele
	}
	if ele, ok := chainCfg.Opts["usdcAda"]; ok && ele != "" {
		ret.UsdcAda = ele
	}
	if ele, ok := chainCfg.Opts["wsolAda"]; ok && ele != "" {
		ret.WsolAda = ele
	}
	if ele, ok := chainCfg.Opts[chain.McsOpt]; ok && ele != "" {
		for _, addr := range strings.Split(ele, ",") {
			ret.McsContract = append(ret.McsContract, addr)
		}
	}

	if v, ok := chainCfg.Opts[chain.Event]; ok && v != "" {
		ret.SolEvent = append(ret.SolEvent, strings.Split(v, "|")...)
	}

	return &ret, nil
}
