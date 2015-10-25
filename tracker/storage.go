// Copyright 2015 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

package tracker

import (
	"hash/fnv"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/chihaya/chihaya/config"
	"github.com/chihaya/chihaya/stats"
	"github.com/chihaya/chihaya/tracker/models"
)

type Torrents struct {
	torrents map[string]*models.Torrent
	sync.RWMutex
}

type Storage struct {
	users  map[string]*models.User
	usersM sync.RWMutex

	shards []Torrents
	size   int32

	clients  map[string]bool
	clientsM sync.RWMutex
}

func NewStorage(cfg *config.Config) *Storage {
	s := &Storage{
		users:   make(map[string]*models.User),
		shards:  make([]Torrents, cfg.TorrentMapShards),
		clients: make(map[string]bool),
	}
	for i := range s.shards {
		s.shards[i].torrents = make(map[string]*models.Torrent)
	}
	return s
}

func (s *Storage) Len() int {
	return int(atomic.LoadInt32(&s.size))
}

func (s *Storage) getShardIndex(infohash string) uint32 {
	idx := fnv.New32()
	idx.Write([]byte(infohash))
	return idx.Sum32() % uint32(len(s.shards))
}

func (s *Storage) getTorrentShard(infohash string, readonly bool) *Torrents {
	shardindex := s.getShardIndex(infohash)
	if readonly {
		s.shards[shardindex].RLock()
	} else {
		s.shards[shardindex].Lock()
	}
	return &s.shards[shardindex]
}

func (s *Storage) TouchTorrent(infohash string) error {
	shard := s.getTorrentShard(infohash, false)
	defer shard.Unlock()

	torrent, exists := shard.torrents[infohash]
	if !exists {
		return models.ErrTorrentDNE
	}

	torrent.LastAction = time.Now().Unix()

	return nil
}

func (s *Storage) FindTorrent(infohash string) (*models.Torrent, error) {
	shard := s.getTorrentShard(infohash, true)
	defer shard.RUnlock()

	torrent, exists := shard.torrents[infohash]
	if !exists {
		return nil, models.ErrTorrentDNE
	}

	return &*torrent, nil
}

func (s *Storage) PutTorrent(torrent *models.Torrent) {
	shard := s.getTorrentShard(torrent.Infohash, false)
	defer shard.Unlock()

	_, exists := shard.torrents[torrent.Infohash]
	if !exists {
		atomic.AddInt32(&s.size, 1)
	}
	shard.torrents[torrent.Infohash] = &*torrent
}

func (s *Storage) DeleteTorrent(infohash string) {
	shard := s.getTorrentShard(infohash, false)
	defer shard.Unlock()

	if _, exists := shard.torrents[infohash]; exists {
		atomic.AddInt32(&s.size, -1)
		delete(shard.torrents, infohash)
	}
}

func (s *Storage) IncrementTorrentSnatches(infohash string) error {
	shard := s.getTorrentShard(infohash, false)
	defer shard.Unlock()

	torrent, exists := shard.torrents[infohash]
	if !exists {
		return models.ErrTorrentDNE
	}

	torrent.Snatches++

	return nil
}

func (s *Storage) PutLeecher(infohash string, p *models.Peer) error {
	shard := s.getTorrentShard(infohash, false)
	defer shard.Unlock()

	torrent, exists := shard.torrents[infohash]
	if !exists {
		return models.ErrTorrentDNE
	}

	torrent.Leechers.Put(*p)

	return nil
}

func (s *Storage) DeleteLeecher(infohash string, p *models.Peer) error {
	shard := s.getTorrentShard(infohash, false)
	defer shard.Unlock()

	torrent, exists := shard.torrents[infohash]
	if !exists {
		return models.ErrTorrentDNE
	}

	torrent.Leechers.Delete(p.Key())

	return nil
}

func (s *Storage) PutSeeder(infohash string, p *models.Peer) error {
	shard := s.getTorrentShard(infohash, false)
	defer shard.Unlock()

	torrent, exists := shard.torrents[infohash]
	if !exists {
		return models.ErrTorrentDNE
	}

	torrent.Seeders.Put(*p)

	return nil
}

func (s *Storage) DeleteSeeder(infohash string, p *models.Peer) error {
	shard := s.getTorrentShard(infohash, false)
	defer shard.Unlock()

	torrent, exists := shard.torrents[infohash]
	if !exists {
		return models.ErrTorrentDNE
	}

	torrent.Seeders.Delete(p.Key())

	return nil
}

func (s *Storage) PurgeInactiveTorrent(infohash string) error {
	shard := s.getTorrentShard(infohash, false)
	defer shard.Unlock()

	torrent, exists := shard.torrents[infohash]
	if !exists {
		return models.ErrTorrentDNE
	}

	if torrent.PeerCount() == 0 {
		atomic.AddInt32(&s.size, -1)
		delete(shard.torrents, infohash)
	}

	return nil
}

func (s *Storage) PurgeInactivePeers(purgeEmptyTorrents bool, before time.Time) error {
	unixtime := before.Unix()

	// Build a list of keys to process.
	index := 0
	maxkeys := s.Len()
	keys := make([]string, maxkeys)
	for i := range s.shards {
		shard := &s.shards[i]
		shard.RLock()
		for infohash := range shard.torrents {
			keys[index] = infohash
			index++
			if index >= maxkeys {
				break
			}
		}
		shard.RUnlock()
		if index >= maxkeys {
			break
		}
	}

	// Process the keys while allowing other goroutines to run.
	for _, infohash := range keys {
		runtime.Gosched()
		shard := s.getTorrentShard(infohash, false)
		torrent := shard.torrents[infohash]

		if torrent == nil {
			// The torrent has already been deleted since keys were computed.
			shard.Unlock()
			continue
		}

		torrent.Seeders.Purge(unixtime)
		torrent.Leechers.Purge(unixtime)

		peers := torrent.PeerCount()
		shard.Unlock()

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
