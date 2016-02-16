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

	"github.com/chihaya/chihaya/config"
	"github.com/chihaya/chihaya/server"
	"github.com/chihaya/chihaya/tracker"
)

var theStore *Store

func init() {
	server.Register("store", constructor)
}

func constructor(srvcfg *config.ServerConfig, tkr *tracker.Tracker) (server.Server, error) {
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
			ClientStore: cs,
			PeerStore:   ps,
			IPStore:     ips,
		}
	}
	return theStore, nil
}

type Config struct {
	Addr              string        `yaml:"addr"`
	RequestTimeout    time.Duration `yaml:"requestTimeout"`
	ReadTimeout       time.Duration `yaml:"readTimeout"`
	WriteTimeout      time.Duration `yaml:"writeTimeout"`
	GCAfter           time.Duration `yaml:"gcAfter"`
	ClientStore       string        `yaml:"clientStore"`
	ClientStoreConfig interface{}   `yaml:"clienStoreConfig"`
	PeerStore         string        `yaml:"peerStore"`
	PeerStoreConfig   interface{}   `yaml:"peerStoreConfig"`
	IPStore           string        `yaml:"ipStore"`
	IPStoreConfig     interface{}   `yaml:"ipStoreConfig"`
}

func newConfig(srvcfg interface{}) (*Config, error) {
	bytes, err := yaml.Marshal(srvcfg)
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
