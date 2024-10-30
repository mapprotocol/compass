package main

import (
	"errors"
	"github.com/mapprotocol/compass/chains/ton"
	"os"
	"strconv"

	"github.com/mapprotocol/compass/chains/tron"

	"github.com/mapprotocol/compass/pkg/util"

	"github.com/mapprotocol/compass/chains/bsc"
	"github.com/mapprotocol/compass/chains/conflux"

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
	"github.com/mapprotocol/compass/mapprotocol"
	"github.com/mapprotocol/compass/msg"
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
	config.SkipErrorFlag,
	config.FilterFlag,
}

var devFlags = []cli.Flag{
	config.TestKeyFlag,
}

var importFlags = []cli.Flag{
	config.EthereumImportFlag,
	config.PrivateKeyFlag,
	config.PasswordFlag,
	config.KeystorePathFlag,
	config.TronFlag,
	config.TronKeyNameFlag,
}

var accountCommand = cli.Command{
	Name:  "accounts",
	Usage: "manage bridge keystore",
	Description: "The accounts command is used to manage the bridge keystore.\n" +
		"\tTo import a tron private key file: compass accounts import --privateKey private_key",
	Subcommands: []*cli.Command{
		{
			Action: wrapHandler(handleImportCmd),
			Name:   "import",
			Usage:  "import bridge keystore",
			Flags:  importFlags,
			Description: "The import subcommand is used to import a keystore for the bridge.\n" +
				"\tA path to the keystore must be provided\n" +
				"\tUse --privateKey to create a keystore from a provided private key.",
		},
	},
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
	Version = "1.2.1"
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
		&oracleCommand,
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
	util.Init(cfg.Other.Env, cfg.Other.MonitorUrl)
	sysErr := make(chan error)
	mapcid, err := strconv.Atoi(cfg.MapChain.Id)
	if err != nil {
		return err
	}
	c := core.NewCore(sysErr, msg.ChainId(mapcid), role)
	// merge map chain
	allChains := make([]config.RawChainConfig, 0, len(cfg.Chains)+1)
	allChains = append(allChains, cfg.MapChain)
	allChains = append(allChains, cfg.Chains...)

	for idx, chain := range allChains {
		ks := chain.KeystorePath
		if ks == "" {
			ks = ctx.String(config.KeyPathFlag.Name)
		}
		chainId, err := strconv.Atoi(chain.Id)
		if err != nil {
			return err
		}
		mapprotocol.MapId = cfg.MapChain.Id
		chain.Opts[config.MapChainID] = cfg.MapChain.Id
		chainConfig := &core.ChainConfig{
			Name:             chain.Name,
			Id:               msg.ChainId(chainId),
			Endpoint:         chain.Endpoint,
			From:             chain.From,
			Network:          chain.Network,
			KeystorePath:     ks,
			NearKeystorePath: chain.KeystorePath,
			BlockstorePath:   ctx.String(config.BlockstorePathFlag.Name),
			FreshStart:       ctx.Bool(config.FreshStartFlag.Name),
			LatestBlock:      ctx.Bool(config.LatestBlockFlag.Name),
			Opts:             chain.Opts,
			SkipError:        ctx.Bool(config.SkipErrorFlag.Name),
			Filter:           ctx.Bool(config.FilterFlag.Name),
			FilterHost:       cfg.Other.Filter,
		}
		var (
			newChain core.Chain
		)

		logger := log.Root().New("chain", chainConfig.Name)
		switch chain.Type {
		case chains.Ethereum:
			newChain, err = ethereum.InitializeChain(chainConfig, logger, sysErr, role)
			if err != nil {
				return err
			}
			if idx == 0 {
				mapprotocol.GlobalMapConn = newChain.(*chain2.Chain).EthClient()
				mapprotocol.Init2GetEth22MapNumber(common.HexToAddress(chainConfig.Opts[chain2.LightNode]))
				mapprotocol.InitOtherChain2MapHeight(common.HexToAddress(chainConfig.Opts[chain2.LightNode]))
				mapprotocol.InitLightManager(common.HexToAddress(chainConfig.Opts[chain2.LightNode]))
				mapprotocol.LightManagerNodeType(common.HexToAddress(chainConfig.Opts[chain2.LightNode]))
			}
		case chains.Near:
			newChain, err = near.InitializeChain(chainConfig, logger, sysErr, role)
		case chains.Bsc:
			newChain, err = bsc.InitializeChain(chainConfig, logger, sysErr, role)
		case chains.Matic:
			newChain, err = matic.InitializeChain(chainConfig, logger, sysErr, role)
		case chains.Klaytn:
			newChain, err = klaytn.InitializeChain(chainConfig, logger, sysErr, role)
		case chains.Eth2:
			newChain, err = eth2.InitializeChain(chainConfig, logger, sysErr, role)
		case chains.Conflux:
			newChain, err = conflux.InitializeChain(chainConfig, logger, sysErr, role)
		case chains.Tron:
			newChain, err = tron.NewChain(chainConfig, logger, sysErr, role)
		case chains.Ton:
			newChain, err = ton.New(chainConfig, logger, sysErr, role)
		default:
			return errors.New("unrecognized Chain Type")
		}
		if err != nil {
			return err
		}

		mapprotocol.OnlineChaId[chainConfig.Id] = chainConfig.Name
		c.AddChain(newChain)
	}
	c.Start()

	return nil
}
