// Copyright 2015 The Chihaya Authors. All rights reserved.
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

// SubnetConfig is the configuration used to specify if local peers should be
// given a preference when responding to an announce.
type SubnetConfig struct {
	PreferredSubnet     bool `json:"preferredSubnet,omitempty"`
	PreferredIPv4Subnet int  `json:"preferredIPv4Subnet,omitempty"`
	PreferredIPv6Subnet int  `json:"preferredIPv6Subnet,omitempty"`
}

// NetConfig is the configuration used to tune networking behaviour.
type NetConfig struct {
	AllowIPSpoofing  bool   `json:"allowIPSpoofing"`
	DualStackedPeers bool   `json:"dualStackedPeers"`
	RealIPHeader     string `json:"realIPHeader"`
	RespectAF        bool   `json:"respectAF"`
	SubnetConfig
}

// StatsConfig is the configuration used to record runtime statistics.
type StatsConfig struct {
	BufferSize        int      `json:"statsBufferSize"`
	IncludeMem        bool     `json:"includeMemStats"`
	VerboseMem        bool     `json:"verboseMemStats"`
	MemUpdateInterval Duration `json:"memStatsInterval"`
}

// WhitelistConfig is the configuration used enable and store a whitelist of
// acceptable torrent client peer ID prefixes.
type WhitelistConfig struct {
	ClientWhitelistEnabled bool     `json:"clientWhitelistEnabled"`
	ClientWhitelist        []string `json:"clientWhitelist,omitempty"`
}

// TrackerConfig is the configuration for tracker functionality.
type TrackerConfig struct {
	CreateOnAnnounce      bool     `json:"createOnAnnounce"`
	PrivateEnabled        bool     `json:"privateEnabled"`
	FreeleechEnabled      bool     `json:"freeleechEnabled"`
	PurgeInactiveTorrents bool     `json:"purgeInactiveTorrents"`
	Announce              Duration `json:"announce"`
	MinAnnounce           Duration `json:"minAnnounce"`
	ReapInterval          Duration `json:"reapInterval"`
	ReapRatio             float64  `json:"reapRatio"`
	NumWantFallback       int      `json:"defaultNumWant"`
	TorrentMapShards      int      `json:"torrentMapShards"`

	NetConfig
	WhitelistConfig
}

// APIConfig is the configuration for an HTTP JSON API server.
type APIConfig struct {
	ListenAddr     string   `json:"apiListenAddr"`
	RequestTimeout Duration `json:"apiRequestTimeout"`
	ReadTimeout    Duration `json:"apiReadTimeout"`
	WriteTimeout   Duration `json:"apiWriteTimeout"`
	ListenLimit    int      `json:"apiListenLimit"`
}

// HTTPConfig is the configuration for the HTTP protocol.
type HTTPConfig struct {
	ListenAddr     string   `json:"httpListenAddr"`
	RequestTimeout Duration `json:"httpRequestTimeout"`
	ReadTimeout    Duration `json:"httpReadTimeout"`
	WriteTimeout   Duration `json:"httpWriteTimeout"`
	ListenLimit    int      `json:"httpListenLimit"`
}

// UDPConfig is the configuration for the UDP protocol.
type UDPConfig struct {
	ListenAddr     string `json:"udpListenAddr"`
	ReadBufferSize int    `json:"udpReadBufferSize"`
}

// Config is the global configuration for an instance of Chihaya.
type Config struct {
	TrackerConfig
	APIConfig
	HTTPConfig
	UDPConfig
	DriverConfig
	StatsConfig
}

// DefaultConfig is a configuration that can be used as a fallback value.
var DefaultConfig = Config{
	TrackerConfig: TrackerConfig{
		CreateOnAnnounce:      true,
		PrivateEnabled:        false,
		FreeleechEnabled:      false,
		PurgeInactiveTorrents: true,
		Announce:              Duration{30 * time.Minute},
		MinAnnounce:           Duration{15 * time.Minute},
		ReapInterval:          Duration{60 * time.Second},
		ReapRatio:             1.25,
		NumWantFallback:       50,
		TorrentMapShards:      1,

		NetConfig: NetConfig{
			AllowIPSpoofing:  true,
			DualStackedPeers: true,
			RespectAF:        false,
		},

		WhitelistConfig: WhitelistConfig{
			ClientWhitelistEnabled: false,
		},
	},

	APIConfig: APIConfig{
		ListenAddr:     ":6880",
		RequestTimeout: Duration{10 * time.Second},
		ReadTimeout:    Duration{10 * time.Second},
		WriteTimeout:   Duration{10 * time.Second},
	},

	HTTPConfig: HTTPConfig{
		ListenAddr:     ":6881",
		RequestTimeout: Duration{10 * time.Second},
		ReadTimeout:    Duration{10 * time.Second},
		WriteTimeout:   Duration{10 * time.Second},
	},

	UDPConfig: UDPConfig{
		ListenAddr: ":6882",
	},

	DriverConfig: DriverConfig{
		Name: "noop",
	},

	StatsConfig: StatsConfig{
		BufferSize: 0,
		IncludeMem: true,
		VerboseMem: false,

		MemUpdateInterval: Duration{5 * time.Second},
	},
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
	conf := DefaultConfig
	err := json.NewDecoder(r).Decode(&conf)
	return &conf, err
}
