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

	"github.com/jzelinskie/trakr/bittorrent"
	"golang.org/x/net/context"
)

// GenericConfig is a block of configuration who's structure is unknown.
type GenericConfig struct {
	name   string      `yaml:"name"`
	config interface{} `yaml:"config"`
}

// Backend is a multi-protocol, customizable BitTorrent Tracker.
type Backend struct {
	AnnounceInterval time.Duration   `yaml:"announce_interval"`
	GCInterval       time.Duration   `yaml:"gc_interval"`
	GCExpiration     time.Duration   `yaml:"gc_expiration"`
	PeerStoreConfig  []GenericConfig `yaml:"storage"`
	PreHooks         []GenericConfig `yaml:"prehooks"`
	PostHooks        []GenericConfig `yaml:"posthooks"`

	peerStore PeerStore
	closing   chan struct{}
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
	// Build an TrackerFuncs from the PreHooks and PostHooks.
	// Create a PeerStore instance.
	// Make TrackerFuncs available to be used by frontends.
	select {
	case <-t.closing:
		return nil
	}

	return nil
}

// TrackerFuncs is the collection of callback functions provided by the Backend
// to (1) generate a response from a parsed request, and (2) observe anything
// after the response has been delivered to the client.
type TrackerFuncs struct {
	HandleAnnounce AnnounceHandler
	HandleScrape   ScrapeHandler
	AfterAnnounce  AnnounceCallback
	AfterScrape    ScrapeCallback
}

// AnnounceHandler is a function that generates a response for an Announce.
type AnnounceHandler func(context.Context, *bittorrent.AnnounceRequest) (*bittorrent.AnnounceResponse, error)

// AnnounceCallback is a function that does something with the results of an
// Announce after it has been completed.
type AnnounceCallback func(*bittorrent.AnnounceRequest, *bittorrent.AnnounceResponse)

// ScrapeHandler is a function that generates a response for a Scrape.
type ScrapeHandler func(context.Context, *bittorrent.ScrapeRequest) (*bittorrent.ScrapeResponse, error)

// ScrapeCallback is a function that does something with the results of a
// Scrape after it has been completed.
type ScrapeCallback func(*bittorrent.ScrapeRequest, *bittorrent.ScrapeResponse)
