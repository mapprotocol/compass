package tron

import (
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/mapprotocol/compass/pkg/abi"
	"github.com/mapprotocol/compass/pkg/contract"

	connection "github.com/mapprotocol/compass/connections/ethereum"
	"github.com/mapprotocol/compass/keystore"

	"github.com/ChainSafe/log15"
	"github.com/ethereum/go-ethereum/log"
	"github.com/lbtsm/gotron-sdk/pkg/client"
	"github.com/mapprotocol/compass/core"
	"github.com/mapprotocol/compass/internal/chain"
	"github.com/mapprotocol/compass/mapprotocol"
	"github.com/mapprotocol/compass/msg"
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
	return c.createChain(chainCfg, logger, sysErr, role)
}

func (c *Chain) createChain(chainCfg *core.ChainConfig, logger log15.Logger, sysErr chan<- error, role mapprotocol.Role, opts ...chain.SyncOpt) (core.Chain, error) {
	config, err := parseCfg(chainCfg)
	if err != nil {
		return nil, err
	}

	conn := NewConnection(config.Endpoint, logger)
	err = conn.Connect()
	if err != nil {
		return nil, err
	}

	ethConn := connection.NewConnection(config.Eth2Endpoint, true, nil, logger, config.GasLimit, config.MaxGasPrice, 0)
	err = ethConn.Connect()
	if err != nil {
		return nil, err
	}

	pswd := keystore.GetPassword(fmt.Sprintf("Enter password for key %s:", chainCfg.From))

	var (
		stop   = make(chan int)
		listen core.Listener
	)
	bs, err := chain.SetupBlockStore(&config.Config, role)
	if err != nil {
		return nil, err
	}
	cs := chain.NewCommonSync(ethConn, &config.Config, logger, stop, sysErr, bs)

	switch role {
	case mapprotocol.RoleOfMessenger:
		listen = newSync(cs, messengerHandler, conn)
	case mapprotocol.RoleOfOracle:
		oAbi, _ := abi.New(mapprotocol.SignerJson)
		oracleCall := contract.New(ethConn, []common.Address{config.OracleNode}, oAbi)
		mapprotocol.SingMapping[config.Id] = oracleCall

		otherAbi, _ := abi.New(mapprotocol.OtherAbi)
		call := contract.New(conn, []common.Address{common.HexToAddress(config.LightNode)}, otherAbi)
		mapprotocol.LightNodeMapping[config.Id] = call
		listen = newSync(cs, oracleHandler, conn)
	}

	return &Chain{
		conn:   conn,
		stop:   stop,
		listen: listen,
		cfg:    chainCfg,
		writer: newWriter(conn, config, logger, stop, sysErr, pswd),
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

func Map2Tron(fromUser, lightNode string, client *client.GrpcClient) mapprotocol.GetHeight {
	return func() (*big.Int, error) {
		call, err := client.TriggerConstantContract(fromUser, lightNode, "headerHeight()", "")
		if err != nil {
			return nil, fmt.Errorf("map2tron call headerHeight failed, err is %v", err.Error())
		}
		return mapprotocol.UnpackHeaderHeightOutput(call.ConstantResult[0])
	}
}
