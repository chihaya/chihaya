// Copyright 2013 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

// Package models implements the models for an abstraction over the
// multiple data stores used by a BitTorrent tracker.
package models

import (
	"errors"
	"net/http"
	"path"
	"strconv"
	"time"

	"github.com/chihaya/chihaya/config"
	"github.com/chihaya/chihaya/models/query"
)

var (
	// ErrMalformedRequest is returned when a request does no have the required
	// parameters.
	ErrMalformedRequest = errors.New("malformed request")
)

// Peer is the internal representation of a participant in a swarm.
type Peer struct {
	ID        string `json:"id"`
	UserID    uint64 `json:"user_id"`
	TorrentID uint64 `json:"torrent_id"`

	IP   string `json:"ip"`
	Port uint64 `json:"port"`

	Uploaded     uint64 `json:"uploaded"`
	Downloaded   uint64 `json:"downloaded`
	Left         uint64 `json:"left"`
	LastAnnounce int64  `json:"last_announce"`
}

// Key is a helper that returns the proper format for keys used for maps
// of peers (i.e. torrent.Seeders & torrent.Leechers).
func (p Peer) Key() string {
	return p.ID + ":" + strconv.FormatUint(p.UserID, 36)
}

// Torrent is the internal representation of a swarm for a given torrent file.
type Torrent struct {
	ID       uint64 `json:"id"`
	Infohash string `json:"infohash"`
	Active   bool   `json:"active"`

	Seeders  map[string]Peer `json:"seeders"`
	Leechers map[string]Peer `json:"leechers"`

	Snatches       uint64  `json:"snatches"`
	UpMultiplier   float64 `json:"up_multiplier"`
	DownMultiplier float64 `json:"down_multiplier"`
	LastAction     int64   `json:"last_action"`
}

// InSeederPool returns true if a peer is within a torrent's pool of seeders.
func (t *Torrent) InSeederPool(p *Peer) bool {
	_, exists := t.Seeders[p.Key()]
	return exists
}

// InLeecherPool returns true if a peer is within a torrent's pool of leechers.
func (t *Torrent) InLeecherPool(p *Peer) bool {
	_, exists := t.Leechers[p.Key()]
	return exists
}

// NewPeer creates a new peer using the information provided by an announce.
func NewPeer(t *Torrent, u *User, a *Announce) *Peer {
	return &Peer{
		ID:           a.PeerID,
		UserID:       u.ID,
		TorrentID:    t.ID,
		IP:           a.IP,
		Port:         a.Port,
		Uploaded:     a.Uploaded,
		Downloaded:   a.Downloaded,
		Left:         a.Left,
		LastAnnounce: time.Now().Unix(),
	}
}

// User is the internal representation of registered user for private trackers.
type User struct {
	ID      uint64 `json:"id"`
	Passkey string `json:"passkey"`

	UpMultiplier   float64 `json:"up_multiplier"`
	DownMultiplier float64 `json:"down_multiplier"`
	Snatches       uint64  `json:"snatches"`
}

// Announce represents all of the data from an announce request.
type Announce struct {
	Config  *config.Config `json:"config"`
	Request *http.Request  `json:"request"`

	Compact    bool   `json:"compact"`
	Downloaded uint64 `json:"downloaded"`
	Event      string `json:"event"`
	IP         string `json:"ip"`
	Infohash   string `json:"infohash"`
	Left       uint64 `json:"left"`
	NumWant    int    `json:"numwant"`
	Passkey    string `json:"passkey"`
	PeerID     string `json:"peer_id"`
	Port       uint64 `json:"port"`
	Uploaded   uint64 `json:"uploaded"`
}

// NewAnnounce parses an HTTP request and generates an Announce.
func NewAnnounce(r *http.Request, conf *config.Config) (*Announce, error) {
	q, err := query.New(r.URL.RawQuery)
	if err != nil {
		return nil, err
	}

	compact := q.Params["compact"] != "0"
	downloaded, downloadedErr := q.Uint64("downloaded")
	event, _ := q.Params["event"]
	infohash, _ := q.Params["info_hash"]
	ip, _ := q.RequestedIP(r)
	left, leftErr := q.Uint64("left")
	numWant := q.RequestedPeerCount(conf.DefaultNumWant)
	dir, _ := path.Split(r.URL.Path)
	peerID, _ := q.Params["peer_id"]
	port, portErr := q.Uint64("port")
	uploaded, uploadedErr := q.Uint64("uploaded")

	if downloadedErr != nil ||
		infohash == "" ||
		leftErr != nil ||
		peerID == "" ||
		portErr != nil ||
		uploadedErr != nil ||
		ip == "" ||
		len(dir) != 34 {
		return nil, ErrMalformedRequest
	}

	return &Announce{
		Config:     conf,
		Request:    r,
		Compact:    compact,
		Downloaded: downloaded,
		Event:      event,
		IP:         ip,
		Infohash:   infohash,
		Left:       left,
		NumWant:    numWant,
		Passkey:    dir[1:33],
		PeerID:     peerID,
		Port:       port,
		Uploaded:   uploaded,
	}, nil
}

// ClientID returns the part of a PeerID that identifies the client software.
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

// AnnounceDelta contains a difference in statistics for a peer.
// It is used for communicating changes to be recorded by the driver.
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

// NewAnnounceDelta does stuff
func NewAnnounceDelta(p *Peer, u *User, a *Announce, t *Torrent, created, snatched bool) *AnnounceDelta {
	rawDeltaUp := p.Uploaded - a.Uploaded
	rawDeltaDown := p.Downloaded - a.Downloaded

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

// Scrape represents all of the data from an scrape request.
type Scrape struct {
	Config  *config.Config `json:"config"`
	Request *http.Request  `json:"request"`

	Passkey    string
	Infohashes []string
}

// NewScrape parses an HTTP request and generates a Scrape.
func NewScrape(r *http.Request, c *config.Config) (*Scrape, error) {
	q, err := query.New(r.URL.RawQuery)
	if err != nil {
		return nil, err
	}

	var passkey string
	if c.Private {
		dir, _ := path.Split(r.URL.Path)
		if len(dir) != 34 {
			return nil, ErrMalformedRequest
		}
		passkey = dir[1:34]
	}

	if q.Infohashes == nil {
		if _, exists := q.Params["infohash"]; !exists {
			// There aren't any infohashes.
			return nil, ErrMalformedRequest
		}
		q.Infohashes = []string{q.Params["infohash"]}
	}

	return &Scrape{
		Config:  c,
		Request: r,

		Passkey:    passkey,
		Infohashes: q.Infohashes,
	}, nil
}
