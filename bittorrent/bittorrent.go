// Package bittorrent implements all of the abstractions used to decouple the
// protocol of a BitTorrent tracker from the logic of handling Announces and
// Scrapes.
package bittorrent

import (
	"net"
	"time"
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
	Event      Event
	InfoHash   InfoHash
	Compact    bool
	NumWant    uint32
	Left       uint64
	Downloaded uint64
	Uploaded   uint64

	Peer
	Params
}

// AnnounceResponse represents the parameters used to create an announce
// response.
type AnnounceResponse struct {
	Compact     bool
	Complete    uint32
	Incomplete  uint32
	Interval    time.Duration
	MinInterval time.Duration
	IPv4Peers   []Peer
	IPv6Peers   []Peer
}

// ScrapeRequest represents the parsed parameters from a scrape request.
type ScrapeRequest struct {
	AddressFamily AddressFamily
	InfoHashes    []InfoHash
	Params        Params
}

// ScrapeResponse represents the parameters used to create a scrape response.
//
// The Scrapes must be in the same order as the InfoHashes in the corresponding
// ScrapeRequest.
type ScrapeResponse struct {
	Files []Scrape
}

// Scrape represents the state of a swarm that is returned in a scrape response.
type Scrape struct {
	InfoHash   InfoHash
	Snatches   uint32
	Complete   uint32
	Incomplete uint32
}

// AddressFamily is the address family of an IP address.
type AddressFamily uint8

// AddressFamily constants.
const (
	IPv4 AddressFamily = iota
	IPv6
)

// IP is a net.IP with an AddressFamily.
type IP struct {
	net.IP
	AddressFamily
}

// Peer represents the connection details of a peer that is returned in an
// announce response.
type Peer struct {
	ID   PeerID
	IP   IP
	Port uint16
}

// Equal reports whether p and x are the same.
func (p Peer) Equal(x Peer) bool { return p.EqualEndpoint(x) && p.ID == x.ID }

// EqualEndpoint reports whether p and x have the same endpoint.
func (p Peer) EqualEndpoint(x Peer) bool { return p.Port == x.Port && p.IP.Equal(x.IP.IP) }

// ClientError represents an error that should be exposed to the client over
// the BitTorrent protocol implementation.
type ClientError string

// Error implements the error interface for ClientError.
func (c ClientError) Error() string { return string(c) }
