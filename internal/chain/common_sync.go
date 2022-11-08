package chain

import (
	"math/big"
	"time"

	metrics "github.com/ChainSafe/chainbridge-utils/metrics/types"
	"github.com/ChainSafe/log15"
	"github.com/mapprotocol/compass/blockstore"
	"github.com/mapprotocol/compass/chains"
)

type CommonSync struct {
	Cfg                Config
	Conn               Connection
	Log                log15.Logger
	Router             chains.Router
	Stop               <-chan int
	MsgCh              chan struct{}
	SysErr             chan<- error // Reports fatal error to core
	LatestBlock        metrics.LatestBlock
	Metrics            *metrics.ChainMetrics
	BlockConfirmations *big.Int
	BlockStore         blockstore.Blockstorer
}

// NewCommonSync creates and returns a listener
func NewCommonSync(conn Connection, cfg *Config, log log15.Logger, stop <-chan int, sysErr chan<- error,
	m *metrics.ChainMetrics, bs blockstore.Blockstorer) *CommonSync {
	return &CommonSync{
		Cfg:                *cfg,
		Conn:               conn,
		Log:                log,
		Stop:               stop,
		SysErr:             sysErr,
		LatestBlock:        metrics.LatestBlock{LastUpdated: time.Now()},
		Metrics:            m,
		BlockConfirmations: cfg.BlockConfirmations,
		MsgCh:              make(chan struct{}),
		BlockStore:         bs,
	}
}

func (c *CommonSync) SetRouter(r chains.Router) {
	c.Router = r
}

func (c *CommonSync) GetLatestBlock() metrics.LatestBlock {
	return c.LatestBlock
}

// WaitUntilMsgHandled this function will block untill message is handled
func (c *CommonSync) WaitUntilMsgHandled(counter int) error {
	c.Log.Debug("WaitUntilMsgHandled", "counter", counter)
	for counter > 0 {
		<-c.MsgCh
		counter -= 1
	}
	return nil
}
