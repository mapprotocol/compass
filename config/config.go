package config

import (
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	"os"
	"path/filepath"

	"github.com/ethereum/go-ethereum/log"
	"github.com/urfave/cli/v2"
)

const DefaultConfigPath = "./config.json"
const DefaultKeystorePath = "./keys"
const MapChainID = "mapChainId"

type Config struct {
	MapChain RawChainConfig   `json:"mapchain"`
	Chains   []RawChainConfig `json:"chains"`
	Other    Construction     `json:"other,omitempty"`
}

// RawChainConfig is parsed directly from the config file and should be using to construct the core.ChainConfig
type RawChainConfig struct {
	Name         string            `json:"name"`
	Type         string            `json:"type"`
	Id           string            `json:"id"`       // ChainID
	Endpoint     string            `json:"endpoint"` // url for rpc endpoint
	From         string            `json:"from"`     // address of key to use
	Network      string            `json:"network"`
	KeystorePath string            `json:"keystorePath"`
	Opts         map[string]string `json:"opts"`
}

type Construction struct {
	MonitorUrl   string `json:"monitor_url,omitempty"`
	Env          string `json:"env,omitempty"`
	BlackListUrl string `json:"black_list_url"`
	Filter       string `json:"filter"`
	BtcUrl       string `json:"btc_url"`
	Butter       string `json:"butter"`
}

func (c *Config) ToJSON(file string) *os.File {
	var (
		newFile *os.File
		err     error
	)

	var raw []byte
	if raw, err = json.Marshal(*c); err != nil {
		log.Warn("error marshalling json", "err", err)
		os.Exit(1)
	}

	newFile, err = os.Create(file)
	if err != nil {
		log.Warn("error creating config file", "err", err)
	}
	_, err = newFile.Write(raw)
	if err != nil {
		log.Warn("error writing to config file", "err", err)
	}

	if err := newFile.Close(); err != nil {
		log.Warn("error closing file", "err", err)
	}
	return newFile
}

func (c *Config) validate() error {
	for _, chain := range c.Chains {
		if chain.Id == "" {
			return fmt.Errorf("required field chain.Id empty for chain %s", chain.Id)
		}
		if chain.Type == "" {
			return fmt.Errorf("required field chain.Type empty for chain %s", chain.Id)
		}
		if chain.Endpoint == "" {
			return fmt.Errorf("required field chain.Endpoint empty for chain %s", chain.Id)
		}
		if chain.Name == "" {
			return fmt.Errorf("required field chain.Name empty for chain %s", chain.Id)
		}
		if chain.From == "" {
			return fmt.Errorf("required field chain.From empty for chain %s", chain.Id)
		}
	}
	// check map chain
	if c.MapChain.Id == "" {
		return fmt.Errorf("required field chain.Id empty for chain %s", c.MapChain.Id)
	}
	if c.MapChain.Endpoint == "" {
		return fmt.Errorf("required field mapchain.Endpoint empty for chain %s", c.MapChain.Id)
	}
	if c.MapChain.From == "" {
		return fmt.Errorf("required field chain.From empty for chain %s", c.MapChain.Id)
	}

	return nil
}

func GetConfig(ctx *cli.Context) (*Config, error) {
	var fig Config
	path := DefaultConfigPath
	if file := ctx.String(ConfigFileFlag.Name); file != "" {
		path = file
	}
	err := loadConfig(path, &fig)
	if err != nil {
		log.Warn("err loading json file", "err", err.Error())
		return &fig, err
	}
	log.Debug("Loaded config", "path", path)
	err = fig.validate()
	// fill map chain config
	fig.MapChain.Type = "ethereum"
	fig.MapChain.Name = "map"
	if ctx.Bool(FilterFlag.Name) && fig.Other.Filter == "" {
		return nil, errors.New("filter url is empty")
	}

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

	log.Debug("Loading configuration", "path", filepath.Clean(fp))

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
