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
	Complete    int32
	Incomplete  int32
	Interval    time.Duration
	MinInterval time.Duration
	IPv4Peers   []Peer
	IPv6Peers   []Peer
}

// AnnounceHandler is a function that generates a response for an Announce.
type AnnounceHandler func(*AnnounceRequest) *AnnounceResponse

// AnnounceCallback is a function that does something with the results of an
// Announce after it has been completed.
type AnnounceCallback func(*AnnounceRequest, *AnnounceResponse)

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
	Snatches   uint32
	Complete   uint32
	Incomplete uint32
}

// ScrapeHandler is a function that generates a response for a Scrape.
type ScrapeHandler func(*ScrapeRequest) *ScrapeResponse

// ScrapeCallback is a function that does something with the results of a
// Scrape after it has been completed.
type ScrapeCallback func(*ScrapeRequest, *ScrapeResponse)

// Peer represents the connection details of a peer that is returned in an
// announce response.
type Peer struct {
	ID   PeerID
	IP   net.IP
	Port uint16
}

// Equal reports whether p and x are the same.
func (p Peer) Equal(x Peer) bool { return p.EqualEndpoint(x) && p.ID == x.ID }

// EqualEndpoint reports whether p and x have the same endpoint.
func (p Peer) EqualEndpoint(x Peer) bool { return p.Port == x.Port && p.IP.Equal(x.IP) }

// Params is used to fetch request optional parameters.
type Params interface {
	String(key string) (string, error)
}

// ClientError represents an error that should be exposed to the client over
// the BitTorrent protocol implementation.
type ClientError string

// Error implements the error interface for ClientError.
func (c ClientError) Error() string { return string(c) }

// Server represents an implementation of the BitTorrent tracker protocol.
type Server interface {
	ListenAndServe() error
	Stop()
}

// ServerFuncs are the collection of protocol-agnostic functions used to handle
// requests in a Server.
type ServerFuncs struct {
	HandleAnnounce AnnounceHandler
	HandleScrape   ScrapeHandler
	AfterAnnounce  AnnounceCallback
	AfterScrape    ScrapeCallback
}
