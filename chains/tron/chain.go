package tron

import (
	"fmt"
	"math/big"
	"strconv"
	gosync "sync"

	"github.com/mapprotocol/compass/internal/mapprotocol"
	"github.com/mapprotocol/compass/pkg/keystore"
	"github.com/mapprotocol/compass/pkg/msg"

	"github.com/ChainSafe/log15"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	connection "github.com/mapprotocol/compass/connections/ethereum"
	"github.com/mapprotocol/compass/core"
	"github.com/mapprotocol/compass/internal/chain"
	"github.com/mapprotocol/compass/internal/constant"
	"github.com/mapprotocol/compass/internal/proof"
	"github.com/mapprotocol/compass/internal/tx"
	"github.com/mapprotocol/compass/pkg/abi"
	"github.com/mapprotocol/compass/pkg/contract"
	"github.com/mapprotocol/compass/pkg/ethclient"
	"github.com/pkg/errors"
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

	pswd := make([]byte, 0)
	if role == mapprotocol.RoleOfMessenger {
		pswd = keystore.GetPassword(fmt.Sprintf("Enter password for key %s:", chainCfg.From))
	}

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
		handler := messenger
		if config.Filter {
			handler = filterMos
		}
		listen = newSync(cs, handler, conn)
	case mapprotocol.RoleOfOracle:
		oAbi, _ := abi.New(mapprotocol.SignerJson)
		oracleCall := contract.New(ethConn, []common.Address{config.OracleNode}, oAbi)
		mapprotocol.SingMapping[config.Id] = oracleCall

		otherAbi, _ := abi.New(mapprotocol.OtherAbi)
		call := contract.New(conn, []common.Address{common.HexToAddress(config.LightNode)}, otherAbi)
		mapprotocol.LightNodeMapping[config.Id] = call
		handler := oracle
		if config.Filter {
			handler = filterOracle
		}
		listen = newSync(cs, handler, conn)
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

func (c *Chain) Connect(id, endpoint, mcs, lightNode, oracleNode string) (*ethclient.Client, error) {
	conn := connection.NewConnection(endpoint, true, nil, nil, big.NewInt(chain.DefaultGasLimit),
		big.NewInt(chain.DefaultGasPrice), chain.DefaultGasMultiplier)
	err := conn.Connect()
	if err != nil {
		return nil, err
	}

	fn := gosync.OnceFunc(func() {
		idInt, _ := strconv.ParseUint(id, 10, 64)
		oracleAbi, _ := abi.New(mapprotocol.OracleAbiJson)
		call := contract.New(conn, []common.Address{common.HexToAddress(mcs)}, oracleAbi)
		mapprotocol.ContractMapping[msg.ChainId(idInt)] = call

		oAbi, _ := abi.New(mapprotocol.SignerJson)
		oracleCall := contract.New(conn, []common.Address{common.HexToAddress(oracleNode)}, oAbi)
		mapprotocol.SingMapping[msg.ChainId(idInt)] = oracleCall

		fn := mapprotocol.Map2EthHeight(constant.ZeroAddress.Hex(), common.HexToAddress(lightNode), conn.Client())
		mapprotocol.Map2OtherHeight[msg.ChainId(idInt)] = fn
	})
	fn()

	return conn.Client(), nil
}

func (c *Chain) Proof(client *ethclient.Client, l *types.Log, endpoint string, proofType int64, selfId,
	toChainID uint64, sign [][]byte) ([]byte, error) {
	orderId := l.Topics[1]
	var (
		bn       = big.NewInt(0).SetUint64(l.BlockNumber)
		receipts []*types.Receipt
		key      = strconv.FormatUint(selfId, 10) + "_" + strconv.FormatUint(l.BlockNumber, 10)
	)

	if v, ok := proof.CacheReceipt.Get(key); ok {
		receipts = v.([]*types.Receipt)
	} else {
		txsHash, err := getTxsByBN(client, bn)
		if err != nil {
			return nil, fmt.Errorf("unable to get tx hashes Logs: %w", err)
		}
		receipts, err = tx.GetReceiptsByTxsHash(client, txsHash)
		if err != nil {
			return nil, fmt.Errorf("unable to get receipts hashes Logs: %w", err)
		}
		proof.CacheReceipt.Add(key, receipts)
	}

	method := chain.GetMethod(l.Topics[0])
	proofType, err := chain.PreSendTx(0, selfId, toChainID, bn, orderId.Bytes())
	if err != nil {
		return nil, err
	}

	tmp := l
	input, err := assembleProof(tmp, receipts, method, msg.ChainId(selfId), msg.ChainId(toChainID), proofType)
	if err != nil {
		return nil, err
	}

	return input, nil
}

func (c *Chain) Maintainer(client *ethclient.Client, selfId, toChainId uint64, srcEndpoint string) ([]byte, error) {
	return nil, errors.New("tron not support maintainer")
}
