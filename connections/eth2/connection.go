package eth2

import (
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"math/big"

	"github.com/mapprotocol/compass/core"

	"github.com/ChainSafe/log15"
	"github.com/mapprotocol/compass/connections/ethereum"
	"github.com/mapprotocol/compass/internal/eth2"
)

type Connection struct {
	endpoint, eth2Endpoint string
	core.Connection
	eth2Conn *eth2.Client
}

// NewConnection returns an uninitialized connection, must call Connection.Connect() before using.
func NewConnection(endpoint, eth2Endpoint string, http bool, kp *keystore.Key, log log15.Logger, gasLimit, gasPrice *big.Int,
	gasMultiplier float64) core.Eth2Connection {
	conn := ethereum.NewConnection(endpoint, http, kp, log, gasLimit, gasPrice, gasMultiplier)
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
