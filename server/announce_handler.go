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

	"github.com/chihaya/chihaya/config"
	"github.com/chihaya/chihaya/storage"
	"github.com/chihaya/chihaya/storage/backend"
)

// announce represents all of the data from an announce request.
type announce struct {
	Compact    bool
	Downloaded uint64
	Event      string
	IP         string
	Infohash   string
	Left       uint64
	NumWant    int
	Passkey    string
	PeerID     string
	Port       uint64
	Uploaded   uint64
}

// newAnnounce parses an HTTP request and generates an Announce.
func newAnnounce(r *http.Request, conf *config.Config) (*announce, error) {
	pq, err := parseQuery(r.URL.RawQuery)
	if err != nil {
		return nil, err
	}

	compact := pq.Params["compact"] == "1"
	downloaded, downloadedErr := pq.getUint64("downloaded")
	event, _ := pq.Params["event"]
	infohash, _ := pq.Params["info_hash"]
	ip, _ := requestedIP(r, pq)
	left, leftErr := pq.getUint64("left")
	numWant := requestedPeerCount(conf.DefaultNumWant, pq)
	passkey, _ := path.Split(r.URL.Path)
	peerID, _ := pq.Params["peer_id"]
	port, portErr := pq.getUint64("port")
	uploaded, uploadedErr := pq.getUint64("uploaded")

	if downloadedErr != nil ||
		infohash == "" ||
		leftErr != nil ||
		peerID == "" ||
		portErr != nil ||
		uploadedErr != nil ||
		ip == "" {
		return nil, errors.New("malformed request")
	}

	return &announce{
		Compact:    compact,
		Downloaded: downloaded,
		Event:      event,
		IP:         ip,
		Left:       left,
		NumWant:    numWant,
		Passkey:    passkey,
		PeerID:     peerID,
		Port:       port,
		Uploaded:   uploaded,
	}, nil
}

func (s Server) serveAnnounce(w http.ResponseWriter, r *http.Request) {
	// Parse the required data from a request
	ann, err := newAnnounce(r, s.conf)
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
	user, err := validateUser(conn, ann.Passkey)
	if err != nil {
		fail(err, w, r)
		return
	}

	// Check if the user's client is whitelisted
	whitelisted, err := conn.ClientWhitelisted(parsePeerID(ann.PeerID))
	if err != nil {
		log.Panicf("server: %s", err)
	}
	if !whitelisted {
		fail(errors.New("client is not approved"), w, r)
		return
	}

	// Find the specified torrent
	torrent, exists, err := conn.FindTorrent(ann.Infohash)
	if err != nil {
		log.Panicf("server: %s", err)
	}
	if !exists {
		fail(errors.New("torrent does not exist"), w, r)
		return
	}

	// If the torrent was pruned and the user is seeding, unprune it
	if !torrent.Active && ann.Left == 0 {
		err := conn.MarkActive(torrent)
		if err != nil {
			log.Panicf("server: %s", err)
		}
	}

	now := time.Now().Unix()
	// Create a new peer object from the request
	peer := &storage.Peer{
		ID:           ann.PeerID,
		UserID:       user.ID,
		TorrentID:    torrent.ID,
		IP:           ann.IP,
		Port:         ann.Port,
		Uploaded:     ann.Uploaded,
		Downloaded:   ann.Downloaded,
		Left:         ann.Left,
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
		if ann.Left == 0 {
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
		if ann.Left == 0 {
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
	case ann.Event == "stopped" || ann.Event == "paused":
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

	case ann.Event == "completed":
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

	case leecher && ann.Left == 0:
		// A leecher completed but the event was never received
		err := conn.LeecherFinished(torrent, peer)
		if err != nil {
			log.Panicf("server: %s", err)
		}
	}

	if ann.IP != peer.IP || ann.Port != peer.Port {
		peer.Port = ann.Port
		peer.IP = ann.IP
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

	if ann.NumWant > 0 && ann.Event != "stopped" && ann.Event != "paused" {
		writeBencoded(w, "peers")
		var peerCount, count int

		if ann.Compact {
			if ann.Left > 0 {
				peerCount = minInt(ann.NumWant, leechCount)
			} else {
				peerCount = minInt(ann.NumWant, leechCount+seedCount-1)
			}
			writeBencoded(w, strconv.Itoa(peerCount*6))
			writeBencoded(w, ":")
		} else {
			writeBencoded(w, "l")
		}

		if ann.Left > 0 {
			// If they're seeding, give them only leechers
			count += writeLeechers(w, user, torrent, ann.NumWant, ann.Compact)
		} else {
			// If they're leeching, prioritize giving them seeders
			count += writeSeeders(w, user, torrent, ann.NumWant, ann.Compact)
			count += writeLeechers(w, user, torrent, ann.NumWant-count, ann.Compact)
		}

		if ann.Compact && peerCount != count {
			log.Panicf("Calculated peer count (%d) != real count (%d)", peerCount, count)
		}

		if !ann.Compact {
			writeBencoded(w, "e")
		}
	}
	writeBencoded(w, "e")

	rawDeltaUp := peer.Uploaded - ann.Uploaded
	rawDeltaDown := peer.Downloaded - ann.Downloaded

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
	if ip, ok := pq.Params["ip"]; ok {
		return ip, nil
	}
	if ip, ok := pq.Params["ipv4"]; ok {
		return ip, nil
	}
	if xRealIPs, ok := pq.Params["X-Real-Ip"]; ok {
		return string(xRealIPs[0]), nil
	}

	portIndex := len(r.RemoteAddr) - 1
	for ; portIndex >= 0; portIndex-- {
		if r.RemoteAddr[portIndex] == ':' {
			break
		}
	}
	if portIndex != -1 {
		return r.RemoteAddr[0:portIndex], nil
	}
	return "", errors.New("failed to parse IP address")
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
