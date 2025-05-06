package xrp

import (
	"github.com/ChainSafe/log15"
	"github.com/mapprotocol/compass/pkg/msg"
)

type Writer struct {
	cfg    *Config
	log    log15.Logger
	stop   <-chan int
	sysErr chan<- error
}

func newWriter(cfg *Config, log log15.Logger, stop <-chan int, sysErr chan<- error) *Writer {
	return &Writer{
		cfg:    cfg,
		log:    log,
		stop:   stop,
		sysErr: sysErr,
	}
}

func (w *Writer) ResolveMessage(m msg.Message) bool {
	w.log.Info("Attempting to resolve message", "type", m.Type, "src", m.Source, "dst", m.Destination)
	switch m.Type {
	case msg.SwapWithMapProof:
		return w.exeMcs(m)
	default:
		w.log.Error("Unknown message type received", "type", m.Type)
		return false
	}
}

func (w *Writer) exeMcs(m msg.Message) bool {
	m.DoneCh <- struct{}{}
	return true
}
