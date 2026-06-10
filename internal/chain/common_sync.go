package chain

import (
	"fmt"

	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/mapprotocol/compass/internal/constant"
	"github.com/mapprotocol/compass/internal/mapprotocol"
	"github.com/mapprotocol/compass/internal/observability"
	"github.com/mapprotocol/compass/internal/stream"
	"github.com/mapprotocol/compass/pkg/msg"

	"github.com/mapprotocol/compass/core"

	"github.com/ChainSafe/log15"
	eth "github.com/ethereum/go-ethereum"
	ethcommon "github.com/ethereum/go-ethereum/common"
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

func OptOfFilterClient(client FilterClient) SyncOpt {
	return func(sync *CommonSync) {
		sync.filterClient = client
	}
}

type CommonSync struct {
	Cfg                       Config
	Conn                      core.Connection
	Log                       log15.Logger
	Router                    core.Router
	Stop                      <-chan int
	MsgCh                     chan struct{}
	SysErr                    chan<- error // Reports fatal error to core
	BlockConfirmations        *big.Int
	BlockStore                blockstore.Blockstorer
	State                     *observability.ChainState // set by chain constructor once role is known
	height                    int64
	syncHeaderToMap           SyncHeader2Map
	mosHandler                Mos
	oracleHandler             OracleHandler
	assembleProof             AssembleProof
	filterClient              FilterClient
	reqTime, cacheBlockNumber int64
}

// RegisterState wires this CommonSync to the package-level Observability. Safe
// to call zero or more times; nil-safety lives on every State.* method, so
// loops that ran before this was wired stay panic-free.
func (c *CommonSync) RegisterState(chain, role string) {
	c.State = observability.RegisterChain(chain, role)
}

func (c *CommonSync) FilterClient() FilterClient {
	return c.filterClient
}

func (c *CommonSync) ListMosLogs(projectID int64, topic string, limit int) (*stream.MosListResp, error) {
	if !c.Cfg.Filter {
		return nil, fmt.Errorf("filter mode disabled")
	}
	filterClient := c.FilterClient()
	if filterClient == nil {
		return nil, fmt.Errorf("filter client is nil")
	}
	return filterClient.ListMosLogs(FilterListRequest{
		StartID:   c.Cfg.StartBlock,
		ProjectID: projectID,
		ChainID:   int64(c.Cfg.Id),
		Topic:     topic,
		Limit:     limit,
	})
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
	if cfg.Filter {
		cs.filterClient = NewRadarFilterClient(cfg.FilterHost, cfg.FilterAPIKey)
	}
	for _, op := range opts {
		op(cs)
	}

	return cs
}

func (c *CommonSync) SetRouter(r core.Router) {
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
	return mapprotocol.MethodOfMessageIn
}

func (c *CommonSync) FilterLatestBlock() (*big.Int, error) {
	if !c.Cfg.Filter {
		if c.Conn == nil {
			return nil, fmt.Errorf("connection is nil")
		}
		return c.Conn.LatestBlock()
	}
	if time.Now().Unix()-c.reqTime < constant.ReqInterval {
		return big.NewInt(c.cacheBlockNumber), nil
	}
	filterClient := c.FilterClient()
	if filterClient == nil {
		return nil, fmt.Errorf("filter client is nil")
	}
	latestBlock, err := filterClient.LatestBlock(int64(c.Cfg.Id))
	if err != nil {
		c.Log.Error("Unable to get latest block", "err", err)
		time.Sleep(constant.BlockRetryInterval)
		return nil, err
	}
	c.Log.Debug("Filter latest block", "block", latestBlock)
	c.cacheBlockNumber = latestBlock.Int64()
	c.reqTime = time.Now().Unix()
	return latestBlock, nil
}

func (c *CommonSync) FilterMaxID() (*big.Int, error) {
	if !c.Cfg.Filter {
		return nil, fmt.Errorf("filter mode disabled")
	}
	filterClient := c.FilterClient()
	if filterClient == nil {
		return nil, fmt.Errorf("filter client is nil")
	}
	maxID, err := filterClient.MaxID(int64(c.Cfg.Id))
	if err != nil {
		return nil, err
	}
	c.Log.Debug("Filter max id", "id", maxID)
	return maxID, nil
}

func StartLatestBlock(cfg *Config, conn core.Connection, logger log15.Logger) error {
	if cfg.Filter {
		cs := &CommonSync{
			Cfg:          *cfg,
			Log:          logger,
			filterClient: NewRadarFilterClient(cfg.FilterHost, cfg.FilterAPIKey),
		}
		maxID, err := cs.FilterMaxID()
		if err != nil {
			return err
		}
		cfg.StartBlock = maxID
		logger.Info("Start from latest filter id", "id", cfg.StartBlock)
		return nil
	}

	if conn == nil {
		return fmt.Errorf("connection is nil")
	}
	curr, err := conn.LatestBlock()
	if err != nil {
		return err
	}
	cfg.StartBlock = curr
	logger.Info("Start from latest block", "block", cfg.StartBlock)
	return nil
}

func parseFilterNumber(data interface{}) (*big.Int, error) {
	switch v := data.(type) {
	case string:
		n, ok := big.NewInt(0).SetString(v, 10)
		if !ok {
			return nil, fmt.Errorf("invalid number %q", v)
		}
		return n, nil
	case float64:
		return big.NewInt(int64(v)), nil
	case map[string]interface{}:
		for _, key := range []string{"max_id", "maxId", "id", "block", "block_number", "blockNumber"} {
			if val, ok := v[key]; ok {
				return parseFilterNumber(val)
			}
		}
		return nil, fmt.Errorf("missing max id field")
	default:
		return nil, fmt.Errorf("unsupported number type %T", data)
	}
}

func (c *CommonSync) Match(target string) int {
	for idx, ele := range append(c.Cfg.McsContract, c.Cfg.LightNode) {
		if ele.Hex() == target {
			return idx
		}
	}
	return -1
}

func GetMethod(topic ethcommon.Hash) string {
	def := &CommonSync{}
	return def.GetMethod(topic)
}
