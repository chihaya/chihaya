// Copyright 2013 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

package server

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"path"
	"strconv"
	"time"

	"github.com/pushrax/chihaya/storage"
)

func (s *Server) serveAnnounce(w http.ResponseWriter, r *http.Request) {
	passkey, _ := path.Split(r.URL.Path)
	user, err := s.FindUser(passkey)
	if err != nil {
		fail(err, w, r)
		return
	}

	aq, err := newAnnounceQuery(r)
	if err != nil {
		fail(errors.New("Malformed request"), w, r)
		return
	}

	peerID := aq.PeerID()
	ok, err := s.dataStore.ClientWhitelisted(peerID)
	if err != nil {
		log.Panicf("server: %s", err)
	}
	if !ok {
		fail(errors.New("Your client is not approved"), w, r)
		return
	}

	torrent, exists, err := s.dataStore.FindTorrent(aq.Infohash())
	if err != nil {
		log.Panicf("server: %s", err)
	}
	if !exists {
		fail(errors.New("This torrent does not exist"), w, r)
		return
	}

	tx, err := s.dataStore.Begin()
	if err != nil {
		log.Panicf("server: %s", err)
	}

	left := aq.Left()
	if torrent.Pruned && left == 0 {
		err := tx.Unprune(torrent.ID)
		if err != nil {
			log.Panicf("server: %s", err)
		}
	} else if torrent.Pruned {
		e := fmt.Errorf("This torrent does not exist (pruned: %t, left: %d)", torrent.Pruned, left)
		fail(e, w, r)
		return
	}

	_ = aq.NumWant(s.conf.DefaultNumWant)

	if s.conf.Slots && user.Slots != -1 && aq.Left() != 0 {
		if user.UsedSlots >= user.Slots {
			fail(errors.New("You've run out of download slots."), w, r)
			return
		}
	}

	_, isLeecher := torrent.Leechers[peerID]
	_, isSeeder := torrent.Seeders[peerID]

	event := aq.Event()
	completed := "completed" == event

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
	} else if completed {
		err := tx.Snatch(user.ID, torrent.ID)
		if err != nil {
			log.Panicf("server: %s", err)
		}
	}

}

// An AnnounceQuery is a parsedQuery that guarantees the existance
// of parameters required for torrent client announces.
type announceQuery struct {
	pq      *parsedQuery
	ip      string
	created int64
}

func newAnnounceQuery(r *http.Request) (*announceQuery, error) {
	pq, err := parseQuery(r.URL.RawQuery)
	if err != nil {
		return nil, err
	}

	infohash, _ := pq.Params["info_hash"]
	if infohash == "" {
		return nil, errors.New("infohash does not exist")
	}
	peerId, _ := pq.Params["peer_id"]
	if peerId == "" {
		return nil, errors.New("peerId does not exist")
	}
	_, err = pq.getUint64("port")
	if err != nil {
		return nil, errors.New("port does not exist")
	}
	_, err = pq.getUint64("uploaded")
	if err != nil {
		return nil, errors.New("uploaded does not exist")
	}
	_, err = pq.getUint64("downloaded")
	if err != nil {
		return nil, errors.New("downloaded does not exist")
	}
	_, err = pq.getUint64("left")
	if err != nil {
		return nil, errors.New("left does not exist")
	}

	aq := &announceQuery{
		pq:      pq,
		created: time.Now().Unix(),
	}
	aq.ip, err = aq.determineIP(r)
	if err != nil {
		return nil, err
	}
	return aq, nil
}

func (aq *announceQuery) Infohash() string {
	infohash, _ := aq.pq.Params["info_hash"]
	if infohash == "" {
		panic("announceQuery missing infohash")
	}
	return infohash
}

func (aq *announceQuery) PeerID() string {
	peerID, _ := aq.pq.Params["peer_id"]
	if peerID == "" {
		panic("announceQuery missing peer_id")
	}
	return peerID
}

func (aq *announceQuery) Port() uint64 {
	port, err := aq.pq.getUint64("port")
	if err != nil {
		panic("announceQuery missing port")
	}
	return port
}

func (aq *announceQuery) IP() string {
	return aq.ip
}

func (aq *announceQuery) Uploaded() uint64 {
	ul, err := aq.pq.getUint64("uploaded")
	if err != nil {
		panic("announceQuery missing uploaded")
	}
	return ul
}
func (aq *announceQuery) Downloaded() uint64 {
	dl, err := aq.pq.getUint64("downloaded")
	if err != nil {
		panic("announceQuery missing downloaded")
	}
	return dl
}
func (aq *announceQuery) Left() uint64 {
	left, err := aq.pq.getUint64("left")
	if err != nil {
		panic("announceQuery missing left")
	}
	return left
}

func (aq *announceQuery) Event() string {
	return aq.pq.Params["event"]
}

func (aq *announceQuery) determineIP(r *http.Request) (string, error) {
	if ip, ok := aq.pq.Params["ip"]; ok {
		return ip, nil
	} else if ip, ok := aq.pq.Params["ipv4"]; ok {
		return ip, nil
	} else if ips, ok := aq.pq.Params["X-Real-Ip"]; ok && len(ips) > 0 {
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

func (aq *announceQuery) NumWant(fallback int) int {
	if numWantStr, exists := aq.pq.Params["numWant"]; exists {
		numWant, err := strconv.Atoi(numWantStr)
		if err != nil {
			return fallback
		}
		return numWant
	} else {
		return fallback
	}
}

func (aq *announceQuery) Peer(uid, tid uint64) *storage.Peer {
	return &storage.Peer{
		ID:        aq.PeerID(),
		UserID:    uid,
		TorrentID: tid,

		IP:   aq.IP(),
		Port: aq.Port(),

		LastAnnounce: aq.created,
		Uploaded:     aq.Uploaded(),
		Downloaded:   aq.Downloaded(),
	}
}
