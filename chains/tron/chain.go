package tron

import (
	"fmt"
	"math/big"
	"os"

	"github.com/mapprotocol/compass/keystore"

	metrics "github.com/ChainSafe/chainbridge-utils/metrics/types"
	"github.com/ChainSafe/log15"
	"github.com/ethereum/go-ethereum/log"
	"github.com/lbtsm/gotron-sdk/pkg/client"
	"github.com/mapprotocol/compass/chains"
	"github.com/mapprotocol/compass/core"
	"github.com/mapprotocol/compass/internal/chain"
	"github.com/mapprotocol/compass/mapprotocol"
	"github.com/mapprotocol/compass/msg"
	"github.com/pkg/errors"
)

func NewChain(chainCfg *core.ChainConfig, logger log15.Logger, sysErr chan<- error, m *metrics.ChainMetrics,
	role mapprotocol.Role) (core.Chain, error) {
	return createChain(chainCfg, logger, sysErr, m, role)
}

func createChain(chainCfg *core.ChainConfig, logger log15.Logger, sysErr chan<- error, m *metrics.ChainMetrics,
	role mapprotocol.Role, opts ...chain.SyncOpt) (core.Chain, error) {
	config, err := parseCfg(chainCfg)
	if err != nil {
		return nil, err
	}

	conn := NewConnection(config.Endpoint, logger)
	err = conn.Connect()
	if err != nil {
		return nil, err
	}

	var pswd []byte
	if pswdStr := os.Getenv(keystore.EnvPassword); pswdStr != "" {
		pswd = []byte(pswdStr)
	} else {
		pswd = keystore.GetPassword(fmt.Sprintf("Enter password for key %s:", chainCfg.From))
	}

	var (
		stop   = make(chan int, 0)
		listen chains.Listener
	)

	if role == mapprotocol.RoleOfMaintainer {
		fn := Map2Tron(config.From, config.LightNode, conn.cli)
		height, err := fn()
		if err != nil {
			return nil, errors.Wrap(err, "Map2Tron get init headerHeight failed")
		}
		logger.Info("Map2other Current situation", "id", config.Id, "height", height, "lightNode", config.LightNode)
		mapprotocol.SyncOtherMap[config.Id] = height
		mapprotocol.Map2OtherHeight[config.Id] = fn
		listen = NewMaintainer(logger)
	} else if role == mapprotocol.RoleOfMessenger {

	}

	return &Chain{
		conn:   conn,
		stop:   stop,
		listen: listen,
		cfg:    chainCfg,
		writer: newWriter(conn, config, logger, stop, sysErr, pswd),
	}, nil
}

type Chain struct {
	cfg    *core.ChainConfig
	conn   core.Connection
	writer *Writer
	stop   chan<- int
	listen chains.Listener
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

// Conn return Connection interface for relayer register
func (c *Chain) Conn() core.Connection {
	return c.conn
}

func Map2Tron(fromUser, lightNode string, client *client.GrpcClient) mapprotocol.GetHeight {
	return func() (*big.Int, error) {
		call, err := client.TriggerConstantContract(fromUser, lightNode, "headerHeight()", "")
		if err != nil {
			return nil, fmt.Errorf("map2tron call headerHeight failed, err is %v", err.Error())
		}
		return mapprotocol.UnpackHeaderHeightOutput(call.ConstantResult[0])
	}
}
