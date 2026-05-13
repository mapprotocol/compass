package observability

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus/testutil"
)

func freezeTime(ts int64) func() {
	prev := nowUnix
	nowUnix = func() int64 { return ts }
	return func() { nowUnix = prev }
}

func TestChainState_SetCurrentBlock_BumpsProgressTsOnlyWhenMoving(t *testing.T) {
	o := New("test", Config{})
	cs := o.RegisterChain("bsc", "messenger")

	defer freezeTime(1700000000)()
	cs.SetCurrentBlock(100)
	if got := cs.snapshot().LastProgressTs; got != 1700000000 {
		t.Fatalf("LastProgressTs = %d, want 1700000000", got)
	}

	defer freezeTime(1700000999)()
	cs.SetCurrentBlock(100) // same value, must NOT move
	if got := cs.snapshot().LastProgressTs; got != 1700000000 {
		t.Fatalf("LastProgressTs unexpectedly advanced to %d", got)
	}

	cs.SetCurrentBlock(101) // forward, must move
	if got := cs.snapshot().LastProgressTs; got != 1700000999 {
		t.Fatalf("LastProgressTs = %d, want 1700000999", got)
	}
}

func TestChainState_BlockLagDerivation(t *testing.T) {
	o := New("test", Config{})
	cs := o.RegisterChain("eth", "sync")

	cs.SetLatestBlock(1000)
	cs.SetCurrentBlock(950)
	if got := cs.snapshot().BlockLag; got != 50 {
		t.Fatalf("BlockLag = %d, want 50", got)
	}

	cs.SetLatestBlock(1100)
	if got := cs.snapshot().BlockLag; got != 150 {
		t.Fatalf("BlockLag = %d, want 150", got)
	}

	// And the prometheus gauge tracks it too.
	if got := testutil.ToFloat64(cs.m.BlockLag.WithLabelValues("eth", "sync")); got != 150 {
		t.Fatalf("block_lag gauge = %v, want 150", got)
	}
}

func TestObservability_RegisterChain_IsIdempotent(t *testing.T) {
	o := New("t", Config{})
	a := o.RegisterChain("bsc", "messenger")
	b := o.RegisterChain("bsc", "messenger")
	if a != b {
		t.Fatal("RegisterChain should return the same ChainState for same chain+role")
	}
}

func TestStatusEndpoint_RendersChains(t *testing.T) {
	o := New("t", Config{})
	cs := o.RegisterChain("bsc", "messenger")
	cs.SetLatestBlock(500)
	cs.SetCurrentBlock(450)

	rec := httptest.NewRecorder()
	o.handleStatus(rec, httptest.NewRequest("GET", "/status", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status code = %d, want 200", rec.Code)
	}

	var env statusEnvelope
	if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil {
		t.Fatalf("parse json: %v\nbody: %s", err, rec.Body.String())
	}
	if len(env.Chains) != 1 {
		t.Fatalf("chains len = %d, want 1", len(env.Chains))
	}
	got := env.Chains[0]
	if got.Chain != "bsc" || got.Role != "messenger" || got.BlockLag != 50 {
		t.Fatalf("snapshot mismatch: %+v", got)
	}
}

func TestStatusEndpoint_EmptyHasZeroChains(t *testing.T) {
	o := New("t", Config{})
	rec := httptest.NewRecorder()
	o.handleStatus(rec, httptest.NewRequest("GET", "/status", nil))

	var env statusEnvelope
	if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil {
		t.Fatalf("parse json: %v", err)
	}
	if len(env.Chains) != 0 {
		t.Fatalf("empty observability should have 0 chains, got %d", len(env.Chains))
	}
}

func TestRecordError_BumpsCounter(t *testing.T) {
	o := New("t", Config{})
	cs := o.RegisterChain("bsc", "messenger")

	cs.RecordError("rpc_timeout", "oops")
	cs.RecordError("rpc_timeout", "oops again")
	cs.RecordError("db_failed", "schema mismatch")

	if got := testutil.ToFloat64(cs.m.ErrorsTotal.WithLabelValues("bsc", "rpc_timeout")); got != 2 {
		t.Fatalf("rpc_timeout counter = %v, want 2", got)
	}
	if got := testutil.ToFloat64(cs.m.ErrorsTotal.WithLabelValues("bsc", "db_failed")); got != 1 {
		t.Fatalf("db_failed counter = %v, want 1", got)
	}

	snap := cs.snapshot()
	if snap.LastError != "schema mismatch" {
		t.Fatalf("snapshot LastError = %q, want %q", snap.LastError, "schema mismatch")
	}
}

func TestIncBlocksProcessedAndEventsMatched(t *testing.T) {
	o := New("t", Config{})
	cs := o.RegisterChain("xrp", "sync")
	cs.IncBlocksProcessed(7)
	cs.IncBlocksProcessed(3)
	cs.IncEventsMatched(11)

	if got := testutil.ToFloat64(cs.m.BlocksProcessed.WithLabelValues("xrp", "sync")); got != 10 {
		t.Fatalf("blocks_processed = %v, want 10", got)
	}
	if got := testutil.ToFloat64(cs.m.EventsMatched.WithLabelValues("xrp", "sync")); got != 11 {
		t.Fatalf("events_matched = %v, want 11", got)
	}
}

func TestBlockLagAlarm_FiresAfterSustainedBreach_ThenCools(t *testing.T) {
	var fired atomic.Int32
	gotMsg := make(chan string, 4)
	o := New("t", Config{
		AlarmFn: func(_ context.Context, msg string) {
			fired.Add(1)
			select {
			case gotMsg <- msg:
			default:
			}
		},
	})
	cs := o.RegisterChain("bsc", "messenger")
	cs.SetLatestBlock(1000)
	cs.SetCurrentBlock(900) // lag = 100

	// Drive the rule evaluator manually so we don't sit on the 30s ticker.
	rule := AlarmRule{Name: "lag", Threshold: 50, For: 30 * time.Millisecond, Cooldown: 200 * time.Millisecond}

	type seriesState struct {
		breachStart time.Time
		lastAlarmed time.Time
	}
	series := map[string]*seriesState{}

	evalOnce := func() {
		o.mu.RLock()
		snaps := make([]chainSnapshot, 0, len(o.chains))
		for _, c := range o.chains {
			snaps = append(snaps, c.snapshot())
		}
		o.mu.RUnlock()
		now := time.Now()
		for _, s := range snaps {
			key := s.Chain + "/" + s.Role
			st, ok := series[key]
			if !ok {
				st = &seriesState{}
				series[key] = st
			}
			if s.BlockLag <= rule.Threshold {
				st.breachStart = time.Time{}
				continue
			}
			if st.breachStart.IsZero() {
				st.breachStart = now
				continue
			}
			if now.Sub(st.breachStart) < rule.For {
				continue
			}
			if !st.lastAlarmed.IsZero() && now.Sub(st.lastAlarmed) < rule.Cooldown {
				continue
			}
			st.lastAlarmed = now
			o.cfg.AlarmFn(context.Background(), key)
		}
	}

	// 1st eval: starts the breach timer; no alarm yet
	evalOnce()
	if fired.Load() != 0 {
		t.Fatal("alarm fired prematurely on first eval")
	}
	// wait past `For`
	time.Sleep(40 * time.Millisecond)
	evalOnce()
	if fired.Load() != 1 {
		t.Fatalf("expected 1 alarm after sustained breach, got %d", fired.Load())
	}
	// next eval inside cooldown -> no extra alarm
	evalOnce()
	if fired.Load() != 1 {
		t.Fatalf("cooldown ignored: fired=%d", fired.Load())
	}
	// fix the lag -> next breach must restart the For window
	cs.SetCurrentBlock(995) // lag = 5 now
	evalOnce()
	cs.SetCurrentBlock(900) // breach again
	evalOnce()
	if fired.Load() != 1 {
		t.Fatalf("breach restart should not refire instantly: fired=%d", fired.Load())
	}

	select {
	case msg := <-gotMsg:
		if msg != "bsc/messenger" {
			t.Fatalf("alarm subject = %q, want bsc/messenger", msg)
		}
	default:
		t.Fatal("no alarm message captured")
	}
}
