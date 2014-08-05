// Copyright 2014 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

package models

import (
	"encoding/json"
	"net"
	"sync"

	"github.com/chihaya/chihaya/stats"
)

// PeerMap is a thread-safe map from PeerKeys to Peers.
type PeerMap struct {
	seeders bool
	peers   map[PeerKey]Peer
	sync.RWMutex
}

// NewPeerMap initializes the map for a new PeerMap.
func NewPeerMap(seeders bool) PeerMap {
	return PeerMap{
		peers:   make(map[PeerKey]Peer),
		seeders: seeders,
	}
}

// Contains is true if a peer is contained with a PeerMap.
func (pm *PeerMap) Contains(pk PeerKey) (exists bool) {
	pm.RLock()
	defer pm.RUnlock()

	_, exists = pm.peers[pk]

	return
}

// LookUp is a thread-safe read from a PeerMap.
func (pm *PeerMap) LookUp(pk PeerKey) (peer Peer, exists bool) {
	pm.RLock()
	defer pm.RUnlock()

	peer, exists = pm.peers[pk]

	return
}

// Put is a thread-safe write to a PeerMap.
func (pm *PeerMap) Put(p Peer) {
	pm.Lock()
	defer pm.Unlock()

	pm.peers[p.Key()] = p
}

// Delete is a thread-safe delete from a PeerMap.
func (pm *PeerMap) Delete(pk PeerKey) {
	pm.Lock()
	defer pm.Unlock()

	delete(pm.peers, pk)
}

// Len returns the number of peers within a PeerMap.
func (pm *PeerMap) Len() int {
	pm.RLock()
	defer pm.RUnlock()

	return len(pm.peers)
}

func (pm *PeerMap) MarshalJSON() ([]byte, error) {
	pm.RLock()
	defer pm.RUnlock()
	return json.Marshal(pm.peers)
}

func (pm *PeerMap) UnmarshalJSON(b []byte) error {
	pm.Lock()
	defer pm.Unlock()

	peers := make(map[PeerKey]Peer)
	err := json.Unmarshal(b, &peers)
	if err != nil {
		return err
	}

	pm.peers = peers
	return nil
}

// Purge iterates over all of the peers within a PeerMap and deletes them if
// they are older than the provided time.
func (pm *PeerMap) Purge(unixtime int64) {

	pm.Lock()
	defer pm.Unlock()

	for key, peer := range pm.peers {
		if peer.LastAnnounce <= unixtime {
			delete(pm.peers, key)
			if pm.seeders {
				stats.RecordPeerEvent(stats.ReapedSeed, peer.HasIPv6())
			} else {
				stats.RecordPeerEvent(stats.ReapedLeech, peer.HasIPv6())
			}
		}
	}

	return
}

// AppendPeers adds peers to given IPv4 or IPv6 lists.
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

// AppendSubnetPeers is an alternative version of AppendPeers used when the
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

// peersEquivalent checks if two peers represent the same entity.
func peersEquivalent(a, b *Peer) bool {
	return a.ID == b.ID || a.UserID != 0 && a.UserID == b.UserID
}
