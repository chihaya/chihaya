// Copyright 2015 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

// Package models implements the common data types used throughout a BitTorrent
// tracker.
package models

import (
	"net"
	"strconv"
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

	// ErrUserDNE is returned when a user does not exist.
	ErrUserDNE = NotFoundError("user does not exist")

	// ErrTorrentDNE is returned when a torrent does not exist.
	ErrTorrentDNE = NotFoundError("torrent does not exist")

	// ErrClientUnapproved is returned when a clientID is not in the whitelist.
	ErrClientUnapproved = ClientError("client is not approved")

	// ErrInvalidPasskey is returned when a passkey is not properly formatted.
	ErrInvalidPasskey = ClientError("passkey is invalid")
)

type ClientError string
type NotFoundError ClientError
type ProtocolError ClientError

func (e ClientError) Error() string   { return string(e) }
func (e NotFoundError) Error() string { return string(e) }
func (e ProtocolError) Error() string { return string(e) }

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
func NewPeerKey(peerID string, ip net.IP, port string) PeerKey {
	return PeerKey(peerID + "//" + ip.String() + ":" + port)
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

// Port returns the port section of the PeerKey.
func (pk PeerKey) Port() string {
	return strings.Split(string(pk), "//")[2]
}

// Peer is a participant in a swarm.
type Peer struct {
	ID        string `json:"id"`
	UserID    uint64 `json:"user_id"`
	TorrentID uint64 `json:"torrent_id"`

	// Always has length net.IPv4len if IPv4, and net.IPv6len if IPv6
	IP net.IP `json:"ip,omitempty"`

	Port uint16 `json:"port"`

	Uploaded     uint64 `json:"uploaded"`
	Downloaded   uint64 `json:"downloaded"`
	Left         uint64 `json:"left"`
	LastAnnounce int64  `json:"last_announce"`
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
	return NewPeerKey(p.ID, p.IP, strconv.FormatUint(p.Port, 10))
}

// Torrent is a swarm for a given torrent file.
type Torrent struct {
	ID       uint64 `json:"id"`
	Infohash string `json:"infohash"`

	Seeders  *PeerMap `json:"seeders"`
	Leechers *PeerMap `json:"leechers"`

	Snatches       uint64  `json:"snatches"`
	UpMultiplier   float64 `json:"up_multiplier"`
	DownMultiplier float64 `json:"down_multiplier"`
	LastAction     int64   `json:"last_action"`
}

// PeerCount returns the total number of peers connected on this Torrent.
func (t *Torrent) PeerCount() int {
	return t.Seeders.Len() + t.Leechers.Len()
}

// User is a registered user for private trackers.
type User struct {
	ID      uint64 `json:"id"`
	Passkey string `json:"passkey"`

	UpMultiplier   float64 `json:"up_multiplier"`
	DownMultiplier float64 `json:"down_multiplier"`
}

// Announce is an Announce by a Peer.
type Announce struct {
	Config *config.Config `json:"config"`

	Compact    bool   `json:"compact"`
	Downloaded uint64 `json:"downloaded"`
	Event      string `json:"event"`
	IPv4       net.IP `json:"ipv4"`
	IPv6       net.IP `json:"ipv6"`
	Infohash   string `json:"infohash"`
	Left       uint64 `json:"left"`
	NumWant    int    `json:"numwant"`
	Passkey    string `json:"passkey"`
	PeerID     string `json:"peer_id"`
	Port       uint16 `json:"port"`
	Uploaded   uint64 `json:"uploaded"`

	Torrent *Torrent `json:"-"`
	User    *User    `json:"-"`
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
			clientID = a.PeerID[0:6]
		}
	}

	return
}

func (a *Announce) HasIPv4() bool {
	return a.IPv4 != nil
}

func (a *Announce) HasIPv6() bool {
	return a.IPv6 != nil
}

// BuildPeer creates the Peer representation of an Announce. When provided nil
// for the user or torrent parameter, it creates a Peer{UserID: 0} or
// Peer{TorrentID: 0}, respectively. BuildPeer creates one peer for each IP
// in the announce, and panics if there are none.
func (a *Announce) BuildPeer(u *User, t *Torrent) {
	a.Peer = &Peer{
		ID:           a.PeerID,
		Port:         a.Port,
		Uploaded:     a.Uploaded,
		Downloaded:   a.Downloaded,
		Left:         a.Left,
		LastAnnounce: time.Now().Unix(),
	}

	if t != nil {
		a.Peer.TorrentID = t.ID
		a.Torrent = t
	}

	if u != nil {
		a.Peer.UserID = u.ID
		a.User = u
	}

	if a.HasIPv4() && a.HasIPv6() {
		a.PeerV4 = a.Peer
		a.PeerV4.IP = a.IPv4
		a.PeerV6 = &*a.Peer
		a.PeerV6.IP = a.IPv6
	} else if a.HasIPv4() {
		a.PeerV4 = a.Peer
		a.PeerV4.IP = a.IPv4
	} else if a.HasIPv6() {
		a.PeerV6 = a.Peer
		a.PeerV6.IP = a.IPv6
	} else {
		panic("models: announce must have an IP")
	}
	return
}

// AnnounceDelta contains the changes to a Peer's state. These changes are
// recorded by the backend driver.
type AnnounceDelta struct {
	Peer    *Peer
	Torrent *Torrent
	User    *User

	// Created is true if this announce created a new peer or changed an existing
	// peer's address
	Created bool
	// Snatched is true if this announce completed the download
	Snatched bool

	// Uploaded contains the upload delta for this announce, in bytes
	Uploaded    uint64
	RawUploaded uint64

	// Downloaded contains the download delta for this announce, in bytes
	Downloaded    uint64
	RawDownloaded uint64
}

// AnnounceResponse contains the information needed to fulfill an announce.
type AnnounceResponse struct {
	Complete, Incomplete  int
	Interval, MinInterval time.Duration
	IPv4Peers, IPv6Peers  PeerList

	Compact bool
}

// Scrape is a Scrape by a Peer.
type Scrape struct {
	Config *config.Config `json:"config"`

	Passkey    string
	Infohashes []string
}

// ScrapeResponse contains the information needed to fulfill a scrape.
type ScrapeResponse struct {
	Files []*Torrent
}
