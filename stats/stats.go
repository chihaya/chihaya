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
)

// DefaultStats is a default instance of stats tracking that uses an unbuffered
// channel for broadcasting events.
var DefaultStats *Stats

func init() {
	DefaultStats = New(0)
}

type PeerStats struct {
	// Stats for all peers.
	Completed uint64 `json:"completed"`
	Joined    uint64 `json:"joined"`
	Left      uint64 `json:"left"`
	Reaped    uint64 `json:"reaped"`

	// Stats for seeds only (subset of total).
	SeedsJoined uint64 `json:"seeds_joined"`
	SeedsLeft   uint64 `json:"seeds_left"`
	SeedsReaped uint64 `json:"seeds_reaped"`
}

type Stats struct {
	Start time.Time `json:"start_time"`

	Announces uint64 `json:"announces"`
	Scrapes   uint64 `json:"scrapes"`

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

		case CompletedIPv4:
			s.IPv4Peers.Completed++
		case NewLeechIPv4:
			s.IPv4Peers.Joined++
		case DeletedLeechIPv4:
			s.IPv4Peers.Left++
		case ReapedLeechIPv4:
			s.IPv4Peers.Reaped++

		case NewSeedIPv4:
			s.IPv4Peers.SeedsJoined++
			s.IPv4Peers.Joined++
		case DeletedSeedIPv4:
			s.IPv4Peers.SeedsLeft++
			s.IPv4Peers.Left++
		case ReapedSeedIPv4:
			s.IPv4Peers.SeedsReaped++
			s.IPv4Peers.Reaped++

		case CompletedIPv6:
			s.IPv6Peers.Completed++
		case NewLeechIPv6:
			s.IPv6Peers.Joined++
		case DeletedLeechIPv6:
			s.IPv6Peers.Left++
		case ReapedLeechIPv6:
			s.IPv6Peers.Reaped++

		case NewSeedIPv6:
			s.IPv6Peers.SeedsJoined++
			s.IPv6Peers.Joined++
		case DeletedSeedIPv6:
			s.IPv6Peers.SeedsLeft++
			s.IPv6Peers.Left++
		case ReapedSeedIPv6:
			s.IPv6Peers.SeedsReaped++
			s.IPv6Peers.Reaped++

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

// RecordEvent broadcasts an event to the default stats tracking.
func RecordEvent(event int) {
	DefaultStats.RecordEvent(event)
}
