// Copyright 2016 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

// Package config implements the opening and parsing of a chihaya configuration.
package config

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

// TrackerConfig represents the configuration of the BitTorrent tracker used by
// chihaya.
type TrackerConfig struct {
	AnnounceInterval    time.Duration `yaml:"announce"`
	MinAnnounceInterval time.Duration `yaml:"min_announce"`
	AnnounceMiddleware  []string      `yaml:"announce_middleware"`
	ScrapeMiddleware    []string      `yaml:"scrape_middleware"`
}

// ServerConfig represents the configuration of the servers started by chihaya.
type ServerConfig struct {
	Name   string      `yaml:"name"`
	Config interface{} `yaml:"config"`
}

// Open is a shortcut to open a file, read it, and allocates a new Config.
// It supports relative and absolute paths. Given "", it returns DefaultConfig.
func Open(path string) (*Config, error) {
	if path == "" {
		return &DefaultConfig, nil
	}

	f, err := os.Open(os.ExpandEnv(path))
	if err != nil {
		return nil, err
	}
	defer f.Close()

	cfg, err := Decode(f)
	if err != nil {
		return nil, err
	}

	return cfg, nil
}

// Decode unmarshals an io.Reader into a newly allocated *Config.
func Decode(r io.Reader) (*Config, error) {
	contents, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}

	cfg := &Config{}
	err = yaml.Unmarshal(contents, cfg)
	if err != nil {
		return nil, err
	}

	return cfg, nil
}
