// Copyright 2016 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.package middleware

package chihaya

import (
	"net"
	"time"

	"github.com/chihaya/chihaya/pkg/event"
)

// PeerID represents a peer ID.
type PeerID string

// InfoHash represents an infohash in hexadecimal notation.
type InfoHash string

// AnnounceRequest represents the parsed parameters from an announce request.
type AnnounceRequest struct {
	Event    event.Event
	InfoHash InfoHash
	PeerID   PeerID

	IPv4, IPv6 net.IP
	Port       uint16

	Compact bool
	NumWant int32

	Left, Downloaded, Uploaded uint64

	Params Params
}

// AnnounceResponse represents the parameters used to create an announce
// response.
type AnnounceResponse struct {
	Compact     bool
	Complete    int32
	Incomplete  int32
	Interval    time.Duration
	MinInterval time.Duration
	IPv4Peers   []Peer
	IPv6Peers   []Peer
}

// ScrapeRequest represents the parsed parameters from a scrape request.
type ScrapeRequest struct {
	InfoHashes []InfoHash
	Params     Params
}

// ScrapeResponse represents the parameters used to create a scrape response.
type ScrapeResponse struct {
	Files map[InfoHash]Scrape
}

// Scrape represents the state of a swarm that is returned in a scrape response.
type Scrape struct {
	Complete   int32
	Incomplete int32
}

// Peer represents the connection details of a peer that is returned in an
// announce response.
type Peer struct {
	ID   PeerID
	IP   net.IP
	Port uint16
}

// Params is used to fetch request parameters.
type Params interface {
	String(key string) (string, error)
}

// Equal reports whether peer and x are the same.
func (peer *Peer) Equal(x *Peer) bool {
	if peer.ID == x.ID && peer.Port == x.Port && peer.IP.Equal(x.IP) {
		return true
	}
	return false
}
