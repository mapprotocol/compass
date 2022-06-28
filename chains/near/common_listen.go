package near

import (
	"errors"
	"math/big"
	"time"

	metrics "github.com/ChainSafe/chainbridge-utils/metrics/types"
	"github.com/ChainSafe/log15"
	"github.com/mapprotocol/compass/blockstore"
	"github.com/mapprotocol/compass/chains"
)

var (
	BlockRetryInterval = time.Second * 5
	BlockRetryLimit    = 5
	ErrFatalPolling    = errors.New("listener block polling failed")
)

type CommonListen struct {
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
	blockStore         blockstore.Blockstorer
}

// NewCommonListen creates and returns a listener
func NewCommonListen(conn Connection, cfg *Config, log log15.Logger, stop <-chan int, sysErr chan<- error,
	m *metrics.ChainMetrics, bs blockstore.Blockstorer) *CommonListen {
	return &CommonListen{
		cfg:                *cfg,
		conn:               conn,
		log:                log,
		stop:               stop,
		sysErr:             sysErr,
		latestBlock:        metrics.LatestBlock{LastUpdated: time.Now()},
		metrics:            m,
		blockConfirmations: cfg.blockConfirmations,
		msgCh:              make(chan struct{}),
		blockStore:         bs,
	}
}

func (c *CommonListen) SetRouter(r chains.Router) {
	c.router = r
}

func (c *CommonListen) GetLatestBlock() metrics.LatestBlock {
	return c.latestBlock
}

// waitUntilMsgHandled this function will block untill message is handled
func (c *CommonListen) waitUntilMsgHandled(counter int) error {
	c.log.Debug("waitUntilMsgHandled", "counter", counter)
	for counter > 0 {
		<-c.msgCh
		counter -= 1
	}
	return nil
}
