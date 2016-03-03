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

func constructor(srvcfg *chihaya.ServerConfig, tkr *tracker.Tracker) (server.Server, error) {
	if theStore == nil {
		cfg, err := newConfig(srvcfg)
		if err != nil {
			return nil, errors.New("store: invalid store config: " + err.Error())
		}

		cs, err := OpenClientStore(cfg)
		if err != nil {
			return nil, err
		}

		ps, err := OpenPeerStore(cfg)
		if err != nil {
			return nil, err
		}

		ips, err := OpenIPStore(cfg)
		if err != nil {
			return nil, err
		}

		theStore = &Store{
			cfg:         cfg,
			tkr:         tkr,
			shutdown:    make(chan struct{}),
			ClientStore: cs,
			PeerStore:   ps,
			IPStore:     ips,
		}
	}
	return theStore, nil
}

type Config struct {
	Addr              string        `yaml:"addr"`
	RequestTimeout    time.Duration `yaml:"request_timeout"`
	ReadTimeout       time.Duration `yaml:"read_timeout"`
	WriteTimeout      time.Duration `yaml:"write_timeout"`
	GCAfter           time.Duration `yaml:"gc_after"`
	ClientStore       string        `yaml:"client_store"`
	ClientStoreConfig interface{}   `yaml:"client_store_config"`
	PeerStore         string        `yaml:"peer_store"`
	PeerStoreConfig   interface{}   `yaml:"peer_store_config"`
	IPStore           string        `yaml:"ip_store"`
	IPStoreConfig     interface{}   `yaml:"ip_store_config"`
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

type Store struct {
	cfg      *Config
	tkr      *tracker.Tracker
	shutdown chan struct{}
	wg       sync.WaitGroup

	PeerStore
	ClientStore
	IPStore
}

func (s *Store) Start() {
}

func (s *Store) Stop() {
	close(s.shutdown)
	s.wg.Wait()
}
