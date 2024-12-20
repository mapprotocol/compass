package sol

import (
	"context"
	"math/big"
	"time"

	"github.com/ChainSafe/log15"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	ethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/gagliardetto/solana-go/rpc"
	"github.com/mapprotocol/compass/internal/constant"
	"github.com/mapprotocol/compass/pkg/ethclient"
)

type Connection struct {
	endpoint                  string
	cli                       *rpc.Client
	log                       log15.Logger
	stop                      chan int
	reqTime, cacheBlockNumber int64
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
	c.log.Info("Connecting to tron chain...", "url", c.endpoint)
	c.cli = rpc.New(c.endpoint)
	return nil
}

func (c *Connection) Keypair() *keystore.Key {
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
	// 1s req
	if time.Now().Unix()-c.reqTime < constant.ReqInterval {
		return big.NewInt(0).SetInt64(c.cacheBlockNumber), nil
	}

	bnum, err := c.cli.GetBlockHeight(context.Background(), rpc.CommitmentFinalized)
	if err != nil {
		return nil, err
	}
	c.cacheBlockNumber = int64(bnum)
	c.reqTime = time.Now().Unix()

	return big.NewInt(0).SetInt64(c.cacheBlockNumber), nil
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
		_ = c.cli.Close()
	}
	close(c.stop)
}