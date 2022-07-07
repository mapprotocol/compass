// Copyright 2021 Compass Systems
// SPDX-License-Identifier: LGPL-3.0-only

package near

import (
	metrics "github.com/ChainSafe/chainbridge-utils/metrics/types"
	"github.com/ChainSafe/log15"
	"github.com/mapprotocol/compass/core"
	"github.com/mapprotocol/compass/msg"
)

var _ core.Writer = &writer{}

var PassedStatus uint8 = 2
var TransferredStatus uint8 = 3
var CancelledStatus uint8 = 4

type writer struct {
	cfg     Config
	conn    Connection
	log     log15.Logger
	stop    <-chan int
	sysErr  chan<- error // Reports fatal error to core
	metrics *metrics.ChainMetrics
}

// NewWriter creates and returns writer
func NewWriter(conn Connection, cfg *Config, log log15.Logger, stop <-chan int, sysErr chan<- error, m *metrics.ChainMetrics) *writer {
	return &writer{
		cfg:     *cfg,
		conn:    conn,
		log:     log,
		stop:    stop,
		sysErr:  sysErr,
		metrics: m,
	}
}

func (w *writer) start() error {
	w.log.Debug("Starting ethereum writer...")
	return nil
}

// ResolveMessage handles any given message based on type
// A bool is returned to indicate failure/success, this should be ignored except for within tests.
func (w *writer) ResolveMessage(m msg.Message) bool {
	w.log.Info("Attempting to resolve message", "type", m.Type, "src", m.Source, "dst", m.Destination, "nonce", m.DepositNonce)

	switch m.Type {
	case msg.SyncFromMap:
		return w.exeSyncMapMsg(m)
	case msg.SwapWithMapProof:
		return w.exeSwapMsg(m)
	default:
		w.log.Error("Unknown message type received", "type", m.Type)
		return false
	}
}
