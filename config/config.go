// Copyright 2014 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

// Package config implements the configuration for a BitTorrent tracker
package config

import (
	"encoding/json"
	"errors"
	"io"
	"os"
	"time"
)

// ErrMissingRequiredParam is used by drivers to indicate that an entry required
// to be within the DriverConfig.Params map is not present.
var ErrMissingRequiredParam = errors.New("A parameter that was required by a driver is not present")

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

	PurgeInactiveTorrents bool `json:"purge_inactive_torrents"`

	Announce        Duration `json:"announce"`
	MinAnnounce     Duration `json:"min_announce"`
	RequestTimeout  Duration `json:"request_timeout"`
	NumWantFallback int      `json:"default_num_want"`

	PreferredSubnet     bool `json:"preferred_subnet,omitempty"`
	PreferredIPv4Subnet int  `json:"preferred_ipv4_subnet,omitempty"`
	PreferredIPv6Subnet int  `json:"preferred_ipv6_subnet,omitempty"`
}

// DefaultConfig is a configuration that can be used as a fallback value.
var DefaultConfig = Config{
	Addr: "127.0.0.1:6881",

	Tracker: DriverConfig{
		Name: "memory",
	},

	Backend: DriverConfig{
		Name: "noop",
	},

	Private:   false,
	Freeleech: false,
	Whitelist: false,

	PurgeInactiveTorrents: true,

	Announce:        Duration{30 * time.Minute},
	MinAnnounce:     Duration{15 * time.Minute},
	RequestTimeout:  Duration{10 * time.Second},
	NumWantFallback: 50,
}

// Open is a shortcut to open a file, read it, and generate a Config.
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

	conf, err := Decode(f)
	if err != nil {
		return nil, err
	}
	return conf, nil
}

// Decode casts an io.Reader into a JSONDecoder and decodes it into a *Config.
func Decode(r io.Reader) (*Config, error) {
	conf := &Config{}
	err := json.NewDecoder(r).Decode(conf)
	if err != nil {
		return nil, err
	}
	return conf, nil
}
