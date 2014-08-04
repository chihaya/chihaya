// Copyright 2014 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

package models

import (
	"net"
	"sync"
	"time"

	"github.com/chihaya/chihaya/config"
	"github.com/chihaya/chihaya/stats"
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

	IP   net.IP `json:"ip,omitempty"` // Always has length net.IPv4len if IPv4, and net.IPv6len if IPv6
	Port uint64 `json:"port"`

	Uploaded     uint64 `json:"uploaded"`
	Downloaded   uint64 `json:"downloaded"`
	Left         uint64 `json:"left"`
	LastAnnounce int64  `json:"last_announce"`
}

func (p *Peer) HasIPv4() bool {
	return !p.HasIPv6()
}

func (p *Peer) HasIPv6() bool {
	return len(p.IP) == net.IPv6len
}

func (p *Peer) Key() PeerKey {
	return NewPeerKey(p.ID, p.HasIPv6())
}

type PeerList []Peer
type PeerKey string

func NewPeerKey(peerID string, ipv6 bool) PeerKey {
	if ipv6 {
		return PeerKey("6:" + peerID)
	}

	return PeerKey("4:" + peerID)
}

// PeerMap is a map from PeerKeys to Peers.
type PeerMap struct {
	peers map[PeerKey]Peer
	sync.RWMutex
}

func NewPeerMap() PeerMap {
	return PeerMap{
		peers: make(map[PeerKey]Peer),
	}
}

func (pm *PeerMap) Contains(pk PeerKey) (exists bool) {
	pm.RLock()
	defer pm.RUnlock()

	_, exists = pm.peers[pk]

	return
}

func (pm *PeerMap) LookUp(pk PeerKey) (peer Peer, exists bool) {
	pm.RLock()
	defer pm.RUnlock()

	peer, exists = pm.peers[pk]

	return
}

func (pm *PeerMap) Put(p Peer) {
	pm.Lock()
	defer pm.Unlock()

	pm.peers[p.Key()] = p
}

func (pm *PeerMap) Delete(pk PeerKey) {
	pm.Lock()
	defer pm.Unlock()

	delete(pm.peers, pk)
}

func (pm *PeerMap) Len() int {
	pm.RLock()
	defer pm.RUnlock()

	return len(pm.peers)
}

func (pm *PeerMap) Purge(unixtime int64) {
	pm.Lock()
	defer pm.Unlock()

	for key, peer := range pm.peers {
		if peer.LastAnnounce <= unixtime {
			delete(pm.peers, key)
			stats.RecordPeerEvent(stats.ReapedSeed, peer.HasIPv6())
		}
	}
}

// AppendPeers implements the logic of adding peers to given IPv4 or IPv6 lists.
func (pm *PeerMap) AppendPeers(ipv4s, ipv6s PeerList, ann *Announce, wanted int) (PeerList, PeerList) {
	if ann.Config.PreferredSubnet {
		return pm.AppendSubnetPeers(ipv4s, ipv6s, ann, wanted)
	}

	pm.Lock()
	defer pm.Unlock()

	count := 0
	for _, peer := range pm.peers {
		if count >= wanted {
			break
		} else if peersEquivalent(&peer, ann.Peer) {
			continue
		}

		if ann.HasIPv6() && peer.HasIPv6() {
			ipv6s = append(ipv6s, peer)
			count++
		} else if peer.HasIPv4() {
			ipv4s = append(ipv4s, peer)
			count++
		}
	}

	return ipv4s, ipv6s
}

// peersEquivalent checks if two peers represent the same entity.
func peersEquivalent(a, b *Peer) bool {
	return a.ID == b.ID || a.UserID != 0 && a.UserID == b.UserID
}

// AppendSubnetPeers is an alternative version of appendPeers used when the
// config variable PreferredSubnet is enabled.
func (pm *PeerMap) AppendSubnetPeers(ipv4s, ipv6s PeerList, ann *Announce, wanted int) (PeerList, PeerList) {
	var subnetIPv4 net.IPNet
	var subnetIPv6 net.IPNet

	if ann.HasIPv4() {
		subnetIPv4 = net.IPNet{ann.IPv4, net.CIDRMask(ann.Config.PreferredIPv4Subnet, 32)}
	}

	if ann.HasIPv6() {
		subnetIPv6 = net.IPNet{ann.IPv6, net.CIDRMask(ann.Config.PreferredIPv6Subnet, 128)}
	}

	pm.Lock()
	defer pm.Unlock()

	// Iterate over the peers twice: first add only peers in the same subnet and
	// if we still need more peers grab ones that haven't already been added.
	count := 0
	for _, checkInSubnet := range [2]bool{true, false} {
		for _, peer := range pm.peers {
			if count >= wanted {
				break
			}

			inSubnet4 := peer.HasIPv4() && subnetIPv4.Contains(peer.IP)
			inSubnet6 := peer.HasIPv6() && subnetIPv6.Contains(peer.IP)

			if peersEquivalent(&peer, ann.Peer) || checkInSubnet != (inSubnet4 || inSubnet6) {
				continue
			}

			if ann.HasIPv6() && peer.HasIPv6() {
				ipv6s = append(ipv6s, peer)
				count++
			} else if peer.HasIPv4() {
				ipv4s = append(ipv4s, peer)
				count++
			}
		}
	}

	return ipv4s, ipv6s
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
	Port       uint64 `json:"port"`
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
