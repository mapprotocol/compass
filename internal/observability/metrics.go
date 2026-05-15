// Package observability bundles Prometheus metrics, a /status JSON snapshot,
// pprof endpoints, and a small alarm rule engine into one HTTP server.
//
// Intended usage from main():
//
//	obs := observability.New("radar", observability.Config{Addr: ":9101"})
//	obs.StartHTTP()
//	defer obs.Stop()
//	chainState := obs.RegisterChain("eth")    // returned holder updated by sync loops
//	chainState.SetCurrentBlock(123)
//	chainState.SetLatestBlock(456)
//	chainState.IncBlocksProcessed()
//	obs.RPCLatency.WithLabelValues("eth", "LatestBlock").Observe(secs)
//
// Two binaries embed two independent registries (no cross-leakage).
package observability

import (
	"sync"

	"github.com/prometheus/client_golang/prometheus"
)

// Metrics holds the Prometheus collectors. One instance per Observability.
type Metrics struct {
	CurrentBlock    *prometheus.GaugeVec     // labels: chain, role
	LatestBlock     *prometheus.GaugeVec     // labels: chain, role
	BlockLag        *prometheus.GaugeVec     // labels: chain, role  (latest - current)
	LastProgressTs  *prometheus.GaugeVec     // labels: chain, role  (unix seconds)
	BlocksProcessed *prometheus.CounterVec   // labels: chain, role
	EventsMatched   *prometheus.CounterVec   // labels: chain, role
	RPCLatency      *prometheus.HistogramVec // labels: chain, method
	DBInsertLatency *prometheus.HistogramVec // labels: table
	ProcessLatency  *prometheus.HistogramVec // labels: chain, stage (e.g. filter,match,insert)
	ErrorsTotal     *prometheus.CounterVec   // labels: chain, kind
	InFlight        *prometheus.GaugeVec     // labels: scope (router, signer, etc.)

	reg *prometheus.Registry
}

func newMetrics(namespace string) *Metrics {
	reg := prometheus.NewRegistry()
	m := &Metrics{reg: reg}

	m.CurrentBlock = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: namespace, Subsystem: "chain", Name: "current_block",
		Help: "Last block this chain loop has processed.",
	}, []string{"chain", "role"})

	m.LatestBlock = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: namespace, Subsystem: "chain", Name: "latest_block",
		Help: "Latest block reported by the chain RPC.",
	}, []string{"chain", "role"})

	m.BlockLag = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: namespace, Subsystem: "chain", Name: "block_lag",
		Help: "latest_block - current_block. Primary 'is this chain slow' indicator.",
	}, []string{"chain", "role"})

	m.LastProgressTs = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: namespace, Subsystem: "chain", Name: "last_progress_unixtime",
		Help: "Unix timestamp of the last time current_block moved forward.",
	}, []string{"chain", "role"})

	m.BlocksProcessed = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace, Subsystem: "chain", Name: "blocks_processed_total",
		Help: "Total blocks the loop has finished processing.",
	}, []string{"chain", "role"})

	m.EventsMatched = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace, Subsystem: "chain", Name: "events_matched_total",
		Help: "Total log/tx events matched against filter rules.",
	}, []string{"chain", "role"})

	m.RPCLatency = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: namespace, Subsystem: "rpc", Name: "duration_seconds",
		Help:    "Duration of outbound RPC calls.",
		Buckets: prometheus.ExponentialBuckets(0.01, 2, 12), // 10ms .. ~40s
	}, []string{"chain", "method"})

	m.DBInsertLatency = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: namespace, Subsystem: "db", Name: "insert_duration_seconds",
		Help:    "Duration of DB write operations.",
		Buckets: prometheus.ExponentialBuckets(0.001, 2, 12), // 1ms .. ~4s
	}, []string{"table"})

	m.ProcessLatency = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: namespace, Subsystem: "process", Name: "duration_seconds",
		Help:    "Duration of internal processing stages.",
		Buckets: prometheus.ExponentialBuckets(0.001, 2, 14),
	}, []string{"chain", "stage"})

	m.ErrorsTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace, Subsystem: "errors", Name: "total",
		Help: "Errors observed, bucketed by chain and kind.",
	}, []string{"chain", "kind"})

	m.InFlight = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: namespace, Subsystem: "queue", Name: "in_flight",
		Help: "Currently in-flight units of work (e.g. router messages).",
	}, []string{"scope"})

	for _, c := range []prometheus.Collector{
		m.CurrentBlock, m.LatestBlock, m.BlockLag, m.LastProgressTs,
		m.BlocksProcessed, m.EventsMatched, m.RPCLatency, m.DBInsertLatency,
		m.ProcessLatency, m.ErrorsTotal, m.InFlight,
	} {
		reg.MustRegister(c)
	}
	return m
}

// Registry exposes the underlying Prometheus registry for /metrics serving.
func (m *Metrics) Registry() *prometheus.Registry { return m.reg }

// ChainState is the live snapshot of one chain's loop. It mirrors a subset
// of the Prometheus gauges into plain Go fields so /status can render JSON
// without scraping the registry, AND updates the gauges in lock-step.
type ChainState struct {
	mu             sync.RWMutex
	Chain          string
	Role           string
	CurrentBlock   int64
	LatestBlock    int64
	LastError      string
	LastErrorTs    int64
	LastProgressTs int64
	StartedTs      int64

	m *Metrics
}

func newChainState(m *Metrics, chain, role string, startedTs int64) *ChainState {
	return &ChainState{m: m, Chain: chain, Role: role, StartedTs: startedTs}
}

// SetCurrentBlock updates the per-chain current_block gauge and bumps the
// last-progress timestamp if it moved forward.
func (s *ChainState) SetCurrentBlock(b int64) {
	if s == nil {
		return
	}
	now := nowUnix()
	s.mu.Lock()
	moved := b > s.CurrentBlock
	s.CurrentBlock = b
	if moved {
		s.LastProgressTs = now
	}
	latest := s.LatestBlock
	s.mu.Unlock()
	s.m.CurrentBlock.WithLabelValues(s.Chain, s.Role).Set(float64(b))
	if moved {
		s.m.LastProgressTs.WithLabelValues(s.Chain, s.Role).Set(float64(now))
	}
	s.m.BlockLag.WithLabelValues(s.Chain, s.Role).Set(float64(latest - b))
}

// SetLatestBlock updates the latest_block gauge and re-derives lag.
func (s *ChainState) SetLatestBlock(b int64) {
	if s == nil {
		return
	}
	s.mu.Lock()
	s.LatestBlock = b
	cur := s.CurrentBlock
	s.mu.Unlock()
	s.m.LatestBlock.WithLabelValues(s.Chain, s.Role).Set(float64(b))
	s.m.BlockLag.WithLabelValues(s.Chain, s.Role).Set(float64(b - cur))
}

// IncBlocksProcessed bumps blocks_processed_total.
func (s *ChainState) IncBlocksProcessed(n int) {
	if s == nil {
		return
	}
	s.m.BlocksProcessed.WithLabelValues(s.Chain, s.Role).Add(float64(n))
}

// IncEventsMatched bumps events_matched_total.
func (s *ChainState) IncEventsMatched(n int) {
	if s == nil {
		return
	}
	s.m.EventsMatched.WithLabelValues(s.Chain, s.Role).Add(float64(n))
}

// ObserveRPC records a single outbound RPC call duration (seconds) for the
// (chain, method) pair. Call sites typically do:
//
//	start := time.Now()
//	... rpc call ...
//	cs.ObserveRPC("LatestBlock", time.Since(start).Seconds())
func (s *ChainState) ObserveRPC(method string, seconds float64) {
	if s == nil {
		return
	}
	s.m.RPCLatency.WithLabelValues(s.Chain, method).Observe(seconds)
}

// ObserveDBInsert records a single DB insert duration (seconds) labeled by
// logical table/sink name.
func (s *ChainState) ObserveDBInsert(table string, seconds float64) {
	if s == nil {
		return
	}
	s.m.DBInsertLatency.WithLabelValues(table).Observe(seconds)
}

// RecordError records an error of the given kind.
func (s *ChainState) RecordError(kind, msg string) {
	if s == nil {
		return
	}
	s.mu.Lock()
	s.LastError = msg
	s.LastErrorTs = nowUnix()
	s.mu.Unlock()
	s.m.ErrorsTotal.WithLabelValues(s.Chain, kind).Inc()
}

// snapshot copies the read-side fields atomically for the /status renderer.
func (s *ChainState) snapshot() chainSnapshot {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return chainSnapshot{
		Chain:          s.Chain,
		Role:           s.Role,
		CurrentBlock:   s.CurrentBlock,
		LatestBlock:    s.LatestBlock,
		BlockLag:       s.LatestBlock - s.CurrentBlock,
		LastProgressTs: s.LastProgressTs,
		LastError:      s.LastError,
		LastErrorTs:    s.LastErrorTs,
		StartedTs:      s.StartedTs,
	}
}
