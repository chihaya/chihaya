// Copyright 2016 Jimmy Zelinskie
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package trakr implements a BitTorrent Tracker that supports multiple
// protocols and configurable Hooks that execute before and after a Response
// has been delievered to a BitTorrent client.
package trakr

import (
	"time"

	"github.com/jzelinskie/trakr/bittorrent/http"
	"github.com/jzelinskie/trakr/bittorrent/udp"
)

// GenericConfig is a block of configuration who's structure is unknown.
type GenericConfig struct {
	name   string      `yaml:"name"`
	config interface{} `yaml:"config"`
}

// MultiTracker is a multi-protocol, customizable BitTorrent Tracker.
type MultiTracker struct {
	AnnounceInterval time.Duration   `yaml:"announce_interval"`
	GCInterval       time.Duration   `yaml:"gc_interval"`
	GCExpiration     time.Duration   `yaml:"gc_expiration"`
	HTTPConfig       http.Config     `yaml:"http"`
	UDPConfig        udp.Config      `yaml:"udp"`
	PeerStoreConfig  []GenericConfig `yaml:"storage"`
	PreHooks         []GenericConfig `yaml:"prehooks"`
	PostHooks        []GenericConfig `yaml:"posthooks"`

	peerStore   PeerStore
	httpTracker http.Tracker
	udpTracker  udp.Tracker
	closing     chan struct{}
}

// Stop provides a thread-safe way to shutdown a currently running
// MultiTracker.
func (t *MultiTracker) Stop() {
	close(t.closing)
}

// ListenAndServe listens on the protocols and addresses specified in the
// HTTPConfig and UDPConfig then blocks serving BitTorrent requests until
// t.Stop() is called or an error is returned.
func (t *MultiTracker) ListenAndServe() error {
	t.closing = make(chan struct{})
	// Build an TrackerFuncs from the PreHooks and PostHooks.
	// Create a PeerStore instance.
	// Create a HTTP Tracker instance.
	// Create a UDP Tracker instance.
	select {
	case <-t.closing:
		return nil
	}

	return nil
}
