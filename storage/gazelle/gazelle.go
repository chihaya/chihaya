// Copyright 2013 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

// Package gazelle provides a driver for a BitTorrent tracker to interface
// with the MySQL database used by Gazelle (github.com/WhatCD/Gazelle).
package gazelle

import (
	"bytes"
	"database/sql"
	"fmt"
	"sync"

	"github.com/pushrax/chihaya/config"
	"github.com/pushrax/chihaya/models"
	"github.com/pushrax/chihaya/storage"

	_ "github.com/go-sql-driver/mysql"
)

type driver struct{}

func (d *driver) New(conf *config.DataStore) storage.Conn {
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
	db.SetMaxIdleConns(conf.MaxIdleConns)

	conn := &Conn{db: db}

	// TODO Buffer sizes
	conn.torrentChannel = make(chan *bytes.Buffer, 1000)
	conn.userChannel = make(chan *bytes.Buffer, 1000)
	conn.transferHistoryChannel = make(chan *bytes.Buffer, 1000)
	conn.transferIpsChannel = make(chan *bytes.Buffer, 1000)
	conn.snatchChannel = make(chan *bytes.Buffer, 100)

	return conn
}

type Conn struct {
	db        *sql.DB
	waitGroup sync.WaitGroup
	terminate bool

	torrentChannel         chan *bytes.Buffer
	userChannel            chan *bytes.Buffer
	transferHistoryChannel chan *bytes.Buffer
	transferIpsChannel     chan *bytes.Buffer
	snatchChannel          chan *bytes.Buffer
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
	return c.db.Close()
}

func (c *Conn) RecordAnnounce(delta *models.AnnounceDelta) error {
	return nil
}

func (c *Conn) RecordSnatch(peer *models.Peer) error {
	return nil
}

func init() {
	storage.Register("gazelle", &driver{})
}
