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

	CompletedIPv4
	NewLeechIPv4
	DeletedLeechIPv4
	ReapedLeechIPv4
	NewSeedIPv4
	DeletedSeedIPv4
	ReapedSeedIPv4

	CompletedIPv6
	NewLeechIPv6
	DeletedLeechIPv6
	ReapedLeechIPv6
	NewSeedIPv6
	DeletedSeedIPv6
	ReapedSeedIPv6

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
	Completed uint64 `json:"completed"` // Number of transitions from leech to seed.
	Joined    uint64 `json:"joined"`    // Total peers that announced.
	Left      uint64 `json:"left"`      // Total peers that paused or stopped.
	Reaped    uint64 `json:"reaped"`    // Total peers cleaned up after inactivity.
	Current   uint64 `json:"current"`   // Current total peer count.

	// Stats for seeds only (subset of total).
	SeedsJoined  uint64 `json:"seeds_joined"`  // Seeds that announced (does not included leechers that completed).
	SeedsLeft    uint64 `json:"seeds_left"`    // Seeds that paused or stopped.
	SeedsReaped  uint64 `json:"seeds_reaped"`  // Seeds cleaned up after inactivity.
	SeedsCurrent uint64 `json:"seeds_current"` // Current seed count.
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
	responseTimeEvents chan time.Duration
}

func New(chanSize int) *Stats {
	s := &Stats{
		Start:  time.Now(),
		events: make(chan int, chanSize),

		responseTimeEvents: make(chan time.Duration, chanSize),
		ResponseTime: PercentileTimes{
			P50: NewPercentile(0.5),
			P90: NewPercentile(0.9),
			P95: NewPercentile(0.95),
		},
	}

	go s.handleEvents()
	go s.handleTimings()

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

func (s *Stats) RecordTiming(event int, duration time.Duration) {
	switch event {
	case ResponseTime:
		s.responseTimeEvents <- duration
	default:
		panic("stats: RecordTiming called with an unknown event")
	}
}

func (s *Stats) handleEvents() {
	for event := range s.events {
		switch event {
		case Announce:
			s.Announces++
		case Scrape:
			s.Scrapes++

		case CompletedIPv4:
			s.IPv4Peers.Completed++
			s.IPv4Peers.SeedsCurrent++
		case NewLeechIPv4:
			s.IPv4Peers.Joined++
			s.IPv4Peers.Current++
		case DeletedLeechIPv4:
			s.IPv4Peers.Left++
			s.IPv4Peers.Current--
		case ReapedLeechIPv4:
			s.IPv4Peers.Reaped++
			s.IPv4Peers.Current--

		case NewSeedIPv4:
			s.IPv4Peers.SeedsJoined++
			s.IPv4Peers.SeedsCurrent++
			s.IPv4Peers.Joined++
			s.IPv4Peers.Current++
		case DeletedSeedIPv4:
			s.IPv4Peers.SeedsLeft++
			s.IPv4Peers.SeedsCurrent--
			s.IPv4Peers.Left++
			s.IPv4Peers.Current--
		case ReapedSeedIPv4:
			s.IPv4Peers.SeedsReaped++
			s.IPv4Peers.SeedsCurrent--
			s.IPv4Peers.Reaped++
			s.IPv4Peers.Current--

		case CompletedIPv6:
			s.IPv6Peers.Completed++
			s.IPv6Peers.SeedsCurrent++
		case NewLeechIPv6:
			s.IPv6Peers.Joined++
			s.IPv6Peers.Current++
		case DeletedLeechIPv6:
			s.IPv6Peers.Left++
			s.IPv6Peers.Current--
		case ReapedLeechIPv6:
			s.IPv6Peers.Reaped++
			s.IPv6Peers.Current--

		case NewSeedIPv6:
			s.IPv6Peers.SeedsJoined++
			s.IPv6Peers.SeedsCurrent++
			s.IPv6Peers.Joined++
			s.IPv6Peers.Current++
		case DeletedSeedIPv6:
			s.IPv6Peers.SeedsLeft++
			s.IPv6Peers.SeedsCurrent--
			s.IPv6Peers.Left++
			s.IPv6Peers.Current--
		case ReapedSeedIPv6:
			s.IPv6Peers.SeedsReaped++
			s.IPv6Peers.SeedsCurrent--
			s.IPv6Peers.Reaped++
			s.IPv6Peers.Current--

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
}

func (s *Stats) handleTimings() {
	for {
		select {
		case duration := <-s.responseTimeEvents:
			f := float64(duration) / float64(time.Millisecond)
			s.ResponseTime.P50.AddSample(f)
			s.ResponseTime.P90.AddSample(f)
			s.ResponseTime.P95.AddSample(f)
		}
	}
}

// RecordEvent broadcasts an event to the default stats queue.
func RecordEvent(event int) {
	DefaultStats.RecordEvent(event)
}

// RecordTiming broadcasts a timing event to the default stats queue.
func RecordTiming(event int, duration time.Duration) {
	DefaultStats.RecordTiming(event, duration)
}
