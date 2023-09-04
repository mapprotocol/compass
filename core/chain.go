// Copyright 2021 Compass Systems
// SPDX-License-Identifier: LGPL-3.0-only

package core

import (
	"math/big"

	"github.com/ChainSafe/chainbridge-utils/crypto/secp256k1"
	metrics "github.com/ChainSafe/chainbridge-utils/metrics/types"
	"github.com/ChainSafe/log15"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/mapprotocol/compass/internal/eth2"
	"github.com/mapprotocol/compass/internal/klaytn"
	"github.com/mapprotocol/compass/msg"
	"github.com/mapprotocol/compass/pkg/ethclient"
)

type Chain interface {
	Start() error // Start chain
	SetRouter(*Router)
	Id() msg.ChainId
	Name() string
	LatestBlock() metrics.LatestBlock
	Stop()
	Conn() Connection
}

type ChainConfig struct {
	Name             string            // Human-readable chain name
	Id               msg.ChainId       // ChainID
	Endpoint         string            // url for rpc endpoint
	Network          string            //
	From             string            // address of key to use
	KeystorePath     string            // Location of key files
	NearKeystorePath string            // Location of key files
	Insecure         bool              // Indicated whether the test keyring should be used
	BlockstorePath   string            // Location of blockstore
	FreshStart       bool              // If true, blockstore is ignored at start.
	LatestBlock      bool              // If true, overrides blockstore or latest block in config and starts from current block
	Opts             map[string]string // Per chain options
	SkipError        bool              // Flag of Skip Error
}

type Connection interface {
	Connect() error
	Keypair() *secp256k1.Keypair
	Opts() *bind.TransactOpts
	CallOpts() *bind.CallOpts
	LockAndUpdateOpts(bool) error
	UnlockOpts()
	Client() *ethclient.Client
	EnsureHasBytecode(address common.Address) error
	LatestBlock() (*big.Int, error)
	WaitForBlock(block *big.Int, delay *big.Int) error
	Close()
}

type KConnection interface {
	Connection
	KClient() *klaytn.Client
}

type Eth2Connection interface {
	Connection
	Eth2Client() *eth2.Client
}

type CreateConn func(string, bool, *secp256k1.Keypair, log15.Logger, *big.Int, *big.Int, float64, string, string) Connection
