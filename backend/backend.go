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

	"log"

	"github.com/jzelinskie/trakr/bittorrent"
	"github.com/jzelinskie/trakr/frontend"
	"golang.org/x/net/context"
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

var _ frontend.TrackerFuncs = &Backend{}

func New(config BackendConfig, peerStore PeerStore) (*Backend, error) {
	// Build TrackerFuncs from the PreHooks and PostHooks
	return &Backend{peerStore: peerStore}, nil
}

// Backend is a multi-protocol, customizable BitTorrent Tracker.
type Backend struct {
	peerStore      PeerStore
	handleAnnounce func(context.Context, *bittorrent.AnnounceRequest) (*bittorrent.AnnounceResponse, error)
	afterAnnounce  func(context.Context, *bittorrent.AnnounceRequest, *bittorrent.AnnounceResponse) error
	handleScrape   func(context.Context, *bittorrent.ScrapeRequest) (*bittorrent.ScrapeResponse, error)
	afterScrape    func(context.Context, *bittorrent.ScrapeRequest, *bittorrent.ScrapeResponse) error
}

// HandleAnnounce generates a response for an Announce.
func (b *Backend) HandleAnnounce(ctx context.Context, req *bittorrent.AnnounceRequest) (*bittorrent.AnnounceResponse, error) {
	return b.handleAnnounce(ctx, req)
}

// AfterAnnounce does something with the results of an Announce after it
// has been completed.
func (b *Backend) AfterAnnounce(ctx context.Context, req *bittorrent.AnnounceRequest, resp *bittorrent.AnnounceResponse) {
	err := b.afterAnnounce(ctx, req, resp)
	if err != nil {
		log.Println("trakr: post-announce hooks failed:", err.Error())
	}
}

// HandleScrape generates a response for a Scrape.
func (b *Backend) HandleScrape(ctx context.Context, req *bittorrent.ScrapeRequest) (*bittorrent.ScrapeResponse, error) {
	return b.handleScrape(ctx, req)
}

// AfterScrape does something with the results of a Scrape after it has been completed.
func (b *Backend) AfterScrape(ctx context.Context, req *bittorrent.ScrapeRequest, resp *bittorrent.ScrapeResponse) {
	err := b.afterScrape(ctx, req, resp)
	if err != nil {
		log.Println("trakr: post-scrape hooks failed:", err.Error())
	}
}
