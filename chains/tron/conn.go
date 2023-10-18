package tron

import (
	"math/big"

	"google.golang.org/grpc"

	"github.com/ChainSafe/chainbridge-utils/crypto/secp256k1"
	"github.com/ChainSafe/log15"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	ethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/lbtsm/gotron-sdk/pkg/client"
	"github.com/mapprotocol/compass/pkg/ethclient"
)

type Connection struct {
	endpoint string
	cli      *client.GrpcClient
	log      log15.Logger
	stop     chan int
}

func NewConnection(endpoint string, log log15.Logger) *Connection {
	return &Connection{
		endpoint: endpoint,
		log:      log,
		stop:     make(chan int),
	}
}

// Connect starts the ethereum WS connection
func (c *Connection) Connect() error {
	c.log.Info("Connecting to ethereum chain...", "url", c.endpoint)
	c.cli = client.NewGrpcClient("grpc.nile.trongrid.io:50051")
	err := c.cli.Start(grpc.WithInsecure())
	if err != nil {
		return err
	}
	return nil
}

func (c *Connection) Keypair() *secp256k1.Keypair {
	return nil
}

func (c *Connection) Client() *ethclient.Client {
	return nil
}

func (c *Connection) Opts() *bind.TransactOpts {
	return nil
}

func (c *Connection) CallOpts() *bind.CallOpts {
	return nil
}

func (c *Connection) UnlockOpts() {
}

func (c *Connection) LockAndUpdateOpts(needNewNonce bool) error {
	return nil
}

// LatestBlock returns the latest block from the current chain
func (c *Connection) LatestBlock() (*big.Int, error) {
	bnum, err := c.cli.GetNowBlock()
	if err != nil {
		return nil, err
	}

	return big.NewInt(0).SetInt64(bnum.GetBlockHeader().GetRawData().Number), nil // todo
}

// EnsureHasBytecode asserts if contract code exists at the specified address
func (c *Connection) EnsureHasBytecode(addr ethcommon.Address) error {
	return nil
}

func (c *Connection) WaitForBlock(targetBlock *big.Int, delay *big.Int) error {
	return nil
}

func (c *Connection) Close() {
	if c.cli != nil {
		_ = c.cli.Conn.Close()
	}
	close(c.stop)
}
