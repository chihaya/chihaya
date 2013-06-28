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

func (d *driver) New(conf *config.Storage) (storage.Conn, error) {
	return &Conn{
		conf: conf,
		pool: &redis.Pool{
			MaxIdle:      3,
			IdleTimeout:  240 * time.Second,
			Dial:         makeDialFunc(conf),
			TestOnBorrow: testOnBorrow,
		},
	}, nil
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

type Conn struct {
	conf *config.Storage
	pool *redis.Pool
}

func (c *Conn) Close() error {
	return c.pool.Close()
}

func (c *Conn) FindUser(passkey string) (*storage.User, bool, error) {
	conn := c.pool.Get()
	defer c.pool.Close()

	key := c.conf.Prefix + "User:" + passkey

	exists, err := redis.Bool(conn.Do("EXISTS", key))
	if err != nil {
		return nil, false, err
	}
	if !exists {
		return nil, false, nil
	}

	reply, err := redis.Values(conn.Do("HGETALL", key))
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
	conn := c.pool.Get()
	defer c.pool.Close()

	key := c.conf.Prefix + "Torrent:" + infohash

	exists, err := redis.Bool(conn.Do("EXISTS", key))
	if err != nil {
		return nil, false, err
	}
	if !exists {
		return nil, false, nil
	}

	reply, err := redis.Values(conn.Do("HGETALL", key))
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

func (c *Conn) UnpruneTorrent(torrent *storage.Torrent) error {
	// TODO
	return nil
}

func init() {
	storage.Register("redis", &driver{})
}
