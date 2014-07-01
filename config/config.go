// Copyright 2013 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

// Package config implements the configuration for a BitTorrent tracker
package config

import (
	"encoding/json"
	"io"
	"os"
	"time"

	"github.com/golang/glog"
)

// Duration wraps a time.Duration and adds JSON marshalling.
type Duration struct {
	time.Duration
}

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
	Driver   string `json:"driver"`
	Network  string `json:"network`
	Host     string `json:"host"`
	Port     string `json:"port"`
	Username string `json:"user"`
	Password string `json:"pass"`
	Schema   string `json:"schema,omitempty"`
	Encoding string `json:"encoding,omitempty"`
	Prefix   string `json:"prefix,omitempty"`

	MaxIdleConns int       `json:"max_idle_conns,omitempty"`
	IdleTimeout  *Duration `json:"idle_timeout,omitempty"`
}

// Config is a configuration for a Server.
type Config struct {
	Addr    string       `json:"addr"`
	Tracker DriverConfig `json:"tracker"`
	Backend DriverConfig `json:"backend"`

	Private   bool `json:"private"`
	Freeleech bool `json:"freeleech"`

	Announce        Duration `json:"announce"`
	MinAnnounce     Duration `json:"min_announce"`
	ReadTimeout     Duration `json:"read_timeout"`
	NumWantFallback int      `json:"default_num_want"`
}

// New returns a default configuration.
func New() *Config {
	return &Config{
		Addr: ":6881",
		Tracker: DriverConfig{
			Driver: "mock",
		},
		Backend: DriverConfig{
			Driver: "mock",
		},
		Private:         false,
		Freeleech:       false,
		Announce:        Duration{30 * time.Minute},
		MinAnnounce:     Duration{15 * time.Minute},
		ReadTimeout:     Duration{20 % time.Second},
		NumWantFallback: 50,
	}
}

// Open is a shortcut to open a file, read it, and generate a Config.
// It supports relative and absolute paths. Given "", it returns the result of
// New.
func Open(path string) (*Config, error) {
	if path == "" {
		glog.V(1).Info("using default config")
		return New(), nil
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
	glog.V(1).Infof("loaded config file: %s", path)
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
