// Copyright 2014 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

// Package stats implements a means of tracking processing statistics for a
// BitTorrent tracker.
package stats

import (
	"time"

	"github.com/chihaya/chihaya/config"
	"github.com/pushrax/flatjson"
)

const (
	Announce = iota
	Scrape

	Completed
	NewLeech
	DeletedLeech
	ReapedLeech
	NewSeed
	DeletedSeed
	ReapedSeed

	NewTorrent
	DeletedTorrent
	ReapedTorrent

	AcceptedConnection
	ClosedConnection

	HandledRequest
	ErroredRequest
	ClientError

	ResponseTime
)

// DefaultStats is a default instance of stats tracking that uses an unbuffered
// channel for broadcasting events unless specified otherwise via a command
// line flag.
var DefaultStats *Stats

type PeerClassStats struct {
	Current uint64 // Current peer count.
	Joined  uint64 // Peers that announced.
	Left    uint64 // Peers that paused or stopped.
	Reaped  uint64 // Peers cleaned up after inactivity.
}

type PeerStats struct {
	PeerClassStats `json:"Peers"` // Stats for all peers.

	Seeds     PeerClassStats // Stats for seeds only.
	Completed uint64         // Number of transitions from leech to seed.
}

type PercentileTimes struct {
	P50 *Percentile
	P90 *Percentile
	P95 *Percentile
}

type Stats struct {
	Started time.Time // Time at which Chihaya was booted.

	OpenConnections     uint64 `json:"Connections.Open"`
	ConnectionsAccepted uint64 `json:"Connections.Accepted"`
	BytesTransmitted    uint64 `json:"BytesTransmitted"`

	RequestsHandled uint64 `json:"Requests.Handled"`
	RequestsErrored uint64 `json:"Requests.Errored"`
	ClientErrors    uint64 `json:"Requests.Bad"`
	ResponseTime    PercentileTimes

	Announces uint64 `json:"Tracker.Announces"`
	Scrapes   uint64 `json:"Tracker.Scrapes"`

	TorrentsAdded   uint64 `json:"Torrents.Added"`
	TorrentsRemoved uint64 `json:"Torrents.Removed"`
	TorrentsReaped  uint64 `json:"Torrents.Reaped"`

	IPv4Peers PeerStats `json:"Peers.IPv4"`
	IPv6Peers PeerStats `json:"Peers.IPv6"`

	*MemStatsWrapper `json:",omitempty"`

	events             chan int
	ipv4PeerEvents     chan int
	ipv6PeerEvents     chan int
	responseTimeEvents chan time.Duration
	recordMemStats     <-chan time.Time

	flattened flatjson.FlatMap
}

func New(cfg config.StatsConfig) *Stats {
	s := &Stats{
		Started: time.Now(),
		events:  make(chan int, cfg.BufferSize),

		ipv4PeerEvents:     make(chan int, cfg.BufferSize),
		ipv6PeerEvents:     make(chan int, cfg.BufferSize),
		responseTimeEvents: make(chan time.Duration, cfg.BufferSize),

		ResponseTime: PercentileTimes{
			P50: NewPercentile(0.5),
			P90: NewPercentile(0.9),
			P95: NewPercentile(0.95),
		},
	}

	if cfg.IncludeMem {
		s.MemStatsWrapper = NewMemStatsWrapper(cfg.VerboseMem)
		s.recordMemStats = time.NewTicker(cfg.MemUpdateInterval.Duration).C
	}

	s.flattened = flatjson.Flatten(s)
	go s.handleEvents()
	return s
}

func (s *Stats) Flattened() flatjson.FlatMap {
	return s.flattened
}

func (s *Stats) Close() {
	close(s.events)
}

func (s *Stats) Uptime() time.Duration {
	return time.Since(s.Started)
}

func (s *Stats) RecordEvent(event int) {
	s.events <- event
}

func (s *Stats) RecordPeerEvent(event int, ipv6 bool) {
	if ipv6 {
		s.ipv6PeerEvents <- event
	} else {
		s.ipv4PeerEvents <- event
	}
}

func (s *Stats) RecordTiming(event int, duration time.Duration) {
	switch event {
	case ResponseTime:
		s.responseTimeEvents <- duration
	default:
		panic("stats: RecordTiming called with an unknown event")
	}
}

func (s *Stats) handleEvents() {
	for {
		select {
		case event := <-s.events:
			s.handleEvent(event)

		case event := <-s.ipv4PeerEvents:
			s.handlePeerEvent(&s.IPv4Peers, event)

		case event := <-s.ipv6PeerEvents:
			s.handlePeerEvent(&s.IPv6Peers, event)

		case duration := <-s.responseTimeEvents:
			f := float64(duration) / float64(time.Millisecond)
			s.ResponseTime.P50.AddSample(f)
			s.ResponseTime.P90.AddSample(f)
			s.ResponseTime.P95.AddSample(f)

		case <-s.recordMemStats:
			s.MemStatsWrapper.Update()
		}
	}
}

func (s *Stats) handleEvent(event int) {
	switch event {
	case Announce:
		s.Announces++

	case Scrape:
		s.Scrapes++

	case NewTorrent:
		s.TorrentsAdded++

	case DeletedTorrent:
		s.TorrentsRemoved++

	case ReapedTorrent:
		s.TorrentsReaped++

	case AcceptedConnection:
		s.ConnectionsAccepted++
		s.OpenConnections++

	case ClosedConnection:
		s.OpenConnections--

	case HandledRequest:
		s.RequestsHandled++

	case ClientError:
		s.ClientErrors++

	case ErroredRequest:
		s.RequestsErrored++

	default:
		panic("stats: RecordEvent called with an unknown event")
	}
}

func (s *Stats) handlePeerEvent(ps *PeerStats, event int) {
	switch event {
	case Completed:
		ps.Completed++
		ps.Seeds.Current++

	case NewLeech:
		ps.Joined++
		ps.Current++

	case DeletedLeech:
		ps.Left++
		ps.Current--

	case ReapedLeech:
		ps.Reaped++
		ps.Current--

	case NewSeed:
		ps.Seeds.Joined++
		ps.Seeds.Current++
		ps.Joined++
		ps.Current++

	case DeletedSeed:
		ps.Seeds.Left++
		ps.Seeds.Current--
		ps.Left++
		ps.Current--

	case ReapedSeed:
		ps.Seeds.Reaped++
		ps.Seeds.Current--
		ps.Reaped++
		ps.Current--

	default:
		panic("stats: RecordPeerEvent called with an unknown event")
	}
}

// RecordEvent broadcasts an event to the default stats queue.
func RecordEvent(event int) {
	DefaultStats.RecordEvent(event)
}

// RecordPeerEvent broadcasts a peer event to the default stats queue.
func RecordPeerEvent(event int, ipv6 bool) {
	DefaultStats.RecordPeerEvent(event, ipv6)
}

// RecordTiming broadcasts a timing event to the default stats queue.
func RecordTiming(event int, duration time.Duration) {
	DefaultStats.RecordTiming(event, duration)
}
