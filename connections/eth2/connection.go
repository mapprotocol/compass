package eth2

import (
	"math/big"

	"github.com/ChainSafe/chainbridge-utils/crypto/secp256k1"
	"github.com/ChainSafe/log15"
	"github.com/mapprotocol/compass/connections/ethereum"
	"github.com/mapprotocol/compass/internal/chain"
	"github.com/mapprotocol/compass/internal/eth2"
)

type Connection struct {
	endpoint, eth2Endpoint string
	chain.Connection
	eth2Conn *eth2.Client
}

// NewConnection returns an uninitialized connection, must call Connection.Connect() before using.
func NewConnection(endpoint, eth2Endpoint string, http bool, kp *secp256k1.Keypair, log log15.Logger, gasLimit, gasPrice *big.Int,
	gasMultiplier float64, gsnApiKey, gsnSpeed string) chain.Eth2Connection {
	conn := ethereum.NewConnection(endpoint, http, kp, log, gasLimit, gasPrice, gasMultiplier, gsnApiKey, gsnSpeed)
	return &Connection{
		Connection:   conn,
		endpoint:     endpoint,
		eth2Endpoint: eth2Endpoint,
	}
}

func (c *Connection) Eth2Client() *eth2.Client {
	return c.eth2Conn
}

// Connect starts the ethereum WS connection
func (c *Connection) Connect() error {
	if err := c.Connection.Connect(); err != nil {
		return err
	}

	client, err := eth2.DialHttp(c.eth2Endpoint)
	if err != nil {
		return err
	}
	c.eth2Conn = client
	return nil
}
