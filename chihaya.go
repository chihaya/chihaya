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
type PeerID [20]byte

// PeerIDFromBytes creates a PeerID from a byte slice.
//
// It panics if b is not 20 bytes long.
func PeerIDFromBytes(b []byte) PeerID {
	if len(b) != 20 {
		panic("peer ID must be 20 bytes")
	}

	var buf [20]byte
	copy(buf[:], b)
	return PeerID(buf)
}

// PeerIDFromString creates a PeerID from a string.
//
// It panics if s is not 20 bytes long.
func PeerIDFromString(s string) PeerID {
	if len(s) != 20 {
		panic("peer ID must be 20 bytes")
	}

	var buf [20]byte
	copy(buf[:], s)
	return PeerID(buf)
}

// InfoHash represents an infohash.
type InfoHash [20]byte

// InfoHashFromBytes creates an InfoHash from a byte slice.
//
// It panics if b is not 20 bytes long.
func InfoHashFromBytes(b []byte) InfoHash {
	if len(b) != 20 {
		panic("infohash must be 20 bytes")
	}

	var buf [20]byte
	copy(buf[:], b)
	return InfoHash(buf)
}

// InfoHashFromString creates an InfoHash from a string.
//
// It panics if s is not 20 bytes long.
func InfoHashFromString(s string) InfoHash {
	if len(s) != 20 {
		panic("infohash must be 20 bytes")
	}

	var buf [20]byte
	copy(buf[:], s)
	return InfoHash(buf)
}

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

// Peer4 returns a Peer using the IPv4 endpoint of the Announce.
// Note that, if the Announce does not contain an IPv4 address, the IP field of
// the returned Peer can be nil.
func (r *AnnounceRequest) Peer4() Peer {
	return Peer{
		IP:   r.IPv4,
		Port: r.Port,
		ID:   r.PeerID,
	}
}

// Peer6 returns a Peer using the IPv6 endpoint of the Announce.
// Note that, if the Announce does not contain an IPv6 address, the IP field of
// the returned Peer can be nil.
func (r *AnnounceRequest) Peer6() Peer {
	return Peer{
		IP:   r.IPv6,
		Port: r.Port,
		ID:   r.PeerID,
	}
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

// Equal reports whether p and x are the same.
func (p Peer) Equal(x Peer) bool {
	return p.EqualEndpoint(x) && p.ID == x.ID
}

// EqualEndpoint reports whether p and x have the same endpoint.
func (p Peer) EqualEndpoint(x Peer) bool {
	return p.Port == x.Port && p.IP.Equal(x.IP)
}

// Params is used to fetch request parameters.
type Params interface {
	String(key string) (string, error)
}
