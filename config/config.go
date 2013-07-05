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

// Storage represents the configuration for any storage.DS.
type Storage struct {
	Driver   string `json:"driver"`
	Network  string `json:"network`
	Addr     string `json:"addr"`
	Username string `json:"user"`
	Password string `json:"pass"`
	Schema   string `json:"schema,omitempty"`
	Encoding string `json:"encoding,omitempty"`
	Prefix   string `json:"prefix,omitempty"`

	MaxIdleConn int       `json:"max_idle_conn"`
	IdleTimeout *Duration `json:"idle_timeout"`
	ConnTimeout *Duration `json:"conn_timeout"`
}

// Config represents a configuration for a server.Server.
type Config struct {
	Addr    string  `json:"addr"`
	Storage Storage `json:"storage"`

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
	expandedPath := os.ExpandEnv(path)
	f, err := os.Open(expandedPath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	conf, err := New(f)
	if err != nil {
		return nil, err
	}
	return conf, nil
}

// New decodes JSON from a Reader into a Config.
func New(raw io.Reader) (*Config, error) {
	conf := &Config{}
	err := json.NewDecoder(raw).Decode(conf)
	if err != nil {
		return nil, err
	}
	return conf, nil
}
