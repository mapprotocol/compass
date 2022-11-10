package writer

import (
	metrics "github.com/ChainSafe/chainbridge-utils/metrics/types"
	"github.com/ChainSafe/log15"
	"github.com/mapprotocol/compass/internal/chain"
	"github.com/mapprotocol/compass/msg"
)

type Writer struct {
	cfg     chain.Config
	conn    chain.Connection
	log     log15.Logger
	stop    <-chan int
	sysErr  chan<- error // Reports fatal error to core
	metrics *metrics.ChainMetrics
}

// New creates and returns Writer
func New(conn chain.Connection, cfg *chain.Config, log log15.Logger, stop <-chan int, sysErr chan<- error,
	m *metrics.ChainMetrics) *Writer {
	return &Writer{
		cfg:     *cfg,
		conn:    conn,
		log:     log,
		stop:    stop,
		sysErr:  sysErr,
		metrics: m,
	}
}

func (w *Writer) start() error {
	w.log.Debug("Starting Writer...")
	return nil
}

// ResolveMessage handles any given message based on type
// A bool is returned to indicate failure/success, this should be ignored except for within tests.
func (w *Writer) ResolveMessage(m msg.Message) bool {
	w.log.Info("Attempting to resolve message", "type", m.Type, "src", m.Source, "dst", m.Destination, "nonce", m.DepositNonce)

	switch m.Type {
	case msg.SyncToMap:
		return w.execToMapMsg(m)
	case msg.SyncFromMap:
		return w.execMap2OtherMsg(m)
	case msg.SwapTransfer:
		fallthrough
	case msg.SwapWithProof:
		fallthrough
	case msg.SwapWithMapProof:
		// same process
		return w.exeSwapMsg(m)
	default:
		w.log.Error("Unknown message type received", "type", m.Type)
		return false
	}
}
