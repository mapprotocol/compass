package main

import (
	"errors"
	"os"
	"strconv"
	"strings"

	log "github.com/ChainSafe/log15"
	"github.com/ethereum/go-ethereum/common"
	"github.com/mapprotocol/compass/chains"
	"github.com/mapprotocol/compass/config"
	"github.com/mapprotocol/compass/core"
	"github.com/mapprotocol/compass/internal/blacklist"
	"github.com/mapprotocol/compass/internal/butter"
	chain2 "github.com/mapprotocol/compass/internal/chain"
	"github.com/mapprotocol/compass/internal/contract"
	"github.com/mapprotocol/compass/internal/mapprotocol"
	"github.com/mapprotocol/compass/internal/observability"
	"github.com/mapprotocol/compass/internal/report"
	"github.com/mapprotocol/compass/pkg/abi"
	contract2 "github.com/mapprotocol/compass/pkg/contract"
	"github.com/mapprotocol/compass/pkg/msg"
	"github.com/mapprotocol/compass/pkg/util"
	"github.com/urfave/cli/v2"
)

var app = cli.NewApp()

var cliFlags = []cli.Flag{
	config.ConfigFileFlag,
	config.VerbosityFlag,
	config.KeystorePathFlag,
	config.KeyPathFlag,
	config.BlockstorePathFlag,
	config.FreshStartFlag,
	config.LatestBlockFlag,
	config.StartLatestFlag,
	config.SkipErrorFlag,
	config.FilterFlag,
	config.FilterAPIKeyFlag,
	config.ButterAPIKeyFlag,
	config.OnlySpecialTokenFlag,
}

var devFlags = []cli.Flag{
	config.TestKeyFlag,
}

var maintainerCommand = cli.Command{
	Name:  "maintainer",
	Usage: "manage maintainer operations",
	Description: "The maintainer command is used to manage maintainer on Map chain.\n" +
		"\tTo register an account : compass relayers register --account '0x0...'",
	Action:      maintainer,
	Subcommands: []*cli.Command{},
	Flags:       append(app.Flags, cliFlags...),
}

var messengerCommand = cli.Command{
	Name:        "messenger",
	Usage:       "manage messenger operations",
	Description: "The messenger command is used to sync the log information of transactions in the block",
	Action:      messenger,
	Flags:       append(app.Flags, cliFlags...),
}

var oracleCommand = cli.Command{
	Name:        "oracle",
	Usage:       "manage oracle operations",
	Description: "The oracle command is used to sync the log information of transactions in the block",
	Action:      oracle,
	Flags:       append(app.Flags, cliFlags...),
}

var (
	Version = "1.3.0"
)

// init initializes CLI
func init() {
	//app.Action = run
	app.Copyright = "Copyright 2021 MAP Protocol 2021 Authors"
	app.Name = "compass"
	app.Usage = "Compass"
	app.Authors = []*cli.Author{{Name: "MAP Protocol 2021"}}
	app.Version = Version
	app.EnableBashCompletion = true
	app.Commands = []*cli.Command{
		&maintainerCommand,
		&messengerCommand,
		&oracleCommand,
		&exposeCommand,
		&swapFailedCommand,
	}

	app.Flags = append(app.Flags, cliFlags...)
	app.Flags = append(app.Flags, devFlags...)
}

func main() {
	if err := app.Run(os.Args); err != nil {
		log.Error(err.Error())
		os.Exit(1)
	}
}

func startLogger(ctx *cli.Context) error {
	logger := log.Root()
	handler := logger.GetHandler()
	var lvl log.Lvl

	if lvlToInt, err := strconv.Atoi(ctx.String(config.VerbosityFlag.Name)); err == nil {
		lvl = log.Lvl(lvlToInt)
	} else if lvl, err = log.LvlFromString(ctx.String(config.VerbosityFlag.Name)); err != nil {
		return err
	}
	log.Root().SetHandler(log.LvlFilterHandler(lvl, handler))

	return nil
}

func maintainer(ctx *cli.Context) error {
	return run(ctx, mapprotocol.RoleOfMaintainer)
}

func messenger(ctx *cli.Context) error {
	return run(ctx, mapprotocol.RoleOfMessenger)
}

func oracle(ctx *cli.Context) error {
	return run(ctx, mapprotocol.RoleOfOracle)
}

func run(ctx *cli.Context, role mapprotocol.Role) error {
	err := startLogger(ctx)
	if err != nil {
		return err
	}
	log.Info("Starting Compass...")

	cfg, err := config.GetConfig(ctx)
	if err != nil {
		return err
	}
	blacklist.Init(cfg.Other.BlackListUrl)
	butter.Init(cfg.Other.ButterAPIKey)
	util.Init(cfg.Other.Env, cfg.Other.MonitorUrl)
	report.Init(cfg.Other.ReportUrl)

	// Stand up observability (metrics + /status + pprof + alarms) before
	// any chain goroutines start so the first tick can already publish.
	obsAddr := ":9102"
	if cfg.Other.ObservabilityAddr != "" {
		obsAddr = cfg.Other.ObservabilityAddr
	}
	obs := observability.New("compass", observability.Config{
		Addr:    obsAddr,
		AlarmFn: util.Alarm,
	})
	observability.SetDefault(obs)
	obs.StartHTTP()
	obs.StartBlockLagAlarms(observability.DefaultBlockLagRule())
	log.Info("Observability HTTP serving", "addr", obsAddr,
		"endpoints", "/metrics /status /healthz /debug/pprof/")
	defer obs.Stop()

	sysErr := make(chan error)
	mapcid, err := strconv.Atoi(cfg.MapChain.Id)
	if err != nil {
		return err
	}
	c := core.NewCore(sysErr, msg.ChainId(mapcid), role)
	// merge map chain
	filterAPIKey := filterAPIKeyFromConfig(ctx, cfg)
	log.Info("Filter API auth configured", "enabled", filterAPIKey != "")
	butter.SetAPIKey(butterAPIKeyFromConfig(ctx, cfg))

	allChains := make([]config.RawChainConfig, 0, len(cfg.Chains)+1)
	allChains = append(allChains, cfg.MapChain)
	allChains = append(allChains, cfg.Chains...)

	for idx, ele := range allChains {
		ks := ele.KeystorePath
		if ks == "" {
			ks = ctx.String(config.KeyPathFlag.Name)
		}
		chainId, err := strconv.Atoi(ele.Id)
		if err != nil {
			return err
		}
		mapprotocol.MapId = cfg.MapChain.Id
		ele.Opts[config.MapChainID] = cfg.MapChain.Id
		chainConfig := &core.ChainConfig{
			Name:             ele.Name,
			Id:               msg.ChainId(chainId),
			Endpoint:         ele.Endpoint,
			From:             ele.From,
			Network:          ele.Network,
			KeystorePath:     ks,
			NearKeystorePath: ele.KeystorePath,
			BlockstorePath:   ctx.String(config.BlockstorePathFlag.Name),
			FreshStart:       ctx.Bool(config.FreshStartFlag.Name),
			LatestBlock:      ctx.Bool(config.LatestBlockFlag.Name),
			StartLatest:      ctx.Bool(config.StartLatestFlag.Name),
			Opts:             ele.Opts,
			SkipError:        ctx.Bool(config.SkipErrorFlag.Name),
			Filter:           ctx.Bool(config.FilterFlag.Name),
			OnlySpecialToken: ctx.Bool(config.OnlySpecialTokenFlag.Name),
			FilterHost:       cfg.Other.Filter,
			FilterAPIKey:     filterAPIKey,
			BtcHost:          cfg.Other.BtcUrl,
			ButterHost:       cfg.Other.Butter,
			PriceHost:        cfg.Other.Price,
			ReportHost:       cfg.Other.ReportUrl,
		}
		var (
			newChain core.Chain
		)

		logger := log.Root().New("ele", chainConfig.Name)
		creator, ok := chains.Create(ele.Type)
		if !ok {
			return errors.New("unrecognized Chain Type")
		}

		newChain, err = creator.New(chainConfig, logger, sysErr, role)
		if err != nil {
			return err
		}

		if idx == 0 {
			mapprotocol.GlobalMapConn = newChain.(*chain2.Chain).EthClient()
			validateAbi, err := abi.New(mapprotocol.ValidateJson)
			if err != nil {
				return err
			}
			contract.InitDefaultValidator(contract2.New(newChain.(*chain2.Chain).Conn(),
				[]common.Address{common.HexToAddress(chainConfig.Opts[chain2.Validate])}, validateAbi))
			mapprotocol.Init2GetEth22MapNumber(common.HexToAddress(chainConfig.Opts[chain2.LightNode]))
			mapprotocol.InitOtherChain2MapHeight(common.HexToAddress(chainConfig.Opts[chain2.LightNode]))
			mapprotocol.InitLightManager(common.HexToAddress(chainConfig.Opts[chain2.LightNode]))
			mapprotocol.LightManagerNodeType(common.HexToAddress(chainConfig.Opts[chain2.LightNode]))
		}

		mapprotocol.OnlineChaId[chainConfig.Id] = chainConfig.Name
		c.AddChain(newChain)
	}
	c.Start()

	return nil
}

func filterAPIKeyFromConfig(ctx *cli.Context, cfg *config.Config) string {
	if ctx.IsSet(config.FilterAPIKeyFlag.Name) {
		if key := strings.TrimSpace(ctx.String(config.FilterAPIKeyFlag.Name)); key != "" {
			return key
		}
	}
	if cfg.Other.FilterAPIKey != "" {
		return strings.TrimSpace(cfg.Other.FilterAPIKey)
	}
	key, err := os.ReadFile(os.ExpandEnv("$HOME/.compass/filter_api_key"))
	if err == nil {
		return strings.TrimSpace(string(key))
	}
	return ""
}

func butterAPIKeyFromConfig(ctx *cli.Context, cfg *config.Config) string {
	if ctx.IsSet(config.ButterAPIKeyFlag.Name) {
		return strings.TrimSpace(ctx.String(config.ButterAPIKeyFlag.Name))
	}
	if cfg.Other.ButterAPIKey != "" {
		return strings.TrimSpace(cfg.Other.ButterAPIKey)
	}
	key, err := os.ReadFile(os.ExpandEnv("$HOME/.compass/butter_api_key"))
	if err == nil {
		return strings.TrimSpace(string(key))
	}
	return ""
}
