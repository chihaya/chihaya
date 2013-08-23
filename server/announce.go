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

	"github.com/pushrax/chihaya/models"
)

func (s Server) serveAnnounce(w http.ResponseWriter, r *http.Request) {
	// Parse the required parameters off of a query
	compact, numWant, infohash, peerID, event, ip, port, uploaded, downloaded, left, err := s.validateAnnounceQuery(r)
	if err != nil {
		fail(err, w, r)
		return
	}

	// Retry failed transactions a specified number of times
	for i := 0; i < s.conf.Cache.TxRetries; i++ {

		// Start a transaction
		tx, err := s.dbConnPool.Get()
		if err != nil {
			log.Panicf("server: %s", err)
		}

		// Validate the user's passkey
		passkey, _ := path.Split(r.URL.Path)
		user, err := validateUser(tx, passkey)
		if err != nil {
			fail(err, w, r)
			return
		}

		// Check if the user's client is whitelisted
		whitelisted, err := tx.ClientWhitelisted(peerID)
		if err != nil {
			log.Panicf("server: %s", err)
		}
		if !whitelisted {
			fail(errors.New("Your client is not approved"), w, r)
			return
		}

		// Find the specified torrent
		torrent, exists, err := tx.FindTorrent(infohash)
		if err != nil {
			log.Panicf("server: %s", err)
		}
		if !exists {
			fail(errors.New("This torrent does not exist"), w, r)
			return
		}

		// If the torrent was pruned and the user is seeding, unprune it
		if !torrent.Active && left == 0 {
			err := tx.MarkActive(torrent)
			if err != nil {
				log.Panicf("server: %s", err)
			}
		}

		// Create a new peer object from the request
		peer := &models.Peer{
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
				err := tx.RemoveLeecher(torrent, peer)
				if err != nil {
					log.Panicf("server: %s", err)
				}
				leecher = false
			} else {
				err := tx.RemoveSeeder(torrent, peer)
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
				err := tx.AddSeeder(torrent, peer)
				if err != nil {
					log.Panicf("server: %s", err)
				}
			} else {
				// Save the peer as a new leecher and increment the user's slots
				err := tx.IncrementSlots(user)
				if err != nil {
					log.Panicf("server: %s", err)
				}
				err = tx.AddLeecher(torrent, peer)
				if err != nil {
					log.Panicf("server: %s", err)
				}
			}
		}

		// Handle any events in the request
		switch {
		case event == "stopped" || event == "paused":
			if seeder {
				err := tx.RemoveSeeder(torrent, peer)
				if err != nil {
					log.Panicf("server: %s", err)
				}
			}
			if leecher {
				err := tx.RemoveLeecher(torrent, peer)
				if err != nil {
					log.Panicf("server: %s", err)
				}
				err = tx.DecrementSlots(user)
				if err != nil {
					log.Panicf("server: %s", err)
				}
			}

		case event == "completed":
			err := tx.RecordSnatch(user, torrent)
			if err != nil {
				log.Panicf("server: %s", err)
			}
			if leecher {
				err := tx.RemoveLeecher(torrent, peer)
				if err != nil {
					log.Panicf("server: %s", err)
				}
				err = tx.AddSeeder(torrent, peer)
				if err != nil {
					log.Panicf("server: %s", err)
				}
			}

		case leecher && left == 0:
			// A leecher completed but the event was never received
			err := tx.RemoveLeecher(torrent, peer)
			if err != nil {
				log.Panicf("server: %s", err)
			}
			err = tx.AddSeeder(torrent, peer)
			if err != nil {
				log.Panicf("server: %s", err)
			}
		}

		if ip != peer.IP || port != peer.Port {
			peer.Port = port
			peer.IP = ip
		}

		// If the transaction failed, retry
		err = tx.Commit()
		if err != nil {
			continue
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

		if numWant > 0 && event != "stopped" && event != "paused" {
			writeBencoded(w, "peers")
			var peerCount, count int

			if compact {
				if left > 0 {
					peerCount = minInt(numWant, leechCount)
				} else {
					peerCount = minInt(numWant, leechCount+seedCount-1)
				}
				writeBencoded(w, strconv.Itoa(peerCount*6))
				writeBencoded(w, ":")
			} else {
				writeBencoded(w, "l")
			}

			if left > 0 {
				// If they're seeding, give them only leechers
				writeLeechers(w, torrent, count, numWant, compact)
			} else {
				// If they're leeching, prioritize giving them seeders
				writeSeeders(w, torrent, count, numWant, compact)
				writeLeechers(w, torrent, count, numWant, compact)
			}

			if compact && peerCount != count {
				log.Panicf("Calculated peer count (%d) != real count (%d)", peerCount, count)
			}

			if !compact {
				writeBencoded(w, "e")
			}
		}
		writeBencoded(w, "e")

		return
	}
}

func (s Server) validateAnnounceQuery(r *http.Request) (compact bool, numWant int, infohash, peerID, event, ip string, port, uploaded, downloaded, left uint64, err error) {
	pq, err := parseQuery(r.URL.RawQuery)
	if err != nil {
		return false, 0, "", "", "", "", 0, 0, 0, 0, err
	}

	compact = pq.Params["compact"] == "1"
	numWant = requestedPeerCount(s.conf.DefaultNumWant, pq)
	infohash, _ = pq.Params["info_hash"]
	peerID, _ = pq.Params["peer_id"]
	event, _ = pq.Params["event"]
	ip, _ = requestedIP(r, pq)
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

func requestedPeerCount(fallback int, pq *parsedQuery) int {
	if numWantStr, exists := pq.Params["numWant"]; exists {
		numWant, err := strconv.Atoi(numWantStr)
		if err != nil {
			return fallback
		}
		return numWant
	}
	return fallback
}

func requestedIP(r *http.Request, pq *parsedQuery) (string, error) {
	ip, ok := pq.Params["ip"]
	ipv4, okv4 := pq.Params["ipv4"]
	xRealIPs, xRealOk := pq.Params["X-Real-Ip"]

	switch {
	case ok:
		return ip, nil

	case okv4:
		return ipv4, nil

	case xRealOk && len(xRealIPs) > 0:
		return string(xRealIPs[0]), nil

	default:
		portIndex := len(r.RemoteAddr) - 1
		for ; portIndex >= 0; portIndex-- {
			if r.RemoteAddr[portIndex] == ':' {
				break
			}
		}
		if portIndex != -1 {
			return r.RemoteAddr[0:portIndex], nil
		}
		return "", errors.New("Failed to parse IP address")
	}
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func writeSeeders(w http.ResponseWriter, t *models.Torrent, count, numWant int, compact bool) {
	for _, seed := range t.Seeders {
		if count >= numWant {
			break
		}
		if compact {
			// TODO writeBencoded(w, compactAddr)
		} else {
			writeBencoded(w, "d")
			writeBencoded(w, "ip")
			writeBencoded(w, seed.IP)
			writeBencoded(w, "peer id")
			writeBencoded(w, seed.ID)
			writeBencoded(w, "port")
			writeBencoded(w, seed.Port)
			writeBencoded(w, "e")
		}
		count++
	}
}

func writeLeechers(w http.ResponseWriter, t *models.Torrent, count, numWant int, compact bool) {
	for _, leech := range t.Leechers {
		if count >= numWant {
			break
		}
		if compact {
			// TODO writeBencoded(w, compactAddr)
		} else {
			writeBencoded(w, "d")
			writeBencoded(w, "ip")
			writeBencoded(w, leech.IP)
			writeBencoded(w, "peer id")
			writeBencoded(w, leech.ID)
			writeBencoded(w, "port")
			writeBencoded(w, leech.Port)
			writeBencoded(w, "e")
		}
		count++
	}
}
