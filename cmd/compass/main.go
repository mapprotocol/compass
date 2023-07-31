// Copyright 2021 Compass Systems
// SPDX-License-Identifier: LGPL-3.0-only
/*
Provides the command-line interface for the chainbridge application.

For configuration and CLI commands see the README: https://github.com/ChainSafe/ChainBridge.
*/
package main

import (
	"errors"
	"fmt"
	"github.com/mapprotocol/compass/chains/conflux"
	"net/http"
	"os"
	"strconv"

	"github.com/mapprotocol/compass/chains/platon"

	"github.com/mapprotocol/compass/chains/bsc"

	"github.com/ChainSafe/chainbridge-utils/metrics/health"
	metrics "github.com/ChainSafe/chainbridge-utils/metrics/types"
	log "github.com/ChainSafe/log15"
	"github.com/ethereum/go-ethereum/common"
	"github.com/mapprotocol/compass/chains"
	"github.com/mapprotocol/compass/chains/eth2"
	"github.com/mapprotocol/compass/chains/ethereum"
	"github.com/mapprotocol/compass/chains/klaytn"
	"github.com/mapprotocol/compass/chains/matic"
	"github.com/mapprotocol/compass/chains/near"
	"github.com/mapprotocol/compass/config"
	"github.com/mapprotocol/compass/core"
	chain2 "github.com/mapprotocol/compass/internal/chain"
	"github.com/mapprotocol/compass/internal/monitor"
	"github.com/mapprotocol/compass/mapprotocol"
	"github.com/mapprotocol/compass/msg"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/cors"
	"github.com/urfave/cli/v2"
)

var app = cli.NewApp()

var cliFlags = []cli.Flag{
	config.ConfigFileFlag,
	config.VerbosityFlag,
	config.KeystorePathFlag,
	config.BlockstorePathFlag,
	config.FreshStartFlag,
	config.LatestBlockFlag,
	config.MetricsFlag,
	config.MetricsPort,
	config.SkipErrorFlag,
}

var generateFlags = []cli.Flag{
	config.PasswordFlag,
	config.Sr25519Flag,
	config.Secp256k1Flag,
	config.Ed25519Flag,
	config.SubkeyNetworkFlag,
}

var devFlags = []cli.Flag{
	config.TestKeyFlag,
}

var importFlags = []cli.Flag{
	config.EthereumImportFlag,
	config.PrivateKeyFlag,
	config.Sr25519Flag,
	config.Secp256k1Flag,
	config.Ed25519Flag,
	config.PasswordFlag,
	config.SubkeyNetworkFlag,
}

var registerFlags = []cli.Flag{
	config.Account,
	config.Value,
}

var bindFlags = []cli.Flag{
	config.Relayer,
	config.Worker,
}

var monitorFlags = []cli.Flag{
	config.ConfigFileFlag,
	config.ExposePortFlag,
}

var accountCommand = cli.Command{
	Name:  "accounts",
	Usage: "manage bridge keystore",
	Description: "The accounts command is used to manage the bridge keystore.\n" +
		"\tTo generate a new account (key type generated is determined on the flag passed in): compass accounts generate\n" +
		"\tTo import a keystore file: compass accounts import path/to/file\n" +
		"\tTo import a geth keystore file: compass accounts import --ethereum path/to/file\n" +
		"\tTo import a private key file: compass accounts import --privateKey private_key\n" +
		"\tTo list keys: compass accounts list",
	Subcommands: []*cli.Command{
		{
			Action: wrapHandler(handleGenerateCmd),
			Name:   "generate",
			Usage:  "generate bridge keystore, key type determined by flag",
			Flags:  generateFlags,
			Description: "The generate subcommand is used to generate the bridge keystore.\n" +
				"\tIf no options are specified, a secp256k1 key will be made.",
		},
		{
			Action: wrapHandler(handleImportCmd),
			Name:   "import",
			Usage:  "import bridge keystore",
			Flags:  importFlags,
			Description: "The import subcommand is used to import a keystore for the bridge.\n" +
				"\tA path to the keystore must be provided\n" +
				"\tUse --ethereum to import an ethereum keystore from external sources such as geth\n" +
				"\tUse --privateKey to create a keystore from a provided private key.",
		},
		{
			Action:      wrapHandler(handleListCmd),
			Name:        "list",
			Usage:       "list bridge keystore",
			Description: "The list subcommand is used to list all of the bridge keystores.\n",
		},
	},
}

var maintainerCommand = cli.Command{
	Name:  "maintainer",
	Usage: "manage maintainer operations",
	Description: "The maintainer command is used to manage maintainer on Map chain.\n" +
		"\tTo register an account : compass relayers register --account '0x0...'",
	Action: maintainer,
	Subcommands: []*cli.Command{
		{
			Action: handleRegisterCmd,
			Name:   "register",
			Usage:  "register account as relayer",
			Flags:  registerFlags,
			Description: "The register subcommand is used to register an account as relayer.\n" +
				"\tA path to the keystore must be provided\n" +
				"\tA path to the config must be provided\n" +
				"\tUse --account to provide an address of an account.",
		},
		{
			Action: handleBindCmd,
			Name:   "bind",
			Usage:  "bind a worker account for relayer",
			Flags:  bindFlags,
			Description: "The bind subcommand is used to bind a worker account for relayer.\n" +
				"\tA path to the keystore must be provided\n" +
				"\tA path to the config must be provided\n" +
				"\tUse --relayer to provide the address of relayer.\n" +
				"\tUse --worker to provide the address of worker.",
		},
	},
	Flags: append(app.Flags, cliFlags...),
}

var messengerCommand = cli.Command{
	Name:        "messenger",
	Usage:       "manage messenger operations",
	Description: "The messenger command is used to sync the log information of transactions in the block",
	Action:      messenger,
	Flags:       append(app.Flags, cliFlags...),
}

var monitorCommand = cli.Command{
	Name:        "expose",
	Usage:       "monitor account balance",
	Description: "The messenger command is used to sync the log information of transactions in the block",
	Action:      expose,
	Flags:       append(app.Flags, monitorFlags...),
}

var (
	Version = "1.1.0"
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
		&accountCommand,
		&maintainerCommand,
		&messengerCommand,
		&monitorCommand,
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

func expose(ctx *cli.Context) error {
	return run(ctx, mapprotocol.RoleOfMonitor)
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

	log.Debug("Config on initialization...", "config", *cfg)

	// Check for test key flag
	var ks string
	var insecure bool
	if key := ctx.String(config.TestKeyFlag.Name); key != "" {
		ks = key
		insecure = true
	} else {
		ks = cfg.KeystorePath
	}

	// Used to signal core shutdown due to fatal error
	sysErr := make(chan error)

	mapcid, err := strconv.Atoi(cfg.MapChain.Id)
	if err != nil {
		return err
	}
	c := core.NewCore(sysErr, msg.ChainId(mapcid))
	// merge map chain
	allChains := make([]config.RawChainConfig, 0, len(cfg.Chains)+1)
	allChains = append(allChains, cfg.MapChain)
	allChains = append(allChains, cfg.Chains...)

	for idx, chain := range allChains {
		chainId, err := strconv.Atoi(chain.Id)
		if err != nil {
			return err
		}
		// write Map chain id to opts
		chain.Opts[config.MapChainID] = cfg.MapChain.Id
		chainConfig := &core.ChainConfig{
			Name:             chain.Name,
			Id:               msg.ChainId(chainId),
			Endpoint:         chain.Endpoint,
			From:             chain.From,
			Network:          chain.Network,
			KeystorePath:     ks,
			NearKeystorePath: chain.KeystorePath,
			Insecure:         insecure,
			BlockstorePath:   ctx.String(config.BlockstorePathFlag.Name),
			FreshStart:       ctx.Bool(config.FreshStartFlag.Name),
			LatestBlock:      ctx.Bool(config.LatestBlockFlag.Name),
			Opts:             chain.Opts,
			SkipError:        ctx.Bool(config.SkipErrorFlag.Name),
		}
		var (
			newChain core.Chain
			m        *metrics.ChainMetrics
		)

		logger := log.Root().New("chain", chainConfig.Name)
		if ctx.Bool(config.MetricsFlag.Name) {
			m = metrics.NewChainMetrics(chain.Name)
		}
		logger.Info("This task set skip error", "skip", ctx.Bool(config.SkipErrorFlag.Name))

		switch chain.Type {
		case chains.Ethereum:
			newChain, err = ethereum.InitializeChain(chainConfig, logger, sysErr, m, role)
			if err != nil {
				return err
			}
			if idx == 0 {
				mapprotocol.GlobalMapConn = newChain.(*ethereum.Chain).EthClient()
				//mapprotocol.Init2MapHeightByLight(common.HexToAddress(chainConfig.Opts[chain2.LightNode]))
				mapprotocol.Init2GetEth22MapNumber(common.HexToAddress(chainConfig.Opts[chain2.LightNode]))
				mapprotocol.InitOtherChain2MapHeight(common.HexToAddress(chainConfig.Opts[chain2.LightNode]))
				mapprotocol.InitOtherChain2MapVerifyRange(common.HexToAddress(chainConfig.Opts[chain2.LightNode]))
				mapprotocol.InitLightManager(common.HexToAddress(chainConfig.Opts[chain2.LightNode]))
			}
		case chains.Near:
			newChain, err = near.InitializeChain(chainConfig, logger, sysErr, m, role)
		case chains.Bsc:
			newChain, err = bsc.InitializeChain(chainConfig, logger, sysErr, m, role)
		case chains.Matic:
			newChain, err = matic.InitializeChain(chainConfig, logger, sysErr, m, role)
		case chains.Klaytn:
			newChain, err = klaytn.InitializeChain(chainConfig, logger, sysErr, m, role)
		case chains.Eth2:
			newChain, err = eth2.InitializeChain(chainConfig, logger, sysErr, m, role)
		case chains.Platon:
			newChain, err = platon.InitializeChain(chainConfig, logger, sysErr, m, role)
		case chains.Conflux:
			newChain, err = conflux.InitializeChain(chainConfig, logger, sysErr, m, role)
		default:
			return errors.New("unrecognized Chain Type")
		}
		if err != nil {
			return err
		}

		mapprotocol.OnlineChaId[chainConfig.Id] = chainConfig.Name
		mapprotocol.OnlineChainCfg[chainConfig.Id] = chainConfig
		c.AddChain(newChain)
	}

	// Start prometheus and health server
	if ctx.Bool(config.MetricsFlag.Name) {
		port := ctx.Int(config.MetricsPort.Name)
		blockTimeoutStr := os.Getenv(config.HealthBlockTimeout)
		blockTimeout := config.DefaultBlockTimeout
		if blockTimeoutStr != "" {
			blockTimeout, err = strconv.ParseInt(blockTimeoutStr, 10, 0)
			if err != nil {
				return err
			}
		}
		h := health.NewHealthServer(port, c.ToUCoreRegistry(), int(blockTimeout))

		go func() {
			http.Handle("/metrics", promhttp.Handler())
			http.HandleFunc("/health", h.HealthStatus)
			err := http.ListenAndServe(fmt.Sprintf(":%d", port), nil)
			if errors.Is(err, http.ErrServerClosed) {
				log.Info("Health status server is shutting down", err)
			} else {
				log.Error("Error serving metrics", "err", err)
			}
		}()
	}

	if role == mapprotocol.RoleOfMonitor {
		port := ctx.Int(config.ExposePortFlag.Name)
		mux := http.NewServeMux()
		mux.HandleFunc("/get/proof", monitor.Handler)

		handler := cors.Default().Handler(mux)
		err := http.ListenAndServe(fmt.Sprintf(":%d", port), handler)
		if errors.Is(err, http.ErrServerClosed) {
			log.Info("Health status server is shutting down", err)
		} else {
			log.Error("Error serving metrics", "err", err)
		}
	}

	c.Start()

	return nil
}
