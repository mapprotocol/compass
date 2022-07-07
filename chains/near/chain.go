package near

import (
	"context"
	"encoding/json"
	"log"
	"math/big"

	metrics "github.com/ChainSafe/chainbridge-utils/metrics/types"
	"github.com/ChainSafe/log15"
	nearclient "github.com/eteu-technologies/near-api-go/pkg/client"
	"github.com/eteu-technologies/near-api-go/pkg/client/block"
	"github.com/eteu-technologies/near-api-go/pkg/types/key"
	"github.com/ethereum/go-ethereum/common"
	"github.com/mapprotocol/atlas/accounts/abi/bind"
	"github.com/mapprotocol/compass/blockstore"
	"github.com/mapprotocol/compass/chains"
	connection "github.com/mapprotocol/compass/connections/near"
	"github.com/mapprotocol/compass/core"
	"github.com/mapprotocol/compass/internal/near"
	"github.com/mapprotocol/compass/keystore"
	"github.com/mapprotocol/compass/mapprotocol"
	"github.com/mapprotocol/compass/msg"
	"github.com/pkg/errors"
)

type Connection interface {
	Connect() error
	Keypair() *key.KeyPair
	Opts() *bind.TransactOpts
	CallOpts() *bind.CallOpts
	LockAndUpdateOpts() error
	UnlockOpts()
	Client() *nearclient.Client
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
func setupBlockstore(cfg *Config, kp *key.KeyPair, role mapprotocol.Role) (*blockstore.Blockstore, error) {
	bs, err := blockstore.NewBlockstore(cfg.blockstorePath, cfg.id, kp.PublicKey.ToPublicKey().Hash(), role)
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

	kp, err := keystore.NearKeyPairFrom(chainCfg.Network, cfg.keystorePath, cfg.from)
	if err != nil {
		return nil, err
	}

	bs, err := setupBlockstore(cfg, &kp, role)
	if err != nil {
		return nil, err
	}

	stop := make(chan int)
	conn := connection.NewConnection(cfg.endpoint, cfg.http, &kp, logger, cfg.gasLimit, cfg.maxGasPrice,
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

	if cfg.lightNode != "" && cfg.id != cfg.mapChainID && syncMap != nil { // 请求获取同步的map 高度
		res, err := conn.Client().ContractViewCallFunction(context.Background(), cfg.lightNode, AbiMethodOfGetHeaderHeight,
			"e30=", block.FinalityFinal())
		if err != nil {
			return nil, errors.Wrap(err, "call near lightNode to get headerHeight failed")
		}

		if res.Error != nil {
			return nil, errors.Wrap(res.Error, "call near lightNode to get headerHeight resp exist error")
		}

		result := &near.Result{}
		err = json.Unmarshal(res.Result, result)
		if err != nil {
			return nil, errors.Wrap(err, "near lightNode headerHeight resp json marshal failed")
		}
		height, _ := new(big.Int).SetString(string(result.Result), 10)
		syncMap[cfg.id] = height
	}

	// simplified a little bit
	var listen chains.Listener
	cs := NewCommonListen(conn, cfg, logger, stop, sysErr, m, bs)
	if role == mapprotocol.RoleOfMessenger {
		listen = NewMessenger(cs) // todo 修改这里 ---------------------
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
	log.Println("Chain setRouter ----------------- ", c.listen)
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
func (c *Chain) EthClient() *nearclient.Client {
	return c.conn.Client()
}

// Conn return Connection interface for relayer register
func (c *Chain) Conn() Connection {
	return c.conn
}
