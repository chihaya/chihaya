// Copyright 2013 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

// Package batter provides a driver for a BitTorrent tracker to interface
// with the postgres database used by batter (github.com/wafflesfm/batter).
package batter

import (
	"database/sql"
	"fmt"

	"github.com/pushrax/chihaya/config"
	"github.com/pushrax/chihaya/storage/web"

	_ "github.com/bmizerany/pq"
)

type driver struct{}

func (d *driver) New(conf *config.DataStore) web.Conn {
	dsn := fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s",
		conf.Host,
		conf.Port,
		conf.Username,
		conf.Password,
		conf.Schema,
	)
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		panic("batter: failed to open connection to postgres")
	}

	if conf.MaxIdleConns != 0 {
		db.SetMaxIdleConns(conf.MaxIdleConns)
	}

	return &Conn{db}
}

type Conn struct {
	*sql.DB
}

func (c *Conn) Start() error {
	return nil
}

func (c *Conn) RecordAnnounce(delta *web.AnnounceDelta) error {
	return nil
}

func init() {
	web.Register("batter", &driver{})
}
