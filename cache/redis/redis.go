// Copyright 2013 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

// Package redis implements the storage interface for a BitTorrent tracker.
//
// The client whitelist is represented as a set with the key name "whitelist"
// with an optional prefix. Torrents and users are represented as hashes.
// Torrents' keys are named "torrent:<infohash>" with an optional prefix.
// Users' keys are named "user:<passkey>" with an optional prefix. The
// seeders and leechers attributes of torrent hashes are strings that represent
// the key for those hashes within redis. This is done because redis cannot
// nest their hash data type.
package redis

import (
	"errors"
	"strconv"
	"time"

	"github.com/garyburd/redigo/redis"

	"github.com/pushrax/chihaya/cache"
	"github.com/pushrax/chihaya/config"
	"github.com/pushrax/chihaya/models"
)

var (
	ErrCreateUser    = errors.New("redis: Incorrect reply length for user")
	ErrCreateTorrent = errors.New("redis: Incorrect reply length for torrent")
	ErrCreatePeer    = errors.New("redis: Incorrect reply length for peer")
	ErrMarkActive    = errors.New("redis: Torrent doesn't exist")

	SeederPrefix  = "seeders:"
	LeecherPrefix = "leechers:"
	TorrentPrefix = "torrent:"
	UserPrefix    = "user:"
	PeerPrefix    = "peer:"
)

type driver struct{}

func (d *driver) New(conf *config.DataStore) cache.Pool {
	return &Pool{
		conf: conf,
		pool: redis.Pool{
			MaxIdle:      conf.MaxIdleConns,
			IdleTimeout:  conf.IdleTimeout.Duration,
			Dial:         makeDialFunc(conf),
			TestOnBorrow: testOnBorrow,
		},
	}
}

func makeDialFunc(conf *config.DataStore) func() (redis.Conn, error) {
	return func() (conn redis.Conn, err error) {
		conn, err = redis.Dial(conf.Network, conf.Host+":"+conf.Port)
		if err != nil {
			return nil, err
		}
		return conn, nil
	}
}

func testOnBorrow(c redis.Conn, t time.Time) error {
	_, err := c.Do("PING")
	return err
}

type Pool struct {
	conf *config.DataStore
	pool redis.Pool
}

func (p *Pool) Close() error {
	return p.pool.Close()
}

func (p *Pool) Get() (cache.Tx, error) {
	retTx := &Tx{
		conf: p.conf,
		done: false,
		Conn: p.pool.Get(),
	}
	// Test valid connection before returning
	_, err := retTx.Do("PING")

	return retTx, err
}

type Tx struct {
	conf  *config.DataStore
	done  bool
	multi bool
	redis.Conn
}

func (tx *Tx) close() {
	if tx.done {
		panic("redis: transaction closed twice")
	}
	tx.done = true
	tx.Conn.Close()
}

func createUser(userVals []string) (*models.User, error) {
	if len(userVals) != 7 {
		return nil, ErrCreateUser
	}
	// This could be a loop+switch
	ID, err := strconv.ParseUint(userVals[0], 10, 64)
	if err != nil {
		return nil, err
	}
	Passkey := userVals[1]
	UpMultiplier, err := strconv.ParseFloat(userVals[2], 64)
	if err != nil {
		return nil, err
	}
	DownMultiplier, err := strconv.ParseFloat(userVals[3], 64)
	if err != nil {
		return nil, err
	}
	Slots, err := strconv.ParseInt(userVals[4], 10, 64)
	if err != nil {
		return nil, err
	}
	SlotsUsed, err := strconv.ParseInt(userVals[5], 10, 64)
	if err != nil {
		return nil, err
	}
	Snatches, err := strconv.ParseUint(userVals[6], 10, 64)
	if err != nil {
		return nil, err
	}
	return &models.User{ID, Passkey, UpMultiplier, DownMultiplier, Slots, SlotsUsed, uint(Snatches)}, nil
}

func (tx *Tx) createTorrent(torrentVals []string) (*models.Torrent, error) {
	if len(torrentVals) != 7 {
		return nil, ErrCreateTorrent
	}
	ID, err := strconv.ParseUint(torrentVals[0], 10, 64)
	if err != nil {
		return nil, err
	}
	Infohash := torrentVals[1]
	Active, err := strconv.ParseBool(torrentVals[2])
	if err != nil {
		return nil, err
	}
	Snatches, err := strconv.ParseUint(torrentVals[3], 10, 32)
	if err != nil {
		return nil, err
	}
	UpMultiplier, err := strconv.ParseFloat(torrentVals[4], 64)
	if err != nil {
		return nil, err
	}
	DownMultiplier, err := strconv.ParseFloat(torrentVals[5], 64)
	if err != nil {
		return nil, err
	}
	LastAction, err := strconv.ParseInt(torrentVals[6], 10, 64)
	if err != nil {
		return nil, err
	}
	seeders, err := tx.getPeers(ID, SeederPrefix)
	if err != nil {
		return nil, err
	}
	leechers, err := tx.getPeers(ID, LeecherPrefix)
	if err != nil {
		return nil, err
	}

	return &models.Torrent{ID, Infohash, Active, seeders, leechers, uint(Snatches), UpMultiplier, DownMultiplier, LastAction}, nil
}

// hashkey relies on combination of peerID, userID, and torrentID being unique
func (tx *Tx) setPeer(peer *models.Peer, peerTypePrefix string) error {
	hashKey := tx.conf.Prefix + peerTypePrefix + getPeerHashKey(peer)
	_, err := tx.Do("HMSET", hashKey,
		"id", peer.ID,
		"user_id", peer.UserID,
		"torrent_id", peer.TorrentID,
		"ip", peer.IP,
		"port", peer.Port,
		"uploaded", peer.Uploaded,
		"downloaded", peer.Downloaded,
		"left", peer.Left,
		"last_announce", peer.LastAnnounce)

	if err != nil {
		return err
	}
	return nil
}

// Will not return an error if the peer doesn't exist
func (tx *Tx) removePeer(peer *models.Peer, peerTypePrefix string) error {
	setKey := tx.conf.Prefix + peerTypePrefix + strconv.FormatUint(peer.TorrentID, 36)
	_, err := tx.Do("SREM", setKey, *peer)
	if err != nil {
		return err
	}
	return nil
}

func (tx *Tx) removePeers(torrentID uint64, peers map[string]models.Peer, peerTypePrefix string) error {
	for _, peer := range peers {
		hashKey := tx.conf.Prefix + peerTypePrefix + getPeerHashKey(&peer)
		_, err := tx.Do("DEL", hashKey)
		if err != nil {
			return err
		}
		delete(peers, peer.ID)
	}
	// Only delete the set if all the peer deletions were successful
	setKey := tx.conf.Prefix + peerTypePrefix + strconv.FormatUint(torrentID, 36)
	_, err := tx.Do("DEL", setKey)
	if err != nil {
		return err
	}

	return nil
}

func getPeerHashKey(peer *models.Peer) string {
	return peer.ID + ":" + strconv.FormatUint(peer.UserID, 36) + ":" + strconv.FormatUint(peer.TorrentID, 36)
}

func (tx *Tx) addPeers(peers map[string]models.Peer, peerTypePrefix string) error {
	for _, peer := range peers {
		setKey := tx.conf.Prefix + peerTypePrefix + strconv.FormatUint(peer.TorrentID, 36)
		_, err := tx.Do("SADD", setKey, getPeerHashKey(&peer))
		if err != nil {
			return err
		}
		tx.setPeer(&peer, peerTypePrefix)
	}
	return nil
}

func createPeer(peerVals []string) (*models.Peer, error) {
	if len(peerVals) != 9 {
		return nil, ErrCreatePeer
	}
	ID := peerVals[0]
	UserID, err := strconv.ParseUint(peerVals[1], 10, 64)
	if err != nil {
		return nil, err
	}
	TorrentID, err := strconv.ParseUint(peerVals[2], 10, 64)
	if err != nil {
		return nil, err
	}
	IP := peerVals[3]

	Port, err := strconv.ParseUint(peerVals[4], 10, 64)
	if err != nil {
		return nil, err
	}
	Uploaded, err := strconv.ParseUint(peerVals[5], 10, 64)
	if err != nil {
		return nil, err
	}
	Downloaded, err := strconv.ParseUint(peerVals[6], 10, 64)
	if err != nil {
		return nil, err
	}
	Left, err := strconv.ParseUint(peerVals[7], 10, 64)
	if err != nil {
		return nil, err
	}
	LastAnnounce, err := strconv.ParseInt(peerVals[8], 10, 64)
	if err != nil {
		return nil, err
	}
	return &models.Peer{ID, UserID, TorrentID, IP, Port, Uploaded, Downloaded, Left, LastAnnounce}, nil

}

func (tx *Tx) getPeers(torrentID uint64, peerTypePrefix string) (peers map[string]models.Peer, err error) {
	peers = make(map[string]models.Peer)
	setKey := tx.conf.Prefix + peerTypePrefix + strconv.FormatUint(torrentID, 36)
	peerStrings, err := redis.Strings(tx.Do("SMEMBERS", setKey))
	if err != nil {
		return peers, err
	}
	// Keys map to peer objects stored in hashes
	for _, peerHashKey := range peerStrings {
		hashKey := tx.conf.Prefix + peerTypePrefix + peerHashKey
		peerVals, err := redis.Strings(tx.Do("HVALS", hashKey))
		if err != nil {
			return peers, err
		}
		peer, err := createPeer(peerVals)
		if err != nil {
			return nil, err
		}
		peers[peer.ID] = *peer
	}
	return
}

func (tx *Tx) AddTorrent(t *models.Torrent) error {
	hashkey := tx.conf.Prefix + TorrentPrefix + t.Infohash
	_, err := tx.Do("HMSET", hashkey,
		"id", t.ID,
		"infohash", t.Infohash,
		"active", t.Active,
		"snatches", t.Snatches,
		"up_multiplier", t.UpMultiplier,
		"down_multiplier", t.DownMultiplier,
		"last_action", t.LastAction)
	if err != nil {
		return err
	}

	tx.addPeers(t.Seeders, SeederPrefix)
	tx.addPeers(t.Leechers, LeecherPrefix)

	return nil
}

func (tx *Tx) RemoveTorrent(t *models.Torrent) error {
	hashkey := tx.conf.Prefix + TorrentPrefix + t.Infohash
	_, err := tx.Do("DEL", hashkey)
	if err != nil {
		return err
	}
	// Remove seeders and leechers as well
	err = tx.removePeers(t.ID, t.Seeders, SeederPrefix)
	if err != nil {
		return err
	}
	err = tx.removePeers(t.ID, t.Leechers, LeecherPrefix)
	if err != nil {
		return err
	}

	return nil
}

func (tx *Tx) AddUser(u *models.User) error {
	hashkey := tx.conf.Prefix + UserPrefix + u.Passkey
	_, err := tx.Do("HMSET", hashkey,
		"id", u.ID,
		"passkey", u.Passkey,
		"up_multiplier", u.UpMultiplier,
		"down_multiplier", u.DownMultiplier,
		"slots", u.Slots,
		"slots_used", u.SlotsUsed,
		"snatches", u.Snatches)
	if err != nil {
		return err
	}
	return nil
}

func (tx *Tx) RemoveUser(u *models.User) error {
	hashkey := tx.conf.Prefix + UserPrefix + u.Passkey
	_, err := tx.Do("DEL", hashkey)
	if err != nil {
		return err
	}
	return nil
}

func (tx *Tx) FindUser(passkey string) (*models.User, bool, error) {
	hashkey := tx.conf.Prefix + UserPrefix + passkey
	userStrings, err := redis.Strings(tx.Do("HVALS", hashkey))
	if err != nil {
		return nil, false, err
	} else if len(userStrings) == 0 {
		return nil, false, nil
	}
	foundUser, err := createUser(userStrings)
	if err != nil {
		return nil, false, err
	}
	return foundUser, true, nil
}

// This is a mulple action command, it's not internally atomic
func (tx *Tx) FindTorrent(infohash string) (*models.Torrent, bool, error) {
	hashkey := tx.conf.Prefix + TorrentPrefix + infohash
	torrentStrings, err := redis.Strings(tx.Do("HVALS", hashkey))
	if err != nil {
		return nil, false, err
	} else if len(torrentStrings) == 0 {
		return nil, false, nil
	}

	foundTorrent, err := tx.createTorrent(torrentStrings)
	if err != nil {
		return nil, false, err
	}
	return foundTorrent, true, nil
}

func (tx *Tx) ClientWhitelisted(peerID string) (exists bool, err error) {
	key := tx.conf.Prefix + "whitelist"
	return redis.Bool(tx.Do("SISMEMBER", key, peerID))
}

func (tx *Tx) WhitelistClient(peerID string) error {
	key := tx.conf.Prefix + "whitelist"
	_, err := tx.Do("SADD", key, peerID)
	return err
}

func (tx *Tx) UnWhitelistClient(peerID string) error {
	key := tx.conf.Prefix + "whitelist"
	_, err := tx.Do("SREM", key, peerID)
	return err
}

// This is a mulple action command, it's not internally atomic
func (tx *Tx) RecordSnatch(user *models.User, torrent *models.Torrent) error {

	torrentKey := tx.conf.Prefix + TorrentPrefix + torrent.Infohash
	snatchCount, err := redis.Int(tx.Do("HINCRBY", torrentKey, 1))
	if err != nil {
		return err
	}
	torrent.Snatches = uint(snatchCount)

	userKey := tx.conf.Prefix + TorrentPrefix + torrent.Infohash
	snatchCount, err = redis.Int(tx.Do("HINCRBY", userKey, 1))
	if err != nil {
		return err
	}
	user.Snatches = uint(snatchCount)
	return nil
}

func (tx *Tx) MarkActive(torrent *models.Torrent) error {
	hashkey := tx.conf.Prefix + TorrentPrefix + torrent.Infohash
	activeExists, err := redis.Int(tx.Do("HSET", hashkey, true))
	if err != nil {
		return err
	}
	// HSET returns 1 if hash didn't exist before
	if activeExists == 1 {
		return ErrMarkActive
	}
	return nil
}

func (tx *Tx) AddLeecher(torrent *models.Torrent, peer *models.Peer) error {
	setKey := tx.conf.Prefix + LeecherPrefix + strconv.FormatUint(torrent.ID, 36)
	_, err := tx.Do("SADD", setKey, getPeerHashKey(peer))
	if err != nil {
		return err
	}
	err = tx.setPeer(peer, LeecherPrefix)
	if err != nil {
		return err
	}
	if torrent.Leechers == nil {
		torrent.Leechers = make(map[string]models.Peer)
	}
	torrent.Leechers[peer.ID] = *peer
	return nil
}

// Setting assumes it is already a leecher, and just needs to be updated
// Maybe eventually there will be a move from leecher to seeder method
func (tx *Tx) SetLeecher(t *models.Torrent, p *models.Peer) error {
	return tx.setPeer(p, LeecherPrefix)
}

func (tx *Tx) RemoveLeecher(t *models.Torrent, p *models.Peer) error {
	err := tx.removePeer(p, LeecherPrefix)
	if err != nil {
		return err
	}
	delete(t.Leechers, p.ID)
	return nil
}

func (tx *Tx) AddSeeder(torrent *models.Torrent, peer *models.Peer) error {
	setKey := tx.conf.Prefix + SeederPrefix + strconv.FormatUint(torrent.ID, 36)
	_, err := tx.Do("SADD", setKey, getPeerHashKey(peer))
	if err != nil {
		return err
	}
	err = tx.setPeer(peer, SeederPrefix)
	if err != nil {
		return err
	}
	if torrent.Seeders == nil {
		torrent.Seeders = make(map[string]models.Peer)
	}
	torrent.Seeders[peer.ID] = *peer
	return nil
}

func (tx *Tx) SetSeeder(t *models.Torrent, p *models.Peer) error {
	return tx.setPeer(p, SeederPrefix)
}

func (tx *Tx) RemoveSeeder(t *models.Torrent, p *models.Peer) error {
	err := tx.removePeer(p, SeederPrefix)
	if err != nil {
		return err
	}
	delete(t.Seeders, p.ID)
	return nil
}

func (tx *Tx) IncrementSlots(u *models.User) error {
	hashkey := tx.conf.Prefix + UserPrefix + u.Passkey
	slotCount, err := redis.Int(tx.Do("HINCRBY", hashkey, 1))
	if err != nil {
		return err
	}
	u.Slots = int64(slotCount)
	return nil
}

func (tx *Tx) DecrementSlots(u *models.User) error {
	hashkey := tx.conf.Prefix + UserPrefix + u.Passkey
	slotCount, err := redis.Int(tx.Do("HINCRBY", hashkey, -1))
	if err != nil {
		return err
	}
	u.Slots = int64(slotCount)
	return nil
}

func init() {
	cache.Register("redis", &driver{})
}
