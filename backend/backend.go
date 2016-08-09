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
// has been delivered to a BitTorrent client.
package backend

import (
	"log"
	"time"

	"golang.org/x/net/context"

	"github.com/jzelinskie/trakr/bittorrent"
	"github.com/jzelinskie/trakr/frontend"
)

type BackendConfig struct {
	AnnounceInterval time.Duration `yaml:"announce_interval"`
}

var _ frontend.TrackerFuncs = &Backend{}

func New(config BackendConfig, peerStore PeerStore, announcePreHooks, announcePostHooks, scrapePreHooks, scrapePostHooks []Hook) (*Backend, error) {
	toReturn := &Backend{
		announceInterval:  config.AnnounceInterval,
		peerStore:         peerStore,
		announcePreHooks:  announcePreHooks,
		announcePostHooks: announcePostHooks,
		scrapePreHooks:    scrapePreHooks,
		scrapePostHooks:   scrapePostHooks,
	}

	if len(toReturn.announcePreHooks) == 0 {
		toReturn.announcePreHooks = []Hook{nopHook{}}
	}

	if len(toReturn.announcePostHooks) == 0 {
		toReturn.announcePostHooks = []Hook{nopHook{}}
	}

	if len(toReturn.scrapePreHooks) == 0 {
		toReturn.scrapePreHooks = []Hook{nopHook{}}
	}

	if len(toReturn.scrapePostHooks) == 0 {
		toReturn.scrapePostHooks = []Hook{nopHook{}}
	}

	return toReturn, nil
}

// Backend is a protocol-agnostic backend of a BitTorrent tracker.
type Backend struct {
	announceInterval  time.Duration
	peerStore         PeerStore
	announcePreHooks  []Hook
	announcePostHooks []Hook
	scrapePreHooks    []Hook
	scrapePostHooks   []Hook
}

// HandleAnnounce generates a response for an Announce.
func (b *Backend) HandleAnnounce(ctx context.Context, req *bittorrent.AnnounceRequest) (*bittorrent.AnnounceResponse, error) {
	resp := &bittorrent.AnnounceResponse{
		Interval: b.announceInterval,
	}
	for _, h := range b.announcePreHooks {
		if err := h.HandleAnnounce(ctx, req, resp); err != nil {
			return nil, err
		}
	}

	return resp, nil
}

// AfterAnnounce does something with the results of an Announce after it
// has been completed.
func (b *Backend) AfterAnnounce(ctx context.Context, req *bittorrent.AnnounceRequest, resp *bittorrent.AnnounceResponse) {
	for _, h := range b.announcePostHooks {
		if err := h.HandleAnnounce(ctx, req, resp); err != nil {
			log.Println("trakr: post-announce hooks failed:", err.Error())
			return
		}
	}
}

// HandleScrape generates a response for a Scrape.
func (b *Backend) HandleScrape(ctx context.Context, req *bittorrent.ScrapeRequest) (*bittorrent.ScrapeResponse, error) {
	resp := &bittorrent.ScrapeResponse{
		Files: make(map[bittorrent.InfoHash]bittorrent.Scrape),
	}
	for _, h := range b.scrapePreHooks {
		if err := h.HandleScrape(ctx, req, resp); err != nil {
			return nil, err
		}
	}

	return resp, nil
}

// AfterScrape does something with the results of a Scrape after it has been completed.
func (b *Backend) AfterScrape(ctx context.Context, req *bittorrent.ScrapeRequest, resp *bittorrent.ScrapeResponse) {
	for _, h := range b.scrapePostHooks {
		if err := h.HandleScrape(ctx, req, resp); err != nil {
			log.Println("trakr: post-scrape hooks failed:", err.Error())
			return
		}
	}
}
