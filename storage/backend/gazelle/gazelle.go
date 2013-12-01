// Copyright 2013 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

// Package gazelle provides a driver for a BitTorrent tracker to interface
// with the MySQL database used by Gazelle (github.com/WhatCD/Gazelle).
package gazelle

import (
	"database/sql"
	"fmt"
	"sync"

	"github.com/chihaya/chihaya/config"
	"github.com/chihaya/chihaya/storage/backend"

	_ "github.com/go-sql-driver/mysql"
)

type driver struct{}

func (d *driver) New(conf *config.DataStore) backend.Conn {
	dsn := fmt.Sprintf(
		"%s:%s@%s:%s/%s?charset=utf8mb4,utf8",
		conf.Username,
		conf.Password,
		conf.Host,
		conf.Port,
		conf.Schema,
	)
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		panic("gazelle: failed to open connection to MySQL")
	}

	if conf.MaxIdleConns != 0 {
		db.SetMaxIdleConns(conf.MaxIdleConns)
	}

	conn := &Conn{DB: db}

	// TODO Buffer sizes
	conn.torrentChannel = make(chan string, 1000)
	conn.userChannel = make(chan string, 1000)
	conn.transferHistoryChannel = make(chan string, 1000)
	conn.transferIpsChannel = make(chan string, 1000)
	conn.snatchChannel = make(chan string, 100)

	return conn
}

type Conn struct {
	waitGroup sync.WaitGroup
	terminate bool

	torrentChannel         chan string
	userChannel            chan string
	transferHistoryChannel chan string
	transferIpsChannel     chan string
	snatchChannel          chan string

	*sql.DB
}

func (c *Conn) Start() error {
	go c.flushTorrents()
	go c.flushUsers()
	go c.flushTransferHistory()
	go c.flushTransferIps()
	go c.flushSnatches()
	return nil
}

func (c *Conn) Close() error {
	c.terminate = true
	c.waitGroup.Wait()
	return c.DB.Close()
}

func (c *Conn) RecordAnnounce(delta *backend.AnnounceDelta) error {
	snatchCount := 0
	if delta.Snatched {
		snatchCount = 1
	}

	c.torrentChannel <- fmt.Sprintf(
		"('%d','%d','%d','%d','%d')",
		delta.Torrent.ID,
		snatchCount,
		len(delta.Torrent.Seeders),
		len(delta.Torrent.Leechers),
		delta.Torrent.LastAction,
	)
	return nil
}

func init() {
	backend.Register("gazelle", &driver{})
}
