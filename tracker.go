// Copyright 2016 Jimmy Zelinskie
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package trakr implements a BitTorrent Tracker that supports multiple
// protocols and configurable Hooks that execute before and after a Response
// has been delievered to a BitTorrent client.
package trakr

import (
	"errors"
	"io"
	"io/ioutil"
	"os"
	"time"

	"github.com/jzelinskie/trakr/bittorrent/http"
	"github.com/jzelinskie/trakr/bittorrent/udp"
	"gopkg.in/yaml.v2"
)

// GenericConfig is a block of configuration who's structure is unknown.
type GenericConfig struct {
	name   string      `yaml:"name"`
	config interface{} `yaml:"config"`
}

// MultiTracker is a multi-protocol, customizable BitTorrent Tracker.
type MultiTracker struct {
	AnnounceInterval time.Duration   `yaml:"announce_interval"`
	GCInterval       time.Duration   `yaml:"gc_interval"`
	GCExpiration     time.Duration   `yaml:"gc_expiration"`
	HTTPConfig       http.Config     `yaml:"http"`
	UDPConfig        udp.Config      `yaml:"udp"`
	PeerStoreConfig  []GenericConfig `yaml:"storage"`
	PreHooks         []GenericConfig `yaml:"prehooks"`
	PostHooks        []GenericConfig `yaml:"posthooks"`

	peerStore   PeerStore
	httpTracker http.Tracker
	udpTracker  udp.Tracker
}

// decodeConfigFile unmarshals an io.Reader into a new MultiTracker.
func decodeConfigFile(r io.Reader) (*MultiTracker, error) {
	contents, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}

	cfgFile := struct {
		mt MultiTracker `yaml:"trakr"`
	}{}
	err = yaml.Unmarshal(contents, cfgFile)
	if err != nil {
		return nil, err
	}

	return &cfgFile.mt, nil
}

// MultiTrackerFromFile returns a new MultiTracker given the path to a YAML
// configuration file.
//
// It supports relative and absolute paths and environment variables.
func MultiTrackerFromFile(path string) (*MultiTracker, error) {
	if path == "" {
		return nil, errors.New("no config path specified")
	}

	f, err := os.Open(os.ExpandEnv(path))
	if err != nil {
		return nil, err
	}
	defer f.Close()

	cfg, err := decodeConfigFile(f)
	if err != nil {
		return nil, err
	}

	return cfg, nil
}

// Stop provides a thread-safe way to shutdown a currently running
// MultiTracker.
func (t *MultiTracker) Stop() {
}

// ListenAndServe listens on the protocols and addresses specified in the
// HTTPConfig and UDPConfig then blocks serving BitTorrent requests until
// t.Stop() is called or an error is returned.
func (t *MultiTracker) ListenAndServe() error {
	// Build an TrackerFuncs from the PreHooks and PostHooks.
	// Create a PeerStore instance.
	// Create a HTTP Tracker instance.
	// Create a UDP Tracker instance.
	return nil
}
