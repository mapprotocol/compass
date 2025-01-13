package expose

import (
	"encoding/json"
	"fmt"
	"github.com/mapprotocol/compass/config"
	"github.com/mapprotocol/compass/internal/constant"
	"github.com/urfave/cli/v2"
	"os"
	"path/filepath"
)

const (
	DefaultConfigPath = "./config.json"
)

type Config struct {
	Chains []RawChainConfig `json:"chains"`
	Other  Construction     `json:"other,omitempty"`
}

type RawChainConfig struct {
	Name       string `json:"name"`
	Type       string `json:"type"`
	Id         string `json:"id"`       // ChainID
	Endpoint   string `json:"endpoint"` // url for rpc endpoint
	Mcs        string `json:"mcs,omitempty"`
	OracleNode string `json:"oracleNode,omitempty"`
}

type Construction struct {
	MonitorUrl string `json:"monitor_url,omitempty"`
	Env        string `json:"env,omitempty"`
	Port       string `json:"port,omitempty"`
	Butter     string `json:"butter,omitempty"`
}

func (c *Config) validate() error {
	for idx, chain := range c.Chains {
		if chain.Id == "" {
			return fmt.Errorf("required field chain.Id empty for chain %s", chain.Id)
		}
		if chain.Type == "" {
			c.Chains[idx].Type = constant.Ethereum
		}
		if chain.Name == "" {
			return fmt.Errorf("required field chain.Name empty for chain %s", chain.Id)
		}
	}
	return nil
}

func Local(ctx *cli.Context) (*Config, error) {
	var fig Config
	path := DefaultConfigPath
	if ctx.String(config.ConfigFileFlag.Name) != "" {
		path = ctx.String(config.ConfigFileFlag.Name)
	}

	err := loadConfig(path, &fig)
	if err != nil {
		return &fig, err
	}

	err = fig.validate()
	if err != nil {
		return nil, err
	}
	return &fig, nil
}

func loadConfig(file string, config *Config) error {
	ext := filepath.Ext(file)
	fp, err := filepath.Abs(file)
	if err != nil {
		return err
	}

	f, err := os.Open(filepath.Clean(fp))
	if err != nil {
		return err
	}

	if ext == ".json" {
		if err = json.NewDecoder(f).Decode(&config); err != nil {
			return err
		}
	} else {
		return fmt.Errorf("unrecognized extention: %s", ext)
	}

	return nil
}
