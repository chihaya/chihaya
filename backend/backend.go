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
package backend

import (
	"time"

	"github.com/jzelinskie/trakr/frontends"
)

// GenericConfig is a block of configuration who's structure is unknown.
type GenericConfig struct {
	name   string      `yaml:"name"`
	config interface{} `yaml:"config"`
}

type BackendConfig struct {
	AnnounceInterval time.Duration   `yaml:"announce_interval"`
	PreHooks         []GenericConfig `yaml:"prehooks"`
	PostHooks        []GenericConfig `yaml:"posthooks"`
}

func New(config BackendConfig, peerStore PeerStore) (*Backend, error) {
	// Build TrackerFuncs from the PreHooks and PostHooks
	return &Backend{peerStore: peerStore}, nil
}

// Backend is a multi-protocol, customizable BitTorrent Tracker.
type Backend struct {
	TrackerFuncs frontends.TrackerFuncs
	peerStore    PeerStore
	closing      chan struct{}
}

// Stop provides a thread-safe way to shutdown a currently running
// Backend.
func (t *Backend) Stop() {
	close(t.closing)
}

// Start starts the Backend.
// It blocks until t.Stop() is called or an error is returned.
func (t *Backend) Start() error {
	t.closing = make(chan struct{})
	select {
	case <-t.closing:
		return nil
	}

	return nil
}
