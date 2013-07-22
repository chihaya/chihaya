// Copyright 2013 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

package server

import (
	"errors"
	"log"
	"net/http"
	"path"
	"strconv"
	"time"

	"github.com/pushrax/chihaya/storage"
)

func (s *Server) serveAnnounce(w http.ResponseWriter, r *http.Request) {
	// Parse the required parameters off of a query
	compact, numWant, infohash, peerID, event, ip, port, uploaded, downloaded, left, err := s.validateAnnounceQuery(r)
	if err != nil {
		fail(err, w, r)
		return
	}

	// Validate the user's passkey
	passkey, _ := path.Split(r.URL.Path)
	user, err := s.FindUser(passkey)
	if err != nil {
		fail(err, w, r)
		return
	}

	// Check if the user's client is whitelisted
	whitelisted, err := s.dataStore.ClientWhitelisted(peerID)
	if err != nil {
		log.Panicf("server: %s", err)
	}
	if !whitelisted {
		fail(errors.New("Your client is not approved"), w, r)
		return
	}

	// Find the specified torrent
	torrent, exists, err := s.dataStore.FindTorrent(infohash)
	if err != nil {
		log.Panicf("server: %s", err)
	}
	if !exists {
		fail(errors.New("This torrent does not exist"), w, r)
		return
	}

	// Begin a data store transaction
	tx, err := s.dataStore.Begin()
	if err != nil {
		log.Panicf("server: %s", err)
	}

	// If the torrent was pruned and the user is seeding, unprune it
	if torrent.Pruned && left == 0 {
		err := tx.Unprune(torrent)
		if err != nil {
			log.Panicf("server: %s", err)
		}
	}

	// Look for the user in in the pool of seeders and leechers
	sp, seeder := torrent.Seeders[peerID]
	lp, leecher := torrent.Leechers[peerID]
	peer := &storage.Peer{}
	switch {
	// Guarantee that no user is in both pools
	case seeder && leecher:
		if left == 0 {
			peer = &sp
			err := tx.RmLeecher(torrent, peer)
			if err != nil {
				log.Panicf("server: %s", err)
			}
			leecher = false
		} else {
			peer = &lp
			err := tx.RmSeeder(torrent, peer)
			if err != nil {
				log.Panicf("server: %s", err)
			}
			seeder = false
		}

	case seeder:
		peer = &sp
		// TODO update stats

	case leecher:
		peer = &lp
		// TODO update stats

	default:
		// The user is a new peer
		peer = &storage.Peer{
			ID:           peerID,
			UserID:       user.ID,
			TorrentID:    torrent.ID,
			IP:           ip,
			Port:         port,
			Uploaded:     uploaded,
			Downloaded:   downloaded,
			Left:         left,
			LastAnnounce: time.Now().Unix(),
		}

		// Check the new user's slots
		if s.conf.Slots && user.Slots != -1 && left != 0 {
			if user.UsedSlots >= user.Slots {
				fail(errors.New("You've run out of download slots."), w, r)
				return
			}
		}

		if left == 0 {
			err := tx.NewSeeder(torrent, peer)
			if err != nil {
				log.Panicf("server: %s", err)
			}
		} else {
			err := tx.IncrementSlots(user)
			if err != nil {
				log.Panicf("server: %s", err)
			}
			err = tx.NewLeecher(torrent, peer)
			if err != nil {
				log.Panicf("server: %s", err)
			}
		}
	}

	// Handle any events given to us by the user
	switch {
	case event == "stopped" || event == "paused":
		if seeder {
			err := tx.RmSeeder(torrent, peer)
			if err != nil {
				log.Panicf("server: %s", err)
			}
		}
		if leecher {
			err := tx.RmLeecher(torrent, peer)
			if err != nil {
				log.Panicf("server: %s", err)
			}
			err = tx.DecrementSlots(user)
			if err != nil {
				log.Panicf("server: %s", err)
			}
		}

	case event == "completed":
		err := tx.Snatch(user, torrent)
		if err != nil {
			log.Panicf("server: %s", err)
		}
		if leecher {
			err := tx.RmLeecher(torrent, peer)
			if err != nil {
				log.Panicf("server: %s", err)
			}
			err = tx.NewSeeder(torrent, peer)
			if err != nil {
				log.Panicf("server: %s", err)
			}
		}

	case leecher && left == 0:
		// Completed event from the peer was never received
		err := tx.RmLeecher(torrent, peer)
		if err != nil {
			log.Panicf("server: %s", err)
		}
		err = tx.NewSeeder(torrent, peer)
		if err != nil {
			log.Panicf("server: %s", err)
		}
	}

	// TODO compact, response, etc...

}

func (s *Server) validateAnnounceQuery(r *http.Request) (compact bool, numWant int, infohash, peerID, event, ip string, port, uploaded, downloaded, left uint64, err error) {
	pq, err := parseQuery(r.URL.RawQuery)
	if err != nil {
		return false, 0, "", "", "", "", 0, 0, 0, 0, err
	}

	compact = pq.Params["compact"] == "1"
	numWant = determineNumWant(s.conf.DefaultNumWant, pq)
	infohash, _ = pq.Params["info_hash"]
	peerID, _ = pq.Params["peer_id"]
	event, _ = pq.Params["event"]
	ip, _ = determineIP(r, pq)
	port, portErr := pq.getUint64("port")
	uploaded, uploadedErr := pq.getUint64("uploaded")
	downloaded, downloadedErr := pq.getUint64("downloaded")
	left, leftErr := pq.getUint64("left")

	if infohash == "" ||
		peerID == "" ||
		ip == "" ||
		portErr != nil ||
		uploadedErr != nil ||
		downloadedErr != nil ||
		leftErr != nil {
		return false, 0, "", "", "", "", 0, 0, 0, 0, errors.New("Malformed request")
	}
	return
}

func determineNumWant(fallback int, pq *parsedQuery) int {
	if numWantStr, exists := pq.Params["numWant"]; exists {
		numWant, err := strconv.Atoi(numWantStr)
		if err != nil {
			return fallback
		}
		return numWant
	} else {
		return fallback
	}
}

func determineIP(r *http.Request, pq *parsedQuery) (string, error) {
	if ip, ok := pq.Params["ip"]; ok {
		return ip, nil
	} else if ip, ok := pq.Params["ipv4"]; ok {
		return ip, nil
	} else if ips, ok := pq.Params["X-Real-Ip"]; ok && len(ips) > 0 {
		return string(ips[0]), nil
	} else {
		portIndex := len(r.RemoteAddr) - 1
		for ; portIndex >= 0; portIndex-- {
			if r.RemoteAddr[portIndex] == ':' {
				break
			}
		}
		if portIndex != -1 {
			return r.RemoteAddr[0:portIndex], nil
		} else {
			return "", errors.New("Failed to parse IP address")
		}
	}
}
