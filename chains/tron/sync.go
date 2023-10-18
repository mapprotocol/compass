package tron

import (
	"time"

	metrics "github.com/ChainSafe/chainbridge-utils/metrics/types"
	"github.com/ChainSafe/log15"
	"github.com/mapprotocol/compass/chains"
)

type Maintainer struct {
	Log log15.Logger
}

func NewMaintainer(log log15.Logger) *Maintainer {
	return &Maintainer{Log: log}
}

func (m *Maintainer) Sync() error {
	m.Log.Debug("Starting listener...")
	go func() {
		time.Sleep(time.Hour * 2400)
	}()

	return nil
}

func (m *Maintainer) SetRouter(r chains.Router) {

}

func (m *Maintainer) GetLatestBlock() metrics.LatestBlock {
	return metrics.LatestBlock{}
}

type Messenger struct {
	Log log15.Logger
}

func NewMessenger(log log15.Logger) *Messenger {
	return &Messenger{Log: log}
}

func (m *Messenger) Sync() error {
	m.Log.Debug("Starting listener...")
	go func() {
		time.Sleep(time.Hour * 2400)
	}()

	return nil
}

func (m *Messenger) SetRouter(r chains.Router) {

}

func (m *Messenger) GetLatestBlock() metrics.LatestBlock {
	return metrics.LatestBlock{}
}
