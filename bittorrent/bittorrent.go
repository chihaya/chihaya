// Package bittorrent implements all of the abstractions used to decouple the
// protocol of a BitTorrent tracker from the logic of handling Announces and
// Scrapes.
package bittorrent

import (
	"fmt"
	"log/slog"
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

// String implements fmt.Stringer, returning the base16 encoded PeerID.
func (p PeerID) String() string {
	return fmt.Sprintf("%x", p[:])
}

// RawString returns a 20-byte string of the raw bytes of the ID.
func (p PeerID) RawString() string {
	return string(p[:])
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

// LogValue renders the current request as a set of log fields.
func (r AnnounceRequest) LogValue() slog.Value {
	return slog.GroupValue(
		slog.String("event", r.Event.String()),
		slog.String("infoHash", r.InfoHash.String()),
		slog.Bool("compact", r.Compact),
		slog.Bool("eventProvided", r.EventProvided),
		slog.Bool("numWantProvided", r.NumWantProvided),
		slog.Bool("ipProvided", r.IPProvided),
		slog.Uint64("numWant", uint64(r.NumWant)),
		slog.Uint64("left", r.Left),
		slog.Uint64("downloaded", r.Downloaded),
		slog.Uint64("uploaded", r.Uploaded),
		slog.Any("peer", &r.Peer),
		slog.String("params", r.RawQuery()),
	)
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

// LogValue renders the current response as a set of log fields.
func (r AnnounceResponse) LogValue() slog.Value {
	ipv4s := make([]slog.Value, 0, len(r.IPv4Peers))
	for _, p := range r.IPv4Peers {
		ipv4s = append(ipv4s, p.LogValue())
	}

	ipv6s := make([]slog.Value, 0, len(r.IPv6Peers))
	for _, p := range r.IPv6Peers {
		ipv6s = append(ipv6s, p.LogValue())
	}

	return slog.GroupValue(
		slog.Bool("compact", r.Compact),
		slog.Uint64("complete", uint64(r.Complete)),
		slog.Uint64("incomplete", uint64(r.Incomplete)),
		slog.Duration("interval", r.Interval),
		slog.Duration("minInterval", r.MinInterval),
		// TODO(jzelinskie): avoid reflection for these
		slog.Any("ipv4Peers", ipv4s),
		slog.Any("ipv6Peers", ipv6s),
	)
}

// ScrapeRequest represents the parsed parameters from a scrape request.
type ScrapeRequest struct {
	AddressFamily AddressFamily
	InfoHashes    []InfoHash
	Params        Params
}

// LogValue renders the request as a set of log fields.
func (r ScrapeRequest) LogValue() slog.Value {
	return slog.GroupValue(
		slog.String("addressFamily", r.AddressFamily.String()),
		slog.Any("infoHashes", r.InfoHashes), // TODO(jzelinskie): avoid reflection for this
		slog.String("params", r.Params.RawQuery()),
	)
}

// ScrapeResponse represents the parameters used to create a scrape response.
//
// The Scrapes must be in the same order as the InfoHashes in the corresponding
// ScrapeRequest.
type ScrapeResponse struct {
	Files []Scrape
}

// LogValue renders the response as a set of log fields.
func (sr ScrapeResponse) LogValue() slog.Value {
	files := make([]slog.Value, 0, len(sr.Files))
	for _, f := range sr.Files {
		files = append(files, f.LogValue())
	}

	return slog.GroupValue(slog.Any("files", files))
}

// Scrape represents the state of a swarm that is returned in a scrape response.
type Scrape struct {
	InfoHash   InfoHash
	Snatches   uint32
	Complete   uint32
	Incomplete uint32
}

// LogValue renders the current swarm as a set of log fields.
func (s Scrape) LogValue() slog.Value {
	return slog.GroupValue(
		slog.String("infoHash", s.InfoHash.String()),
		slog.Uint64("snatches", uint64(s.Snatches)),
		slog.Uint64("complete", uint64(s.Complete)),
		slog.Uint64("incomplete", uint64(s.Incomplete)),
	)
}

// AddressFamily is the address family of an IP address.
type AddressFamily uint8

func (af AddressFamily) String() string {
	switch af {
	case IPv4:
		return "IPv4"
	case IPv6:
		return "IPv6"
	default:
		panic("tried to print unknown AddressFamily")
	}
}

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

func (ip IP) String() string {
	return ip.IP.String()
}

// Peer represents the connection details of a peer that is returned in an
// announce response.
type Peer struct {
	ID   PeerID
	IP   IP
	Port uint16
}

// String implements fmt.Stringer to return a human-readable representation.
// The string will have the format <PeerID>@[<IP>]:<port>, for example
// "0102030405060708090a0b0c0d0e0f1011121314@[10.11.12.13]:1234"
func (p Peer) String() string {
	return fmt.Sprintf("%s@[%s]:%d", p.ID.String(), p.IP.String(), p.Port)
}

// LogValue renders the Peer as a set of log fields.
func (p Peer) LogValue() slog.Value {
	return slog.GroupValue(
		slog.String("ID", p.ID.String()),
		slog.String("IP", p.IP.String()),
		slog.Int("port", int(p.Port)),
	)
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
