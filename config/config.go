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

type Client struct {
	Name   string `json:"name"`
	PeerID string `json:"peer_id"`
}

type Storage struct {
	Driver   string `json:"driver"`
	Network  string `json:"network`
	Addr     string `json:"addr"`
	Username string `json:"user"`
	Password string `json:"pass"`
	Schema   string `json:"schema,omitempty"`
	Encoding string `json:"encoding,omitempty"`
	Prefix   string `json:"prefix,omitempty"`

	ConnectTimeout *Duration `json:"conn_timeout,omitempty"`
	ReadTimeout    *Duration `json:"read_timeout,omitempty"`
	WriteTimeout   *Duration `json:"write_timeout,omitempty"`
}

type Config struct {
	Addr    string  `json:"addr"`
	Storage Storage `json:"storage"`

	Private   bool `json:"private"`
	Freeleech bool `json:"freeleech"`

	Announce    Duration `json:"announce"`
	MinAnnounce Duration `json:"min_announce"`
	ReadTimeout Duration `json:"read_timeout"`

	Whitelist []Client `json:"whitelist"`
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

func (c *Config) Whitelisted(peerId string) (matched bool) {
	for _, client := range c.Whitelist {
		length := len(client.PeerID)
		if length <= len(peerId) {
			matched = true
			for i := 0; i < length; i++ {
				if peerId[i] != client.PeerID[i] {
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
