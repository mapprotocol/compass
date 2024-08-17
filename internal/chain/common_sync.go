package chain

import (
	"fmt"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/mapprotocol/compass/internal/constant"
	"github.com/mapprotocol/compass/msg"
	"math/big"
	"time"

	"github.com/mapprotocol/compass/core"

	"github.com/ChainSafe/log15"
	eth "github.com/ethereum/go-ethereum"
	ethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/mapprotocol/compass/chains"
	"github.com/mapprotocol/compass/mapprotocol"
	"github.com/mapprotocol/compass/pkg/blockstore"
)

type (
	SyncOpt        func(*CommonSync)
	SyncHeader2Map func(*Maintainer, *big.Int) error
	Mos            func(*Messenger, *big.Int) (int, error)
	AssembleProof  func(*Messenger, *types.Log, int64, uint64, [][]byte) (*msg.Message, error)
	OracleHandler  func(*Oracle, *big.Int) error
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

func OptOfOracleHandler(fn OracleHandler) SyncOpt {
	return func(sync *CommonSync) {
		sync.oracleHandler = fn
	}
}

func OptOfAssembleProof(fn AssembleProof) SyncOpt {
	return func(sync *CommonSync) {
		sync.assembleProof = fn
	}
}

type CommonSync struct {
	Cfg                       Config
	Conn                      core.Connection
	Log                       log15.Logger
	Router                    chains.Router
	Stop                      <-chan int
	MsgCh                     chan struct{}
	SysErr                    chan<- error // Reports fatal error to core
	BlockConfirmations        *big.Int
	BlockStore                blockstore.Blockstorer
	height                    int64
	syncHeaderToMap           SyncHeader2Map
	mosHandler                Mos
	oracleHandler             OracleHandler
	assembleProof             AssembleProof
	reqTime, cacheBlockNumber int64
}

// NewCommonSync creates and returns a listener
func NewCommonSync(conn core.Connection, cfg *Config, log log15.Logger, stop <-chan int, sysErr chan<- error,
	bs blockstore.Blockstorer, opts ...SyncOpt) *CommonSync {
	cs := &CommonSync{
		Cfg:                *cfg,
		Conn:               conn,
		Log:                log,
		Stop:               stop,
		SysErr:             sysErr,
		BlockConfirmations: cfg.BlockConfirmations,
		MsgCh:              make(chan struct{}),
		BlockStore:         bs,
		height:             1,
		mosHandler:         defaultMosHandler,
	}
	for _, op := range opts {
		op(cs)
	}

	return cs
}

func (c *CommonSync) SetRouter(r chains.Router) {
	c.Router = r
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
func (c *CommonSync) BuildQuery(contract ethcommon.Address, sig []constant.EventSig, startBlock *big.Int, endBlock *big.Int) eth.FilterQuery {
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
	} else if topic == mapprotocol.HashOfMessageIn {
		method = mapprotocol.MethodOfTransferInWithIndex
	}

	return method
}

func (c *CommonSync) FilterLatestBlock() (*big.Int, error) {
	if time.Now().Unix()-c.reqTime < constant.ReqInterval {
		return big.NewInt(c.cacheBlockNumber), nil
	}
	data, err := Request(fmt.Sprintf("%s/%s", c.Cfg.FilterHost, fmt.Sprintf("%s?chain_id=%d", constant.FilterBlockUrl, c.Cfg.Id)))
	if err != nil {
		c.Log.Error("Unable to get latest block", "err", err)
		time.Sleep(constant.BlockRetryInterval)
		return nil, err
	}
	c.Log.Debug("Filter latest block", "block", data)
	latestBlock, ok := big.NewInt(0).SetString(data.(string), 10)
	if !ok {
		return nil, fmt.Errorf("get latest failed, block is %v", data)
	}
	c.cacheBlockNumber = latestBlock.Int64()
	c.reqTime = time.Now().Unix()
	return latestBlock, nil
}

func (c *CommonSync) Match(target string) int {
	for idx, ele := range c.Cfg.McsContract {
		if ele.Hex() == target {
			return idx
		}
	}
	return -1
}
