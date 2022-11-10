package matic

import (
	"github.com/ethereum/go-ethereum/log"
	"github.com/mapprotocol/compass/chains"
	"github.com/mapprotocol/compass/internal/chain"
	"github.com/mapprotocol/compass/internal/writer"
	"github.com/mapprotocol/compass/mapprotocol"

	"github.com/ChainSafe/chainbridge-utils/crypto/secp256k1"
	metrics "github.com/ChainSafe/chainbridge-utils/metrics/types"
	"github.com/ChainSafe/log15"
	connection "github.com/mapprotocol/compass/connections/ethereum"
	"github.com/mapprotocol/compass/core"
	"github.com/mapprotocol/compass/keystore"
	"github.com/mapprotocol/compass/msg"
	"github.com/mapprotocol/compass/pkg/ethclient"
)

var _ core.Chain = new(Chain)

type Chain struct {
	cfg    *core.ChainConfig // The config of the chain
	conn   chain.Connection  // The chains connection
	writer *writer.Writer    // The writer of the chain
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

	var listen chains.Listener
	cs := chain.NewCommonSync(conn, cfg, logger, stop, sysErr, m, bs)
	if role == mapprotocol.RoleOfMaintainer { // 请求获取同步的map高度
		//fn := mapprotocol.Map2EthHeight(cfg.From, cfg.LightNode, conn.Client())
		//height, err := fn()
		//if err != nil {
		//	return nil, errors.Wrap(err, "bsc get init headerHeight failed")
		//}
		//logger.Info("map2 Current situation", "chain", cfg.Name, "height", height, "lightNode", cfg.LightNode)
		//mapprotocol.SyncOtherMap[cfg.Id] = height
		//mapprotocol.Map2OtherHeight[cfg.Id] = fn
		listen = NewMaintainer(cs)
	} else if role == mapprotocol.RoleOfMessenger {
		err = conn.EnsureHasBytecode(cfg.McsContract)
		if err != nil {
			return nil, err
		}
		listen = NewMessenger(cs)
	}
	w := writer.New(conn, cfg, logger, stop, sysErr, m)

	return &Chain{
		cfg:    chainCfg,
		conn:   conn,
		writer: w,
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

	log.Debug("Successfully started chain")
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
func (c *Chain) Conn() chain.Connection {
	return c.conn
}
