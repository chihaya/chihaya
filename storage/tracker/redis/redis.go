// Copyright 2013 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

// Package redis implements the storage interface for a BitTorrent tracker.
//
// This interface is configured by a config.DataStore.
// To get a handle to this interface, call New on the initialized driver and
// then Get() on returned the tracker.Pool.
//
// Torrents, Users, and Peers are all stored in Redis hash types. All Redis
// keys can have an optional prefix specified during configuration.
// The relationship between Torrents and Peers is a Redis set that holds
// the peers' keys. There are two sets per torrent, one for seeders and
// one for leechers. The Redis sets are keyed by type and the torrent's ID.
//
// The whitelist is a Redis set with the key "whitelist" that holds client IDs.
// Operations on the whitelist do not parse the client ID from a peer ID.
//
// Some functions in this interface are not atomic. The data being modified may
// change while the function is executing. This will not cause the function to
// return an error; instead the function will complete and return valid, stale
// data.
package redis

import (
  "errors"
  "strconv"
  "time"

  "github.com/garyburd/redigo/redis"

  "github.com/pushrax/chihaya/config"
  "github.com/pushrax/chihaya/storage"
  "github.com/pushrax/chihaya/storage/tracker"
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

// New creates and returns a tracker.Pool.
func (d *driver) New(conf *config.DataStore) tracker.Pool {
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

// makeDialFunc configures and returns a new redis.Dial struct using the specified configuration.
func makeDialFunc(conf *config.DataStore) func() (redis.Conn, error) {
  return func() (conn redis.Conn, err error) {
    conn, err = redis.Dial(conf.Network, conf.Host+":"+conf.Port)
    if err != nil {
      return nil, err
    }
    return conn, nil
  }
}

// testOnBorrow pings the Redis instance
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

func (p *Pool) Get() (tracker.Conn, error) {
  newConn := &Conn{
    conf: p.conf,
    done: false,
    Conn: p.pool.Get(),
  }
  return newConn, nil
}

type Conn struct {
  conf *config.DataStore
  done bool
  redis.Conn
}

func (conn *Conn) close() {
  if conn.done {
    panic("redis: connection closed twice")
  }
  conn.done = true
  conn.Conn.Close()
}

// createUser takes a string slice of length 14 and returns a pointer to a new
// storage.User or an error.
// This function is used to create a user from a Redis hash response(HGETALL).
// The order of strings the in the slice must follow the pattern:
//  [<field name>, <field value>, <field name>, <field value>, ...]
// If the field value string cannot be converted to the correct type,
// createUser will return a nil user and the conversion error.
func createUser(userVals []string) (*storage.User, error) {
  if len(userVals) != 14 {
    return nil, ErrCreateUser
  }
  var user storage.User
  var err error
  for index, userString := range userVals {
    switch userString {
    case "id":
      user.ID, err = strconv.ParseUint(userVals[index+1], 10, 64)
    case "passkey":
      user.Passkey = userVals[index+1]
    case "up_multiplier":
      user.UpMultiplier, err = strconv.ParseFloat(userVals[index+1], 64)
    case "down_multiplier":
      user.DownMultiplier, err = strconv.ParseFloat(userVals[index+1], 64)
    case "slots":
      user.Slots, err = strconv.ParseInt(userVals[index+1], 10, 64)
    case "slots_used":
      user.SlotsUsed, err = strconv.ParseInt(userVals[index+1], 10, 64)
    case "snatches":
      user.Snatches, err = strconv.ParseUint(userVals[index+1], 10, 64)
    }
    if err != nil {
      return nil, err
    }
  }
  return &user, nil
}

// createTorrent takes a string slice of length 14 and returns a pointer to a new storage.Torrent
// or an error.
// This function can be used to create a torrent from a Redis hash response(HGETALL).
// The order of strings the in the slice must follow the pattern:
//  [<field name>, <field value>, <field name>, <field value>, ...]
// This function calls multiple redis commands, it's not internally atomic.
// If the field values cannot be converted to the correct type,
// createTorrent will return a nil user and the conversion error.
// After converting the torrent fields, the seeders and leechers are populated by redis.getPeers
func (conn *Conn) createTorrent(torrentVals []string) (*storage.Torrent, error) {
  if len(torrentVals) != 14 {
    return nil, ErrCreateTorrent
  }
  var torrent storage.Torrent
  var err error
  for index, torrentString := range torrentVals {
    switch torrentString {
    case "id":
      torrent.ID, err = strconv.ParseUint(torrentVals[index+1], 10, 64)
    case "infohash":
      torrent.Infohash = torrentVals[index+1]
    case "active":
      torrent.Active, err = strconv.ParseBool(torrentVals[index+1])
    case "snatches":
      torrent.Snatches, err = strconv.ParseUint(torrentVals[index+1], 10, 32)
    case "up_multiplier":
      torrent.UpMultiplier, err = strconv.ParseFloat(torrentVals[index+1], 64)
    case "down_multiplier":
      torrent.DownMultiplier, err = strconv.ParseFloat(torrentVals[index+1], 64)
    case "last_action":
      torrent.LastAction, err = strconv.ParseInt(torrentVals[index+1], 10, 64)
    }
    if err != nil {
      return nil, err
    }
  }
  torrent.Seeders, err = conn.getPeers(torrent.ID, SeedersPrefix)
  if err != nil {
    return nil, err
  }
  torrent.Leechers, err = conn.getPeers(torrent.ID, LeechersPrefix)
  if err != nil {
    return nil, err
  }
  return &torrent, nil
}

// setPeer writes or overwrites peer information, stored as a Redis hash.
// The hash fields names are the same as the JSON tags on the storage.Peer struct.
func (conn *Conn) setPeer(peer *storage.Peer) error {
  hashKey := conn.conf.Prefix + getPeerHashKey(peer)
  _, err := conn.Do("HMSET", hashKey,
    "id", peer.ID,
    "user_id", peer.UserID,
    "torrent_id", peer.TorrentID,
    "ip", peer.IP,
    "port", peer.Port,
    "uploaded", peer.Uploaded,
    "downloaded", peer.Downloaded,
    "left", peer.Left,
    "last_announce", peer.LastAnnounce)

  return err
}

// removePeer removes the given peer from the specified peer set (seeder or leecher),
// and removes the peer information.
// This function calls multiple redis commands, it's not internally atomic.
// This function will not return an error if the peer to remove doesn't exist.
func (conn *Conn) removePeer(peer *storage.Peer, peerTypePrefix string) error {
  setKey := conn.conf.Prefix + getPeerSetKey(peerTypePrefix, peer)
  _, err := conn.Do("SREM", setKey, getPeerHashKey(peer))
  if err != nil {
    return err
  }
  hashKey := conn.conf.Prefix + getPeerHashKey(peer)
  _, err = conn.Do("DEL", hashKey)
  return nil
}

// removePeers removes all peers from specified peer set (seeders or leechers),
// removes the peer information, and then removes the associated peer from the given map.
// This function will not return an error if the peer to remove doesn't exist.
// This function will only delete the peer set if all the individual peer deletions were successful
// This function calls multiple redis commands, it's not internally atomic.
func (conn *Conn) removePeers(torrentID uint64, peers map[string]storage.Peer, peerTypePrefix string) error {
  for _, peer := range peers {
    hashKey := conn.conf.Prefix + getPeerHashKey(&peer)
    _, err := conn.Do("DEL", hashKey)
    if err != nil {
      return err
    }
    delete(peers, storage.PeerMapKey(&peer))
  }
  setKey := conn.conf.Prefix + peerTypePrefix + strconv.FormatUint(torrentID, 36)
  _, err := conn.Do("DEL", setKey)
  if err != nil {
    return err
  }
  return nil
}

// getPeerHashKey returns a string with the peer.ID, encoded peer.UserID, and encoded peer.TorrentID,
// concatenated and delimited by colons
// This key corresponds to a Redis hash type with fields containing a peer's data.
// The peer hashkey relies on the combination of peerID, userID, and torrentID being unique.
func getPeerHashKey(peer *storage.Peer) string {
  return peer.ID + ":" + strconv.FormatUint(peer.UserID, 36) + ":" + strconv.FormatUint(peer.TorrentID, 36)
}

// getPeerSetKey returns a string that is the peer's encoded torrentID appended to the typePrefix
// This key corresponds to a torrent's pool of leechers or seeders
func getPeerSetKey(typePrefix string, peer *storage.Peer) string {
  return typePrefix + strconv.FormatUint(peer.TorrentID, 36)
}

// addPeers adds each peer's key to the specified peer set and saves the peer's information.
// This function will not return an error if the peer already exists in the set.
// This function calls multiple redis commands, it's not internally atomic.
func (conn *Conn) addPeers(peers map[string]storage.Peer, peerTypePrefix string) error {
  for _, peer := range peers {
    setKey := conn.conf.Prefix + getPeerSetKey(peerTypePrefix, &peer)
    _, err := conn.Do("SADD", setKey, getPeerHashKey(&peer))
    if err != nil {
      return err
    }
    conn.setPeer(&peer)
  }
  return nil
}

// createPeer takes a slice of length 9 and returns a pointer to a new storage.Peer or an error.
// This function is used to create a peer from a Redis hash response(HGETALL).
// The order of strings the in the slice must follow the pattern:
//  [<field name>, <field value>, <field name>, <field value>, ...]
// If the field value string cannot be converted to the correct type,
// the function will return a nil peer and the conversion error.
func createPeer(peerVals []string) (*storage.Peer, error) {
  if len(peerVals) != 18 {
    return nil, ErrCreatePeer
  }
  var peer storage.Peer
  var err error
  for index, peerString := range peerVals {
    switch peerString {
    case "id":
      peer.ID = peerVals[index+1]
    case "user_id":
      peer.UserID, err = strconv.ParseUint(peerVals[index+1], 10, 64)
    case "torrent_id":
      peer.TorrentID, err = strconv.ParseUint(peerVals[index+1], 10, 64)
    case "ip":
      peer.IP = peerVals[index+1]
    case "port":
      peer.Port, err = strconv.ParseUint(peerVals[index+1], 10, 64)
    case "uploaded":
      peer.Uploaded, err = strconv.ParseUint(peerVals[index+1], 10, 64)
    case "downloaded":
      peer.Downloaded, err = strconv.ParseUint(peerVals[index+1], 10, 64)
    case "left":
      peer.Left, err = strconv.ParseUint(peerVals[index+1], 10, 64)
    case "last_announce":
      peer.LastAnnounce, err = strconv.ParseInt(peerVals[index+1], 10, 64)
    }
    if err != nil {
      return nil, err
    }
  }
  return &peer, nil
}

// getPeers returns a map of peers from a specified torrent's peer set(seeders or leechers).
// This is a multiple action command, it's not internally atomic.
func (conn *Conn) getPeers(torrentID uint64, peerTypePrefix string) (peers map[string]storage.Peer, err error) {
  peers = make(map[string]storage.Peer)
  setKey := conn.conf.Prefix + peerTypePrefix + strconv.FormatUint(torrentID, 36)
  peerStrings, err := redis.Strings(conn.Do("SMEMBERS", setKey))
  if err != nil {
    return nil, err
  }
  // Keys map to peer objects stored in hashes
  for _, peerHashKey := range peerStrings {
    hashKey := conn.conf.Prefix + peerHashKey
    peerVals, err := redis.Strings(conn.Do("HGETALL", hashKey))
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
    peers[storage.PeerMapKey(peer)] = *peer
  }
  return
}

// AddTorrent writes/overwrites torrent information and saves peers from both peer sets.
// The hash fields names are the same as the JSON tags on the storage.Torrent struct.
// This is a multiple action command, it's not internally atomic.
func (conn *Conn) AddTorrent(t *storage.Torrent) error {
  hashkey := conn.conf.Prefix + TorrentPrefix + t.Infohash
  _, err := conn.Do("HMSET", hashkey,
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

  err = conn.addPeers(t.Seeders, SeedersPrefix)
  if err != nil {
    return err
  }
  err = conn.addPeers(t.Leechers, LeechersPrefix)
  if err != nil {
    return err
  }
  return nil
}

// RemoveTorrent deletes the torrent's Redis hash and then deletes all peers.
// This function will not return an error if the torrent has already been removed.
// This is a multiple action command, it's not internally atomic.
func (conn *Conn) RemoveTorrent(t *storage.Torrent) error {
  hashkey := conn.conf.Prefix + TorrentPrefix + t.Infohash
  _, err := conn.Do("DEL", hashkey)
  if err != nil {
    return err
  }
  // Remove seeders and leechers as well
  err = conn.removePeers(t.ID, t.Seeders, SeedersPrefix)
  if err != nil {
    return err
  }
  err = conn.removePeers(t.ID, t.Leechers, LeechersPrefix)
  if err != nil {
    return err
  }
  return nil
}

// AddUser writes/overwrites user information to a Redis hash.
// The hash fields names are the same as the JSON tags on the storage.user struct.
func (conn *Conn) AddUser(u *storage.User) error {
  hashkey := conn.conf.Prefix + UserPrefix + u.Passkey
  _, err := conn.Do("HMSET", hashkey,
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

// RemoveUser removes the user's hash from Redis.
// This function does not return an error if the user doesn't exist.
func (conn *Conn) RemoveUser(u *storage.User) error {
  hashkey := conn.conf.Prefix + UserPrefix + u.Passkey
  _, err := conn.Do("DEL", hashkey)
  if err != nil {
    return err
  }
  return nil
}

// FindUser returns a pointer to a new user struct and true if the user exists,
// or nil and false if the user doesn't exist.
// This function does not return an error if the torrent doesn't exist.
func (conn *Conn) FindUser(passkey string) (*storage.User, bool, error) {
  hashkey := conn.conf.Prefix + UserPrefix + passkey
  // Consider using HGETALL instead of HVALS here for robustness
  userStrings, err := redis.Strings(conn.Do("HGETALL", hashkey))
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

// FindTorrent returns a pointer to a new torrent struct and true if the torrent exists,
// or nil and false if the torrent doesn't exist.
// This is a multiple action command, it's not internally atomic.
func (conn *Conn) FindTorrent(infohash string) (*storage.Torrent, bool, error) {
  hashkey := conn.conf.Prefix + TorrentPrefix + infohash
  torrentStrings, err := redis.Strings(conn.Do("HGETALL", hashkey))
  if err != nil {
    return nil, false, err
  } else if len(torrentStrings) == 0 {
    return nil, false, nil
  }

  foundTorrent, err := conn.createTorrent(torrentStrings)
  if err != nil {
    return nil, false, err
  }
  return foundTorrent, true, nil
}

// ClientWhitelisted returns true if the ClientID exists in the Client set.
// This function does not parse the client ID from the peer ID.
// The clientID must match exactly to a member of the set.
func (conn *Conn) ClientWhitelisted(peerID string) (exists bool, err error) {
  key := conn.conf.Prefix + "whitelist"
  return redis.Bool(conn.Do("SISMEMBER", key, peerID))
}

// WhitelistClient adds a client ID to the client whitelist set.
// This function does not return an error if the client ID is already in the set.
func (conn *Conn) WhitelistClient(peerID string) error {
  key := conn.conf.Prefix + "whitelist"
  _, err := conn.Do("SADD", key, peerID)
  return err
}

// UnWhitelistClient removes a client ID from the client whitelist set
// This function does not return an error if the client ID is not in the set.
func (conn *Conn) UnWhitelistClient(peerID string) error {
  key := conn.conf.Prefix + "whitelist"
  _, err := conn.Do("SREM", key, peerID)
  return err
}

// RecordSnatch increments the snatch counter on the torrent and user by one.
// This modifies the arguments as well as the hash field in Redis.
// This is a multiple action command, it's not internally atomic.
func (conn *Conn) RecordSnatch(user *storage.User, torrent *storage.Torrent) error {

  torrentKey := conn.conf.Prefix + TorrentPrefix + torrent.Infohash
  snatchCount, err := redis.Int(conn.Do("HINCRBY", torrentKey, "snatches", 1))
  if err != nil {
    return err
  }
  torrent.Snatches = uint64(snatchCount)

  userKey := conn.conf.Prefix + UserPrefix + user.Passkey
  snatchCount, err = redis.Int(conn.Do("HINCRBY", userKey, "snatches", 1))
  if err != nil {
    return err
  }
  user.Snatches = uint64(snatchCount)
  return nil
}

// MarkActive sets the active field of the torrent to true.
// This modifies the argument as well as the hash field in Redis.
// This function will return ErrMarkActive if the torrent does not exist.
func (conn *Conn) MarkActive(torrent *storage.Torrent) error {
  hashkey := conn.conf.Prefix + TorrentPrefix + torrent.Infohash
  activeExists, err := redis.Int(conn.Do("HSET", hashkey, "active", true))
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

// MarkInactive sets the active field of the torrent to false.
// This modifies the argument as well as the hash field in Redis.
// This function will return ErrMarkActive if the torrent does not exist.
func (conn *Conn) MarkInactive(torrent *storage.Torrent) error {
  hashkey := conn.conf.Prefix + TorrentPrefix + torrent.Infohash
  activeExists, err := redis.Int(conn.Do("HSET", hashkey, "active", false))
  if err != nil {
    return err
  }
  torrent.Active = false
  // HSET returns 1 if hash didn't exist before
  if activeExists == 1 {
    // Clean-up incomplete torrent
    _, err = conn.Do("DEL", hashkey)
    if err != nil {
      return err
    }
    return ErrMarkActive
  }
  return nil
}

// AddLeecher adds a new peer to a torrent's leecher set.
// This modifies the torrent argument, as well as the torrent's set and peer's hash in Redis.
// This function does not return an error if the leecher already exists.
// This is a multiple action command, it's not internally atomic.
func (conn *Conn) AddLeecher(torrent *storage.Torrent, peer *storage.Peer) error {
  setKey := conn.conf.Prefix + LeechersPrefix + strconv.FormatUint(torrent.ID, 36)
  _, err := conn.Do("SADD", setKey, getPeerHashKey(peer))
  if err != nil {
    return err
  }
  err = conn.setPeer(peer)
  if err != nil {
    return err
  }
  if torrent.Leechers == nil {
    torrent.Leechers = make(map[string]storage.Peer)
  }
  torrent.Leechers[storage.PeerMapKey(peer)] = *peer
  return nil
}

// SetLeecher updates a torrent's leecher.
// This modifies the torrent argument, as well as the peer's hash in Redis.
// Setting assumes that the peer is already a leecher, and only needs to be updated.
// This function does not return an error if the leecher does not exist or is not in the torrent's leecher set.
func (conn *Conn) SetLeecher(t *storage.Torrent, p *storage.Peer) error {
  err := conn.setPeer(p)
  if err != nil {
    return err
  }
  t.Leechers[storage.PeerMapKey(p)] = *p
  return nil
}

// RemoveLeecher removes the given peer from a torrent's leecher set.
// This modifies the torrent argument, as well as the torrent's set and peer's hash in Redis.
// This function does not return an error if the peer doesn't exist, or is not in the set.
func (conn *Conn) RemoveLeecher(t *storage.Torrent, p *storage.Peer) error {
  err := conn.removePeer(p, LeechersPrefix)
  if err != nil {
    return err
  }
  delete(t.Leechers, storage.PeerMapKey(p))
  return nil
}

// LeecherFinished moves a peer's hashkey from a torrent's leecher set to the seeder set and updates the peer.
// This modifies the torrent argument, as well as the torrent's set and peer's hash in Redis.
// This function does not return an error if the peer doesn't exist or is not in the torrent's leecher set.
func (conn *Conn) LeecherFinished(torrent *storage.Torrent, peer *storage.Peer) error {
  torrentIdKey := strconv.FormatUint(torrent.ID, 36)
  seederSetKey := conn.conf.Prefix + SeedersPrefix + torrentIdKey
  leecherSetKey := conn.conf.Prefix + LeechersPrefix + torrentIdKey

  _, err := conn.Do("SMOVE", leecherSetKey, seederSetKey, getPeerHashKey(peer))
  if err != nil {
    return err
  }
  torrent.Seeders[storage.PeerMapKey(peer)] = *peer
  delete(torrent.Leechers, storage.PeerMapKey(peer))

  err = conn.setPeer(peer)
  return err
}

// AddSeeder adds a new peer to a torrent's seeder set.
// This modifies the torrent argument, as well as the torrent's set and peer's hash in Redis.
// This function does not return an error if the seeder already exists.
// This is a multiple action command, it's not internally atomic.
func (conn *Conn) AddSeeder(torrent *storage.Torrent, peer *storage.Peer) error {
  setKey := conn.conf.Prefix + SeedersPrefix + strconv.FormatUint(torrent.ID, 36)
  _, err := conn.Do("SADD", setKey, getPeerHashKey(peer))
  if err != nil {
    return err
  }
  err = conn.setPeer(peer)
  if err != nil {
    return err
  }
  if torrent.Seeders == nil {
    torrent.Seeders = make(map[string]storage.Peer)
  }
  torrent.Seeders[storage.PeerMapKey(peer)] = *peer
  return nil
}

// SetSeeder updates a torrent's seeder.
// This modifies the torrent argument, as well as the peer's hash in Redis.
// Setting assumes that the peer is already a seeder, and only needs to be updated.
// This function does not return an error if the seeder does not exist or is not in the torrent's seeder set.
func (conn *Conn) SetSeeder(t *storage.Torrent, p *storage.Peer) error {
  err := conn.setPeer(p)
  if err != nil {
    return err
  }
  t.Seeders[storage.PeerMapKey(p)] = *p
  return nil
}

// RemoveSeeder removes the given peer from a torrent's seeder set.
// This modifies the torrent argument, as well as the torrent's set and peer's hash in Redis.
// This function does not return an error if the peer doesn't exist, or is not in the set.
func (conn *Conn) RemoveSeeder(t *storage.Torrent, p *storage.Peer) error {
  err := conn.removePeer(p, SeedersPrefix)
  if err != nil {
    return err
  }
  delete(t.Seeders, storage.PeerMapKey(p))
  return nil
}

// IncrementSlots increment a user's Slots by one.
// This function modifies the argument as well as the hash field in Redis.
func (conn *Conn) IncrementSlots(u *storage.User) error {
  hashkey := conn.conf.Prefix + UserPrefix + u.Passkey
  slotCount, err := redis.Int(conn.Do("HINCRBY", hashkey, "slots", 1))
  if err != nil {
    return err
  }
  u.Slots = int64(slotCount)
  return nil
}

// IncrementSlots increment a user's Slots by one.
// This function modifies the argument as well as the hash field in Redis.
func (conn *Conn) DecrementSlots(u *storage.User) error {
  hashkey := conn.conf.Prefix + UserPrefix + u.Passkey
  slotCount, err := redis.Int(conn.Do("HINCRBY", hashkey, "slots", -1))
  if err != nil {
    return err
  }
  u.Slots = int64(slotCount)
  return nil
}

// init registers the redis driver
func init() {
  tracker.Register("redis", &driver{})
}
