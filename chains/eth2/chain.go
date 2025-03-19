package eth2

import (
	"fmt"
	"github.com/ChainSafe/log15"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/mapprotocol/compass/connections/eth2"
	"github.com/mapprotocol/compass/core"
	"github.com/mapprotocol/compass/internal/chain"
	"github.com/mapprotocol/compass/internal/constant"
	ieth "github.com/mapprotocol/compass/internal/eth2"
	"github.com/mapprotocol/compass/internal/mapprotocol"
	"github.com/mapprotocol/compass/internal/tx"
	"github.com/mapprotocol/compass/pkg/abi"
	"github.com/mapprotocol/compass/pkg/contract"
	"github.com/mapprotocol/compass/pkg/ethclient"
	"github.com/mapprotocol/compass/pkg/keystore"
	"github.com/mapprotocol/compass/pkg/msg"
	"github.com/pkg/errors"
	"math/big"
	"strconv"
	"sync"
)

var _ core.Chain = new(Chain)

type Chain struct {
	cfg    *core.ChainConfig   // The config of the chain
	conn   core.Eth2Connection // The chains connection
	writer *chain.Writer       // The writer of the chain
	listen core.Listener       // The listener of this chain
	stop   chan<- int
}

func New() *Chain {
	return &Chain{}
}

func (c *Chain) New(chainCfg *core.ChainConfig, logger log15.Logger, sysErr chan<- error, role mapprotocol.Role) (core.Chain, error) {
	cfg, err := chain.ParseConfig(chainCfg)
	if err != nil {
		return nil, err
	}

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

	if chainCfg.LatestBlock {
		curr, err := conn.LatestBlock()
		if err != nil {
			return nil, err
		}
		cfg.StartBlock = curr
	}

	// simplified a little bit
	var listen core.Listener
	cs := chain.NewCommonSync(conn, cfg, logger, stop, sysErr, bs, chain.OptOfOracleHandler(chain.DefaultOracleHandler))
	switch role {
	case mapprotocol.RoleOfMaintainer:
		fn := mapprotocol.Map2EthHeight(cfg.From, cfg.LightNode, conn.Client())
		height, err := fn()
		if err != nil {
			return nil, errors.Wrap(err, "eth2 get init headerHeight failed")
		}
		logger.Info("map2eth2 Current situation", "height", height, "lightNode", cfg.LightNode)
		mapprotocol.SyncOtherMap[cfg.Id] = height
		mapprotocol.Map2OtherHeight[cfg.Id] = fn
		listen = NewMaintainer(cs, conn.Eth2Client())
	case mapprotocol.RoleOfMessenger:
		oracleAbi, _ := abi.New(mapprotocol.OracleAbiJson)
		call := contract.New(conn, cfg.McsContract, oracleAbi)
		mapprotocol.ContractMapping[cfg.Id] = call
		listen = NewMessenger(cs)
	case mapprotocol.RoleOfOracle:
		oAbi, _ := abi.New(mapprotocol.SignerJson)
		oracleCall := contract.New(conn, []common.Address{cfg.OracleNode}, oAbi)
		mapprotocol.SingMapping[cfg.Id] = oracleCall

		otherAbi, _ := abi.New(mapprotocol.OtherAbi)
		call := contract.New(conn, []common.Address{cfg.LightNode}, otherAbi)
		mapprotocol.LightNodeMapping[cfg.Id] = call
		listen = chain.NewOracle(cs)
	}
	wri := chain.NewWriter(conn, cfg, logger, stop, sysErr)

	return &Chain{
		cfg:    chainCfg,
		conn:   conn,
		writer: wri,
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

// EthClient return EthClient for global map connection
func (c *Chain) EthClient() *ethclient.Client {
	return c.conn.Client()
}

// Conn return Connection interface for relayer register
func (c *Chain) Conn() core.Connection {
	return c.conn
}

func (c *Chain) Connect(id, endpoint, mcs, lightNode, oracleNode string) (*ethclient.Client, error) {
	conn := eth2.NewConnection(endpoint, "", true, nil, nil, big.NewInt(chain.DefaultGasLimit),
		big.NewInt(chain.DefaultGasPrice), chain.DefaultGasMultiplier)
	err := conn.Connect()
	if err != nil {
		return nil, err
	}

	fn := sync.OnceFunc(func() {
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

func (c *Chain) Proof(client *ethclient.Client, log *types.Log, endpoint string, proofType int64, selfId,
	toChainID uint64, sign [][]byte) ([]byte, error) {
	var (
		orderId     = log.Topics[1]
		method      = chain.GetMethod(log.Topics[0])
		blockNumber = big.NewInt(0).SetUint64(log.BlockNumber)
	)

	header, err := client.EthLatestHeaderByNumber(endpoint, blockNumber)
	if err != nil {
		return nil, err
	}
	// when syncToMap we need to assemble a tx proof
	txsHash, err := mapprotocol.GetTxsByBn(client, blockNumber)
	if err != nil {
		return nil, fmt.Errorf("unable to get tx hashes Logs: %w", err)
	}
	receipts, err := tx.GetReceiptsByTxsHash(client, txsHash)
	if err != nil {
		return nil, fmt.Errorf("unable to get receipts hashes Logs: %w", err)
	}

	var orderId32 [32]byte
	for i, v := range orderId {
		orderId32[i] = v
	}

	ret, err := ieth.AssembleProof(*ieth.ConvertHeader(header), log, receipts, method, msg.ChainId(selfId), proofType, sign, orderId32)
	if err != nil {
		return nil, fmt.Errorf("unable to Parse Log: %w", err)
	}

	return ret, nil
}

func (c *Chain) Maintainer(client *ethclient.Client, selfId, toChainId uint64, srcEndpoint string) ([]byte, error) {
	return nil, errors.New("eth not support maintainer")
}
