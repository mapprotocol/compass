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
	"context"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum"

	"github.com/mapprotocol/compass/chains"
	"github.com/mapprotocol/compass/mapprotocol"

	"github.com/ChainSafe/chainbridge-utils/crypto/secp256k1"
	metrics "github.com/ChainSafe/chainbridge-utils/metrics/types"
	"github.com/ChainSafe/log15"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/mapprotocol/compass/blockstore"
	connection "github.com/mapprotocol/compass/connections/ethereum"
	"github.com/mapprotocol/compass/core"
	"github.com/mapprotocol/compass/keystore"
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
	writer *writer           // The writer of the chain
	stop   chan<- int
	listen chains.Listener // The listener of this chain
}

// checkBlockstore queries the blockstore for the latest known block. If the latest block is
// greater than cfg.startBlock, then cfg.startBlock is replaced with the latest known block.
func setupBlockstore(cfg *Config, kp *secp256k1.Keypair, role mapprotocol.Role) (*blockstore.Blockstore, error) {
	bs, err := blockstore.NewBlockstore(cfg.blockstorePath, cfg.id, kp.Address(), role)
	if err != nil {
		return nil, err
	}

	if !cfg.freshStart {
		latestBlock, err := bs.TryLoadLatestBlock()
		if err != nil {
			return nil, err
		}

		if latestBlock.Cmp(cfg.startBlock) == 1 {
			cfg.startBlock = latestBlock
		}
	}

	return bs, nil
}

func InitializeChain(chainCfg *core.ChainConfig, logger log15.Logger, sysErr chan<- error, m *metrics.ChainMetrics,
	role mapprotocol.Role, syncMap map[msg.ChainId]*big.Int) (*Chain, error) {
	cfg, err := parseChainConfig(chainCfg)
	if err != nil {
		return nil, err
	}

	kpI, err := keystore.KeypairFromAddress(cfg.from, keystore.EthChain, cfg.keystorePath, chainCfg.Insecure)
	if err != nil {
		return nil, err
	}
	kp, _ := kpI.(*secp256k1.Keypair)

	bs, err := setupBlockstore(cfg, kp, role)
	if err != nil {
		return nil, err
	}

	stop := make(chan int)
	conn := connection.NewConnection(cfg.endpoint, cfg.http, kp, logger, cfg.gasLimit, cfg.maxGasPrice,
		cfg.gasMultiplier, cfg.egsApiKey, cfg.egsSpeed)
	err = conn.Connect()
	if err != nil {
		return nil, err
	}

	err = conn.EnsureHasBytecode(cfg.bridgeContract)
	if err != nil {
		return nil, err
	}

	if chainCfg.LatestBlock {
		curr, err := conn.LatestBlock()
		if err != nil {
			return nil, err
		}
		cfg.startBlock = curr
	}

	if cfg.lightNode.Hex() != "" && cfg.id != cfg.mapChainID && syncMap != nil { // ?????????????????????map??????
		from := common.HexToAddress(cfg.from)
		input, err := mapprotocol.PackLightNodeInput(mapprotocol.MethodOfHeaderHeight)
		if err != nil {
			return nil, fmt.Errorf("pack lightNode headerHeight Input failed, err is %v", err.Error())
		}
		output, err := conn.Client().CallContract(context.Background(),
			ethereum.CallMsg{
				From: from,
				To:   &cfg.lightNode,
				Data: input,
			},
			nil,
		)
		if err != nil {
			return nil, fmt.Errorf("headerHeight callContract failed, err is %v", err.Error())
		}

		resp, err := mapprotocol.ABILightNode.Methods[mapprotocol.MethodOfHeaderHeight].Outputs.Unpack(output)
		if err != nil {
			return nil, fmt.Errorf("headerHeight unpack failed, err is %v", err.Error())
		}
		var height *big.Int
		err = mapprotocol.ABILightNode.Methods[mapprotocol.MethodOfHeaderHeight].Outputs.Copy(&height, resp)
		if err != nil {
			return nil, fmt.Errorf("headerHeight outputs Copy failed, err is %v", err.Error())
		}
		syncMap[cfg.id] = height
	}

	if cfg.id == cfg.mapChainID && role == mapprotocol.RoleOfMaintainer {
		cfg.syncMap = syncMap
		logger.Info("map to other chain, sync height collect ", "set", syncMap)
		minHeight := big.NewInt(0)
		for cId, height := range cfg.syncMap {
			if minHeight.Cmp(height) == -1 {
				logger.Info("map to other chain find min sync height ", "chainId", cId, "syncedHeight",
					minHeight, "currentHeight", height)
				minHeight = height
			}
		}
		if cfg.startBlock.Cmp(minHeight) != 0 { // When the synchronized height is less than or more than the local starting height, use height
			cfg.startBlock = big.NewInt(minHeight.Int64() + 1)
			logger.Info("-----------------", "+1", big.NewInt(minHeight.Int64()+1))
		}
	}

	// simplified a little bit
	var listen chains.Listener
	cs := NewCommonSync(conn, cfg, logger, stop, sysErr, m, bs)
	if role == mapprotocol.RoleOfMessenger {
		listen = NewMessenger(cs)
	} else { // Maintainer is used by default
		listen = NewMaintainer(cs)
	}
	writer := NewWriter(conn, cfg, logger, stop, sysErr, m)

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

	err = c.writer.start()
	if err != nil {
		return err
	}

	c.writer.log.Debug("Successfully started chain")
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
