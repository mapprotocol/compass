package bsc

import (
	"math/big"
	"time"

	metrics "github.com/ChainSafe/chainbridge-utils/metrics/types"
	"github.com/ChainSafe/log15"
	"github.com/mapprotocol/compass/blockstore"
	"github.com/mapprotocol/compass/chains"
	"github.com/mapprotocol/compass/internal/chain"
)

type CommonSync struct {
	cfg                chain.Config
	conn               chain.Connection
	log                log15.Logger
	router             chains.Router
	stop               <-chan int
	msgCh              chan struct{}
	sysErr             chan<- error // Reports fatal error to core
	latestBlock        metrics.LatestBlock
	metrics            *metrics.ChainMetrics
	blockConfirmations *big.Int
	blockStore         blockstore.Blockstorer
}

// NewCommonSync creates and returns a listener
func NewCommonSync(conn chain.Connection, cfg *chain.Config, log log15.Logger, stop <-chan int, sysErr chan<- error,
	m *metrics.ChainMetrics, bs blockstore.Blockstorer) *CommonSync {
	return &CommonSync{
		cfg:                *cfg,
		conn:               conn,
		log:                log,
		stop:               stop,
		sysErr:             sysErr,
		latestBlock:        metrics.LatestBlock{LastUpdated: time.Now()},
		metrics:            m,
		blockConfirmations: cfg.BlockConfirmations,
		msgCh:              make(chan struct{}),
		blockStore:         bs,
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
