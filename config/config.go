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

// DataStore represents the configuration used to connect to a data store.
type DataStore struct {
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

// Config represents a configuration for a server.Server.
type Config struct {
	Addr    string    `json:"addr"`
	Tracker DataStore `json:"tracker"`
	Backend DataStore `json:"backend"`

	Private   bool `json:"private"`
	Freeleech bool `json:"freeleech"`

	Announce       Duration `json:"announce"`
	MinAnnounce    Duration `json:"min_announce"`
	ReadTimeout    Duration `json:"read_timeout"`
	DefaultNumWant int      `json:"default_num_want"`
}

// Open is a shortcut to open a file, read it, and generate a Config.
// It supports relative and absolute paths.
func Open(path string) (*Config, error) {
	f, err := os.Open(os.ExpandEnv(path))
	if err != nil {
		return nil, err
	}
	defer f.Close()

	conf, err := newConfig(f)
	if err != nil {
		return nil, err
	}
	return conf, nil
}

// New decodes JSON from a Reader into a Config.
func newConfig(raw io.Reader) (*Config, error) {
	conf := &Config{}
	err := json.NewDecoder(raw).Decode(conf)
	if err != nil {
		return nil, err
	}
	return conf, nil
}
