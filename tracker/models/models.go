// Copyright 2015 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

// Package models implements the common data types used throughout a BitTorrent
// tracker.
package models

import (
	"net"
	"strings"
	"time"

	"github.com/chihaya/chihaya/config"
)

var (
	// ErrMalformedRequest is returned when a request does not contain the
	// required parameters needed to create a model.
	ErrMalformedRequest = ClientError("malformed request")

	// ErrBadRequest is returned when a request is invalid in the peer's
	// current state. For example, announcing a "completed" event while
	// not a leecher or a "stopped" event while not active.
	ErrBadRequest = ClientError("bad request")

	// ErrTorrentDNE is returned when a torrent does not exist.
	ErrTorrentDNE = NotFoundError("torrent does not exist")

	// ErrClientUnapproved is returned when a clientID is not in the whitelist.
	ErrClientUnapproved = ClientError("client is not approved")
)

type ClientError string
type NotFoundError ClientError
type ProtocolError ClientError

func (e ClientError) Error() string   { return string(e) }
func (e NotFoundError) Error() string { return string(e) }
func (e ProtocolError) Error() string { return string(e) }

// IsPublicError determines whether an error should be propogated to the client.
func IsPublicError(err error) bool {
	_, cl := err.(ClientError)
	_, nf := err.(NotFoundError)
	_, pc := err.(ProtocolError)
	return cl || nf || pc
}

// PeerList represents a list of peers: either seeders or leechers.
type PeerList []Peer

// PeerKey is the key used to uniquely identify a peer in a swarm.
type PeerKey string

// NewPeerKey creates a properly formatted PeerKey.
func NewPeerKey(peerID string, ip net.IP) PeerKey {
	return PeerKey(peerID + "//" + ip.String())
}

// IP parses and returns the IP address for a given PeerKey.
func (pk PeerKey) IP() net.IP {
	ip := net.ParseIP(strings.Split(string(pk), "//")[1])
	if rval := ip.To4(); rval != nil {
		return rval
	}
	return ip
}

// PeerID returns the PeerID section of a PeerKey.
func (pk PeerKey) PeerID() string {
	return strings.Split(string(pk), "//")[0]
}

// Endpoint is an IP and port pair.
//
// IP always has length net.IPv4len if IPv4, and net.IPv6len if IPv6.
type Endpoint struct {
	IP   net.IP `json:"ip"`
	Port uint16 `json:"port"`
}

// Peer represents a participant in a BitTorrent swarm.
type Peer struct {
	ID           string `json:"id"`
	Uploaded     uint64 `json:"uploaded"`
	Downloaded   uint64 `json:"downloaded"`
	Left         uint64 `json:"left"`
	LastAnnounce int64  `json:"lastAnnounce"`
	Endpoint
}

// HasIPv4 determines if a peer's IP address can be represented as an IPv4
// address.
func (p *Peer) HasIPv4() bool {
	return !p.HasIPv6()
}

// HasIPv6 determines if a peer's IP address can be represented as an IPv6
// address.
func (p *Peer) HasIPv6() bool {
	return len(p.IP) == net.IPv6len
}

// Key returns a PeerKey for the given peer.
func (p *Peer) Key() PeerKey {
	return NewPeerKey(p.ID, p.IP)
}

// Torrent represents a BitTorrent swarm and its metadata.
type Torrent struct {
	Infohash   string `json:"infohash"`
	Snatches   uint64 `json:"snatches"`
	LastAction int64  `json:"lastAction"`

	Seeders  *PeerMap `json:"seeders"`
	Leechers *PeerMap `json:"leechers"`
}

// PeerCount returns the total number of peers connected on this Torrent.
func (t *Torrent) PeerCount() int {
	return t.Seeders.Len() + t.Leechers.Len()
}

// Announce is an Announce by a Peer.
type Announce struct {
	Config *config.Config `json:"config"`

	Compact    bool     `json:"compact"`
	Downloaded uint64   `json:"downloaded"`
	Event      string   `json:"event"`
	IPv4       Endpoint `json:"ipv4"`
	IPv6       Endpoint `json:"ipv6"`
	Infohash   string   `json:"infohash"`
	Left       uint64   `json:"left"`
	NumWant    int      `json:"numwant"`
	PeerID     string   `json:"peer_id"`
	Uploaded   uint64   `json:"uploaded"`
	JWT        string   `json:"jwt"`

	Torrent *Torrent `json:"-"`
	Peer    *Peer    `json:"-"`
	PeerV4  *Peer    `json:"-"` // Only valid if HasIPv4() is true.
	PeerV6  *Peer    `json:"-"` // Only valid if HasIPv6() is true.
}

// ClientID returns the part of a PeerID that identifies a Peer's client
// software.
func (a *Announce) ClientID() (clientID string) {
	length := len(a.PeerID)
	if length >= 6 {
		if a.PeerID[0] == '-' {
			if length >= 7 {
				clientID = a.PeerID[1:7]
			}
		} else {
			clientID = a.PeerID[:6]
		}
	}

	return
}

// HasIPv4 determines whether or not an announce has an IPv4 endpoint.
func (a *Announce) HasIPv4() bool {
	return a.IPv4.IP != nil
}

// HasIPv6 determines whether or not an announce has an IPv6 endpoint.
func (a *Announce) HasIPv6() bool {
	return a.IPv6.IP != nil
}

// BuildPeer creates the Peer representation of an Announce. BuildPeer creates
// one peer for each IP in the announce, and panics if there are none.
func (a *Announce) BuildPeer(t *Torrent) {
	a.Peer = &Peer{
		ID:           a.PeerID,
		Uploaded:     a.Uploaded,
		Downloaded:   a.Downloaded,
		Left:         a.Left,
		LastAnnounce: time.Now().Unix(),
	}

	if t != nil {
		a.Torrent = t
	}

	if a.HasIPv4() && a.HasIPv6() {
		a.PeerV4 = a.Peer
		a.PeerV4.Endpoint = a.IPv4
		a.PeerV6 = &*a.Peer
		a.PeerV6.Endpoint = a.IPv6
	} else if a.HasIPv4() {
		a.PeerV4 = a.Peer
		a.PeerV4.Endpoint = a.IPv4
	} else if a.HasIPv6() {
		a.PeerV6 = a.Peer
		a.PeerV6.Endpoint = a.IPv6
	} else {
		panic("models: announce must have an IP")
	}

	return
}

// AnnounceResponse contains the information needed to fulfill an announce.
type AnnounceResponse struct {
	Announce              *Announce
	Complete, Incomplete  int
	Interval, MinInterval time.Duration
	IPv4Peers, IPv6Peers  PeerList

	Compact bool
}

// Scrape is a Scrape by a Peer.
type Scrape struct {
	Config     *config.Config `json:"config"`
	Infohashes []string
}

// ScrapeResponse contains the information needed to fulfill a scrape.
type ScrapeResponse struct {
	Files []*Torrent
}
