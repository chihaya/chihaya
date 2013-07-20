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
)

func (s *Server) serveAnnounce(w http.ResponseWriter, r *http.Request) {
	compact, numWant, infohash, peerID, event, ip, port, uploaded, downloaded, left, err := s.validateAnnounceQuery(r)
	if err != nil {
		fail(err, w, r)
		return
	}

	passkey, _ := path.Split(r.URL.Path)
	user, err := s.FindUser(passkey)
	if err != nil {
		fail(err, w, r)
		return
	}

	whitelisted, err := s.dataStore.ClientWhitelisted(peerID)
	if err != nil {
		log.Panicf("server: %s", err)
	}
	if !whitelisted {
		fail(errors.New("Your client is not approved"), w, r)
		return
	}

	torrent, exists, err := s.dataStore.FindTorrent(infohash)
	if err != nil {
		log.Panicf("server: %s", err)
	}
	if !exists {
		fail(errors.New("This torrent does not exist"), w, r)
		return
	}

	if s.conf.Slots && user.Slots != -1 && left != 0 {
		if user.UsedSlots >= user.Slots {
			fail(errors.New("You've run out of download slots."), w, r)
			return
		}
	}

	tx, err := s.dataStore.Begin()
	if err != nil {
		log.Panicf("server: %s", err)
	}

	if torrent.Pruned && left == 0 {
		err := tx.Unprune(torrent.ID)
		if err != nil {
			log.Panicf("server: %s", err)
		}
	}

	_, isLeecher := torrent.Leechers[peerID]
	_, isSeeder := torrent.Seeders[peerID]
	if event == "stopped" || event == "paused" {
		if left == 0 {
			err := tx.RmSeeder(torrent.ID, peerID)
			if err != nil {
				log.Panicf("server: %s", err)
			}
		} else {
			err := tx.RmLeecher(torrent.ID, peerID)
			if err != nil {
				log.Panicf("server: %s", err)
			}
			err = tx.DecrementSlots(user.ID)
			if err != nil {
				log.Panicf("server: %s", err)
			}
		}
	} else if event == "completed" {
		err := tx.Snatch(user.ID, torrent.ID)
		if err != nil {
			log.Panicf("server: %s", err)
		}
	}
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
