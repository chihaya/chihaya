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

	ResponseTime
)

// DefaultStats is a default instance of stats tracking that uses an unbuffered
// channel for broadcasting events.
var DefaultStats *Stats

func init() {
	DefaultStats = New(0)
}

type PeerStats struct {
	// Stats for all peers.
	Current   uint64 `json:"current"`   // Current total peer count.
	Joined    uint64 `json:"joined"`    // Total peers that announced.
	Left      uint64 `json:"left"`      // Total peers that paused or stopped.
	Reaped    uint64 `json:"reaped"`    // Total peers cleaned up after inactivity.
	Completed uint64 `json:"completed"` // Number of transitions from leech to seed.

	// Stats for seeds only (subset of total).
	SeedsCurrent uint64 `json:"seeds_current"` // Current seed count.
	SeedsJoined  uint64 `json:"seeds_joined"`  // Seeds that announced (does not included leechers that completed).
	SeedsLeft    uint64 `json:"seeds_left"`    // Seeds that paused or stopped.
	SeedsReaped  uint64 `json:"seeds_reaped"`  // Seeds cleaned up after inactivity.
}

type PercentileTimes struct {
	P50 *Percentile
	P90 *Percentile
	P95 *Percentile
}

type Stats struct {
	Start time.Time `json:"start_time"` // Time at which Chihaya was booted.

	Announces uint64 `json:"announces"` // Total number of announces.
	Scrapes   uint64 `json:"scrapes"`   // Total number of scrapes.

	IPv4Peers PeerStats `json:"ipv4_peers"`
	IPv6Peers PeerStats `json:"ipv6_peers"`

	TorrentsAdded   uint64 `json:"torrents_added"`
	TorrentsRemoved uint64 `json:"torrents_removed"`
	TorrentsReaped  uint64 `json:"torrents_reaped"`

	OpenConnections     uint64 `json:"open_connections"`
	ConnectionsAccepted uint64 `json:"connections_accepted"`
	BytesTransmitted    uint64 `json:"bytes_transmitted"`

	RequestsHandled uint64 `json:"requests_handled"`
	RequestsErrored uint64 `json:"requests_errored"`

	ResponseTime PercentileTimes `json:"response_time"`

	events             chan int
	ipv4PeerEvents     chan int
	ipv6PeerEvents     chan int
	responseTimeEvents chan time.Duration
}

func New(chanSize int) *Stats {
	s := &Stats{
		Start:  time.Now(),
		events: make(chan int, chanSize),

		ipv4PeerEvents:     make(chan int, chanSize),
		ipv6PeerEvents:     make(chan int, chanSize),
		responseTimeEvents: make(chan time.Duration, chanSize),

		ResponseTime: PercentileTimes{
			P50: NewPercentile(0.5),
			P90: NewPercentile(0.9),
			P95: NewPercentile(0.95),
		},
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
		ps.SeedsCurrent++

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
		ps.SeedsJoined++
		ps.SeedsCurrent++
		ps.Joined++
		ps.Current++

	case DeletedSeed:
		ps.SeedsLeft++
		ps.SeedsCurrent--
		ps.Left++
		ps.Current--

	case ReapedSeed:
		ps.SeedsReaped++
		ps.SeedsCurrent--
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
