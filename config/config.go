// Copyright 2014 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

// Package config implements the configuration for a BitTorrent tracker
package config

import (
	"encoding/json"
	"io"
	"os"
	"time"
)

// Duration wraps a time.Duration and adds JSON marshalling.
type Duration struct{ time.Duration }

// MarshalJSON transforms a duration into JSON.
func (d *Duration) MarshalJSON() ([]byte, error) {
	return json.Marshal(d.String())
}

// UnmarshalJSON transform JSON into a Duration.
func (d *Duration) UnmarshalJSON(b []byte) error {
	var str string
	err := json.Unmarshal(b, &str)
	d.Duration, err = time.ParseDuration(str)
	return err
}

// DriverConfig is the configuration used to connect to a tracker.Driver or
// a backend.Driver.
type DriverConfig struct {
	Name   string            `json:"driver"`
	Params map[string]string `json:"params,omitempty"`
}

// Config is a configuration for a Server.
type Config struct {
	Addr    string       `json:"addr"`
	Tracker DriverConfig `json:"tracker"`
	Backend DriverConfig `json:"backend"`

	Private   bool `json:"private"`
	Freeleech bool `json:"freeleech"`
	Whitelist bool `json:"whitelist"`

	Announce        Duration `json:"announce"`
	MinAnnounce     Duration `json:"min_announce"`
	RequestTimeout  Duration `json:"request_timeout"`
	NumWantFallback int      `json:"default_num_want"`
}

var DefaultConfig = Config{
	Addr: "127.0.0.1:6881",
	Tracker: DriverConfig{
		Name: "mock",
	},
	Backend: DriverConfig{
		Name: "mock",
	},
	Private:         false,
	Freeleech:       false,
	Whitelist:       false,
	Announce:        Duration{30 * time.Minute},
	MinAnnounce:     Duration{15 * time.Minute},
	RequestTimeout:  Duration{10 * time.Second},
	NumWantFallback: 50,
}

// Open is a shortcut to open a file, read it, and generate a Config.
// It supports relative and absolute paths. Given "", it returns the result of
// New.
func Open(path string) (*Config, error) {
	if path == "" {
		return &DefaultConfig, nil
	}

	f, err := os.Open(os.ExpandEnv(path))
	if err != nil {
		return nil, err
	}
	defer f.Close()

	conf, err := Decode(f)
	if err != nil {
		return nil, err
	}
	return conf, nil
}

// Decode attempts to decode a JSON encoded reader into a *Config.
func Decode(r io.Reader) (*Config, error) {
	conf := &Config{}
	err := json.NewDecoder(r).Decode(conf)
	if err != nil {
		return nil, err
	}
	return conf, nil
}
