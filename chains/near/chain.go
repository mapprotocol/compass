package near

import (
	"github.com/mapprotocol/compass/internal/mapprotocol"
	"github.com/mapprotocol/compass/pkg/keystore"
	"github.com/mapprotocol/compass/pkg/msg"
	"github.com/mapprotocol/compass/pkg/redis"
	"math/big"

	"github.com/pkg/errors"

	"github.com/ChainSafe/log15"
	"github.com/mapprotocol/atlas/accounts/abi/bind"
	connection "github.com/mapprotocol/compass/connections/near"
	"github.com/mapprotocol/compass/core"
	"github.com/mapprotocol/compass/pkg/blockstore"
	nearclient "github.com/mapprotocol/near-api-go/pkg/client"
	"github.com/mapprotocol/near-api-go/pkg/types/key"
)

type Connection interface {
	Connect() error
	Keypair() *key.KeyPair
	Opts() *bind.TransactOpts
	CallOpts() *bind.CallOpts
	LockAndUpdateOpts(bool) error
	UnlockOpts()
	Client() *nearclient.Client
	EnsureHasBytecode(address string) error
	LatestBlock() (*big.Int, error)
	WaitForBlock(block *big.Int, delay *big.Int) error
	Close()
}

type Chain struct {
	cfg    *core.ChainConfig // The config of the chain
	conn   Connection        // The chains connection
	writer *writer           // The writer of the chain
	stop   chan<- int
	listen core.Listener // The listener of this chain
}

func New() *Chain {
	return &Chain{}
}

func setupBlockstore(cfg *Config, kp *key.KeyPair, role mapprotocol.Role) (*blockstore.Blockstore, error) {
	bs, err := blockstore.NewBlockstore(cfg.BlockstorePath, cfg.Id, kp.PublicKey.ToPublicKey().Hash(), role)
	if err != nil {
		return nil, err
	}

	if !cfg.FreshStart {
		latestBlock, err := bs.TryLoadLatestBlock()
		if err != nil {
			return nil, err
		}

		if latestBlock.Cmp(cfg.StartBlock) == 1 {
			cfg.StartBlock = latestBlock
		}
	}

	return bs, nil
}

func (c *Chain) New(chainCfg *core.ChainConfig, logger log15.Logger, sysErr chan<- error,
	role mapprotocol.Role) (core.Chain, error) {
	cfg, err := parseChainConfig(chainCfg)
	if err != nil {
		return nil, err
	}

	kp, err := keystore.NearKeyPairFrom(chainCfg.Network, cfg.KeystorePath, cfg.From)
	if err != nil {
		return nil, err
	}

	bs, err := setupBlockstore(cfg, &kp, role)
	if err != nil {
		return nil, err
	}

	stop := make(chan int)
	conn := connection.NewConnection(cfg.Endpoint, cfg.Http, &kp, logger, cfg.GasLimit, cfg.MaxGasPrice, big.NewFloat(cfg.GasMultiplier))
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

	// simplified a little bit
	var listen core.Listener
	cs := NewCommonListen(conn, cfg, logger, stop, sysErr, bs)
	if role == mapprotocol.RoleOfMessenger {
		redis.Init(cfg.redisUrl)
		listen = NewMessenger(cs)
	} else if role == mapprotocol.RoleOfMaintainer {
		fn := mapprotocol.Map2NearHeight(cfg.lightNode, conn.Client())
		height, err := fn()
		if err != nil {
			return nil, errors.Wrap(err, "near get init headerHeight failed")
		}
		logger.Info("Map2Near Current situation", "height", height, "lightNode", cfg.lightNode)
		mapprotocol.SyncOtherMap[cfg.Id] = height
		mapprotocol.Map2OtherHeight[cfg.Id] = fn
		listen = NewMaintainer(cs)
	}
	writer := NewWriter(conn, cfg, logger, stop, sysErr)

	return &Chain{
		cfg:    chainCfg,
		conn:   conn,
		writer: writer,
		stop:   stop,
		listen: listen,
	}, nil
}

func (c *Chain) SetRouter(r core.Router) {
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
func (c *Chain) Conn() core.Connection {
	return nil
}
