// Copyright 2013 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

// Package redis implements the storage interface for a BitTorrent tracker.
//
// The client whitelist is represented as a set with the key name "whitelist"
// with an optional prefix. Torrents and users are JSON-formatted strings.
// Torrents' keys are named "torrent:<infohash>" with an optional prefix.
// Users' keys are named "user:<passkey>" with an optional prefix.
package redis

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/garyburd/redigo/redis"

	"github.com/pushrax/chihaya/config"
	"github.com/pushrax/chihaya/storage"
)

type driver struct{}

func (d *driver) New(conf *config.Storage) storage.DS {
	return &DS{
		conf: conf,
		Pool: redis.Pool{
			MaxIdle:      conf.MaxIdleConn,
			IdleTimeout:  conf.IdleTimeout.Duration,
			Dial:         makeDialFunc(conf),
			TestOnBorrow: testOnBorrow,
		},
	}
}

func makeDialFunc(conf *config.Storage) func() (redis.Conn, error) {
	return func() (redis.Conn, error) {
		var (
			conn redis.Conn
			err  error
		)

		if conf.ConnTimeout != nil {
			conn, err = redis.DialTimeout(
				conf.Network,
				conf.Addr,
				conf.ConnTimeout.Duration, // Connect Timeout
				conf.ConnTimeout.Duration, // Read Timeout
				conf.ConnTimeout.Duration, // Write Timeout
			)
		} else {
			conn, err = redis.Dial(conf.Network, conf.Addr)
		}
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

type DS struct {
	conf *config.Storage
	redis.Pool
}

func (ds *DS) FindUser(passkey string) (*storage.User, bool, error) {
	conn := ds.Get()
	defer conn.Close()

	key := ds.conf.Prefix + "user:" + passkey
	reply, err := redis.String(conn.Do("GET", key))
	if err != nil {
		if err == redis.ErrNil {
			return nil, false, nil
		}
		return nil, false, err
	}

	user := &storage.User{}
	err = json.NewDecoder(strings.NewReader(reply)).Decode(user)
	if err != nil {
		return nil, true, err
	}
	return user, true, nil
}

func (ds *DS) FindTorrent(infohash string) (*storage.Torrent, bool, error) {
	conn := ds.Get()
	defer conn.Close()

	key := ds.conf.Prefix + "torrent:" + infohash
	reply, err := redis.String(conn.Do("GET", key))
	if err != nil {
		if err == redis.ErrNil {
			return nil, false, nil
		}
		return nil, false, err
	}

	torrent := &storage.Torrent{}
	err = json.NewDecoder(strings.NewReader(reply)).Decode(torrent)
	if err != nil {
		return nil, true, err
	}
	return torrent, true, nil
}

func (ds *DS) ClientWhitelisted(peerID string) (bool, error) {
	conn := ds.Get()
	defer conn.Close()

	key := ds.conf.Prefix + "whitelist"
	exists, err := redis.Bool(conn.Do("SISMEMBER", key, peerID))
	if err != nil {
		return false, err
	}
	return exists, nil
}

type Tx struct {
	conf *config.Storage
	done bool
	redis.Conn
}

func (ds *DS) Begin() (storage.Tx, error) {
	conn := ds.Get()
	err := conn.Send("MULTI")
	if err != nil {
		return nil, err
	}
	return &Tx{
		conf: ds.conf,
		Conn: conn,
	}, nil
}

func (tx *Tx) close() {
	if tx.done {
		panic("redis: transaction closed twice")
	}
	tx.done = true
	tx.Conn.Close()
}

func (tx *Tx) Commit() error {
	if tx.done {
		return storage.ErrTxDone
	}
	_, err := tx.Do("EXEC")
	if err != nil {
		return err
	}

	tx.close()
	return nil
}

// Redis doesn't need to rollback. Exec is atomic.
func (tx *Tx) Rollback() error {
	if tx.done {
		return storage.ErrTxDone
	}
	tx.close()
	return nil
}

func (tx *Tx) Snatch(user *storage.User, torrent *storage.Torrent) error {
	if tx.done {
		return storage.ErrTxDone
	}
	// TODO
	return nil
}

func (tx *Tx) Active(t *storage.Torrent) error {
	if tx.done {
		return storage.ErrTxDone
	}
	key := tx.conf.Prefix + "torrent:" + t.Infohash
	err := activeScript.Send(tx.Conn, key)
	return err
}

func (tx *Tx) NewLeecher(t *storage.Torrent, p *storage.Peer) error {
	if tx.done {
		return storage.ErrTxDone
	}
	// TODO
	return nil
}

func (tx *Tx) SetLeecher(t *storage.Torrent, p *storage.Peer) error {
	if tx.done {
		return storage.ErrTxDone
	}
	// TODO
	return nil
}

func (tx *Tx) RmLeecher(t *storage.Torrent, p *storage.Peer) error {
	if tx.done {
		return storage.ErrTxDone
	}
	// TODO
	return nil
}

func (tx *Tx) NewSeeder(t *storage.Torrent, p *storage.Peer) error {
	if tx.done {
		return storage.ErrTxDone
	}
	// TODO
	return nil
}

func (tx *Tx) SetSeeder(t *storage.Torrent, p *storage.Peer) error {
	if tx.done {
		return storage.ErrTxDone
	}
	// TODO
	return nil
}

func (tx *Tx) RmSeeder(t *storage.Torrent, p *storage.Peer) error {
	if tx.done {
		return storage.ErrTxDone
	}
	key := tx.conf.Prefix + "torrent:" + t.Infohash
	err := rmSeederScript.Send(tx.Conn, key, p.ID)
	return err
}

func (tx *Tx) IncrementSlots(u *storage.User) error {
	if tx.done {
		return storage.ErrTxDone
	}
	key := tx.conf.Prefix + "user:" + u.Passkey
	err := incSlotsScript.Send(tx.Conn, key)
	return err
}

func (tx *Tx) DecrementSlots(u *storage.User) error {
	if tx.done {
		return storage.ErrTxDone
	}
	key := tx.conf.Prefix + "user:" + u.Passkey
	err := decSlotsScript.Send(tx.Conn, key)
	return err
}

func init() {
	storage.Register("redis", &driver{})
}
