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

type PeerStats struct {
	// Stats for all peers.
	Completed uint64
	Joined    uint64
	Left      uint64
	Reaped    uint64

	// Stats for seeds only (subset of total).
	SeedsJoined uint64
	SeedsLeft   uint64
	SeedsReaped uint64
}

type Stats struct {
	Start time.Time

	Announces uint64
	Scrapes   uint64

	IPv4Peers PeerStats
	IPv6Peers PeerStats

	TorrentsAdded   uint64
	TorrentsRemoved uint64
	TorrentsReaped  uint64

	ActiveConnections   uint64
	ConnectionsAccepted uint64
	BytesTransmitted    uint64

	RequestsHandled uint64
	RequestsErrored uint64

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
		case DeletedSeedIPv4:
			s.IPv4Peers.SeedsLeft++
		case ReapedSeedIPv4:
			s.IPv4Peers.SeedsReaped++

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
		case DeletedSeedIPv6:
			s.IPv6Peers.SeedsLeft++
		case ReapedSeedIPv6:
			s.IPv6Peers.SeedsReaped++

		case NewTorrent:
			s.TorrentsAdded++
		case DeletedTorrent:
			s.TorrentsRemoved++
		case ReapedTorrent:
			s.TorrentsReaped++

		case AcceptedConnection:
			s.ConnectionsAccepted++
			s.ActiveConnections++

		case ClosedConnection:
			s.ActiveConnections--

		case HandledRequest:
			s.RequestsHandled++

		case ErroredRequest:
			s.RequestsErrored++

		default:
			panic("stats: RecordEvent called with an unknown event")
		}
	}
}
