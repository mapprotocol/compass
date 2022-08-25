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

	gconfig "github.com/mapprotocol/compass/config"
	"github.com/mapprotocol/compass/connections/ethereum/egs"
	"github.com/mapprotocol/compass/core"
	"github.com/mapprotocol/compass/msg"
)

const DefaultGasLimit = 6721975
const DefaultGasPrice = 20000000000
const DefaultBlockConfirmations = 10
const DefaultGasMultiplier = 1

// Chain specific options
var (
	McsOpt                = "mcs"
	RedisOpt              = "redis"
	MaxGasPriceOpt        = "maxGasPrice"
	GasLimitOpt           = "gasLimit"
	GasMultiplier         = "gasMultiplier"
	HttpOpt               = "http"
	StartBlockOpt         = "startBlock"
	BlockConfirmationsOpt = "blockConfirmations"
	EGSApiKey             = "egsApiKey"
	EGSSpeed              = "egsSpeed"
	SyncToMap             = "syncToMap"
	SyncIDList            = "syncIdList"
	LightNode             = "lightnode"
	Event                 = "event"
)

// Config encapsulates all necessary parameters in ethereum compatible forms
type Config struct {
	name               string      // Human-readable chain name
	id                 msg.ChainId // ChainID
	endpoint           string      // url for rpc endpoint
	from               string      // address of key to use
	keystorePath       string      // Location of keyfiles
	blockstorePath     string
	freshStart         bool // Disables loading from blockstore at start
	mcsContract        string
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
		mcsContract:        "",
		gasLimit:           big.NewInt(DefaultGasLimit),
		maxGasPrice:        big.NewInt(DefaultGasPrice),
		gasMultiplier:      big.NewFloat(DefaultGasMultiplier),
		http:               false,
		startBlock:         big.NewInt(0),
		blockConfirmations: big.NewInt(0),
		egsApiKey:          "",
		egsSpeed:           "",
		redisUrl:           "",
	}

	if contract, ok := chainCfg.Opts[McsOpt]; ok && contract != "" {
		config.mcsContract = contract
		delete(chainCfg.Opts, McsOpt)
	} else {
		return nil, fmt.Errorf("must provide opts.bridge field for ethereum config")
	}

	if v, ok := chainCfg.Opts[RedisOpt]; ok && v != "" {
		config.redisUrl = v
		delete(chainCfg.Opts, RedisOpt)
	}

	if gasPrice, ok := chainCfg.Opts[MaxGasPriceOpt]; ok {
		price := big.NewInt(0)
		_, pass := price.SetString(gasPrice, 10)
		if pass {
			config.maxGasPrice = price
			delete(chainCfg.Opts, MaxGasPriceOpt)
		} else {
			return nil, errors.New("unable to parse max gas price")
		}
	}

	if gasLimit, ok := chainCfg.Opts[GasLimitOpt]; ok {
		limit := big.NewInt(0)
		_, pass := limit.SetString(gasLimit, 10)
		if pass {
			config.gasLimit = limit
			delete(chainCfg.Opts, GasLimitOpt)
		} else {
			return nil, errors.New("unable to parse gas limit")
		}
	}

	if gasMultiplier, ok := chainCfg.Opts[GasMultiplier]; ok {
		multilier := big.NewFloat(1)
		_, pass := multilier.SetString(gasMultiplier)
		if pass {
			config.gasMultiplier = multilier
			delete(chainCfg.Opts, GasMultiplier)
		} else {
			return nil, errors.New("unable to parse gasMultiplier to float")
		}
	}

	if HTTP, ok := chainCfg.Opts[HttpOpt]; ok && HTTP == "true" {
		config.http = true
		delete(chainCfg.Opts, HttpOpt)
	} else if HTTP, ok := chainCfg.Opts[HttpOpt]; ok && HTTP == "false" {
		config.http = false
		delete(chainCfg.Opts, HttpOpt)
	}

	if startBlock, ok := chainCfg.Opts[StartBlockOpt]; ok && startBlock != "" {
		block := big.NewInt(0)
		_, pass := block.SetString(startBlock, 10)
		if pass {
			config.startBlock = block
			delete(chainCfg.Opts, StartBlockOpt)
		} else {
			return nil, fmt.Errorf("unable to parse %s", StartBlockOpt)
		}
	}

	if blockConfirmations, ok := chainCfg.Opts[BlockConfirmationsOpt]; ok && blockConfirmations != "" {
		val := big.NewInt(DefaultBlockConfirmations)
		_, pass := val.SetString(blockConfirmations, 10)
		if pass {
			config.blockConfirmations = val
			delete(chainCfg.Opts, BlockConfirmationsOpt)
		} else {
			return nil, fmt.Errorf("unable to parse %s", BlockConfirmationsOpt)
		}
	} else {
		config.blockConfirmations = big.NewInt(DefaultBlockConfirmations)
		delete(chainCfg.Opts, BlockConfirmationsOpt)
	}

	if gsnApiKey, ok := chainCfg.Opts[EGSApiKey]; ok && gsnApiKey != "" {
		config.egsApiKey = gsnApiKey
		delete(chainCfg.Opts, EGSApiKey)
	}

	if speed, ok := chainCfg.Opts[EGSSpeed]; ok && speed == egs.Average || speed == egs.Fast || speed == egs.Fastest {
		config.egsSpeed = speed
		delete(chainCfg.Opts, EGSSpeed)
	} else {
		// Default to "fast"
		config.egsSpeed = egs.Fast
		delete(chainCfg.Opts, EGSSpeed)
	}

	if syncToMap, ok := chainCfg.Opts[SyncToMap]; ok && syncToMap == "true" {
		config.syncToMap = true
		delete(chainCfg.Opts, SyncToMap)
	} else {
		config.syncToMap = false
		delete(chainCfg.Opts, SyncToMap)
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

	if syncIDList, ok := chainCfg.Opts[SyncIDList]; ok && syncIDList != "[]" {
		err := json.Unmarshal([]byte(syncIDList), &config.syncChainIDList)
		if err != nil {
			return nil, err
		}
		delete(chainCfg.Opts, SyncIDList)
	}

	if lightNode, ok := chainCfg.Opts[LightNode]; ok && lightNode != "" {
		config.lightNode = lightNode
		delete(chainCfg.Opts, LightNode)
	}

	if v, ok := chainCfg.Opts[Event]; ok && v != "" {
		vs := strings.Split(v, "|")
		for _, s := range vs {
			config.events = append(config.events, s)
		}
		delete(chainCfg.Opts, Event)
	}

	if len(chainCfg.Opts) != 0 {
		return nil, fmt.Errorf("unknown Opts Encountered: %#v", chainCfg.Opts)
	}

	return config, nil
}
