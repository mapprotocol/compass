// Copyright 2021 Compass Systems
// SPDX-License-Identifier: LGPL-3.0-only

package near

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"strconv"
	"strings"

	"github.com/mapprotocol/compass/internal/chain"

	gconfig "github.com/mapprotocol/compass/config"
	"github.com/mapprotocol/compass/connections/ethereum/egs"
	"github.com/mapprotocol/compass/core"
	"github.com/mapprotocol/compass/msg"
)

type Config struct {
	name               string      // Human-readable chain name
	id                 msg.ChainId // ChainID
	endpoint           string      // url for rpc endpoint
	from               string      // address of key to use
	keystorePath       string      // Location of keyfiles
	blockstorePath     string
	freshStart         bool // Disables loading from blockstore at start
	mcsContract        []string
	gasLimit           *big.Int
	maxGasPrice        *big.Int
	gasMultiplier      *big.Float
	http               bool // Config for type of connection
	startBlock         *big.Int
	blockConfirmations *big.Int
	egsApiKey          string // API key for ethgasstation to query gas prices
	egsSpeed           string // The speed which a transaction should be processed: average, fast, fastest. Default: fast
	syncToMap          bool   // Whether sync blockchain headers to Map
	mapChainID         msg.ChainId
	syncChainIDList    []msg.ChainId // chain ids which map sync to
	lightNode          string        // the lightnode to sync header
	redisUrl           string
	events             []string
	skipError          bool
}

// parseChainConfig uses a core.ChainConfig to construct a corresponding Config
func parseChainConfig(chainCfg *core.ChainConfig) (*Config, error) {
	config := &Config{
		name:               chainCfg.Name,
		id:                 chainCfg.Id,
		from:               chainCfg.From,
		keystorePath:       chainCfg.NearKeystorePath,
		blockstorePath:     chainCfg.BlockstorePath,
		freshStart:         chainCfg.FreshStart,
		endpoint:           chainCfg.Endpoint,
		mcsContract:        []string{},
		gasLimit:           big.NewInt(chain.DefaultGasLimit),
		maxGasPrice:        big.NewInt(chain.DefaultGasPrice),
		gasMultiplier:      big.NewFloat(chain.DefaultGasMultiplier),
		http:               false,
		startBlock:         big.NewInt(0),
		blockConfirmations: big.NewInt(0),
		egsApiKey:          "",
		egsSpeed:           "",
		redisUrl:           "",
		skipError:          chainCfg.SkipError,
	}

	if contract, ok := chainCfg.Opts[chain.McsOpt]; ok && contract != "" {
		for _, addr := range strings.Split(contract, ",") {
			config.mcsContract = append(config.mcsContract, addr)
		}

		delete(chainCfg.Opts, chain.McsOpt)
	} else {
		return nil, fmt.Errorf("must provide opts.mcs field for ethereum config")
	}

	if v, ok := chainCfg.Opts[chain.RedisOpt]; ok && v != "" {
		config.redisUrl = v
		delete(chainCfg.Opts, chain.RedisOpt)
	}

	if gasPrice, ok := chainCfg.Opts[chain.MaxGasPriceOpt]; ok {
		price := big.NewInt(0)
		_, pass := price.SetString(gasPrice, 10)
		if pass {
			config.maxGasPrice = price
			delete(chainCfg.Opts, chain.MaxGasPriceOpt)
		} else {
			return nil, errors.New("unable to parse max gas price")
		}
	}

	if gasLimit, ok := chainCfg.Opts[chain.GasLimitOpt]; ok {
		limit := big.NewInt(0)
		_, pass := limit.SetString(gasLimit, 10)
		if pass {
			config.gasLimit = limit
			delete(chainCfg.Opts, chain.GasLimitOpt)
		} else {
			return nil, errors.New("unable to parse gas limit")
		}
	}

	if gasMultiplier, ok := chainCfg.Opts[chain.GasMultiplier]; ok {
		multilier := big.NewFloat(1)
		_, pass := multilier.SetString(gasMultiplier)
		if pass {
			config.gasMultiplier = multilier
			delete(chainCfg.Opts, chain.GasMultiplier)
		} else {
			return nil, errors.New("unable to parse gasMultiplier to float")
		}
	}

	if HTTP, ok := chainCfg.Opts[chain.HttpOpt]; ok && HTTP == "true" {
		config.http = true
		delete(chainCfg.Opts, chain.HttpOpt)
	} else if HTTP, ok := chainCfg.Opts[chain.HttpOpt]; ok && HTTP == "false" {
		config.http = false
		delete(chainCfg.Opts, chain.HttpOpt)
	}

	if startBlock, ok := chainCfg.Opts[chain.StartBlockOpt]; ok && startBlock != "" {
		block := big.NewInt(0)
		_, pass := block.SetString(startBlock, 10)
		if pass {
			config.startBlock = block
			delete(chainCfg.Opts, chain.StartBlockOpt)
		} else {
			return nil, fmt.Errorf("unable to parse %s", chain.StartBlockOpt)
		}
	}

	if blockConfirmations, ok := chainCfg.Opts[chain.BlockConfirmationsOpt]; ok && blockConfirmations != "" {
		val := big.NewInt(chain.DefaultBlockConfirmations)
		_, pass := val.SetString(blockConfirmations, 10)
		if pass {
			config.blockConfirmations = val
			delete(chainCfg.Opts, chain.BlockConfirmationsOpt)
		} else {
			return nil, fmt.Errorf("unable to parse %s", chain.BlockConfirmationsOpt)
		}
	} else {
		config.blockConfirmations = big.NewInt(chain.DefaultBlockConfirmations)
		delete(chainCfg.Opts, chain.BlockConfirmationsOpt)
	}

	if gsnApiKey, ok := chainCfg.Opts[chain.EGSApiKey]; ok && gsnApiKey != "" {
		config.egsApiKey = gsnApiKey
		delete(chainCfg.Opts, chain.EGSApiKey)
	}

	if speed, ok := chainCfg.Opts[chain.EGSSpeed]; ok && speed == egs.Average || speed == egs.Fast || speed == egs.Fastest {
		config.egsSpeed = speed
		delete(chainCfg.Opts, chain.EGSSpeed)
	} else {
		// Default to "fast"
		config.egsSpeed = egs.Fast
		delete(chainCfg.Opts, chain.EGSSpeed)
	}

	if syncToMap, ok := chainCfg.Opts[chain.SyncToMap]; ok && syncToMap == "true" {
		config.syncToMap = true
		delete(chainCfg.Opts, chain.SyncToMap)
	} else {
		config.syncToMap = false
		delete(chainCfg.Opts, chain.SyncToMap)
	}

	if mapChainID, ok := chainCfg.Opts[gconfig.MapChainID]; ok {
		// key exist anyway
		chainId, errr := strconv.Atoi(mapChainID)
		if errr != nil {
			return nil, errr
		}
		config.mapChainID = msg.ChainId(chainId)
		delete(chainCfg.Opts, gconfig.MapChainID)
	}

	if syncIDList, ok := chainCfg.Opts[chain.SyncIDList]; ok && syncIDList != "[]" {
		err := json.Unmarshal([]byte(syncIDList), &config.syncChainIDList)
		if err != nil {
			return nil, err
		}
		delete(chainCfg.Opts, chain.SyncIDList)
	}

	if lightNode, ok := chainCfg.Opts[chain.LightNode]; ok && lightNode != "" {
		config.lightNode = lightNode
		delete(chainCfg.Opts, chain.LightNode)
	}

	if v, ok := chainCfg.Opts[chain.Event]; ok && v != "" {
		vs := strings.Split(v, "|")
		for _, s := range vs {
			config.events = append(config.events, s)
		}
		delete(chainCfg.Opts, chain.Event)
	}

	return config, nil
}
