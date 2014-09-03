// Copyright 2014 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

package tracker

import (
	"runtime"
	"sync"
	"time"

	"github.com/chihaya/chihaya/stats"
	"github.com/chihaya/chihaya/tracker/models"
)

type Storage struct {
	users  map[string]*models.User
	usersM sync.RWMutex

	torrents  map[string]*models.Torrent
	torrentsM sync.RWMutex

	clients  map[string]bool
	clientsM sync.RWMutex
}

func NewStorage() *Storage {
	return &Storage{
		users:    make(map[string]*models.User),
		torrents: make(map[string]*models.Torrent),
		clients:  make(map[string]bool),
	}
}

func (s *Storage) TouchTorrent(infohash string) error {
	s.torrentsM.Lock()
	defer s.torrentsM.Unlock()

	torrent, exists := s.torrents[infohash]
	if !exists {
		return models.ErrTorrentDNE
	}

	torrent.LastAction = time.Now().Unix()

	return nil
}

func (s *Storage) FindTorrent(infohash string) (*models.Torrent, error) {
	s.torrentsM.RLock()
	defer s.torrentsM.RUnlock()

	torrent, exists := s.torrents[infohash]
	if !exists {
		return nil, models.ErrTorrentDNE
	}

	return &*torrent, nil
}

func (s *Storage) PutTorrent(torrent *models.Torrent) {
	s.torrentsM.Lock()
	defer s.torrentsM.Unlock()

	s.torrents[torrent.Infohash] = &*torrent
}

func (s *Storage) DeleteTorrent(infohash string) {
	s.torrentsM.Lock()
	defer s.torrentsM.Unlock()

	delete(s.torrents, infohash)
}

func (s *Storage) IncrementTorrentSnatches(infohash string) error {
	s.torrentsM.Lock()
	defer s.torrentsM.Unlock()

	torrent, exists := s.torrents[infohash]
	if !exists {
		return models.ErrTorrentDNE
	}

	torrent.Snatches++

	return nil
}

func (s *Storage) PutLeecher(infohash string, p *models.Peer) error {
	s.torrentsM.Lock()
	defer s.torrentsM.Unlock()

	torrent, exists := s.torrents[infohash]
	if !exists {
		return models.ErrTorrentDNE
	}

	torrent.Leechers.Put(*p)

	return nil
}

func (s *Storage) DeleteLeecher(infohash string, p *models.Peer) error {
	s.torrentsM.Lock()
	defer s.torrentsM.Unlock()

	torrent, exists := s.torrents[infohash]
	if !exists {
		return models.ErrTorrentDNE
	}

	torrent.Leechers.Delete(p.Key())

	return nil
}

func (s *Storage) PutSeeder(infohash string, p *models.Peer) error {
	s.torrentsM.Lock()
	defer s.torrentsM.Unlock()

	torrent, exists := s.torrents[infohash]
	if !exists {
		return models.ErrTorrentDNE
	}

	torrent.Seeders.Put(*p)

	return nil
}

func (s *Storage) DeleteSeeder(infohash string, p *models.Peer) error {
	s.torrentsM.Lock()
	defer s.torrentsM.Unlock()

	torrent, exists := s.torrents[infohash]
	if !exists {
		return models.ErrTorrentDNE
	}

	torrent.Seeders.Delete(p.Key())

	return nil
}

func (s *Storage) PurgeInactiveTorrent(infohash string) error {
	s.torrentsM.Lock()
	defer s.torrentsM.Unlock()

	torrent, exists := s.torrents[infohash]
	if !exists {
		return models.ErrTorrentDNE
	}

	if torrent.PeerCount() == 0 {
		delete(s.torrents, infohash)
	}

	return nil
}

func (s *Storage) PurgeInactivePeers(purgeEmptyTorrents bool, before time.Time) error {
	unixtime := before.Unix()

	// Build a list of keys to process.
	s.torrentsM.RLock()
	index := 0
	keys := make([]string, len(s.torrents))

	for infohash := range s.torrents {
		keys[index] = infohash
		index++
	}
	s.torrentsM.RUnlock()

	// Process the keys while allowing other goroutines to run.
	for _, infohash := range keys {
		runtime.Gosched()

		s.torrentsM.Lock()
		torrent := s.torrents[infohash]

		if torrent == nil {
			// The torrent has already been deleted since keys were computed.
			s.torrentsM.Unlock()
			continue
		}

		torrent.Seeders.Purge(unixtime)
		torrent.Leechers.Purge(unixtime)

		peers := torrent.PeerCount()
		s.torrentsM.Unlock()

		if purgeEmptyTorrents && peers == 0 {
			s.PurgeInactiveTorrent(infohash)
			stats.RecordEvent(stats.ReapedTorrent)
		}
	}

	return nil
}

func (s *Storage) FindUser(passkey string) (*models.User, error) {
	s.usersM.RLock()
	defer s.usersM.RUnlock()

	user, exists := s.users[passkey]
	if !exists {
		return nil, models.ErrUserDNE
	}

	return &*user, nil
}

func (s *Storage) PutUser(user *models.User) {
	s.usersM.Lock()
	defer s.usersM.Unlock()

	s.users[user.Passkey] = &*user
}

func (s *Storage) DeleteUser(passkey string) {
	s.usersM.Lock()
	defer s.usersM.Unlock()

	delete(s.users, passkey)
}

func (s *Storage) ClientApproved(peerID string) error {
	s.clientsM.RLock()
	defer s.clientsM.RUnlock()

	_, exists := s.clients[peerID]
	if !exists {
		return models.ErrClientUnapproved
	}

	return nil
}

func (s *Storage) PutClient(peerID string) {
	s.clientsM.Lock()
	defer s.clientsM.Unlock()

	s.clients[peerID] = true
}

func (s *Storage) DeleteClient(peerID string) {
	s.clientsM.Lock()
	defer s.clientsM.Unlock()

	delete(s.clients, peerID)
}
