// Copyright 2013 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

package redis

import (
	"github.com/garyburd/redigo/redis"

	"github.com/pushrax/chihaya/config"
	"github.com/pushrax/chihaya/storage"
)

type driver struct{}

func (d *driver) New(conf *config.Storage) (storage.Conn, error) {
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
	return &Conn{
		conn,
	}, nil
}

type Conn struct {
	conn redis.Conn
}

func init() {
	storage.Register("redis", &driver{})
}
