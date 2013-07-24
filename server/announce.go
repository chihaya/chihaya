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

func (s Server) serveAnnounce(w http.ResponseWriter, r *http.Request) {
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

	// Create a new peer object from the request
	peer := &storage.Peer{
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

	// Look for the user in in the pool of seeders and leechers
	_, seeder := torrent.Seeders[peerID]
	_, leecher := torrent.Leechers[peerID]

	switch {
	// Guarantee that no user is in both pools
	case seeder && leecher:
		if left == 0 {
			err := tx.RmLeecher(torrent, peer)
			if err != nil {
				log.Panicf("server: %s", err)
			}
			leecher = false
		} else {
			err := tx.RmSeeder(torrent, peer)
			if err != nil {
				log.Panicf("server: %s", err)
			}
			seeder = false
		}

	case seeder:
		// Update the peer with the stats from the request
		err := tx.SetSeeder(torrent, peer)
		if err != nil {
			log.Panicf("server: %s", err)
		}

	case leecher:
		// Update the peer with the stats from the request
		err := tx.SetLeecher(torrent, peer)
		if err != nil {
			log.Panicf("server: %s", err)
		}

	default:
		// Check the user's slots to see if they're allowed to leech
		if s.conf.Slots && user.Slots != -1 && left != 0 {
			if user.SlotsUsed >= user.Slots {
				fail(errors.New("You've run out of download slots."), w, r)
				return
			}
		}

		if left == 0 {
			// Save the peer as a new seeder
			err := tx.NewSeeder(torrent, peer)
			if err != nil {
				log.Panicf("server: %s", err)
			}
		} else {
			// Save the peer as a new leecher and increment the user's slots
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

	// Handle any events in the request
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
		// A leecher completed but the event was never received
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

func (s Server) validateAnnounceQuery(r *http.Request) (compact bool, numWant int, infohash, peerID, event, ip string, port, uploaded, downloaded, left uint64, err error) {
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
