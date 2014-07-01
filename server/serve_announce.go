// Copyright 2013 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

package server

import (
	"fmt"
	"io"
	"net/http"

	"github.com/golang/glog"

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

	if s.conf.Private {
		glog.V(5).Infof(
			"announce: ip: %s user: %s torrent: %s",
			announce.IP,
			user.ID,
			torrent.ID,
		)
	} else {
		glog.V(5).Infof("announce: ip: %s torrent: %s", announce.IP, torrent.ID)
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
	seedCount := len(t.Seeders)
	leechCount := len(t.Leechers)

	var peerCount int
	if a.Left == 0 {
		peerCount = minInt(a.NumWant, leechCount)
	} else {
		peerCount = minInt(a.NumWant, leechCount+seedCount-1)
	}

	bencoder := bencode.NewEncoder(w)
	fmt.Fprintf(w, "d")
	bencoder.Encode("complete")
	bencoder.Encode(seedCount)
	bencoder.Encode("incomplete")
	bencoder.Encode(leechCount)
	bencoder.Encode("interval")
	bencoder.Encode(a.Config.Announce.Duration)
	bencoder.Encode("min interval")
	bencoder.Encode(a.Config.MinAnnounce.Duration)

	if a.NumWant > 0 && a.Event != "stopped" && a.Event != "paused" {
		if a.Compact {
			writePeersCompact(w, a, u, t, peerCount)
		} else {
			writePeersList(w, a, u, t, peerCount)
		}
	}

	fmt.Fprintf(w, "e")
}

func writePeersCompact(w io.Writer, a *models.Announce, u *models.User, t *models.Torrent, peerCount int) {
	ipv4s, ipv6s := getPeers(a, u, t, peerCount)

	if len(ipv4s) > 0 {
		// 6 is the number of bytes that represents 1 compact IPv4 address.
		fmt.Fprintf(w, "peers%d:", len(ipv4s)*6)

		for _, peer := range ipv4s {
			if ip := peer.IP.To4(); ip != nil {
				w.Write(ip)
				w.Write([]byte{byte(peer.Port >> 8), byte(peer.Port & 0xff)})
			}
		}
	}

	if len(ipv6s) > 0 {
		// 18 is the number of bytes that represents 1 compact IPv6 address.
		fmt.Fprintf(w, "peers6%d:", len(ipv6s)*18)

		for _, peer := range ipv6s {
			if ip := peer.IP.To16(); ip != nil {
				w.Write(ip)
				w.Write([]byte{byte(peer.Port >> 8), byte(peer.Port & 0xff)})
			}
		}
	}
}

func getPeers(a *models.Announce, u *models.User, t *models.Torrent, peerCount int) (ipv4s, ipv6s []*models.Peer) {
	if a.Left == 0 {
		// If they're seeding, give them only leechers.
		splitPeers(&ipv4s, &ipv6s, a, u, t.Leechers, peerCount)
	} else {
		// If they're leeching, prioritize giving them seeders.
		count := splitPeers(&ipv4s, &ipv6s, a, u, t.Seeders, peerCount)
		splitPeers(&ipv4s, &ipv6s, a, u, t.Leechers, peerCount-count)
	}

	return
}

func splitPeers(ipv4s, ipv6s *[]*models.Peer, a *models.Announce, u *models.User, peers map[string]models.Peer, peerCount int) (count int) {
	for _, peer := range peers {
		if count >= peerCount {
			break
		}

		if a.Config.Private && peer.UserID == u.ID {
			continue
		}

		if ip := peer.IP.To4(); len(ip) == 4 {
			*ipv4s = append(*ipv4s, &peer)
		} else if ip := peer.IP.To16(); len(ip) == 16 {
			*ipv6s = append(*ipv6s, &peer)
		}

		count++
	}

	return
}

func writePeersList(w io.Writer, a *models.Announce, u *models.User, t *models.Torrent, peerCount int) {
	bencoder := bencode.NewEncoder(w)
	ipv4s, ipv6s := getPeers(a, u, t, peerCount)

	bencoder.Encode("peers")
	fmt.Fprintf(w, "l")

	for _, peer := range ipv4s {
		writePeerDict(w, peer)
	}
	for _, peer := range ipv6s {
		writePeerDict(w, peer)
	}

	fmt.Fprintf(w, "e")
}

func writePeerDict(w io.Writer, peer *models.Peer) {
	bencoder := bencode.NewEncoder(w)
	fmt.Fprintf(w, "d")
	bencoder.Encode("ip")
	bencoder.Encode(peer.IP.String())
	bencoder.Encode("peer id")
	bencoder.Encode(peer.ID)
	bencoder.Encode("port")
	bencoder.Encode(peer.Port)
	fmt.Fprintf(w, "e")
}

func minInt(a, b int) int {
	if a < b {
		return a
	}

	return b
}
