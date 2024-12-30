package ton

import (
	"github.com/ChainSafe/log15"
	"github.com/ethereum/go-ethereum/log"

	"github.com/mapprotocol/compass/core"
	"github.com/mapprotocol/compass/internal/chain"
	"github.com/mapprotocol/compass/keystore"
	"github.com/mapprotocol/compass/mapprotocol"
	"github.com/mapprotocol/compass/msg"
)

var _ core.Chain = new(Chain)

type Chain struct {
	cfg    *core.ChainConfig // The config of the chain
	conn   core.Connection   // The chains connection
	writer *Writer           // The writer of the chain
	listen core.Listener     // The listener of this chain
	stop   chan<- int
}

func New() *Chain {
	return &Chain{}
}

func (c *Chain) New(chainCfg *core.ChainConfig, logger log15.Logger, sysErr chan<- error, role mapprotocol.Role) (core.Chain, error) {
	cfg, err := parseConfig(chainCfg)
	if err != nil {
		return nil, err
	}

	kpI, err := keystore.KeypairFromEth(cfg.KeystorePath)
	if err != nil {
		return nil, err
	}

	password := keystore.GetPassword("Enter password for TON words:")
	conn := NewConnection(cfg.Endpoint, cfg.Words, string(password), kpI, logger)
	err = conn.Connect()
	if err != nil {
		return nil, err
	}

	var (
		stop   = make(chan int)
		listen core.Listener
	)
	bs, err := chain.SetupBlockStore(&cfg.Config, role)
	if err != nil {
		return nil, err
	}
	cs := chain.NewCommonSync(nil, &cfg.Config, logger, stop, sysErr, bs)

	switch role {
	case mapprotocol.RoleOfMessenger:
		listen = newSync(cs, messengerHandler, conn, cfg)
	case mapprotocol.RoleOfOracle:
		listen = newSync(cs, oracleHandler, conn, cfg)
	}
	mapprotocol.MosMapping[cfg.Id] = cfg.McsContract[0]

	return &Chain{
		cfg:    chainCfg,
		conn:   conn,
		writer: newWriter(conn, cfg, logger, stop, sysErr),
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

	log.Debug("Successfully started chain")
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
