package observability

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/pprof"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Config holds the public surface of an Observability instance.
type Config struct {
	// Addr is the listen address for the HTTP server. Empty disables it
	// (everything else still works in-process).
	Addr string
	// Namespace is the Prometheus metric prefix (e.g. "radar", "compass").
	Namespace string
	// AlarmFn is invoked when the rule engine fires. nil disables alarms.
	AlarmFn func(ctx context.Context, msg string)
}

// Observability wraps Metrics + /status + pprof + alarm rules behind a
// single HTTP server that the main binary embeds.
type Observability struct {
	Metrics *Metrics
	cfg     Config

	mu     sync.RWMutex
	chains map[string]*ChainState // key: chain+role

	server  *http.Server
	stopCh  chan struct{}
	stopped bool
}

// New constructs an Observability instance with a fresh registry.
func New(namespace string, cfg Config) *Observability {
	if cfg.Namespace == "" {
		cfg.Namespace = namespace
	}
	return &Observability{
		Metrics: newMetrics(cfg.Namespace),
		cfg:     cfg,
		chains:  make(map[string]*ChainState),
		stopCh:  make(chan struct{}),
	}
}

// RegisterChain creates (or returns the existing) ChainState for chain+role.
func (o *Observability) RegisterChain(chain, role string) *ChainState {
	key := chain + "/" + role
	o.mu.Lock()
	defer o.mu.Unlock()
	if existing, ok := o.chains[key]; ok {
		return existing
	}
	cs := newChainState(o.Metrics, chain, role, nowUnix())
	o.chains[key] = cs
	return cs
}

// StartHTTP starts the embedded HTTP server.
// Returns immediately; the server runs in its own goroutine.
// Routes:
//
//	/metrics            Prometheus exposition
//	/status             JSON snapshot of all chain states
//	/healthz            Liveness (200 OK if process is up)
//	/debug/pprof/*      Go runtime profiling endpoints
func (o *Observability) StartHTTP() {
	if o.cfg.Addr == "" {
		return
	}
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.HandlerFor(o.Metrics.Registry(), promhttp.HandlerOpts{}))
	mux.HandleFunc("/status", o.handleStatus)
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	mux.HandleFunc("/debug/pprof/", pprof.Index)
	mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	mux.HandleFunc("/debug/pprof/trace", pprof.Trace)

	o.server = &http.Server{
		Addr:              o.cfg.Addr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}
	go func() {
		_ = o.server.ListenAndServe()
	}()
}

// Stop shuts the HTTP server down with a short grace period.
func (o *Observability) Stop() {
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.stopped {
		return
	}
	o.stopped = true
	close(o.stopCh)
	if o.server != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		_ = o.server.Shutdown(ctx)
	}
}

// chainSnapshot is the JSON shape rendered by /status. Plain types so the
// reader (jq, curl) doesn't need any tooling beyond stdlib JSON.
type chainSnapshot struct {
	Chain          string `json:"chain"`
	Role           string `json:"role"`
	CurrentBlock   int64  `json:"current_block"`
	LatestBlock    int64  `json:"latest_block"`
	BlockLag       int64  `json:"block_lag"`
	LastProgressTs int64  `json:"last_progress_unixtime"`
	LastError      string `json:"last_error,omitempty"`
	LastErrorTs    int64  `json:"last_error_unixtime,omitempty"`
	StartedTs      int64  `json:"started_unixtime"`
}

type statusEnvelope struct {
	UptimeSeconds int64           `json:"uptime_seconds"`
	NowUnixtime   int64           `json:"now_unixtime"`
	Chains        []chainSnapshot `json:"chains"`
}

var startedAt = nowUnix()

func (o *Observability) handleStatus(w http.ResponseWriter, _ *http.Request) {
	snap := statusEnvelope{
		UptimeSeconds: nowUnix() - startedAt,
		NowUnixtime:   nowUnix(),
	}
	o.mu.RLock()
	for _, cs := range o.chains {
		snap.Chains = append(snap.Chains, cs.snapshot())
	}
	o.mu.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	_ = enc.Encode(snap)
}

// nowUnix is a seam for tests.
var nowUnix = func() int64 { return time.Now().Unix() }
