package sol

import (
	"fmt"
	"github.com/ChainSafe/log15"
	"github.com/ethereum/go-ethereum/log"
	"github.com/mapprotocol/compass/core"
	"github.com/mapprotocol/compass/internal/chain"
	"github.com/mapprotocol/compass/internal/mapprotocol"
	"github.com/mapprotocol/compass/pkg/keystore"
	"github.com/mapprotocol/compass/pkg/msg"
	"github.com/mapprotocol/compass/pkg/util"
)

type Chain struct {
	cfg    *core.ChainConfig
	conn   core.Connection
	writer *Writer
	stop   chan<- int
	listen core.Listener
}

func New() *Chain {
	return &Chain{}
}

func (c *Chain) New(chainCfg *core.ChainConfig, logger log15.Logger, sysErr chan<- error, role mapprotocol.Role) (core.Chain, error) {
	return createChain(chainCfg, logger, sysErr, role)
}

func createChain(chainCfg *core.ChainConfig, logger log15.Logger, sysErr chan<- error, role mapprotocol.Role) (core.Chain, error) {
	config, err := parseCfg(chainCfg)
	if err != nil {
		return nil, err
	}

	conn := NewConnection(config.Endpoint, logger)
	err = conn.Connect()
	if err != nil {
		return nil, err
	}

	pswd := make([]byte, 0)
	if role == mapprotocol.RoleOfMessenger {
		pswd = keystore.GetPassword(fmt.Sprintf("Enter password for key %s:", chainCfg.From))

		pri, err := util.DecryptKeystoreText(string(pswd), config.KeystorePath)
		if err != nil {
			return nil, err
		}
		config.Pri = pri
	}

	var (
		stop   = make(chan int)
		listen core.Listener
	)
	bs, err := chain.SetupBlockStore(&config.Config, role)
	if err != nil {
		return nil, err
	}
	cs := chain.NewCommonSync(nil, &config.Config, logger, stop, sysErr, bs)

	switch role {
	case mapprotocol.RoleOfMessenger:
		listen = newSync(cs, mosHandler, conn, config, conn.cli)
	case mapprotocol.RoleOfOracle:
		listen = newSync(cs, oracleHandler, conn, config, conn.cli)
	}
	mapprotocol.MosMapping[config.Id] = config.McsContract[0]

	return &Chain{
		conn:   conn,
		stop:   stop,
		listen: listen,
		cfg:    chainCfg,
		writer: newWriter(conn, config, logger, stop, sysErr),
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

// Conn return Connection interface for relayer register
func (c *Chain) Conn() core.Connection {
	return c.conn
}
