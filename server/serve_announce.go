// Copyright 2013 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

package server

import (
	"errors"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/chihaya/chihaya/storage"
	"github.com/chihaya/chihaya/storage/backend"
)

func (s Server) serveAnnounce(w http.ResponseWriter, r *http.Request) {
	// Parse the required data from a request
	announce, err := newAnnounce(r, s.conf)
	if err != nil {
		fail(err, w, r)
		return
	}

	// Get a connection to the tracker db
	conn, err := s.trackerPool.Get()
	if err != nil {
		log.Panicf("server: %s", err)
	}

	// Validate the user's passkey
	user, err := validateUser(conn, announce.Passkey)
	if err != nil {
		fail(err, w, r)
		return
	}

	// Check if the user's client is whitelisted
	whitelisted, err := conn.ClientWhitelisted(parsePeerID(announce.PeerID))
	if err != nil {
		log.Panicf("server: %s", err)
	}
	if !whitelisted {
		fail(errors.New("client is not approved"), w, r)
		return
	}

	// Find the specified torrent
	torrent, exists, err := conn.FindTorrent(announce.Infohash)
	if err != nil {
		log.Panicf("server: %s", err)
	}
	if !exists {
		fail(errors.New("torrent does not exist"), w, r)
		return
	}

	// If the torrent was pruned and the user is seeding, unprune it
	if !torrent.Active && announce.Left == 0 {
		err := conn.MarkActive(torrent)
		if err != nil {
			log.Panicf("server: %s", err)
		}
	}

	now := time.Now().Unix()
	// Create a new peer object from the request
	peer := &storage.Peer{
		ID:           announce.PeerID,
		UserID:       user.ID,
		TorrentID:    torrent.ID,
		IP:           announce.IP,
		Port:         announce.Port,
		Uploaded:     announce.Uploaded,
		Downloaded:   announce.Downloaded,
		Left:         announce.Left,
		LastAnnounce: now,
	}
	delta := &backend.AnnounceDelta{
		Peer:      peer,
		Torrent:   torrent,
		User:      user,
		Timestamp: now,
	}

	// Look for the user in in the pool of seeders and leechers
	_, seeder := torrent.Seeders[storage.PeerMapKey(peer)]
	_, leecher := torrent.Leechers[storage.PeerMapKey(peer)]

	switch {
	// Guarantee that no user is in both pools
	case seeder && leecher:
		if announce.Left == 0 {
			err := conn.RemoveLeecher(torrent, peer)
			if err != nil {
				log.Panicf("server: %s", err)
			}
			leecher = false
		} else {
			err := conn.RemoveSeeder(torrent, peer)
			if err != nil {
				log.Panicf("server: %s", err)
			}
			seeder = false
		}

	case seeder:
		// Update the peer with the stats from the request
		err := conn.SetSeeder(torrent, peer)
		if err != nil {
			log.Panicf("server: %s", err)
		}

	case leecher:
		// Update the peer with the stats from the request
		err := conn.SetLeecher(torrent, peer)
		if err != nil {
			log.Panicf("server: %s", err)
		}

	default:
		if announce.Left == 0 {
			// Save the peer as a new seeder
			err := conn.AddSeeder(torrent, peer)
			if err != nil {
				log.Panicf("server: %s", err)
			}
		} else {
			err = conn.AddLeecher(torrent, peer)
			if err != nil {
				log.Panicf("server: %s", err)
			}
		}
		delta.Created = true
	}

	// Handle any events in the request
	switch {
	case announce.Event == "stopped" || announce.Event == "paused":
		if seeder {
			err := conn.RemoveSeeder(torrent, peer)
			if err != nil {
				log.Panicf("server: %s", err)
			}
		}
		if leecher {
			err := conn.RemoveLeecher(torrent, peer)
			if err != nil {
				log.Panicf("server: %s", err)
			}
		}

	case announce.Event == "completed":
		err := conn.RecordSnatch(user, torrent)
		if err != nil {
			log.Panicf("server: %s", err)
		}
		delta.Snatched = true
		if leecher {
			err := conn.LeecherFinished(torrent, peer)
			if err != nil {
				log.Panicf("server: %s", err)
			}
		}

	case leecher && announce.Left == 0:
		// A leecher completed but the event was never received
		err := conn.LeecherFinished(torrent, peer)
		if err != nil {
			log.Panicf("server: %s", err)
		}
	}

	if announce.IP != peer.IP || announce.Port != peer.Port {
		peer.Port = announce.Port
		peer.IP = announce.IP
	}

	// Generate the response
	seedCount := len(torrent.Seeders)
	leechCount := len(torrent.Leechers)

	writeBencoded(w, "d")
	writeBencoded(w, "complete")
	writeBencoded(w, seedCount)
	writeBencoded(w, "incomplete")
	writeBencoded(w, leechCount)
	writeBencoded(w, "interval")
	writeBencoded(w, s.conf.Announce.Duration)
	writeBencoded(w, "min interval")
	writeBencoded(w, s.conf.MinAnnounce.Duration)

	if announce.NumWant > 0 && announce.Event != "stopped" && announce.Event != "paused" {
		writeBencoded(w, "peers")
		var peerCount, count int

		if announce.Compact {
			if announce.Left > 0 {
				peerCount = minInt(announce.NumWant, leechCount)
			} else {
				peerCount = minInt(announce.NumWant, leechCount+seedCount-1)
			}
			writeBencoded(w, strconv.Itoa(peerCount*6))
			writeBencoded(w, ":")
		} else {
			writeBencoded(w, "l")
		}

		if announce.Left > 0 {
			// If they're seeding, give them only leechers
			count += writeLeechers(w, user, torrent, announce.NumWant, announce.Compact)
		} else {
			// If they're leeching, prioritize giving them seeders
			count += writeSeeders(w, user, torrent, announce.NumWant, announce.Compact)
			count += writeLeechers(w, user, torrent, announce.NumWant-count, announce.Compact)
		}

		if announce.Compact && peerCount != count {
			log.Panicf("calculated peer count (%d) != real count (%d)", peerCount, count)
		}

		if !announce.Compact {
			writeBencoded(w, "e")
		}
	}
	writeBencoded(w, "e")

	rawDeltaUp := peer.Uploaded - announce.Uploaded
	rawDeltaDown := peer.Downloaded - announce.Downloaded

	// Restarting a torrent may cause a delta to be negative.
	if rawDeltaUp < 0 {
		rawDeltaUp = 0
	}
	if rawDeltaDown < 0 {
		rawDeltaDown = 0
	}

	delta.Uploaded = uint64(float64(rawDeltaUp) * user.UpMultiplier * torrent.UpMultiplier)
	delta.Downloaded = uint64(float64(rawDeltaDown) * user.DownMultiplier * torrent.DownMultiplier)

	s.backendConn.RecordAnnounce(delta)
}

func minInt(a, b int) int {
	if a < b {
		return a
	}

	return b
}

func writeSeeders(w http.ResponseWriter, user *storage.User, t *storage.Torrent, numWant int, compact bool) int {
	count := 0
	for _, peer := range t.Seeders {
		if count >= numWant {
			break
		}

		if peer.UserID == user.ID {
			continue
		}

		if compact {
			// TODO writeBencoded(w, compactAddr)
		} else {
			writeBencoded(w, "d")
			writeBencoded(w, "ip")
			writeBencoded(w, peer.IP)
			writeBencoded(w, "peer id")
			writeBencoded(w, peer.ID)
			writeBencoded(w, "port")
			writeBencoded(w, peer.Port)
			writeBencoded(w, "e")
		}
		count++
	}

	return count
}

func writeLeechers(w http.ResponseWriter, user *storage.User, t *storage.Torrent, numWant int, compact bool) int {
	count := 0
	for _, peer := range t.Leechers {
		if count >= numWant {
			break
		}

		if peer.UserID == user.ID {
			continue
		}

		if compact {
			// TODO writeBencoded(w, compactAddr)
		} else {
			writeBencoded(w, "d")
			writeBencoded(w, "ip")
			writeBencoded(w, peer.IP)
			writeBencoded(w, "peer id")
			writeBencoded(w, peer.ID)
			writeBencoded(w, "port")
			writeBencoded(w, peer.Port)
			writeBencoded(w, "e")
		}
		count++
	}

	return count
}
