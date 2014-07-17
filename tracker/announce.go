// Copyright 2014 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

package tracker

import (
	"net"

	"github.com/chihaya/chihaya/tracker/models"
)

func (t *Tracker) HandleAnnounce(ann *models.Announce, w Writer) error {
	conn, err := t.Pool.Get()
	if err != nil {
		return err
	}

	if t.cfg.Whitelist {
		err = conn.FindClient(ann.ClientID())
		if err == ErrClientUnapproved {
			w.WriteError(err)
			return nil
		} else if err != nil {
			return err
		}
	}

	var user *models.User
	if t.cfg.Private {
		user, err = conn.FindUser(ann.Passkey)
		if err == ErrUserDNE {
			w.WriteError(err)
			return nil
		} else if err != nil {
			return err
		}
	}

	var torrent *models.Torrent
	torrent, err = conn.FindTorrent(ann.Infohash)
	switch {
	case !t.cfg.Private && err == ErrTorrentDNE:
		torrent = &models.Torrent{
			Infohash: ann.Infohash,
			Seeders:  models.PeerMap{},
			Leechers: models.PeerMap{},
		}

		err = conn.PutTorrent(torrent)
		if err != nil {
			return err
		}

	case t.cfg.Private && err == ErrTorrentDNE:
		w.WriteError(err)
		return nil

	case err != nil:
		return err
	}

	peer := models.NewPeer(ann, user, torrent)

	created, err := updateTorrent(conn, ann, peer, torrent)
	if err != nil {
		return err
	}

	snatched, err := handleEvent(conn, ann, peer, user, torrent)
	if err != nil {
		return err
	}

	if t.cfg.Private {
		delta := models.NewAnnounceDelta(ann, peer, user, torrent, created, snatched)
		err = t.backend.RecordAnnounce(delta)
		if err != nil {
			return err
		}
	} else if t.cfg.PurgeInactiveTorrents && torrent.PeerCount() == 0 {
		// Rather than deleting the torrent explicitly, let the tracker driver
		// ensure there are no race conditions.
		conn.PurgeInactiveTorrent(torrent.Infohash)
	}

	return w.WriteAnnounce(newAnnounceResponse(ann, peer, torrent))
}

func updateTorrent(c Conn, ann *models.Announce, p *models.Peer, t *models.Torrent) (created bool, err error) {
	c.TouchTorrent(t.Infohash)

	switch {
	case t.InSeederPool(p):
		err = c.PutSeeder(t.Infohash, p)
		if err != nil {
			return
		}
		t.Seeders[p.Key()] = *p

	case t.InLeecherPool(p):
		err = c.PutLeecher(t.Infohash, p)
		if err != nil {
			return
		}
		t.Leechers[p.Key()] = *p

	default:
		if ann.Left == 0 {
			err = c.PutSeeder(t.Infohash, p)
			if err != nil {
				return
			}
			t.Seeders[p.Key()] = *p
		} else {
			err = c.PutLeecher(t.Infohash, p)
			if err != nil {
				return
			}
			t.Leechers[p.Key()] = *p
		}
		created = true
	}

	return
}

func handleEvent(c Conn, ann *models.Announce, p *models.Peer, u *models.User, t *models.Torrent) (snatched bool, err error) {
	switch {
	case ann.Event == "stopped" || ann.Event == "paused":
		if t.InSeederPool(p) {
			err = c.DeleteSeeder(t.Infohash, p.Key())
			if err != nil {
				return
			}
			delete(t.Seeders, p.Key())
		} else if t.InLeecherPool(p) {
			err = c.DeleteLeecher(t.Infohash, p.Key())
			if err != nil {
				return
			}
			delete(t.Leechers, p.Key())
		}

	case ann.Event == "completed":
		err = c.IncrementSnatches(t.Infohash)
		if err != nil {
			return
		}
		snatched = true
		t.Snatches++

		if t.InLeecherPool(p) {
			err = leecherFinished(c, t.Infohash, p)
			if err != nil {
				return
			}
		}

	case t.InLeecherPool(p) && ann.Left == 0:
		// A leecher completed but the event was never received.
		err = leecherFinished(c, t.Infohash, p)
		if err != nil {
			return
		}
	}

	return
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

func appendPeers(ipv4s, ipv6s models.PeerList, ann *models.Announce, announcer *models.Peer, peers models.PeerMap, wanted int) (models.PeerList, models.PeerList) {
	if ann.Config.PreferredSubnet {
		return appendSubnetPeers(ipv4s, ipv6s, ann, announcer, peers, wanted)
	}

	count := 0

	for _, peer := range peers {
		if count >= wanted {
			break
		}

		if peer.ID == announcer.ID || peer.UserID != 0 && peer.UserID == announcer.UserID {
			continue
		}

		if ip := peer.IP.To4(); ip != nil {
			ipv4s = append(ipv4s, peer)
		} else if ip := peer.IP.To16(); ip != nil {
			ipv6s = append(ipv6s, peer)
		}

		count++
	}

	return ipv4s, ipv6s
}

func appendSubnetPeers(ipv4s, ipv6s models.PeerList, ann *models.Announce, announcer *models.Peer, peers models.PeerMap, wanted int) (models.PeerList, models.PeerList) {
	var (
		subnet     net.IPNet
		ipv4Subnet bool
		ipv6Subnet bool
	)

	if aip := announcer.IP.To4(); len(aip) == 4 {
		subnet = net.IPNet{aip, net.CIDRMask(ann.Config.PreferredIPv4Subnet, 32)}
		ipv4Subnet = true
	} else if aip := announcer.IP.To16(); len(aip) == 16 {
		subnet = net.IPNet{aip, net.CIDRMask(ann.Config.PreferredIPv6Subnet, 128)}
		ipv6Subnet = true
	}

	// Iterate over the peers twice: first add only peers in the same subnet and
	// if we still need more peers grab any that have already been added.
	count := 0
	for _, peersLeftInSubnet := range [2]bool{true, false} {
		for _, peer := range peers {
			if count >= wanted {
				break
			}

			if peer.ID == announcer.ID || peer.UserID != 0 && peer.UserID == announcer.UserID {
				continue
			}

			if ip := peer.IP.To4(); len(ip) == 4 {
				if peersLeftInSubnet && ipv4Subnet {
					if subnet.Contains(ip) {
						ipv4s = append(ipv4s, peer)
						count++
					}
				} else if !peersLeftInSubnet && !subnet.Contains(ip) {
					ipv4s = append(ipv4s, peer)
					count++
				}
			} else if ip := peer.IP.To16(); len(ip) == 16 {
				if peersLeftInSubnet && ipv6Subnet {
					ipv6s = append(ipv6s, peer)
					count++
				} else if !peersLeftInSubnet && !subnet.Contains(ip) {
					ipv6s = append(ipv6s, peer)
					count++
				}
			}
		}
	}

	return ipv4s, ipv6s
}
