// Copyright 2015 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

// Package memory implements a storage driver for Chihaya by using a sharded
// map in memory that is not persistent.
package memory

import (
	"hash/fnv"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/chihaya/chihaya/config"
	"github.com/chihaya/chihaya/stats"
	"github.com/chihaya/chihaya/store"
	"github.com/chihaya/chihaya/tracker/models"
)

type torrents struct {
	torrents map[string]*models.Torrent
	sync.RWMutex
}

type driver struct{}

// Memory is a sharded map data store that is stored in memory and is not
// persistent.
type Memory struct {
	users  map[string]*models.User
	usersM sync.RWMutex

	shards []torrents
	size   int32

	clients  map[string]bool
	clientsM sync.RWMutex
}

func (d *driver) New(cfg *config.DriverConfig) (store.Conn, error) {
	numShards, ok := cfg.Params["torrent_map_shards"].(int)
	if !ok {
		return nil, config.ErrMissingRequiredParam
	}

	m := &Memory{
		users:   make(map[string]*models.User),
		shards:  make([]torrents, numShards),
		clients: make(map[string]bool),
	}

	for i := range m.shards {
		m.shards[i].torrents = make(map[string]*models.Torrent)
	}

	return m, nil
}

func (m *Memory) Close() error {
	return nil
}

// Len returns the total size of the torrents stored in memory.
func (m *Memory) Len() int {
	return int(atomic.LoadInt32(&m.size))
}

func (m *Memory) getShardIndex(infohash string) uint32 {
	idx := fnv.New32()
	idx.Write([]byte(infohash))
	return idx.Sum32() % uint32(len(m.shards))
}

func (m *Memory) getTorrentShard(infohash string, readonly bool) *torrents {
	shardindex := m.getShardIndex(infohash)
	if readonly {
		m.shards[shardindex].RLock()
	} else {
		m.shards[shardindex].Lock()
	}
	return &m.shards[shardindex]
}

func (m *Memory) TouchTorrent(infohash string) error {
	shard := m.getTorrentShard(infohash, false)
	defer shard.Unlock()

	torrent, exists := shard.torrents[infohash]
	if !exists {
		return models.ErrTorrentDNE
	}

	torrent.LastAction = time.Now().Unix()

	return nil
}

func (m *Memory) FindTorrent(infohash string) (*models.Torrent, error) {
	shard := m.getTorrentShard(infohash, true)
	defer shard.RUnlock()

	torrent, exists := shard.torrents[infohash]
	if !exists {
		return nil, models.ErrTorrentDNE
	}

	return &*torrent, nil
}

func (m *Memory) PutTorrent(torrent *models.Torrent) error {
	shard := m.getTorrentShard(torrent.Infohash, false)
	defer shard.Unlock()

	_, exists := shard.torrents[torrent.Infohash]
	if !exists {
		atomic.AddInt32(&m.size, 1)
	}
	shard.torrents[torrent.Infohash] = &*torrent

	return nil
}

func (m *Memory) DeleteTorrent(infohash string) error {
	shard := m.getTorrentShard(infohash, false)
	defer shard.Unlock()

	if _, exists := shard.torrents[infohash]; exists {
		atomic.AddInt32(&m.size, -1)
		delete(shard.torrents, infohash)
	}

	return nil
}

func (m *Memory) IncrementSnatches(infohash string) error {
	shard := m.getTorrentShard(infohash, false)
	defer shard.Unlock()

	torrent, exists := shard.torrents[infohash]
	if !exists {
		return models.ErrTorrentDNE
	}

	torrent.Snatches++

	return nil
}

func (m *Memory) PutLeecher(infohash string, p *models.Peer) error {
	shard := m.getTorrentShard(infohash, false)
	defer shard.Unlock()

	torrent, exists := shard.torrents[infohash]
	if !exists {
		return models.ErrTorrentDNE
	}

	torrent.Leechers.Put(*p)

	return nil
}

func (m *Memory) DeleteLeecher(infohash string, pk models.PeerKey) error {
	shard := m.getTorrentShard(infohash, false)
	defer shard.Unlock()

	torrent, exists := shard.torrents[infohash]
	if !exists {
		return models.ErrTorrentDNE
	}

	torrent.Leechers.Delete(pk)

	return nil
}

func (m *Memory) PutSeeder(infohash string, p *models.Peer) error {
	shard := m.getTorrentShard(infohash, false)
	defer shard.Unlock()

	torrent, exists := shard.torrents[infohash]
	if !exists {
		return models.ErrTorrentDNE
	}

	torrent.Seeders.Put(*p)

	return nil
}

func (m *Memory) DeleteSeeder(infohash string, pk models.PeerKey) error {
	shard := m.getTorrentShard(infohash, false)
	defer shard.Unlock()

	torrent, exists := shard.torrents[infohash]
	if !exists {
		return models.ErrTorrentDNE
	}

	torrent.Seeders.Delete(pk)

	return nil
}

func (m *Memory) PurgeInactiveTorrent(infohash string) error {
	shard := m.getTorrentShard(infohash, false)
	defer shard.Unlock()

	torrent, exists := shard.torrents[infohash]
	if !exists {
		return models.ErrTorrentDNE
	}

	if torrent.PeerCount() == 0 {
		delete(shard.torrents, infohash)
	}

	return nil
}

func (m *Memory) PurgeInactivePeers(purgeEmptyTorrents bool, before time.Time) error {
	unixtime := before.Unix()

	// Build a list of keys to process.
	index := 0
	maxkeys := m.Len()
	keys := make([]string, maxkeys)
	for i := range m.shards {
		shard := &m.shards[i]
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
		shard := m.getTorrentShard(infohash, false)
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
			m.PurgeInactiveTorrent(infohash)
			stats.RecordEvent(stats.ReapedTorrent)
		}
	}

	return nil
}

func (m *Memory) FindUser(passkey string) (*models.User, error) {
	m.usersM.RLock()
	defer m.usersM.RUnlock()

	user, exists := m.users[passkey]
	if !exists {
		return nil, models.ErrUserDNE
	}

	return &*user, nil
}

func (m *Memory) PutUser(user *models.User) error {
	m.usersM.Lock()
	defer m.usersM.Unlock()

	m.users[user.Passkey] = &*user

	return nil
}

func (m *Memory) DeleteUser(passkey string) error {
	m.usersM.Lock()
	defer m.usersM.Unlock()

	delete(m.users, passkey)

	return nil
}

func (m *Memory) FindClient(peerID string) error {
	m.clientsM.RLock()
	defer m.clientsM.RUnlock()

	_, exists := m.clients[peerID]
	if !exists {
		return models.ErrClientUnapproved
	}

	return nil
}

func (m *Memory) PutClient(peerID string) error {
	m.clientsM.Lock()
	defer m.clientsM.Unlock()

	m.clients[peerID] = true

	return nil
}

func (m *Memory) DeleteClient(peerID string) error {
	m.clientsM.Lock()
	defer m.clientsM.Unlock()

	delete(m.clients, peerID)

	return nil
}

func init() {
	store.Register("memory", &driver{})
}
