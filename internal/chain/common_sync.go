package chain

import (
	"math/big"
	"time"

	"github.com/mapprotocol/compass/core"

	eth "github.com/ethereum/go-ethereum"
	ethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/mapprotocol/compass/mapprotocol"
	utils "github.com/mapprotocol/compass/shared/ethereum"

	metrics "github.com/ChainSafe/chainbridge-utils/metrics/types"
	"github.com/ChainSafe/log15"
	"github.com/mapprotocol/compass/chains"
	"github.com/mapprotocol/compass/pkg/blockstore"
)

type (
	SyncOpt        func(*CommonSync)
	SyncMap2Other  func(*Maintainer, *big.Int) error
	SyncHeader2Map func(*Maintainer, *big.Int) error
	Mos            func(*Messenger, *big.Int) (int, error)
)

func OptOfInitHeight(height int64) SyncOpt {
	return func(sync *CommonSync) {
		sync.height = height
	}
}

func OptOfSync2Map(fn SyncHeader2Map) SyncOpt {
	return func(sync *CommonSync) {
		sync.syncHeaderToMap = fn
	}
}

func OptOfMos(fn Mos) SyncOpt {
	return func(sync *CommonSync) {
		sync.mosHandler = fn
	}
}

type CommonSync struct {
	Cfg                Config
	Conn               core.Connection
	Log                log15.Logger
	Router             chains.Router
	Stop               <-chan int
	MsgCh              chan struct{}
	SysErr             chan<- error // Reports fatal error to core
	LatestBlock        metrics.LatestBlock
	Metrics            *metrics.ChainMetrics
	BlockConfirmations *big.Int
	BlockStore         blockstore.Blockstorer
	height             int64
	syncHeaderToMap    SyncHeader2Map
	mosHandler         Mos
}

// NewCommonSync creates and returns a listener
func NewCommonSync(conn core.Connection, cfg *Config, log log15.Logger, stop <-chan int, sysErr chan<- error,
	m *metrics.ChainMetrics, bs blockstore.Blockstorer, opts ...SyncOpt) *CommonSync {
	cs := &CommonSync{
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
		height:             1,
	}
	for _, op := range opts {
		op(cs)
	}

	return cs
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

// BuildQuery constructs a query for the bridgeContract by hashing sig to get the event topic
func (c *CommonSync) BuildQuery(contract ethcommon.Address, sig []utils.EventSig, startBlock *big.Int, endBlock *big.Int) eth.FilterQuery {
	topics := make([]ethcommon.Hash, 0, len(sig))
	for _, s := range sig {
		topics = append(topics, s.GetTopic())
	}
	query := eth.FilterQuery{
		FromBlock: startBlock,
		ToBlock:   endBlock,
		Addresses: []ethcommon.Address{contract},
		Topics:    [][]ethcommon.Hash{topics},
	}
	return query
}

func (c *CommonSync) GetMethod(topic ethcommon.Hash) string {
	method := mapprotocol.MethodOfTransferIn
	if topic == mapprotocol.HashOfDepositIn {
		method = mapprotocol.MethodOfDepositIn
	} else if topic == mapprotocol.HashOfSwapIn {
		method = mapprotocol.MethodOfSwapIn
	}

	return method
}
