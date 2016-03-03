// Copyright 2016 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

package chihaya

import (
	"io"
	"io/ioutil"
	"os"
	"time"

	"gopkg.in/yaml.v2"
)

// DefaultConfig is a sane configuration used as a fallback or for testing.
var DefaultConfig = Config{
	Tracker: TrackerConfig{
		AnnounceInterval:    30 * time.Minute,
		MinAnnounceInterval: 20 * time.Minute,
		AnnounceMiddleware:  []string{},
		ScrapeMiddleware:    []string{},
	},
	Servers: []ServerConfig{},
}

// Config represents the global configuration of a chihaya binary.
type Config struct {
	Tracker TrackerConfig  `yaml:"tracker"`
	Servers []ServerConfig `yaml:"servers"`
}

// TrackerConfig represents the configuration of protocol-agnostic BitTorrent
// Tracker used by Servers started by chihaya.
type TrackerConfig struct {
	AnnounceInterval    time.Duration `yaml:"announce"`
	MinAnnounceInterval time.Duration `yaml:"min_announce"`
	AnnounceMiddleware  []string      `yaml:"announce_middleware"`
	ScrapeMiddleware    []string      `yaml:"scrape_middleware"`
}

// ServerConfig represents the configuration of the Servers started by chihaya.
type ServerConfig struct {
	Name   string      `yaml:"name"`
	Config interface{} `yaml:"config"`
}

// ConfigFile represents a YAML configuration file that namespaces all chihaya
// configuration under the "chihaya" namespace.
type ConfigFile struct {
	Chihaya Config `yaml:"chihaya"`
}

// DecodeConfigFile unmarshals an io.Reader into a new Config.
func DecodeConfigFile(r io.Reader) (*Config, error) {
	contents, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}

	cfgFile := &ConfigFile{}
	err = yaml.Unmarshal(contents, cfgFile)
	if err != nil {
		return nil, err
	}

	return &cfgFile.Chihaya, nil
}

// OpenConfigFile returns a new Config given the path to a YAML configuration
// file.
// It supports relative and absolute paths and environment variables.
// Given "", it returns DefaultConfig.
func OpenConfigFile(path string) (*Config, error) {
	if path == "" {
		return &DefaultConfig, nil
	}

	f, err := os.Open(os.ExpandEnv(path))
	if err != nil {
		return nil, err
	}
	defer f.Close()

	cfg, err := DecodeConfigFile(f)
	if err != nil {
		return nil, err
	}

	return cfg, nil
}
