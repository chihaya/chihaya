// Package bittorrent implements all of the abstractions used to decouple the
// protocol of a BitTorrent tracker from the logic of handling Announces and
// Scrapes.
package bittorrent

import (
	"encoding/binary"
	"fmt"
	"net/netip"
	"time"

	"github.com/chihaya/chihaya/pkg/iputil"
	"github.com/chihaya/chihaya/pkg/log"
)

// PeerID represents a peer ID.
type PeerID [20]byte

// String implements fmt.Stringer, returning the base16 encoded PeerID.
func (p PeerID) String() string { return fmt.Sprintf("%x", p[:]) }

// MarshalBinary returns a 20-byte string of the raw bytes of the ID.
func (p PeerID) MarshalBinary() []byte { return p[:] }

// PeerIDFromBytes creates a PeerID from bytes.
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

// String implements fmt.Stringer, returning the base16 encoded InfoHash.
func (i InfoHash) String() string { return fmt.Sprintf("%x", i[:]) }

// MarshalBinary returns a 20-byte string of the raw bytes of the InfoHash.
func (i InfoHash) MarshalBinary() []byte { return i[:] }

// AnnounceRequest represents the parsed parameters from an announce request.
type AnnounceRequest struct {
	Event           Event
	InfoHash        InfoHash
	Compact         bool
	EventProvided   bool
	NumWantProvided bool
	IPProvided      bool
	NumWant         uint32
	Left            uint64
	Downloaded      uint64
	Uploaded        uint64

	Peer
	Params
}

// LogFields renders the current response as a set of log fields.
func (r AnnounceRequest) LogFields() log.Fields {
	return log.Fields{
		"event":           r.Event,
		"infoHash":        r.InfoHash,
		"compact":         r.Compact,
		"eventProvided":   r.EventProvided,
		"numWantProvided": r.NumWantProvided,
		"ipProvided":      r.IPProvided,
		"numWant":         r.NumWant,
		"left":            r.Left,
		"downloaded":      r.Downloaded,
		"uploaded":        r.Uploaded,
		"peer":            r.Peer,
		"params":          r.Params,
	}
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

// LogFields renders the current response as a set of log fields.
func (r AnnounceResponse) LogFields() log.Fields {
	return log.Fields{
		"compact":     r.Compact,
		"complete":    r.Complete,
		"interval":    r.Interval,
		"minInterval": r.MinInterval,
		"ipv4Peers":   r.IPv4Peers,
		"ipv6Peers":   r.IPv6Peers,
	}
}

// ScrapeRequest represents the parsed parameters from a scrape request.
type ScrapeRequest struct {
	Peer
	InfoHashes []InfoHash
	Params     Params
}

// LogFields renders the current response as a set of log fields.
func (r ScrapeRequest) LogFields() log.Fields {
	return log.Fields{
		"peer":       r.Peer,
		"infoHashes": r.InfoHashes,
		"params":     r.Params,
	}
}

// ScrapeResponse represents the parameters used to create a scrape response.
//
// The Scrapes must be in the same order as the InfoHashes in the corresponding
// ScrapeRequest.
type ScrapeResponse struct {
	Files []Scrape
}

// LogFields renders the current response as a set of Logrus fields.
func (sr ScrapeResponse) LogFields() log.Fields {
	return log.Fields{
		"files": sr.Files,
	}
}

// Scrape represents the state of a swarm that is returned in a scrape response.
type Scrape struct {
	InfoHash   InfoHash
	Snatches   uint32
	Complete   uint32
	Incomplete uint32
}

// Peer represents the connection details of a peer that is returned in an
// announce response.
type Peer struct {
	ID       PeerID
	AddrPort netip.AddrPort
}

// String implements fmt.Stringer to return a human-readable representation.
// The string will have the format <PeerID>@[<IP>]:<port>, for example
// "0102030405060708090a0b0c0d0e0f1011121314@[10.11.12.13]:1234"
func (p Peer) String() string {
	return fmt.Sprintf("%s@[%s]:%d", p.ID, p.AddrPort.Addr(), p.AddrPort.Port())
}

// MarshalBinary encodes a Peer into a memory-efficient byte representation.
//
// The format is:
//    20-byte PeerID
//    2-byte Big Endian Port
//    4-byte or 16-byte IP address
func (p Peer) MarshalBinary() []byte {
	ip := p.AddrPort.Addr().Unmap()
	b := make([]byte, 20+2+(ip.BitLen()/8))
	copy(b[:20], p.ID[:])
	binary.BigEndian.PutUint16(b[20:22], p.AddrPort.Port())
	copy(b[22:], ip.AsSlice())
	return b
}

// PeerFromBytes parses a Peer from its raw representation.
func PeerFromBytes(b []byte) Peer {
	return Peer{
		ID: PeerIDFromBytes(b[:20]),
		AddrPort: netip.AddrPortFrom(
			iputil.MustAddrFromSlice(b[22:]).Unmap(),
			binary.BigEndian.Uint16(b[20:22]),
		),
	}
}

// LogFields renders the current peer as a set of Logrus fields.
func (p Peer) LogFields() log.Fields {
	return log.Fields{
		"ID":   p.ID,
		"IP":   p.AddrPort.Addr().String(),
		"port": p.AddrPort.Port(),
	}
}

// Equal reports whether p and x are the same.
func (p Peer) Equal(x Peer) bool { return p.EqualEndpoint(x) && p.ID == x.ID }

// EqualEndpoint reports whether p and x have the same endpoint.
func (p Peer) EqualEndpoint(x Peer) bool {
	return p.AddrPort.Port() == x.AddrPort.Port() &&
		p.AddrPort.Addr().Compare(x.AddrPort.Addr()) == 0
}

// ClientError represents an error that should be exposed to the client over
// the BitTorrent protocol implementation.
type ClientError string

// Error implements the error interface for ClientError.
func (c ClientError) Error() string { return string(c) }
