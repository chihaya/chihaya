// Copyright 2014 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

package http

import (
	"fmt"
	"io"
	"net/http"

	"github.com/julienschmidt/httprouter"

	"github.com/chihaya/bencode"
	"github.com/chihaya/chihaya/drivers/tracker"
	"github.com/chihaya/chihaya/models"
)

func (t *Tracker) ServeAnnounce(w http.ResponseWriter, r *http.Request, p httprouter.Params) (int, error) {
	ann, err := models.NewAnnounce(t.cfg, r, p)
	if err == models.ErrMalformedRequest {
		fail(w, r, err)
		return http.StatusOK, nil
	} else if err != nil {
		return http.StatusInternalServerError, err
	}

	conn, err := t.tp.Get()
	if err != nil {
		return http.StatusInternalServerError, err
	}

	if t.cfg.Whitelist {
		err = conn.FindClient(ann.ClientID())
		if err == tracker.ErrClientUnapproved {
			fail(w, r, err)
			return http.StatusOK, nil
		} else if err != nil {
			return http.StatusInternalServerError, err
		}
	}

	var user *models.User
	if t.cfg.Private {
		user, err = conn.FindUser(ann.Passkey)
		if err == tracker.ErrUserDNE {
			fail(w, r, err)
			return http.StatusOK, nil
		} else if err != nil {
			return http.StatusInternalServerError, err
		}
	}

	var torrent *models.Torrent
	torrent, err = conn.FindTorrent(ann.Infohash)
	switch {
	case t.cfg.Private && err == tracker.ErrTorrentDNE:
		torrent = &models.Torrent{
			Infohash: ann.Infohash,
			Seeders:  make(map[string]models.Peer),
			Leechers: make(map[string]models.Peer),
		}

		err = conn.PutTorrent(torrent)
		if err != nil {
			return http.StatusInternalServerError, err
		}

	case !t.cfg.Private && err == tracker.ErrTorrentDNE:
		fail(w, r, err)
		return http.StatusOK, nil

	case err != nil:
		return http.StatusInternalServerError, err
	}

	peer := models.NewPeer(ann, user, torrent)

	created, err := updateTorrent(conn, ann, peer, torrent)
	if err != nil {
		return http.StatusInternalServerError, err
	}

	snatched, err := handleEvent(conn, ann, peer, user, torrent)
	if err != nil {
		return http.StatusInternalServerError, err
	}

	if t.cfg.Private {
		delta := models.NewAnnounceDelta(ann, peer, user, torrent, created, snatched)
		err = t.bc.RecordAnnounce(delta)
		if err != nil {
			return http.StatusInternalServerError, err
		}
	}

	writeAnnounceResponse(w, ann, user, torrent)

	return http.StatusOK, nil
}

func updateTorrent(c tracker.Conn, a *models.Announce, p *models.Peer, t *models.Torrent) (created bool, err error) {
	if !t.Active && a.Left == 0 {
		err = c.MarkActive(t.Infohash)
		if err != nil {
			return
		}
		t.Active = true
	}

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
		if a.Left == 0 {
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

func handleEvent(c tracker.Conn, a *models.Announce, p *models.Peer, u *models.User, t *models.Torrent) (snatched bool, err error) {
	switch {
	case a.Event == "stopped" || a.Event == "paused":
		if t.InSeederPool(p) {
			err = c.DeleteSeeder(t.Infohash, p.Key())
			if err != nil {
				return
			}
			delete(t.Seeders, p.Key())
		}
		if t.InLeecherPool(p) {
			err = c.DeleteLeecher(t.Infohash, p.Key())
			if err != nil {
				return
			}
			delete(t.Leechers, p.Key())
		}

	case a.Event == "completed":
		err = c.IncrementSnatches(t.Infohash)
		if err != nil {
			return
		}
		snatched = true
		t.Snatches++

		if t.InLeecherPool(p) {
			err = tracker.LeecherFinished(c, t.Infohash, p)
			if err != nil {
				return
			}
		}

	case t.InLeecherPool(p) && a.Left == 0:
		// A leecher completed but the event was never received.
		err = tracker.LeecherFinished(c, t.Infohash, p)
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
	bencoder := bencode.NewEncoder(w)
	ipv4s, ipv6s := getPeers(a, u, t, peerCount)

	if len(ipv4s) > 0 {
		// 6 is the number of bytes that represents 1 compact IPv4 address.
		bencoder.Encode("peers")
		fmt.Fprintf(w, "%d:", len(ipv4s)*6)

		for _, peer := range ipv4s {
			if ip := peer.IP.To4(); ip != nil {
				w.Write(ip)
				w.Write([]byte{byte(peer.Port >> 8), byte(peer.Port & 0xff)})
			}
		}
	}

	if len(ipv6s) > 0 {
		// 18 is the number of bytes that represents 1 compact IPv6 address.
		bencoder.Encode("peers6")
		fmt.Fprintf(w, "%d:", len(ipv6s)*18)

		for _, peer := range ipv6s {
			if ip := peer.IP.To16(); ip != nil {
				w.Write(ip)
				w.Write([]byte{byte(peer.Port >> 8), byte(peer.Port & 0xff)})
			}
		}
	}
}

func getPeers(a *models.Announce, u *models.User, t *models.Torrent, peerCount int) (ipv4s, ipv6s []models.Peer) {
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

func splitPeers(ipv4s, ipv6s *[]models.Peer, a *models.Announce, u *models.User, peers map[string]models.Peer, peerCount int) (count int) {
	for _, peer := range peers {
		if count >= peerCount {
			break
		}

		if a.Config.Private && peer.UserID == u.ID {
			continue
		}

		if ip := peer.IP.To4(); len(ip) == 4 {
			*ipv4s = append(*ipv4s, peer)
		} else if ip := peer.IP.To16(); len(ip) == 16 {
			*ipv6s = append(*ipv6s, peer)
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
		writePeerDict(w, &peer)
	}
	for _, peer := range ipv6s {
		writePeerDict(w, &peer)
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
