// Copyright 2016 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

package server

import (
	"sync"

	"github.com/chihaya/chihaya/config"
	"github.com/chihaya/chihaya/tracker"
)

// Pool represents a running pool of servers.
type Pool struct {
	servers []Server
	wg      sync.WaitGroup
}

// StartPool creates a new pool of servers specified by the provided config and
// runs them.
func StartPool(cfgs []config.ServerConfig, tkr *tracker.Tracker) (*Pool, error) {
	var toReturn Pool

	for _, cfg := range cfgs {
		srv, err := New(&cfg, tkr)
		if err != nil {
			return nil, err
		}

		toReturn.wg.Add(1)
		go func(srv Server) {
			defer toReturn.wg.Done()
			srv.Start()
		}(srv)

		toReturn.servers = append(toReturn.servers, srv)
	}

	return &toReturn, nil
}

// Stop safely shuts down a pool of servers.
func (p *Pool) Stop() {
	for _, srv := range p.servers {
		srv.Stop()
	}
	p.wg.Wait()
}
