// Copyright 2013 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

// Package config implements the configuration for a BitTorrent tracker
package config

import (
	"encoding/json"
	"os"
	"time"
)

type Duration struct {
	time.Duration
}

func (d *Duration) MarshalJSON() ([]byte, error) {
	return json.Marshal(d.String())
}

func (d *Duration) UnmarshalJSON(b []byte) error {
	var str string
	err := json.Unmarshal(b, &str)
	d.Duration, err = time.ParseDuration(str)
	return err
}

type Storage struct {
	Driver   string `json:"driver"`
	Protocol string `json:"protocol"`
	Addr     string `json:"addr"`
	Username string `json:"user"`
	Password string `json:"pass"`
	Schema   string `json:"schema"`
	Encoding string `json:"encoding"`
}

type Config struct {
	Addr    string  `json:"addr"`
	Storage Storage `json:"storage"`

	Private   bool `json:"private"`
	Freeleech bool `json:"freeleech"`

	Announce    Duration `json:"announce"`
	MinAnnounce Duration `json:"min_announce"`
	ReadTimeout Duration `json:"read_timeout"`

	Whitelist []string `json:"whitelist"`
}

func New(path string) (*Config, error) {
	expandedPath := os.ExpandEnv(path)
	f, err := os.Open(expandedPath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	conf := &Config{}
	err = json.NewDecoder(f).Decode(conf)
	if err != nil {
		return nil, err
	}
	return conf, nil
}

func (c *Config) Whitelisted(peerId string) bool {
	var (
		widLen  int
		matched bool
	)

	for _, whitelistedId := range c.Whitelist {
		widLen = len(whitelistedId)
		if widLen <= len(peerId) {
			matched = true
			for i := 0; i < widLen; i++ {
				if peerId[i] != whitelistedId[i] {
					matched = false
					break
				}
			}
			if matched {
				return true
			}
		}
	}
	return false
}
