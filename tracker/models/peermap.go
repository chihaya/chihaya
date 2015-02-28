// Copyright 2015 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

package models

import (
	"net"
	"sync"
	"sync/atomic"

	"github.com/chihaya/chihaya/config"
	"github.com/chihaya/chihaya/stats"
)

// PeerMap is a thread-safe map from PeerKeys to Peers. When PreferredSubnet is
// enabled, it is a thread-safe map of maps from MaskedIPs to Peerkeys to Peers.
type PeerMap struct {
	Peers   map[string]map[PeerKey]Peer `json:"peers"`
	Seeders bool                        `json:"seeders"`
	Config  config.SubnetConfig         `json:"config"`
	Size    int32                       `json:"size"`
	sync.RWMutex
}

// NewPeerMap initializes the map for a new PeerMap.
func NewPeerMap(seeders bool, cfg *config.Config) *PeerMap {
	pm := &PeerMap{
		Peers:   make(map[string]map[PeerKey]Peer),
		Seeders: seeders,
		Config:  cfg.NetConfig.SubnetConfig,
	}

	if !pm.Config.PreferredSubnet {
		pm.Peers[""] = make(map[PeerKey]Peer)
	}

	return pm
}

// Contains is true if a peer is contained with a PeerMap.
func (pm *PeerMap) Contains(pk PeerKey) bool {
	pm.RLock()
	defer pm.RUnlock()

	if pm.Config.PreferredSubnet {
		maskedIP := pm.mask(pk.IP())
		peers, exists := pm.Peers[maskedIP]
		if !exists {
			return false
		}

		_, exists = peers[pk]
		return exists
	}

	_, exists := pm.Peers[""][pk]
	return exists
}

func (pm *PeerMap) mask(ip net.IP) string {
	if !pm.Config.PreferredSubnet {
		return ""
	}

	var maskedIP net.IP
	if len(ip) == net.IPv6len {
		maskedIP = ip.Mask(net.CIDRMask(pm.Config.PreferredIPv6Subnet, 128))
	} else {
		maskedIP = ip.Mask(net.CIDRMask(pm.Config.PreferredIPv4Subnet, 32))
	}

	return maskedIP.String()
}

// LookUp is a thread-safe read from a PeerMap.
func (pm *PeerMap) LookUp(pk PeerKey) (peer Peer, exists bool) {
	pm.RLock()
	defer pm.RUnlock()

	maskedIP := pm.mask(pk.IP())
	peers, exists := pm.Peers[maskedIP]
	if !exists {
		return Peer{}, false
	}
	peer, exists = peers[pk]

	return
}

// Put is a thread-safe write to a PeerMap.
func (pm *PeerMap) Put(p Peer) {
	pm.Lock()
	defer pm.Unlock()

	maskedIP := pm.mask(p.IP)
	_, exists := pm.Peers[maskedIP]
	if !exists {
		pm.Peers[maskedIP] = make(map[PeerKey]Peer)
	}
	_, exists = pm.Peers[maskedIP][p.Key()]
	if !exists {
		atomic.AddInt32(&(pm.Size), 1)
	}
	pm.Peers[maskedIP][p.Key()] = p
}

// Delete is a thread-safe delete from a PeerMap.
func (pm *PeerMap) Delete(pk PeerKey) {
	pm.Lock()
	defer pm.Unlock()

	maskedIP := pm.mask(pk.IP())
	_, exists := pm.Peers[maskedIP][pk]
	if exists {
		atomic.AddInt32(&(pm.Size), -1)
		delete(pm.Peers[maskedIP], pk)
	}
}

// Len returns the number of peers within a PeerMap.
func (pm *PeerMap) Len() int {
	return int(atomic.LoadInt32(&pm.Size))
}

// Purge iterates over all of the peers within a PeerMap and deletes them if
// they are older than the provided time.
func (pm *PeerMap) Purge(unixtime int64) {
	pm.Lock()
	defer pm.Unlock()

	for _, subnetmap := range pm.Peers {
		for key, peer := range subnetmap {
			if peer.LastAnnounce <= unixtime {
				atomic.AddInt32(&(pm.Size), -1)
				delete(subnetmap, key)
				if pm.Seeders {
					stats.RecordPeerEvent(stats.ReapedSeed, peer.HasIPv6())
				} else {
					stats.RecordPeerEvent(stats.ReapedLeech, peer.HasIPv6())
				}
			}
		}
	}
}

// AppendPeers adds peers to given IPv4 or IPv6 lists.
func (pm *PeerMap) AppendPeers(ipv4s, ipv6s PeerList, ann *Announce, wanted int) (PeerList, PeerList) {
	maskedIP := pm.mask(ann.Peer.IP)

	pm.RLock()
	defer pm.RUnlock()

	count := 0
	// Attempt to append all the peers in the same subnet.
	for _, peer := range pm.Peers[maskedIP] {
		if count >= wanted {
			break
		} else if peersEquivalent(&peer, ann.Peer) {
			continue
		} else {
			count += AppendPeer(&ipv4s, &ipv6s, ann, &peer)
		}
	}

	// Add any more peers out of the other subnets.
	for subnet, peers := range pm.Peers {
		if subnet == maskedIP {
			continue
		} else {
			for _, peer := range peers {
				if count >= wanted {
					break
				} else if peersEquivalent(&peer, ann.Peer) {
					continue
				} else {
					count += AppendPeer(&ipv4s, &ipv6s, ann, &peer)
				}
			}
		}
	}

	return ipv4s, ipv6s
}

// AppendPeer adds a peer to its corresponding peerlist.
func AppendPeer(ipv4s, ipv6s *PeerList, ann *Announce, peer *Peer) int {
	if ann.HasIPv6() && peer.HasIPv6() {
		*ipv6s = append(*ipv6s, *peer)
		return 1
	} else if ann.Config.RespectAF && ann.HasIPv4() && peer.HasIPv4() {
		*ipv4s = append(*ipv4s, *peer)
		return 1
	} else if !ann.Config.RespectAF && peer.HasIPv4() {
		*ipv4s = append(*ipv4s, *peer)
		return 1
	}

	return 0
}

// peersEquivalent checks if two peers represent the same entity.
func peersEquivalent(a, b *Peer) bool {
	return a.ID == b.ID || a.UserID != 0 && a.UserID == b.UserID
}
