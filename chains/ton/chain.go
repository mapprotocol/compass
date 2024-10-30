package ton

import (
	"github.com/ChainSafe/log15"
	"github.com/ethereum/go-ethereum/log"
	"github.com/mapprotocol/compass/chains"
	"github.com/mapprotocol/compass/connections/eth2"
	"github.com/mapprotocol/compass/core"
	"github.com/mapprotocol/compass/internal/chain"
	"github.com/mapprotocol/compass/keystore"
	"github.com/mapprotocol/compass/mapprotocol"
	"github.com/mapprotocol/compass/msg"
	"github.com/mapprotocol/compass/pkg/abi"
	"github.com/mapprotocol/compass/pkg/contract"
)

var _ core.Chain = new(Chain)

type Chain struct {
	cfg    *core.ChainConfig   // The config of the chain
	conn   core.Eth2Connection // The chains connection
	writer *Writer             // The writer of the chain
	listen chains.Listener     // The listener of this chain
	stop   chan<- int
}

func New(chainCfg *core.ChainConfig, logger log15.Logger, sysErr chan<- error, role mapprotocol.Role) (*Chain, error) {
	cfg, err := chain.ParseConfig(chainCfg)
	if err != nil {
		return nil, err
	}
	// todo ton 自己解析配置，密钥，ton链接

	kpI, err := keystore.KeypairFromEth(cfg.KeystorePath)
	if err != nil {
		return nil, err
	}
	//kp, _ := kpI.(*secp256k1.Keypair)
	bs, err := chain.SetupBlockStore(cfg, role)
	if err != nil {
		return nil, err
	}

	stop := make(chan int)
	conn := eth2.NewConnection(cfg.Endpoint, cfg.Eth2Endpoint, cfg.Http, kpI, logger, cfg.GasLimit, cfg.MaxGasPrice,
		cfg.GasMultiplier)
	err = conn.Connect()
	if err != nil {
		return nil, err
	}

	// simplified a little bit
	var listen chains.Listener
	cs := chain.NewCommonSync(conn, cfg, logger, stop, sysErr, bs, chain.OptOfOracleHandler(chain.DefaultOracleHandler))
	switch role {

	case mapprotocol.RoleOfMessenger:
		oracleAbi, _ := abi.New(mapprotocol.OracleAbiJson)
		call := contract.New(conn, cfg.McsContract, oracleAbi)
		mapprotocol.ContractMapping[cfg.Id] = call
		listen = NewMessenger(cs) // todo ton
	case mapprotocol.RoleOfOracle:
		listen = chain.NewOracle(cs)
	}
	wri := newWriter(conn, config, logger, stop, sysErr) // todo ton

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
