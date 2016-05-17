// Copyright 2016 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

package store

import (
	"errors"
	"log"
	"sync"
	"time"

	"gopkg.in/yaml.v2"

	"github.com/chihaya/chihaya"
	"github.com/chihaya/chihaya/server"
	"github.com/chihaya/chihaya/tracker"
)

var theStore *Store

func init() {
	server.Register("store", constructor)
}

// ErrResourceDoesNotExist is the error returned by all delete methods in the
// store if the requested resource does not exist.
var ErrResourceDoesNotExist = errors.New("resource does not exist")

func constructor(srvcfg *chihaya.ServerConfig, tkr *tracker.Tracker) (server.Server, error) {
	if theStore == nil {
		cfg, err := newConfig(srvcfg)
		if err != nil {
			return nil, errors.New("store: invalid store config: " + err.Error())
		}

		ps, err := OpenPeerStore(&cfg.PeerStore)
		if err != nil {
			return nil, err
		}

		ips, err := OpenIPStore(&cfg.IPStore)
		if err != nil {
			return nil, err
		}

		ss, err := OpenStringStore(&cfg.StringStore)
		if err != nil {
			return nil, err
		}

		theStore = &Store{
			cfg:         cfg,
			tkr:         tkr,
			shutdown:    make(chan struct{}),
			PeerStore:   ps,
			IPStore:     ips,
			StringStore: ss,
		}
	}
	return theStore, nil
}

// Config represents the configuration for the store.
type Config struct {
	Addr           string        `yaml:"addr"`
	RequestTimeout time.Duration `yaml:"request_timeout"`
	ReadTimeout    time.Duration `yaml:"read_timeout"`
	WriteTimeout   time.Duration `yaml:"write_timeout"`
	GCAfter        time.Duration `yaml:"gc_after"`
	PeerStore      DriverConfig  `yaml:"peer_store"`
	IPStore        DriverConfig  `yaml:"ip_store"`
	StringStore    DriverConfig  `yaml:"string_store"`
}

// DriverConfig represents the configuration for a store driver.
type DriverConfig struct {
	Name   string      `yaml:"name"`
	Config interface{} `yaml:"config"`
}

func newConfig(srvcfg *chihaya.ServerConfig) (*Config, error) {
	bytes, err := yaml.Marshal(srvcfg.Config)
	if err != nil {
		return nil, err
	}

	var cfg Config
	err = yaml.Unmarshal(bytes, &cfg)
	if err != nil {
		return nil, err
	}

	return &cfg, nil
}

// MustGetStore is used by middleware to access the store.
//
// This function calls log.Fatal if a server hasn't been already created by
// the server package.
func MustGetStore() *Store {
	if theStore == nil {
		log.Fatal("store middleware used without store server")
	}
	return theStore
}

// Store provides storage for a tracker.
type Store struct {
	cfg      *Config
	tkr      *tracker.Tracker
	shutdown chan struct{}
	wg       sync.WaitGroup

	PeerStore
	IPStore
	StringStore
}

// Start starts the store drivers and blocks until all of them exit.
func (s *Store) Start() {
	<-s.shutdown
	s.wg.Wait()
	log.Println("Store server shut down cleanly")
}

// Stop stops the store drivers and waits for them to exit.
func (s *Store) Stop() {
	close(s.shutdown)
	s.wg.Wait()
}
