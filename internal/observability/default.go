package observability

import "sync/atomic"

// default observability used by package-level helpers. Set once in main()
// via SetDefault, then chains/loops call RegisterChain / metrics helpers
// without threading the *Observability through every constructor.

var defaultObs atomic.Pointer[Observability]

// SetDefault installs the process-wide Observability used by package-level
// helpers (RegisterChain, etc.). Safe to call once at startup.
func SetDefault(o *Observability) { defaultObs.Store(o) }

// Default returns the installed Observability, or a no-op fallback so
// callers that run before SetDefault (or in unit tests) don't panic.
func Default() *Observability {
	if o := defaultObs.Load(); o != nil {
		return o
	}
	return noopOnce()
}

var noop atomic.Pointer[Observability]

func noopOnce() *Observability {
	if o := noop.Load(); o != nil {
		return o
	}
	o := New("noop", Config{}) // not started; gauges/counters still work in-memory
	noop.CompareAndSwap(nil, o)
	return noop.Load()
}

// RegisterChain is shorthand for Default().RegisterChain.
func RegisterChain(chain, role string) *ChainState {
	return Default().RegisterChain(chain, role)
}
