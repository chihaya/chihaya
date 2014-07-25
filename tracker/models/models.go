// Copyright 2014 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

package models

import (
	"net"
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

func (e ClientError) Error() string   { return string(e) }
func (e NotFoundError) Error() string { return string(e) }

// Peer is a participant in a swarm.
type Peer struct {
	ID        string `json:"id"`
	UserID    uint64 `json:"user_id"`
	TorrentID uint64 `json:"torrent_id"`

	IPv4 net.IP `json:"ipv4,omitempty"`
	IPv6 net.IP `json:"ipv6,omitempty"`
	Port uint64 `json:"port"`

	Uploaded     uint64 `json:"uploaded"`
	Downloaded   uint64 `json:"downloaded"`
	Left         uint64 `json:"left"`
	LastAnnounce int64  `json:"last_announce"`
}

type PeerList []Peer

// PeerMap is a map from PeerIDs to Peers.
type PeerMap map[string]Peer

// NewPeer returns the Peer representation of an Announce. When provided nil
// for the announce parameter, it panics. When provided nil for the user or
// torrent parameter, it returns a Peer{UserID: 0} or Peer{TorrentID: 0}
// respectively.
func NewPeer(a *Announce, u *User, t *Torrent) *Peer {
	if a == nil {
		panic("tracker: announce cannot equal nil")
	}

	var userID uint64
	if u != nil {
		userID = u.ID
	}

	var torrentID uint64
	if t != nil {
		torrentID = t.ID
	}

	return &Peer{
		ID:           a.PeerID,
		UserID:       userID,
		TorrentID:    torrentID,
		IPv4:         a.IPv4,
		IPv6:         a.IPv6,
		Port:         a.Port,
		Uploaded:     a.Uploaded,
		Downloaded:   a.Downloaded,
		Left:         a.Left,
		LastAnnounce: time.Now().Unix(),
	}
}

func (p *Peer) HasIPv4() bool {
	return p.IPv4 != nil
}

func (p *Peer) HasIPv6() bool {
	return p.IPv6 != nil
}

// Torrent is a swarm for a given torrent file.
type Torrent struct {
	ID       uint64 `json:"id"`
	Infohash string `json:"infohash"`

	Seeders  PeerMap `json:"seeders"`
	Leechers PeerMap `json:"leechers"`

	Snatches       uint64  `json:"snatches"`
	UpMultiplier   float64 `json:"up_multiplier"`
	DownMultiplier float64 `json:"down_multiplier"`
	LastAction     int64   `json:"last_action"`
}

// InSeederPool returns true if a peer is within a Torrent's map of seeders.
func (t *Torrent) InSeederPool(p *Peer) (exists bool) {
	_, exists = t.Seeders[p.ID]
	return
}

// InLeecherPool returns true if a peer is within a Torrent's map of leechers.
func (t *Torrent) InLeecherPool(p *Peer) (exists bool) {
	_, exists = t.Leechers[p.ID]
	return
}

// PeerCount returns the total number of peers connected on this Torrent.
func (t *Torrent) PeerCount() int {
	return len(t.Seeders) + len(t.Leechers)
}

// User is a registered user for private trackers.
type User struct {
	ID      uint64 `json:"id"`
	Passkey string `json:"passkey"`

	UpMultiplier   float64 `json:"up_multiplier"`
	DownMultiplier float64 `json:"down_multiplier"`
	Snatches       uint64  `json:"snatches"`
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
	Port       uint64 `json:"port"`
	Uploaded   uint64 `json:"uploaded"`
}

// ClientID returns the part of a PeerID that identifies a Peer's client
// software.
func (a Announce) ClientID() (clientID string) {
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

	// Uploaded contains the raw upload delta for this announce, in bytes
	Uploaded uint64
	// Downloaded contains the raw download delta for this announce, in bytes
	Downloaded uint64
}

// AnnounceResponse contains the information needed to fulfill an announce.
type AnnounceResponse struct {
	Complete, Incomplete  int
	Interval, MinInterval time.Duration
	IPv4Peers, IPv6Peers  PeerList

	Compact bool
}

// NewAnnounceDelta calculates a Peer's download and upload deltas between
// Announces and generates an AnnounceDelta.
func NewAnnounceDelta(a *Announce, p *Peer, u *User, t *Torrent, created, snatched bool) *AnnounceDelta {
	var (
		rawDeltaUp   = p.Uploaded - a.Uploaded
		rawDeltaDown uint64
	)

	if !a.Config.FreeleechEnabled {
		rawDeltaDown = p.Downloaded - a.Downloaded
	}

	// Restarting a torrent may cause a delta to be negative.
	if rawDeltaUp < 0 {
		rawDeltaUp = 0
	}

	if rawDeltaDown < 0 {
		rawDeltaDown = 0
	}

	return &AnnounceDelta{
		Peer:    p,
		Torrent: t,
		User:    u,

		Created:  created,
		Snatched: snatched,

		Uploaded:   uint64(float64(rawDeltaUp) * u.UpMultiplier * t.UpMultiplier),
		Downloaded: uint64(float64(rawDeltaDown) * u.DownMultiplier * t.DownMultiplier),
	}
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
