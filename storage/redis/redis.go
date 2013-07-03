// Copyright 2013 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

package redis

import (
	"time"

	"github.com/garyburd/redigo/redis"

	"github.com/pushrax/chihaya/config"
	"github.com/pushrax/chihaya/storage"
)

type driver struct{}

func (d *driver) New(conf *config.Storage) storage.Pool {
	return &Pool{
		conf: conf,
		pool: redis.Pool{
			MaxIdle:      3,
			IdleTimeout:  240 * time.Second,
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

		if conf.ConnectTimeout != nil &&
			conf.ReadTimeout != nil &&
			conf.WriteTimeout != nil {

			conn, err = redis.DialTimeout(
				conf.Network,
				conf.Addr,
				conf.ConnectTimeout.Duration,
				conf.ReadTimeout.Duration,
				conf.WriteTimeout.Duration,
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

type Pool struct {
	conf *config.Storage
	pool redis.Pool
}

func (p *Pool) Get() storage.Conn {
	return &Conn{
		conf: p.conf,
		Conn: p.pool.Get(),
	}
}

func (p *Pool) Close() error {
	return p.pool.Close()
}

type Conn struct {
	conf *config.Storage
	redis.Conn
}

func (c *Conn) FindUser(passkey string) (*storage.User, bool, error) {
	key := c.conf.Prefix + "User:" + passkey

	exists, err := redis.Bool(c.Do("EXISTS", key))
	if err != nil {
		return nil, false, err
	}
	if !exists {
		return nil, false, nil
	}

	reply, err := redis.Values(c.Do("HGETALL", key))
	if err != nil {
		return nil, true, err
	}
	user := &storage.User{}
	err = redis.ScanStruct(reply, user)
	if err != nil {
		return nil, true, err
	}
	return user, true, nil
}

func (c *Conn) FindTorrent(infohash string) (*storage.Torrent, bool, error) {
	key := c.conf.Prefix + "Torrent:" + infohash

	exists, err := redis.Bool(c.Do("EXISTS", key))
	if err != nil {
		return nil, false, err
	}
	if !exists {
		return nil, false, nil
	}

	reply, err := redis.Values(c.Do("HGETALL", key))
	if err != nil {
		return nil, true, err
	}
	torrent := &storage.Torrent{}
	err = redis.ScanStruct(reply, torrent)
	if err != nil {
		return nil, true, err
	}
	return torrent, true, nil
}

type Tx struct {
	conn *Conn
}

func (c *Conn) NewTx() (storage.Tx, error) {
	err := c.Send("MULTI")
	if err != nil {
		return nil, err
	}
	return &Tx{c}, nil
}

func (t *Tx) UnpruneTorrent(torrent *storage.Torrent) error {
	key := t.conn.conf.Prefix + "Torrent:" + torrent.Infohash
	err := t.conn.Send("HSET " + key + " Status 0")
	if err != nil {
		return err
	}
	return nil
}

func (t *Tx) Commit() error {
	_, err := t.conn.Do("EXEC")
	if err != nil {
		return err
	}
	return nil
}

func init() {
	storage.Register("redis", &driver{})
}
