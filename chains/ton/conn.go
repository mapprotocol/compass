package ton

import (
	"context"
	"math/big"
	"strings"

	"github.com/ChainSafe/log15"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	ethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/xssnick/tonutils-go/liteclient"
	"github.com/xssnick/tonutils-go/ton"
	tonwallet "github.com/xssnick/tonutils-go/ton/wallet"

	"github.com/mapprotocol/compass/pkg/ethclient"
)

type Connection struct {
	endpoint string
	words    string
	password string
	kp       *keystore.Key
	client   ton.APIClientWrapped
	wallet   *tonwallet.Wallet
	log      log15.Logger
	stop     chan int
}

func NewConnection(endpoint, words, password string, kp *keystore.Key, log log15.Logger) *Connection {
	return &Connection{
		endpoint: endpoint,
		words:    words,
		password: password,
		kp:       kp,
		log:      log,
		stop:     make(chan int),
	}
}

func (c *Connection) Connect() error {
	cfg, err := liteclient.GetConfigFromUrl(context.Background(), c.endpoint)
	if err != nil {
		return err
	}

	pool := liteclient.NewConnectionPool()
	err = pool.AddConnectionsFromConfig(context.Background(), cfg)
	if err != nil {
		return err
	}
	cli := ton.NewAPIClient(pool, ton.ProofCheckPolicySecure).WithRetry()
	cli.SetTrustedBlockFromConfig(cfg)
	c.client = cli

	// seed words of account, you can generate them with any wallet or using wallet.NewSeed() method
	seed := strings.Split(c.words, " ")

	w, err := tonwallet.FromSeedWithPassword(cli, seed, c.password, tonwallet.V3R2)
	if err != nil {
		return err
	}
	c.wallet = w
	return nil
}

func (c *Connection) Keypair() *keystore.Key {
	return c.kp
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
	return big.NewInt(0), nil
}

// EnsureHasBytecode asserts if contract code exists at the specified address
func (c *Connection) EnsureHasBytecode(addr ethcommon.Address) error {
	return nil
}

func (c *Connection) WaitForBlock(targetBlock *big.Int, delay *big.Int) error {
	return nil
}

func (c *Connection) Close() {
	close(c.stop)
}
