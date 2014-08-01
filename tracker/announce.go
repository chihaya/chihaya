// Copyright 2014 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

package tracker

import (
	"net"

	"github.com/chihaya/chihaya/stats"
	"github.com/chihaya/chihaya/tracker/models"
)

// HandleAnnounce encapsulates all of the logic of handling a BitTorrent
// client's Announce without being coupled to any transport protocol.
func (tkr *Tracker) HandleAnnounce(ann *models.Announce, w Writer) error {
	conn, err := tkr.Pool.Get()
	if err != nil {
		return err
	}

	defer conn.Close()

	if tkr.cfg.ClientWhitelistEnabled {
		if err = conn.FindClient(ann.ClientID()); err != nil {
			return err
		}
	}

	var user *models.User
	if tkr.cfg.PrivateEnabled {
		if user, err = conn.FindUser(ann.Passkey); err != nil {
			return err
		}
	}

	var torrent *models.Torrent
	torrent, err = conn.FindTorrent(ann.Infohash)
	switch {
	case !tkr.cfg.PrivateEnabled && err == models.ErrTorrentDNE:
		torrent = &models.Torrent{
			Infohash: ann.Infohash,
			Seeders:  models.PeerMap{},
			Leechers: models.PeerMap{},
		}

		err = conn.PutTorrent(torrent)
		if err != nil {
			return err
		}
		stats.RecordEvent(stats.NewTorrent)

	case err != nil:
		return err
	}

	peer := models.NewPeer(ann, user, torrent)

	var createdIPv4, createdIPv6 bool
	if peer.HasIPv4() {
		createdIPv4, err = updateSwarm(conn, ann, peer, torrent, "ipv4")
		if err != nil {
			return err
		}
	}
	if peer.HasIPv6() {
		createdIPv6, err = updateSwarm(conn, ann, peer, torrent, "ipv6")
		if err != nil {
			return err
		}
	}
	created := createdIPv4 || createdIPv6

	var snatchedIPv4, snatchedIPv6 bool
	if peer.HasIPv4() {
		snatchedIPv4, err = handleEvent(conn, ann, peer, user, torrent, "ipv4")
		if err != nil {
			return err
		}
	}
	if peer.HasIPv6() {
		snatchedIPv6, err = handleEvent(conn, ann, peer, user, torrent, "ipv6")
		if err != nil {
			return err
		}
	}
	snatched := snatchedIPv4 || snatchedIPv6

	if tkr.cfg.PrivateEnabled {
		delta := models.NewAnnounceDelta(ann, peer, user, torrent, created, snatched)
		err = tkr.backend.RecordAnnounce(delta)
		if err != nil {
			return err
		}
	} else if tkr.cfg.PurgeInactiveTorrents && torrent.PeerCount() == 0 {
		// Rather than deleting the torrent explicitly, let the tracker driver
		// ensure there are no race conditions.
		conn.PurgeInactiveTorrent(torrent.Infohash)
		stats.RecordEvent(stats.DeletedTorrent)
	}

	return w.WriteAnnounce(newAnnounceResponse(ann, peer, torrent))
}

// updateSwarm handles the changes to a torrent's swarm given an announce.
func updateSwarm(c Conn, ann *models.Announce, p *models.Peer, t *models.Torrent, ipv string) (created bool, err error) {
	c.TouchTorrent(t.Infohash)

	switch {
	case t.InSeederPool(p.ID, ipv):
		err = c.PutSeeder(t.Infohash, ipv, p)
		if err != nil {
			return
		}
		t.Seeders[models.NewPeerKey(p.ID, ipv)] = *p

	case t.InLeecherPool(p.ID, ipv):
		err = c.PutLeecher(t.Infohash, ipv, p)
		if err != nil {
			return
		}
		t.Leechers[models.NewPeerKey(p.ID, ipv)] = *p

	default:
		if ann.Event != "" && ann.Event != "started" {
			err = models.ErrBadRequest
			return
		}

		if ann.Left == 0 {
			err = c.PutSeeder(t.Infohash, ipv, p)
			if err != nil {
				return
			}
			t.Seeders[models.NewPeerKey(p.ID, ipv)] = *p
			stats.RecordPeerEvent(stats.NewSeed, ipv)

		} else {
			err = c.PutLeecher(t.Infohash, ipv, p)
			if err != nil {
				return
			}
			t.Leechers[models.NewPeerKey(p.ID, ipv)] = *p
			stats.RecordPeerEvent(stats.NewLeech, ipv)
		}
		created = true
	}

	return
}

// handleEvent checks to see whether an announce has an event and if it does,
// properly handles that event.
func handleEvent(c Conn, ann *models.Announce, p *models.Peer, u *models.User, t *models.Torrent, ipv string) (snatched bool, err error) {
	peerkey := models.NewPeerKey(p.ID, ipv)

	switch {
	case ann.Event == "stopped" || ann.Event == "paused":
		// updateSwarm checks if the peer is active on the torrent,
		// so one of these branches must be followed.
		if t.InSeederPool(p.ID, ipv) {
			err = c.DeleteSeeder(t.Infohash, peerkey)
			if err != nil {
				return
			}
			delete(t.Seeders, models.NewPeerKey(p.ID, ipv))
			stats.RecordPeerEvent(stats.DeletedSeed, ipv)

		} else if t.InLeecherPool(p.ID, ipv) {
			err = c.DeleteLeecher(t.Infohash, peerkey)
			if err != nil {
				return
			}
			delete(t.Leechers, models.NewPeerKey(p.ID, ipv))
			stats.RecordPeerEvent(stats.DeletedLeech, ipv)
		}

	case ann.Event == "completed":
		err = c.IncrementTorrentSnatches(t.Infohash)
		if err != nil {
			return
		}
		t.Snatches++

		if ann.Config.PrivateEnabled {
			err = c.IncrementUserSnatches(u.Passkey)
			if err != nil {
				return
			}
			u.Snatches++
		}

		if t.InLeecherPool(p.ID, ipv) {
			err = leecherFinished(c, t, p, ipv)
		} else {
			err = models.ErrBadRequest
		}

		// If one of the dual-stacked peers is already a seeder, they have already
		// snatched.
		if !(t.InSeederPool(p.ID, "ipv4") || t.InSeederPool(p.ID, "ipv6")) {
			snatched = true
		}

	case t.InLeecherPool(p.ID, ipv) && ann.Left == 0:
		// A leecher completed but the event was never received.
		err = leecherFinished(c, t, p, ipv)
	}

	return
}

// leecherFinished moves a peer from the leeching pool to the seeder pool.
func leecherFinished(c Conn, t *models.Torrent, p *models.Peer, ipv string) error {
	peerkey := models.NewPeerKey(p.ID, ipv)
	if err := c.DeleteLeecher(t.Infohash, peerkey); err != nil {
		return err
	}
	delete(t.Leechers, peerkey)

	if err := c.PutSeeder(t.Infohash, ipv, p); err != nil {
		return err
	}
	t.Seeders[peerkey] = *p

	stats.RecordPeerEvent(stats.Completed, ipv)
	return nil
}

func newAnnounceResponse(ann *models.Announce, announcer *models.Peer, t *models.Torrent) *models.AnnounceResponse {
	seedCount := len(t.Seeders)
	leechCount := len(t.Leechers)

	res := &models.AnnounceResponse{
		Complete:    seedCount,
		Incomplete:  leechCount,
		Interval:    ann.Config.Announce.Duration,
		MinInterval: ann.Config.MinAnnounce.Duration,
		Compact:     ann.Compact,
	}

	if ann.NumWant > 0 && ann.Event != "stopped" && ann.Event != "paused" {
		res.IPv4Peers, res.IPv6Peers = getPeers(ann, announcer, t, ann.NumWant)
	}

	return res
}

// getPeers returns lists IPv4 and IPv6 peers on a given torrent sized according
// to the wanted parameter.
func getPeers(ann *models.Announce, announcer *models.Peer, t *models.Torrent, wanted int) (ipv4s, ipv6s models.PeerList) {
	ipv4s, ipv6s = models.PeerList{}, models.PeerList{}

	if ann.Left == 0 {
		// If they're seeding, give them only leechers.
		return appendPeers(ipv4s, ipv6s, ann, announcer, t.Leechers, wanted)
	}

	// If they're leeching, prioritize giving them seeders.
	ipv4s, ipv6s = appendPeers(ipv4s, ipv6s, ann, announcer, t.Seeders, wanted)
	return appendPeers(ipv4s, ipv6s, ann, announcer, t.Leechers, wanted-len(ipv4s)-len(ipv6s))
}

// appendPeers implements the logic of adding peers to the IPv4 or IPv6 lists.
func appendPeers(ipv4s, ipv6s models.PeerList, ann *models.Announce, announcer *models.Peer, peers models.PeerMap, wanted int) (models.PeerList, models.PeerList) {
	if ann.Config.PreferredSubnet {
		return appendSubnetPeers(ipv4s, ipv6s, ann, announcer, peers, wanted)
	}

	count := 0

	for _, peer := range peers {
		if count >= wanted {
			break
		} else if peersEquivalent(&peer, announcer) {
			continue
		}

		if announcer.HasIPv6() && peer.HasIPv6() {
			ipv6s = append(ipv6s, peer)
			count++
		} else if peer.HasIPv4() {
			ipv4s = append(ipv4s, peer)
			count++
		}
	}

	return ipv4s, ipv6s
}

// appendSubnetPeers is an alternative version of appendPeers used when the
// config variable PreferredSubnet is enabled.
func appendSubnetPeers(ipv4s, ipv6s models.PeerList, ann *models.Announce, announcer *models.Peer, peers models.PeerMap, wanted int) (models.PeerList, models.PeerList) {
	var subnetIPv4 net.IPNet
	var subnetIPv6 net.IPNet

	if announcer.HasIPv4() {
		subnetIPv4 = net.IPNet{announcer.IPv4, net.CIDRMask(ann.Config.PreferredIPv4Subnet, 32)}
	}

	if announcer.HasIPv6() {
		subnetIPv6 = net.IPNet{announcer.IPv6, net.CIDRMask(ann.Config.PreferredIPv6Subnet, 128)}
	}

	// Iterate over the peers twice: first add only peers in the same subnet and
	// if we still need more peers grab ones that haven't already been added.
	count := 0
	for _, checkInSubnet := range [2]bool{true, false} {
		for _, peer := range peers {
			if count >= wanted {
				break
			}

			inSubnet4 := peer.HasIPv4() && subnetIPv4.Contains(peer.IPv4)
			inSubnet6 := peer.HasIPv6() && subnetIPv6.Contains(peer.IPv6)

			if peersEquivalent(&peer, announcer) || checkInSubnet != (inSubnet4 || inSubnet6) {
				continue
			}

			if announcer.HasIPv6() && peer.HasIPv6() {
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
func peersEquivalent(a, b *models.Peer) bool {
	return a.ID == b.ID || a.UserID != 0 && a.UserID == b.UserID
}
