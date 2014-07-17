// Copyright 2014 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

package tracker

import (
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
			Seeders:  make(map[string]models.Peer),
			Leechers: make(map[string]models.Peer),
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
			err = LeecherFinished(c, t.Infohash, p)
			if err != nil {
				return
			}
		}

	case t.InLeecherPool(p) && ann.Left == 0:
		// A leecher completed but the event was never received.
		err = LeecherFinished(c, t.Infohash, p)
		if err != nil {
			return
		}
	}

	return
}

func newAnnounceResponse(ann *models.Announce, announcer *models.Peer, t *models.Torrent) *AnnounceResponse {
	seedCount := len(t.Seeders)
	leechCount := len(t.Leechers)

	res := &AnnounceResponse{
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

func getPeers(ann *models.Announce, announcer *models.Peer, t *models.Torrent, wanted int) (ipv4s, ipv6s PeerList) {
	ipv4s, ipv6s = PeerList{}, PeerList{}

	if ann.Left == 0 {
		// If they're seeding, give them only leechers.
		return appendPeers(ipv4s, ipv6s, announcer, t.Leechers, wanted)
	}

	// If they're leeching, prioritize giving them seeders.
	ipv4s, ipv6s = appendPeers(ipv4s, ipv6s, announcer, t.Seeders, wanted)
	return appendPeers(ipv4s, ipv6s, announcer, t.Leechers, wanted-len(ipv4s)-len(ipv6s))
}

func appendPeers(ipv4s, ipv6s PeerList, announcer *models.Peer, peers map[string]models.Peer, wanted int) (PeerList, PeerList) {
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
