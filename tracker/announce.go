// Copyright 2015 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

package tracker

import (
	"github.com/chihaya/chihaya/stats"
	"github.com/chihaya/chihaya/tracker/models"
)

// HandleAnnounce encapsulates all of the logic of handling a BitTorrent
// client's Announce without being coupled to any transport protocol.
func (tkr *Tracker) HandleAnnounce(ann *models.Announce, w Writer) (err error) {
	if tkr.Config.ClientWhitelistEnabled {
		if err = tkr.ClientApproved(ann.ClientID()); err != nil {
			return err
		}
	}

	if tkr.Config.JWKSetURI != "" {
		err := tkr.validateJWT(ann.JWT, ann.Infohash)
		if err != nil {
			return err
		}
	}

	torrent, err := tkr.FindTorrent(ann.Infohash)
	if err == models.ErrTorrentDNE && tkr.Config.CreateOnAnnounce {
		torrent = &models.Torrent{
			Infohash: ann.Infohash,
			Seeders:  models.NewPeerMap(true, tkr.Config),
			Leechers: models.NewPeerMap(false, tkr.Config),
		}

		tkr.PutTorrent(torrent)
		stats.RecordEvent(stats.NewTorrent)
	} else if err != nil {
		return err
	}

	ann.BuildPeer(torrent)

	_, err = tkr.updateSwarm(ann)
	if err != nil {
		return err
	}

	_, err = tkr.handleEvent(ann)
	if err != nil {
		return err
	}

	if tkr.Config.PurgeInactiveTorrents && torrent.PeerCount() == 0 {
		// Rather than deleting the torrent explicitly, let the tracker driver
		// ensure there are no race conditions.
		tkr.PurgeInactiveTorrent(torrent.Infohash)
		stats.RecordEvent(stats.DeletedTorrent)
	}

	stats.RecordEvent(stats.Announce)
	return w.WriteAnnounce(newAnnounceResponse(ann))
}

// updateSwarm handles the changes to a torrent's swarm given an announce.
func (tkr *Tracker) updateSwarm(ann *models.Announce) (created bool, err error) {
	var createdv4, createdv6 bool
	tkr.TouchTorrent(ann.Torrent.Infohash)

	if ann.HasIPv4() {
		createdv4, err = tkr.updatePeer(ann, ann.PeerV4)
		if err != nil {
			return
		}
	}
	if ann.HasIPv6() {
		createdv6, err = tkr.updatePeer(ann, ann.PeerV6)
		if err != nil {
			return
		}
	}

	return createdv4 || createdv6, nil
}

func (tkr *Tracker) updatePeer(ann *models.Announce, peer *models.Peer) (created bool, err error) {
	p, t := ann.Peer, ann.Torrent

	switch {
	case t.Seeders.Contains(p.Key()):
		err = tkr.PutSeeder(t.Infohash, p)
		if err != nil {
			return
		}

	case t.Leechers.Contains(p.Key()):
		err = tkr.PutLeecher(t.Infohash, p)
		if err != nil {
			return
		}

	default:
		if ann.Left == 0 {
			err = tkr.PutSeeder(t.Infohash, p)
			if err != nil {
				return
			}
			stats.RecordPeerEvent(stats.NewSeed, p.HasIPv6())

		} else {
			err = tkr.PutLeecher(t.Infohash, p)
			if err != nil {
				return
			}
			stats.RecordPeerEvent(stats.NewLeech, p.HasIPv6())
		}
		created = true
	}
	return
}

// handleEvent checks to see whether an announce has an event and if it does,
// properly handles that event.
func (tkr *Tracker) handleEvent(ann *models.Announce) (snatched bool, err error) {
	var snatchedv4, snatchedv6 bool

	if ann.HasIPv4() {
		snatchedv4, err = tkr.handlePeerEvent(ann, ann.PeerV4)
		if err != nil {
			return
		}
	}
	if ann.HasIPv6() {
		snatchedv6, err = tkr.handlePeerEvent(ann, ann.PeerV6)
		if err != nil {
			return
		}
	}

	if snatchedv4 || snatchedv6 {
		err = tkr.IncrementTorrentSnatches(ann.Torrent.Infohash)
		if err != nil {
			return
		}
		ann.Torrent.Snatches++
		return true, nil
	}
	return false, nil
}

func (tkr *Tracker) handlePeerEvent(ann *models.Announce, p *models.Peer) (snatched bool, err error) {
	p, t := ann.Peer, ann.Torrent

	switch {
	case ann.Event == "stopped" || ann.Event == "paused":
		// updateSwarm checks if the peer is active on the torrent,
		// so one of these branches must be followed.
		if t.Seeders.Contains(p.Key()) {
			err = tkr.DeleteSeeder(t.Infohash, p)
			if err != nil {
				return
			}
			stats.RecordPeerEvent(stats.DeletedSeed, p.HasIPv6())

		} else if t.Leechers.Contains(p.Key()) {
			err = tkr.DeleteLeecher(t.Infohash, p)
			if err != nil {
				return
			}
			stats.RecordPeerEvent(stats.DeletedLeech, p.HasIPv6())
		}

	case t.Leechers.Contains(p.Key()) && (ann.Event == "completed" || ann.Left == 0):
		// A leecher has completed or this is the first time we've seen them since
		// they've completed.
		err = tkr.leecherFinished(t, p)
		if err != nil {
			return
		}

		// Only mark as snatched if we receive the completed event.
		if ann.Event == "completed" {
			snatched = true
		}
	}

	return
}

// leecherFinished moves a peer from the leeching pool to the seeder pool.
func (tkr *Tracker) leecherFinished(t *models.Torrent, p *models.Peer) error {
	if err := tkr.DeleteLeecher(t.Infohash, p); err != nil {
		return err
	}

	if err := tkr.PutSeeder(t.Infohash, p); err != nil {
		return err
	}

	stats.RecordPeerEvent(stats.Completed, p.HasIPv6())
	return nil
}

func newAnnounceResponse(ann *models.Announce) *models.AnnounceResponse {
	seedCount := ann.Torrent.Seeders.Len()
	leechCount := ann.Torrent.Leechers.Len()

	res := &models.AnnounceResponse{
		Announce:    ann,
		Complete:    seedCount,
		Incomplete:  leechCount,
		Interval:    ann.Config.Announce.Duration,
		MinInterval: ann.Config.MinAnnounce.Duration,
		Compact:     ann.Compact,
	}

	if ann.NumWant > 0 && ann.Event != "stopped" && ann.Event != "paused" {
		res.IPv4Peers, res.IPv6Peers = getPeers(ann)

		if len(res.IPv4Peers)+len(res.IPv6Peers) == 0 {
			models.AppendPeer(&res.IPv4Peers, &res.IPv6Peers, ann, ann.Peer)
		}
	}

	return res
}

// getPeers returns lists IPv4 and IPv6 peers on a given torrent sized according
// to the wanted parameter.
func getPeers(ann *models.Announce) (ipv4s, ipv6s models.PeerList) {
	ipv4s, ipv6s = models.PeerList{}, models.PeerList{}

	if ann.Left == 0 {
		// If they're seeding, give them only leechers.
		return ann.Torrent.Leechers.AppendPeers(ipv4s, ipv6s, ann, ann.NumWant)
	}

	// If they're leeching, prioritize giving them seeders.
	ipv4s, ipv6s = ann.Torrent.Seeders.AppendPeers(ipv4s, ipv6s, ann, ann.NumWant)
	return ann.Torrent.Leechers.AppendPeers(ipv4s, ipv6s, ann, ann.NumWant-len(ipv4s)-len(ipv6s))
}
