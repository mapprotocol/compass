package observability

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// AlarmRule is the alert specification this package supports today.
// We deliberately keep it simple — one threshold over a sustained window
// per (chain, role). If you need richer rules (rate, deltas, etc.) extend
// here rather than introducing a full PromQL evaluator.
type AlarmRule struct {
	Name      string        // human-readable rule name, included in the message
	Threshold int64         // lag threshold in blocks
	For       time.Duration // how long the breach must be sustained before firing
	Cooldown  time.Duration // minimum time between two alarms for the same series
}

// DefaultBlockLagRule mirrors the user's spec: block_lag > 50 sustained for
// 5 minutes, with a 10-minute cooldown so a stuck chain doesn't spam.
func DefaultBlockLagRule() AlarmRule {
	return AlarmRule{
		Name:      "block_lag_high",
		Threshold: 50,
		For:       5 * time.Minute,
		Cooldown:  10 * time.Minute,
	}
}

// StartBlockLagAlarms launches a goroutine that polls every chain state and
// fires the configured rule via cfg.AlarmFn. The goroutine exits on Stop().
func (o *Observability) StartBlockLagAlarms(rule AlarmRule) {
	if o.cfg.AlarmFn == nil {
		return
	}
	go o.runBlockLagLoop(rule)
}

func (o *Observability) runBlockLagLoop(rule AlarmRule) {
	type seriesState struct {
		breachStart time.Time // zero = not in breach
		lastAlarmed time.Time
	}
	var (
		mu     sync.Mutex
		series = map[string]*seriesState{}
	)

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-o.stopCh:
			return
		case <-ticker.C:
		}

		o.mu.RLock()
		snaps := make([]chainSnapshot, 0, len(o.chains))
		for _, cs := range o.chains {
			snaps = append(snaps, cs.snapshot())
		}
		o.mu.RUnlock()

		now := time.Now()
		for _, s := range snaps {
			key := s.Chain + "/" + s.Role
			mu.Lock()
			st, ok := series[key]
			if !ok {
				st = &seriesState{}
				series[key] = st
			}
			breached := s.BlockLag > rule.Threshold
			if !breached {
				st.breachStart = time.Time{}
				mu.Unlock()
				continue
			}
			if st.breachStart.IsZero() {
				st.breachStart = now
				mu.Unlock()
				continue
			}
			if now.Sub(st.breachStart) < rule.For {
				mu.Unlock()
				continue
			}
			if !st.lastAlarmed.IsZero() && now.Sub(st.lastAlarmed) < rule.Cooldown {
				mu.Unlock()
				continue
			}
			st.lastAlarmed = now
			mu.Unlock()

			msg := fmt.Sprintf(
				"[%s] %s: block_lag=%d > %d sustained for %s (current=%d latest=%d)",
				rule.Name, key, s.BlockLag, rule.Threshold,
				now.Sub(st.breachStart).Truncate(time.Second),
				s.CurrentBlock, s.LatestBlock,
			)
			func() {
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				o.cfg.AlarmFn(ctx, msg)
			}()
		}
	}
}
