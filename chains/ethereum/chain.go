// Copyright 2021 Compass Systems
// SPDX-License-Identifier: LGPL-3.0-only
/*
package ethereum
The ethereum package contains the logic for interacting with ethereum chains.

There are 3 major components: the connection, the listener, and the writer.
The currently supported transfer types are Fungible (ERC20), Non-Fungible (ERC721), and generic.

Connection

The connection contains the ethereum RPC client and can be accessed by both the writer and listener.

Listener

The listener polls for each new block and looks for deposit events in the bridge contract. If a deposit occurs, the listener will fetch additional information from the handler before constructing a message and forwarding it to the router.

Writer

The writer recieves the message and creates a proposals on-chain. Once a proposal is made, the writer then watches for a finalization event and will attempt to execute the proposal if a matching event occurs. The writer skips over any proposals it has already seen.
*/

package ethereum

import (
	"github.com/mapprotocol/compass/internal/chain"
	"github.com/mapprotocol/compass/internal/monitor"
	w "github.com/mapprotocol/compass/internal/writer"
	"math/big"

	"github.com/pkg/errors"

	"github.com/ChainSafe/chainbridge-utils/crypto/secp256k1"
	metrics "github.com/ChainSafe/chainbridge-utils/metrics/types"
	"github.com/ChainSafe/log15"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/mapprotocol/compass/chains"
	connection "github.com/mapprotocol/compass/connections/ethereum"
	"github.com/mapprotocol/compass/core"
	"github.com/mapprotocol/compass/keystore"
	"github.com/mapprotocol/compass/mapprotocol"
	"github.com/mapprotocol/compass/msg"
	"github.com/mapprotocol/compass/pkg/ethclient"
)

var _ core.Chain = &Chain{}

var _ Connection = &connection.Connection{}

type Connection interface {
	Connect() error
	Keypair() *secp256k1.Keypair
	Opts() *bind.TransactOpts
	CallOpts() *bind.CallOpts
	LockAndUpdateOpts() error
	UnlockOpts()
	Client() *ethclient.Client
	EnsureHasBytecode(address common.Address) error
	LatestBlock() (*big.Int, error)
	WaitForBlock(block *big.Int, delay *big.Int) error
	Close()
}

type Chain struct {
	cfg    *core.ChainConfig // The config of the chain
	conn   Connection        // The chains connection
	writer *w.Writer         // The writer of the chain
	stop   chan<- int
	listen chains.Listener // The listener of this chain
}

func InitializeChain(chainCfg *core.ChainConfig, logger log15.Logger, sysErr chan<- error, m *metrics.ChainMetrics,
	role mapprotocol.Role) (*Chain, error) {
	cfg, err := chain.ParseConfig(chainCfg)
	if err != nil {
		return nil, err
	}

	kpI, err := keystore.KeypairFromAddress(cfg.From, keystore.EthChain, cfg.KeystorePath, chainCfg.Insecure)
	if err != nil {
		return nil, err
	}
	kp, _ := kpI.(*secp256k1.Keypair)

	bs, err := chain.SetupBlockStore(cfg, kp, role)
	if err != nil {
		return nil, err
	}

	stop := make(chan int)
	conn := connection.NewConnection(cfg.Endpoint, cfg.Http, kp, logger, cfg.GasLimit, cfg.MaxGasPrice,
		cfg.GasMultiplier, cfg.EgsApiKey, cfg.EgsSpeed)
	err = conn.Connect()
	if err != nil {
		return nil, err
	}

	if chainCfg.LatestBlock {
		curr, err := conn.LatestBlock()
		if err != nil {
			return nil, err
		}
		cfg.StartBlock = curr
	}

	if role == mapprotocol.RoleOfMaintainer && cfg.Id != cfg.MapChainID { // 请求获取同步的map高度
		fn := mapprotocol.Map2EthHeight(cfg.From, cfg.LightNode, conn.Client())
		height, err := fn()
		if err != nil {
			return nil, errors.Wrap(err, "eth get init headerHeight failed")
		}
		logger.Info("map2Other Current situation", "chain", cfg.Name, "height", height)
		mapprotocol.SyncOtherMap[cfg.Id] = height
		mapprotocol.Map2OtherHeight[cfg.Id] = fn
	}

	// simplified a little bit
	var listen chains.Listener
	cs := chain.NewCommonSync(conn, cfg, logger, stop, sysErr, m, bs)
	if role == mapprotocol.RoleOfMessenger {
		err = conn.EnsureHasBytecode(cfg.McsContract)
		if err != nil {
			return nil, err
		}
		// verify range
		if cfg.Id != cfg.MapChainID {
			fn := mapprotocol.Map2EthVerifyRange(cfg.From, cfg.LightNode, conn.Client())
			left, right, err := fn()
			if err != nil {
				return nil, errors.Wrap(err, "eth init verify range failed")
			}
			logger.Info("Map2eth Current verify range", "chain", cfg.Name, "left", left, "right", right, "lightNode", cfg.LightNode)
			mapprotocol.Map2OtherVerifyRange[cfg.Id] = fn
		}
		listen = NewMessenger(cs)
	} else if role == mapprotocol.RoleOfMaintainer { // Maintainer is used by default
		listen = NewMaintainer(cs)
	} else if role == mapprotocol.RoleOfMonitor {
		listen = monitor.New(cs)
	}
	writer := w.New(conn, cfg, logger, stop, sysErr, m)

	return &Chain{
		cfg:    chainCfg,
		conn:   conn,
		writer: writer,
		stop:   stop,
		listen: listen,
	}, nil
}

func (c *Chain) SetRouter(r *core.Router) {
	r.Listen(c.cfg.Id, c.writer)
	c.listen.SetRouter(r)
}

func (c *Chain) Start() error {
	err := c.listen.Sync()
	if err != nil {
		return err
	}

	return nil
}

func (c *Chain) Id() msg.ChainId {
	return c.cfg.Id
}

func (c *Chain) Name() string {
	return c.cfg.Name
}

func (c *Chain) LatestBlock() metrics.LatestBlock {
	return c.listen.GetLatestBlock()
}

// Stop signals to any running routines to exit
func (c *Chain) Stop() {
	close(c.stop)
	if c.conn != nil {
		c.conn.Close()
	}
}

// EthClient return EthClient for global map connection
func (c *Chain) EthClient() *ethclient.Client {
	return c.conn.Client()
}

// Conn return Connection interface for relayer register
func (c *Chain) Conn() Connection {
	return c.conn
}
