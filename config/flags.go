// Copyright 2021 Compass Systems
// SPDX-License-Identifier: LGPL-3.0-only

package config

import (
	log "github.com/ChainSafe/log15"
	"github.com/urfave/cli/v2"
)

// Env vars
var (
	HealthBlockTimeout = "BLOCK_TIMEOUT"
)

var (
	ConfigFileFlag = &cli.StringFlag{
		Name:  "config",
		Usage: "JSON configuration file",
	}

	VerbosityFlag = &cli.StringFlag{
		Name:  "verbosity",
		Usage: "Supports levels crit (silent) to trce (trace)",
		Value: log.LvlInfo.String(),
	}

	KeystorePathFlag = &cli.StringFlag{
		Name:  "keystore",
		Usage: "Path to keystore directory",
		Value: DefaultKeystorePath,
	}

	BlockstorePathFlag = &cli.StringFlag{
		Name:  "blockstore",
		Usage: "Specify path for blockstore",
		Value: "", // Empty will use home dir
	}

	FreshStartFlag = &cli.BoolFlag{
		Name:  "fresh",
		Usage: "Disables loading from blockstore at start. Opts will still be used if specified.",
	}

	LatestBlockFlag = &cli.BoolFlag{
		Name:  "latest",
		Usage: "Overrides blockstore and start block, starts from latest block",
	}

	SkipErrorFlag = &cli.BoolFlag{
		Name:  "skipError",
		Usage: "Skip Error",
	}
)

// Metrics flags
var (
	MetricsFlag = &cli.BoolFlag{
		Name:  "metrics",
		Usage: "Enables metric server",
	}

	MetricsPort = &cli.IntFlag{
		Name:  "metricsPort",
		Usage: "Port to serve metrics on",
		Value: 8001,
	}
)

// Generate subcommand flags
var (
	PasswordFlag = &cli.StringFlag{
		Name:  "password",
		Usage: "Password used to encrypt the keystore. Used with --generate, --import, or --unlock",
	}
	Sr25519Flag = &cli.BoolFlag{
		Name:  "sr25519",
		Usage: "Specify account/key type as sr25519.",
	}
	Secp256k1Flag = &cli.BoolFlag{
		Name:  "secp256k1",
		Usage: "Specify account/key type as secp256k1.",
	}
	Ed25519Flag = &cli.BoolFlag{
		Name:  "ed25519",
		Usage: "Specify account/key type as near.",
	}
)

var (
	EthereumImportFlag = &cli.BoolFlag{
		Name:  "ethereum",
		Usage: "Import an existing ethereum keystore, such as from geth.",
	}
	PrivateKeyFlag = &cli.StringFlag{
		Name:  "privateKey",
		Usage: "Import a hex representation of a private key into a keystore.",
	}
	SubkeyNetworkFlag = &cli.StringFlag{
		Name:        "network",
		Usage:       "Specify the network to use for the address encoding (substrate/polkadot/centrifuge)",
		DefaultText: "substrate",
	}
)

var (
	Account = &cli.StringFlag{
		Name:  "account",
		Usage: "the address of account",
	}

	Value = &cli.Int64Flag{
		Name:  "amount",
		Usage: "the amount of Map to stake for register",
		Value: 100000,
	}
)

var (
	Relayer = &cli.StringFlag{
		Name:  "relayer",
		Usage: "the address of the relayer account",
	}

	Worker = &cli.StringFlag{
		Name:  "worker",
		Usage: "the address of the worker account",
	}
)

// Test Setting Flags
var (
	TestKeyFlag = &cli.StringFlag{
		Name:  "testkey",
		Usage: "Applies a predetermined test keystore to the chains.",
	}
)

var (
	ExposePortFlag = &cli.IntFlag{
		Name:  "exposePort",
		Usage: "Port to serve on",
		Value: 8002,
	}
)
