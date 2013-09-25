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

	SeedersPrefix  = "seeders:"
	LeechersPrefix = "leechers:"
	TorrentPrefix  = "torrent:"
	UserPrefix     = "user:"
	PeerPrefix     = "peer:"
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
	return retTx, nil
}

type Tx struct {
	conf *config.DataStore
	done bool
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
	var user models.User
	convErrors := make([]error, 7)
	user.ID, convErrors[0] = strconv.ParseUint(userVals[0], 10, 64)
	user.Passkey = userVals[1]
	user.UpMultiplier, convErrors[2] = strconv.ParseFloat(userVals[2], 64)
	user.DownMultiplier, convErrors[3] = strconv.ParseFloat(userVals[3], 64)
	user.Slots, convErrors[4] = strconv.ParseInt(userVals[4], 10, 64)
	user.SlotsUsed, convErrors[5] = strconv.ParseInt(userVals[5], 10, 64)
	user.Snatches, convErrors[6] = strconv.ParseUint(userVals[6], 10, 64)

	for i := 0; i < 7; i++ {
		if convErrors[i] != nil {
			return nil, convErrors[i]
		}
	}
	return &user, nil
}

// This is a multiple action command, it's not internally atomic
func (tx *Tx) createTorrent(torrentVals []string) (*models.Torrent, error) {
	if len(torrentVals) != 7 {
		return nil, ErrCreateTorrent
	}
	var torrent models.Torrent
	convErrors := make([]error, 9)
	torrent.ID, convErrors[0] = strconv.ParseUint(torrentVals[0], 10, 64)
	torrent.Infohash = torrentVals[1]
	torrent.Active, convErrors[2] = strconv.ParseBool(torrentVals[2])
	torrent.Snatches, convErrors[3] = strconv.ParseUint(torrentVals[3], 10, 32)
	torrent.UpMultiplier, convErrors[4] = strconv.ParseFloat(torrentVals[4], 64)
	torrent.DownMultiplier, convErrors[5] = strconv.ParseFloat(torrentVals[5], 64)
	torrent.LastAction, convErrors[6] = strconv.ParseInt(torrentVals[6], 10, 64)
	torrent.Seeders, convErrors[7] = tx.getPeers(torrent.ID, SeedersPrefix)
	torrent.Leechers, convErrors[8] = tx.getPeers(torrent.ID, LeechersPrefix)

	for i := 0; i < 9; i++ {
		if convErrors[i] != nil {
			return nil, convErrors[i]
		}
	}
	return &torrent, nil
}

// The peer hashkey relies on the combination of peerID, userID, and torrentID being unique
func (tx *Tx) setPeer(peer *models.Peer) error {
	hashKey := tx.conf.Prefix + getPeerHashKey(peer)
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
	setKey := tx.conf.Prefix + getPeerSetKey(peerTypePrefix, peer)
	_, err := tx.Do("SREM", setKey, getPeerHashKey(peer))
	if err != nil {
		return err
	}
	hashKey := tx.conf.Prefix + getPeerHashKey(peer)
	_, err = tx.Do("DEL", hashKey)
	return nil
}

// This is a multiple action command, it's not internally atomic
func (tx *Tx) removePeers(torrentID uint64, peers map[string]models.Peer, peerTypePrefix string) error {
	for _, peer := range peers {
		hashKey := tx.conf.Prefix + getPeerHashKey(&peer)
		_, err := tx.Do("DEL", hashKey)
		if err != nil {
			return err
		}
		delete(peers, models.PeerMapKey(&peer))
	}
	// Will only delete the set if all the peer deletions were successful
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

func getPeerSetKey(typePrefix string, peer *models.Peer) string {
	return typePrefix + strconv.FormatUint(peer.TorrentID, 36)
}

// This is a multiple action command, it's not internally atomic
func (tx *Tx) addPeers(peers map[string]models.Peer, peerTypePrefix string) error {
	for _, peer := range peers {
		setKey := tx.conf.Prefix + getPeerSetKey(peerTypePrefix, &peer)
		_, err := tx.Do("SADD", setKey, getPeerHashKey(&peer))
		if err != nil {
			return err
		}
		tx.setPeer(&peer)
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
	return &models.Peer{ID: ID, UserID: UserID, TorrentID: TorrentID, IP: IP, Port: Port,
		Uploaded: Uploaded, Downloaded: Downloaded, Left: Left, LastAnnounce: LastAnnounce}, nil

}

// This is a multiple action command, it's not internally atomic
func (tx *Tx) getPeers(torrentID uint64, peerTypePrefix string) (peers map[string]models.Peer, err error) {
	peers = make(map[string]models.Peer)
	setKey := tx.conf.Prefix + peerTypePrefix + strconv.FormatUint(torrentID, 36)
	peerStrings, err := redis.Strings(tx.Do("SMEMBERS", setKey))
	if err != nil {
		return nil, err
	}
	// Keys map to peer objects stored in hashes
	for _, peerHashKey := range peerStrings {
		hashKey := tx.conf.Prefix + peerHashKey
		peerVals, err := redis.Strings(tx.Do("HVALS", hashKey))
		if err != nil {
			return nil, err
		}
		if len(peerVals) == 0 {
			continue
		}
		peer, err := createPeer(peerVals)
		if err != nil {
			return nil, err
		}
		peers[models.PeerMapKey(peer)] = *peer
	}
	return
}

// This is a multiple action command, it's not internally atomic
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

	err = tx.addPeers(t.Seeders, SeedersPrefix)
	if err != nil {
		return err
	}
	err = tx.addPeers(t.Leechers, LeechersPrefix)
	if err != nil {
		return err
	}
	return nil
}

// This is a multiple action command, it's not internally atomic
func (tx *Tx) RemoveTorrent(t *models.Torrent) error {
	hashkey := tx.conf.Prefix + TorrentPrefix + t.Infohash
	_, err := tx.Do("DEL", hashkey)
	if err != nil {
		return err
	}
	// Remove seeders and leechers as well
	err = tx.removePeers(t.ID, t.Seeders, SeedersPrefix)
	if err != nil {
		return err
	}
	err = tx.removePeers(t.ID, t.Leechers, LeechersPrefix)
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

// This is a multiple action command, it's not internally atomic
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

// This is a multiple action command, it's not internally atomic
func (tx *Tx) RecordSnatch(user *models.User, torrent *models.Torrent) error {

	torrentKey := tx.conf.Prefix + TorrentPrefix + torrent.Infohash
	snatchCount, err := redis.Int(tx.Do("HINCRBY", torrentKey, "snatches", 1))
	if err != nil {
		return err
	}
	torrent.Snatches = uint64(snatchCount)

	userKey := tx.conf.Prefix + UserPrefix + user.Passkey
	snatchCount, err = redis.Int(tx.Do("HINCRBY", userKey, "snatches", 1))
	if err != nil {
		return err
	}
	user.Snatches = uint64(snatchCount)
	return nil
}

func (tx *Tx) MarkActive(torrent *models.Torrent) error {
	hashkey := tx.conf.Prefix + TorrentPrefix + torrent.Infohash
	activeExists, err := redis.Int(tx.Do("HSET", hashkey, "active", true))
	if err != nil {
		return err
	}
	torrent.Active = true
	// HSET returns 1 if hash didn't exist before
	if activeExists == 1 {
		return ErrMarkActive
	}
	return nil
}

func (tx *Tx) MarkInactive(torrent *models.Torrent) error {
	hashkey := tx.conf.Prefix + TorrentPrefix + torrent.Infohash
	activeExists, err := redis.Int(tx.Do("HSET", hashkey, "active", false))
	if err != nil {
		return err
	}
	torrent.Active = false
	// HSET returns 1 if hash didn't exist before
	if activeExists == 1 {
		// Clean-up incomplete torrent
		_, err = tx.Do("DEL", hashkey)
		if err != nil {
			return err
		}
		return ErrMarkActive
	}
	return nil
}

// This is a multiple action command, it's not internally atomic
func (tx *Tx) AddLeecher(torrent *models.Torrent, peer *models.Peer) error {
	setKey := tx.conf.Prefix + LeechersPrefix + strconv.FormatUint(torrent.ID, 36)
	_, err := tx.Do("SADD", setKey, getPeerHashKey(peer))
	if err != nil {
		return err
	}
	err = tx.setPeer(peer)
	if err != nil {
		return err
	}
	if torrent.Leechers == nil {
		torrent.Leechers = make(map[string]models.Peer)
	}
	torrent.Leechers[models.PeerMapKey(peer)] = *peer
	return nil
}

// Setting assumes it is already a leecher, and just needs to be updated
// Maybe eventually there will be a move from leecher to seeder method
func (tx *Tx) SetLeecher(t *models.Torrent, p *models.Peer) error {
	err := tx.setPeer(p)
	if err != nil {
		return err
	}
	t.Leechers[models.PeerMapKey(p)] = *p
	return nil
}

func (tx *Tx) RemoveLeecher(t *models.Torrent, p *models.Peer) error {
	err := tx.removePeer(p, LeechersPrefix)
	if err != nil {
		return err
	}
	delete(t.Leechers, models.PeerMapKey(p))
	return nil
}

func (tx *Tx) LeecherFinished(torrent *models.Torrent, peer *models.Peer) error {
	torrentIdKey := strconv.FormatUint(torrent.ID, 36)
	seederSetKey := tx.conf.Prefix + SeedersPrefix + torrentIdKey
	leecherSetKey := tx.conf.Prefix + LeechersPrefix + torrentIdKey

	_, err := tx.Do("SMOVE", leecherSetKey, seederSetKey, getPeerHashKey(peer))
	if err != nil {
		return err
	}
	torrent.Seeders[models.PeerMapKey(peer)] = *peer
	delete(torrent.Leechers, models.PeerMapKey(peer))

	err = tx.setPeer(peer)
	return err
}

// This is a multiple action command, it's not internally atomic
func (tx *Tx) AddSeeder(torrent *models.Torrent, peer *models.Peer) error {
	setKey := tx.conf.Prefix + SeedersPrefix + strconv.FormatUint(torrent.ID, 36)
	_, err := tx.Do("SADD", setKey, getPeerHashKey(peer))
	if err != nil {
		return err
	}
	err = tx.setPeer(peer)
	if err != nil {
		return err
	}
	if torrent.Seeders == nil {
		torrent.Seeders = make(map[string]models.Peer)
	}
	torrent.Seeders[models.PeerMapKey(peer)] = *peer
	return nil
}

func (tx *Tx) SetSeeder(t *models.Torrent, p *models.Peer) error {
	err := tx.setPeer(p)
	if err != nil {
		return err
	}
	t.Seeders[models.PeerMapKey(p)] = *p
	return nil
}

func (tx *Tx) RemoveSeeder(t *models.Torrent, p *models.Peer) error {
	err := tx.removePeer(p, SeedersPrefix)
	if err != nil {
		return err
	}
	delete(t.Seeders, models.PeerMapKey(p))
	return nil
}

func (tx *Tx) IncrementSlots(u *models.User) error {
	hashkey := tx.conf.Prefix + UserPrefix + u.Passkey
	slotCount, err := redis.Int(tx.Do("HINCRBY", hashkey, "slots", 1))
	if err != nil {
		return err
	}
	u.Slots = int64(slotCount)
	return nil
}

func (tx *Tx) DecrementSlots(u *models.User) error {
	hashkey := tx.conf.Prefix + UserPrefix + u.Passkey
	slotCount, err := redis.Int(tx.Do("HINCRBY", hashkey, "slots", -1))
	if err != nil {
		return err
	}
	u.Slots = int64(slotCount)
	return nil
}

func init() {
	cache.Register("redis", &driver{})
}
