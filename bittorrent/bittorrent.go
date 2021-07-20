// Package bittorrent implements all of the abstractions used to decouple the
// protocol of a BitTorrent tracker from the logic of handling Announces and
// Scrapes.
package bittorrent

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"net"
	"strings"
	"time"

	"inet.af/netaddr"

	"github.com/chihaya/chihaya/pkg/log"
	"github.com/jzelinskie/stringz"
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

// String implements fmt.Stringer, returning a string of hex encoded bytes.
func (p PeerID) String() string {
	var b strings.Builder
	b.Grow(40) // 2 chars * 20 bytes

	w := hex.NewEncoder(&b)
	w.Write(p[:])

	return b.String()
}

// RawString implements returns the bytes of a PeerID interpretted as a string.
// of the ID.
func (p PeerID) RawString() string {
	return string(p[:])
}

// ClientID represents the part of a PeerID that identifies a Peer's client
// software.
type ClientID [6]byte

// ClientID returns the client-identifying section of a PeerID.
func (p PeerID) ClientID() ClientID {
	var cid ClientID
	length := len(p)
	if length >= 6 {
		if p[0] == '-' {
			if length >= 7 {
				copy(cid[:], p[1:7])
			}
		} else {
			copy(cid[:], p[:6])
		}
	}

	return cid
}

// PeerIDFromRawString creates a PeerID from a string.
//
// It panics if s is not 20 bytes long.
func PeerIDFromRawString(s string) PeerID {
	if len(s) != 20 {
		panic("peer ID must be 20 bytes")
	}

	var buf [20]byte
	copy(buf[:], s)
	return PeerID(buf)
}

// PeerIDFromHexString creates a PeerID from a hex string.
//
// It panics if s is not 40 bytes long.
func PeerIDFromHexString(s string) PeerID {
	if len(s) != 40 {
		panic("peer ID must be 40 bytes")
	}

	var buf [20]byte
	hex.Decode(buf[:], []byte(s))

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
func (i InfoHash) String() string {
	return fmt.Sprintf("%x", i[:])
}

// RawString returns a 20-byte string of the raw bytes of the InfoHash.
func (i InfoHash) RawString() string {
	return string(i[:])
}

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
	InfoHashes []InfoHash
	Params     Params
	Peer
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
	ID     PeerID
	IPPort netaddr.IPPort
}

// PeerFromRawString parses a Peer from its raw string representation.
func PeerFromRawString(s string) Peer {
	ip, ok := netaddr.FromStdIP(net.IP(s[22:]))
	if !ok {
		panic("failed to parse IP")
	}

	return Peer{
		ID:     PeerIDFromRawString(s[:20]),
		IPPort: netaddr.IPPortFrom(ip, binary.BigEndian.Uint16([]byte(s[20:22]))),
	}
}

// PeerFromString parses a Peer from a human-friendly string representation of
// a Peer.
//
// This function panics is the string fails to parse.
func PeerFromString(s string) Peer {
	var hexPeerID, ipport string
	if err := stringz.Unpack(strings.Split(s, "@"), &hexPeerID, &ipport); err != nil {
		panic("failed to scan peer string: " + err.Error())
	}

	return Peer{
		ID:     PeerIDFromHexString(hexPeerID),
		IPPort: netaddr.MustParseIPPort(ipport),
	}
}

// String implements fmt.Stringer for a human-friendly representation of a
// Peer.
func (p Peer) String() string {
	return fmt.Sprintf("%s@%s", p.ID, p.IPPort)
}

// RawString implements a memory-efficient representation of a Peer.
//
// The format is:
//    20-byte PeerID
//    2-byte Big Endian Port
//    4-byte or 16-byte IP address
func (p Peer) RawString() string {
	var b strings.Builder
	ip := p.IPPort.IP()
	switch {
	case ip.Is4(), ip.Is4in6():
		b.Grow(20 + 2 + 4) // PeerID + Port + IPv4
	case ip.Is6():
		b.Grow(20 + 2 + 16) // PeerID + Port + IPv6
	default:
		panic("unknown IP version")
	}

	if _, err := b.WriteString(p.ID.RawString()); err != nil {
		panic("failed to write peer ID to strings.Builder: " + err.Error())
	}

	if err := binary.Write(&b, binary.BigEndian, p.IPPort.Port()); err != nil {
		panic("failed to write port to strings.Builder: " + err.Error())
	}

	binaryIP, err := ip.MarshalBinary()
	if err != nil {
		panic("netaddr.IP.MarshalBinary() returned an error: " + err.Error())
	}
	if _, err := b.Write(binaryIP); err != nil {
		panic("failed to write binary IP to strings.Builder: " + err.Error())
	}

	return b.String()
}

// LogFields renders the current peer as a set of Logrus fields.
func (p Peer) LogFields() log.Fields {
	return log.Fields{
		"id":   p.ID,
		"ip":   p.IPPort.IP(),
		"port": p.IPPort.Port(),
	}
}

// Equal reports whether p and x are the same.
func (p Peer) Equal(x Peer) bool { return p.EqualEndpoint(x) && p.ID == x.ID }

// EqualEndpoint reports whether p and x have the same endpoint.
func (p Peer) EqualEndpoint(x Peer) bool {
	return p.IPPort.Port() == x.IPPort.Port() && p.IPPort.IP().Compare(x.IPPort.IP()) == 0
}

// ClientError represents an error that should be exposed to the client over
// the BitTorrent protocol implementation.
type ClientError string

// Error implements the error interface for ClientError.
func (c ClientError) Error() string { return string(c) }
