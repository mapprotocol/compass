package ethereum

import (
	"errors"
	"math/big"
	"time"

	metrics "github.com/ChainSafe/chainbridge-utils/metrics/types"
	"github.com/ChainSafe/log15"
	"github.com/mapprotocol/compass/chains"
)

var BlockRetryInterval = time.Second * 5
var BlockRetryLimit = 5
var ErrFatalPolling = errors.New("listener block polling failed")
var MarkOfMaintainer = "maintainer"
var MarkOfMessenger = "messenger"

type CommonSync struct {
	cfg                Config
	conn               Connection
	log                log15.Logger
	router             chains.Router
	stop               <-chan int
	msgCh              chan struct{}
	sysErr             chan<- error // Reports fatal error to core
	latestBlock        metrics.LatestBlock
	metrics            *metrics.ChainMetrics
	blockConfirmations *big.Int
}

// NewCommonSync creates and returns a listener
func NewCommonSync(conn Connection, cfg *Config, log log15.Logger, stop <-chan int, sysErr chan<- error, m *metrics.ChainMetrics) *CommonSync {
	return &CommonSync{
		cfg:                *cfg,
		conn:               conn,
		log:                log,
		stop:               stop,
		sysErr:             sysErr,
		latestBlock:        metrics.LatestBlock{LastUpdated: time.Now()},
		metrics:            m,
		blockConfirmations: cfg.blockConfirmations,
		msgCh:              make(chan struct{}),
	}
}

func (c *CommonSync) SetRouter(r chains.Router) {
	c.router = r
}

func (c *CommonSync) GetLatestBlock() metrics.LatestBlock {
	return c.latestBlock
}

// waitUntilMsgHandled this function will block untill message is handled
func (c *CommonSync) waitUntilMsgHandled(counter int) error {
	c.log.Debug("waitUntilMsgHandled", "counter", counter)
	for counter > 0 {
		<-c.msgCh
		counter -= 1
	}
	return nil
}
