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

	peer := models.NewPeer(announce, user, torrent)

	created, err := updateTorrent(conn, announce, peer, torrent)
	if err != nil {
		fail(err, w, r)
		return
	}

	snatched, err := handleEvent(conn, announce, peer, user, torrent)
	if err != nil {
		fail(err, w, r)
		return
	}

	if s.conf.Private {
		delta := models.NewAnnounceDelta(announce, peer, user, torrent, created, snatched)
		s.backendConn.RecordAnnounce(delta)
	}

	writeAnnounceResponse(w, announce, user, torrent)

	w.(http.Flusher).Flush()

	if log.V(5) {
		log.Infof(
			"announce: ip: %s, user: %s, torrent: %s",
			announce.IP,
			user.ID,
			torrent.ID,
		)
	}
}

func updateTorrent(c tracker.Conn, a *models.Announce, p *models.Peer, t *models.Torrent) (created bool, err error) {
	if !t.Active && a.Left == 0 {
		err = c.MarkActive(t)
		if err != nil {
			return
		}
	}

	switch {
	case t.InSeederPool(p):
		err = c.SetSeeder(t, p)
		if err != nil {
			return
		}

	case t.InLeecherPool(p):
		err = c.SetLeecher(t, p)
		if err != nil {
			return
		}

	default:
		if a.Left == 0 {
			err = c.AddSeeder(t, p)
			if err != nil {
				return
			}
		} else {
			err = c.AddLeecher(t, p)
			if err != nil {
				return
			}
		}
		created = true
	}

	return
}

func handleEvent(c tracker.Conn, a *models.Announce, p *models.Peer, u *models.User, t *models.Torrent) (snatched bool, err error) {
	switch {
	case a.Event == "stopped" || a.Event == "paused":
		if t.InSeederPool(p) {
			err = c.RemoveSeeder(t, p)
			if err != nil {
				return
			}
		}
		if t.InLeecherPool(p) {
			err = c.RemoveLeecher(t, p)
			if err != nil {
				return
			}
		}

	case a.Event == "completed":
		err = c.IncrementSnatches(t)
		if err != nil {
			return
		}
		snatched = true

		if t.InLeecherPool(p) {
			err = tracker.LeecherFinished(c, t, p)
			if err != nil {
				return
			}
		}

	case t.InLeecherPool(p) && a.Left == 0:
		// A leecher completed but the event was never received
		err = tracker.LeecherFinished(c, t, p)
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
		if a.Left == 0 {
			peerCount = minInt(a.NumWant, leechCount)
		} else {
			peerCount = minInt(a.NumWant, leechCount+seedCount-1)
		}

		if a.Compact {
			// 6 is the number of bytes 1 compact peer takes up.
			bencoder.Encode(strconv.Itoa(peerCount * 6))
			bencoder.Encode(":")
		} else {
			bencoder.Encode("l")
		}

		if a.Left == 0 {
			// If they're seeding, give them only leechers
			writePeers(w, u, t.Leechers, peerCount, a.Compact)
		} else {
			// If they're leeching, prioritize giving them seeders
			count := writePeers(w, u, t.Seeders, peerCount, a.Compact)
			writePeers(w, u, t.Leechers, peerCount-count, a.Compact)
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
