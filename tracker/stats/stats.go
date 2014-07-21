// Copyright 2014 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

// Package stats implements a means of tracking processing statistics for a
// BitTorrent tracker.
package stats

import "time"

const (
	Announce = iota
	Scrape
	Completed
	NewPeer
	DeletedPeer
	NewSeeder
	DeletedSeeder
	NewTorrent
	DeletedTorrent
)

type Stats struct {
	Start     time.Time
	Announces uint64
	Scrapes   uint64
	Completed uint64
	Peers     uint64
	Seeders   uint64
	Torrents  uint64

	events chan int
}

func New(chanSize int) *Stats {
	s := &Stats{
		Start:  time.Now(),
		events: make(chan int, chanSize),
	}

	go s.handleEvents()

	return s
}

func (s *Stats) Close() {
	close(s.events)
}

func (s *Stats) Uptime() time.Duration {
	return time.Since(s.Start)
}

func (s *Stats) RecordEvent(event int) {
	s.events <- event
}

func (s *Stats) handleEvents() {
	for event := range s.events {
		switch event {
		case Announce:
			s.Announces++

		case Scrape:
			s.Scrapes++

		case Completed:
			s.Completed++

		case NewPeer:
			s.Peers++

		case DeletedPeer:
			s.Peers--

		case NewSeeder:
			s.Seeders++

		case DeletedSeeder:
			s.Seeders--

		case NewTorrent:
			s.Torrents++

		case DeletedTorrent:
			s.Torrents--

		default:
			panic("stats: RecordEvent called with an unknown event")
		}
	}
}
