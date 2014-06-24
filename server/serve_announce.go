// Copyright 2013 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

package server

import (
	"io"
	"net"
	"net/http"
	"strconv"

	log "github.com/golang/glog"

	"github.com/chihaya/chihaya/bencode"
	"github.com/chihaya/chihaya/drivers/tracker"
	"github.com/chihaya/chihaya/models"
)

func (s Server) serveAnnounce(w http.ResponseWriter, r *http.Request) {
	announce, err := models.NewAnnounce(r, s.conf)
	if err != nil {
		fail(err, w, r)
		return
	}

	conn, err := s.trackerPool.Get()
	if err != nil {
		fail(err, w, r)
		return
	}

	err = conn.ClientWhitelisted(announce.ClientID())
	if err != nil {
		fail(err, w, r)
		return
	}

	var user *models.User
	if s.conf.Private {
		user, err = conn.FindUser(announce.Passkey)
		if err != nil {
			fail(err, w, r)
			return
		}
	}

	torrent, err := conn.FindTorrent(announce.Infohash)
	if err != nil {
		fail(err, w, r)
		return
	}

	peer := models.NewPeer(torrent, user, announce)

	created, err := updateTorrent(peer, torrent, conn, announce)
	if err != nil {
		fail(err, w, r)
		return
	}

	snatched, err := handleEvent(announce, user, torrent, peer, conn)
	if err != nil {
		fail(err, w, r)
		return
	}

	writeAnnounceResponse(w, announce, user, torrent)

	delta := models.NewAnnounceDelta(peer, user, announce, torrent, created, snatched)
	s.backendConn.RecordAnnounce(delta)

	log.V(3).Infof("chihaya: handled announce from %s", announce.IP)
}

func updateTorrent(p *models.Peer, t *models.Torrent, conn tracker.Conn, a *models.Announce) (created bool, err error) {
	if !t.Active && a.Left == 0 {
		err = conn.MarkActive(t)
		if err != nil {
			return
		}
	}

	switch {
	case t.InSeederPool(p):
		err = conn.SetSeeder(t, p)
		if err != nil {
			return
		}

	case t.InLeecherPool(p):
		err = conn.SetLeecher(t, p)
		if err != nil {
			return
		}

	default:
		if a.Left == 0 {
			err = conn.AddSeeder(t, p)
			if err != nil {
				return
			}
		} else {
			err = conn.AddLeecher(t, p)
			if err != nil {
				return
			}
		}
		created = true
	}

	return
}

func handleEvent(a *models.Announce, u *models.User, t *models.Torrent, p *models.Peer, conn tracker.Conn) (snatched bool, err error) {
	switch {
	case a.Event == "stopped" || a.Event == "paused":
		if t.InSeederPool(p) {
			err = conn.RemoveSeeder(t, p)
			if err != nil {
				return
			}
		}
		if t.InLeecherPool(p) {
			err = conn.RemoveLeecher(t, p)
			if err != nil {
				return
			}
		}

	case a.Event == "completed":
		err = conn.IncrementSnatches(t)
		if err != nil {
			return
		}
		snatched = true

		if t.InLeecherPool(p) {
			err = tracker.LeecherFinished(conn, t, p)
			if err != nil {
				return
			}
		}

	case t.InLeecherPool(p) && a.Left == 0:
		// A leecher completed but the event was never received
		err = tracker.LeecherFinished(conn, t, p)
		if err != nil {
			return
		}
	}

	return
}

func writeAnnounceResponse(w io.Writer, a *models.Announce, u *models.User, t *models.Torrent) {
	bencoder := bencode.NewEncoder(w)
	seedCount := len(t.Seeders)
	leechCount := len(t.Leechers)

	bencoder.Encode("d")
	bencoder.Encode("complete")
	bencoder.Encode(seedCount)
	bencoder.Encode("incomplete")
	bencoder.Encode(leechCount)
	bencoder.Encode("interval")
	bencoder.Encode(a.Config.Announce.Duration)
	bencoder.Encode("min interval")
	bencoder.Encode(a.Config.MinAnnounce.Duration)

	if a.NumWant > 0 && a.Event != "stopped" && a.Event != "paused" {
		bencoder.Encode("peers")

		var peerCount int
		if a.Compact {
			if a.Left == 0 {
				peerCount = minInt(a.NumWant, leechCount)
			} else {
				peerCount = minInt(a.NumWant, leechCount+seedCount-1)
			}
			// 6 is the number of bytes 1 compact peer takes up.
			bencoder.Encode(strconv.Itoa(peerCount * 6))
			bencoder.Encode(":")
		} else {
			bencoder.Encode("l")
		}

		var count int
		if a.Left == 0 {
			// If they're seeding, give them only leechers
			count = writePeers(w, u, t.Leechers, a.NumWant, a.Compact)
		} else {
			// If they're leeching, prioritize giving them seeders
			count += writePeers(w, u, t.Seeders, a.NumWant, a.Compact)
			count += writePeers(w, u, t.Leechers, a.NumWant-count, a.Compact)
		}
		if a.Compact && peerCount != count {
			log.Errorf("calculated peer count (%d) != real count (%d)", peerCount, count)
		}

		if !a.Compact {
			bencoder.Encode("e")
		}
	}
	bencoder.Encode("e")
}

func writePeers(w io.Writer, user *models.User, peers map[string]models.Peer, numWant int, compact bool) (count int) {
	bencoder := bencode.NewEncoder(w)
	for _, peer := range peers {
		if count >= numWant {
			break
		}

		if peer.UserID == user.ID {
			continue
		}

		if compact {
			if ip := net.ParseIP(peer.IP); ip != nil {
				w.Write(ip)
				w.Write([]byte{byte(peer.Port >> 8), byte(peer.Port & 0xff)})
			}
		} else {
			bencoder.Encode("d")
			bencoder.Encode("ip")
			bencoder.Encode(peer.IP)
			bencoder.Encode("peer id")
			bencoder.Encode(peer.ID)
			bencoder.Encode("port")
			bencoder.Encode(peer.Port)
			bencoder.Encode("e")
		}
		count++
	}

	return
}

func minInt(a, b int) int {
	if a < b {
		return a
	}

	return b
}
