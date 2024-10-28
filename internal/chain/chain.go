package chain

import (
	"github.com/ChainSafe/log15"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/mapprotocol/compass/chains"
	"github.com/mapprotocol/compass/core"
	"github.com/mapprotocol/compass/keystore"
	"github.com/mapprotocol/compass/mapprotocol"
	"github.com/mapprotocol/compass/msg"
	"github.com/mapprotocol/compass/pkg/abi"
	"github.com/mapprotocol/compass/pkg/contract"
	"github.com/mapprotocol/compass/pkg/ethclient"
	"github.com/pkg/errors"
)

type Chain struct {
	cfg    *core.ChainConfig // The config of the Chain
	conn   core.Connection   // The chains connection
	writer *Writer           // The writer of the Chain
	stop   chan<- int
	listen chains.Listener // The listener of this Chain
}

func New(chainCfg *core.ChainConfig, logger log15.Logger, sysErr chan<- error, role mapprotocol.Role,
	createConn core.CreateConn, opts ...SyncOpt) (*Chain, error) {
	cfg, err := ParseConfig(chainCfg)
	if err != nil {
		return nil, err
	}

	kpI, err := keystore.KeypairFromEth(cfg.KeystorePath)
	if err != nil {
		return nil, err
	}

	bs, err := SetupBlockStore(cfg, role)
	if err != nil {
		return nil, err
	}

	stop := make(chan int)
	conn := createConn(cfg.Endpoint, cfg.Http, kpI, logger, cfg.GasLimit, cfg.MaxGasPrice, cfg.GasMultiplier)
	err = conn.Connect()
	if err != nil {
		return nil, err
	}

	if (chainCfg.LatestBlock || cfg.StartBlock == nil || cfg.StartBlock.Int64() == 0) && !chainCfg.Filter {
		curr, err := conn.LatestBlock()
		if err != nil {
			return nil, err
		}
		cfg.StartBlock = curr
	}

	var listen chains.Listener
	cs := NewCommonSync(conn, cfg, logger, stop, sysErr, bs, opts...)
	switch role {
	case mapprotocol.RoleOfMaintainer:
		if cfg.Id != cfg.MapChainID {
			fn := mapprotocol.Map2EthHeight(cfg.From, cfg.LightNode, conn.Client())
			height, err := fn()
			if err != nil {
				return nil, errors.Wrap(err, "Map2Other get init headerHeight failed")
			}
			logger.Info("Map2other Current situation", "id", cfg.Id, "height", height, "lightNode", cfg.LightNode)
			mapprotocol.SyncOtherMap[cfg.Id] = height
			mapprotocol.Map2OtherHeight[cfg.Id] = fn
		}
		listen = NewMaintainer(cs)
	case mapprotocol.RoleOfMessenger:
		oracleAbi, _ := abi.New(mapprotocol.OracleAbiJson)
		call := contract.New(conn, cfg.McsContract, oracleAbi)
		mapprotocol.ContractMapping[cfg.Id] = call
		listen = NewMessenger(cs)
	case mapprotocol.RoleOfOracle:
		otherAbi, _ := abi.New(mapprotocol.OtherAbi)
		call := contract.New(conn, []common.Address{cfg.LightNode}, otherAbi)
		mapprotocol.LightNodeMapping[cfg.Id] = call
		listen = NewOracle(cs)
	}
	oAbi, _ := abi.New(mapprotocol.SignerJson)
	oracleCall := contract.New(conn, []common.Address{cfg.OracleNode}, oAbi)
	mapprotocol.SingMapping[cfg.Id] = oracleCall
	wri := NewWriter(conn, cfg, logger, stop, sysErr)

	return &Chain{
		cfg:    chainCfg,
		conn:   conn,
		writer: wri,
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

	log.Debug("Successfully started Chain")
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
func (c *Chain) EthClient() *ethclient.Client {
	return c.conn.Client()
}

// Conn return Connection interface for relayer register
func (c *Chain) Conn() core.Connection {
	return c.conn
}
