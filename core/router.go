// Copyright 2021 Compass Systems
// SPDX-License-Identifier: LGPL-3.0-only

package core

import (
	"fmt"
	"sync"

	log "github.com/ChainSafe/log15"
	"github.com/mapprotocol/compass/msg"
)

// Writer consumes a message and makes the requried on-chain interactions.
type Writer interface {
	ResolveMessage(message msg.Message) bool
}

// Router forwards messages from their source to their destination
type Router struct {
	registry map[msg.ChainId]Writer
	lock     *sync.RWMutex
	log      log.Logger
	mapcid   msg.ChainId
}

func NewRouter(log log.Logger, mapcid msg.ChainId) *Router {
	return &Router{
		registry: make(map[msg.ChainId]Writer),
		lock:     &sync.RWMutex{},
		log:      log,
		mapcid:   mapcid,
	}
}

// Send passes a message to the destination Writer if it exists
func (r *Router) Send(msg msg.Message) error {
	r.lock.Lock()
	defer r.lock.Unlock()

	r.log.Trace("Routing message", "src", msg.Source, "dest", msg.Destination, "nonce", msg.DepositNonce)
	dest := msg.Destination
	// if neither src or dest are from map, redirect to map chain
	// todo return err ?
	if msg.Source != r.mapcid && msg.Destination != r.mapcid {
		dest = r.mapcid
	}

	w := r.registry[dest]
	if w == nil {
		return fmt.Errorf("unknown destination chainId: %d", msg.Destination)
	}

	go w.ResolveMessage(msg)
	return nil
}

// Listen registers a Writer with a ChainId which Router.Send can then use to propagate messages
func (r *Router) Listen(id msg.ChainId, w Writer) {
	r.lock.Lock()
	defer r.lock.Unlock()
	r.log.Debug("Registering new chain in router", "id", id)
	r.registry[id] = w
}
