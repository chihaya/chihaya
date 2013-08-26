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
	"encoding/json"
	"strings"
	"time"

	"github.com/garyburd/redigo/redis"

	"github.com/pushrax/chihaya/cache"
	"github.com/pushrax/chihaya/config"
	"github.com/pushrax/chihaya/models"
)

type driver struct{}

func (d *driver) New(conf *config.DataStore) cache.Pool {
	return &Pool{
		conf: conf,
		pool: redis.Pool{
			MaxIdle:      conf.MaxIdleConn,
			IdleTimeout:  conf.IdleTimeout.Duration,
			Dial:         makeDialFunc(conf),
			TestOnBorrow: testOnBorrow,
		},
	}
}

func makeDialFunc(conf *config.DataStore) func() (redis.Conn, error) {
	return func() (conn redis.Conn, err error) {
		conn, err = redis.Dial(conf.Network, conf.Addr)
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
	return &Tx{
		conf:  p.conf,
		done:  false,
		multi: false,
		Conn:  p.pool.Get(),
	}, nil
}

// Tx represents a transaction for Redis with one gotcha:
// all reads must be done prior to any writes. Writes will
// check if the MULTI command has been sent to redis and will
// send it if it hasn't.
//
// Internally a transaction looks like:
// WATCH keyA
// GET keyA
// WATCH keyB
// GET keyB
// MULTI
// SET keyA
// SET keyB
// EXEC
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

func (tx *Tx) initiateWrite() error {
	if tx.done {
		return cache.ErrTxDone
	}
	if tx.multi != true {
		return tx.Send("MULTI")
	}
	return nil
}

func (tx *Tx) initiateRead() error {
	if tx.done {
		return cache.ErrTxDone
	}
	if tx.multi == true {
		panic("Tried to read during MULTI")
	}
	return nil
}

func (tx *Tx) Commit() error {
	if tx.done {
		return cache.ErrTxDone
	}
	if tx.multi == true {
		_, err := tx.Do("EXEC")
		if err != nil {
			return err
		}
	}
	tx.close()
	return nil
}

func (tx *Tx) Rollback() error {
	if tx.done {
		return cache.ErrTxDone
	}
	// Redis doesn't need to do anything. Exec is atomic.
	tx.close()
	return nil
}

func (tx *Tx) FindUser(passkey string) (*models.User, bool, error) {
	err := tx.initiateRead()
	if err != nil {
		return nil, false, err
	}

	key := tx.conf.Prefix + "user:" + passkey
	_, err = tx.Do("WATCH", key)
	if err != nil {
		return nil, false, err
	}
	reply, err := redis.String(tx.Do("GET", key))
	if err != nil {
		if err == redis.ErrNil {
			return nil, false, nil
		}
		return nil, false, err
	}

	user := &models.User{}
	err = json.NewDecoder(strings.NewReader(reply)).Decode(user)
	if err != nil {
		return nil, true, err
	}
	return user, true, nil
}

func (tx *Tx) FindTorrent(infohash string) (*models.Torrent, bool, error) {
	err := tx.initiateRead()
	if err != nil {
		return nil, false, err
	}

	key := tx.conf.Prefix + "torrent:" + infohash
	_, err = tx.Do("WATCH", key)
	if err != nil {
		return nil, false, err
	}
	reply, err := redis.String(tx.Do("GET", key))
	if err != nil {
		if err == redis.ErrNil {
			return nil, false, nil
		}
		return nil, false, err
	}

	torrent := &models.Torrent{}
	err = json.NewDecoder(strings.NewReader(reply)).Decode(torrent)
	if err != nil {
		return nil, true, err
	}
	return torrent, true, nil
}

func (tx *Tx) ClientWhitelisted(peerID string) (exists bool, err error) {
	err = tx.initiateRead()
	if err != nil {
		return false, err
	}

	key := tx.conf.Prefix + "whitelist"
	_, err = tx.Do("WATCH", key)
	if err != nil {
		return
	}

	// TODO
	return
}

func (tx *Tx) RecordSnatch(user *models.User, torrent *models.Torrent) error {
	if err := tx.initiateWrite(); err != nil {
		return err
	}

	// TODO
	return nil
}

func (tx *Tx) MarkActive(t *models.Torrent) error {
	if err := tx.initiateWrite(); err != nil {
		return err
	}

	// TODO
	return nil
}

func (tx *Tx) AddLeecher(t *models.Torrent, p *models.Peer) error {
	if err := tx.initiateWrite(); err != nil {
		return err
	}

	// TODO
	return nil
}

func (tx *Tx) SetLeecher(t *models.Torrent, p *models.Peer) error {
	if err := tx.initiateWrite(); err != nil {
		return err
	}

	// TODO
	return nil
}

func (tx *Tx) RemoveLeecher(t *models.Torrent, p *models.Peer) error {
	if err := tx.initiateWrite(); err != nil {
		return err
	}

	// TODO
	return nil
}

func (tx *Tx) AddSeeder(t *models.Torrent, p *models.Peer) error {
	if err := tx.initiateWrite(); err != nil {
		return err
	}

	// TODO
	return nil
}

func (tx *Tx) SetSeeder(t *models.Torrent, p *models.Peer) error {
	if err := tx.initiateWrite(); err != nil {
		return err
	}

	// TODO
	return nil
}

func (tx *Tx) RemoveSeeder(t *models.Torrent, p *models.Peer) error {
	if err := tx.initiateWrite(); err != nil {
		return err
	}

	// TODO
	return nil
}

func (tx *Tx) IncrementSlots(u *models.User) error {
	if err := tx.initiateWrite(); err != nil {
		return err
	}

	// TODO
	return nil
}

func (tx *Tx) DecrementSlots(u *models.User) error {
	if err := tx.initiateWrite(); err != nil {
		return err
	}

	// TODO
	return nil
}

func init() {
	cache.Register("redis", &driver{})
}
