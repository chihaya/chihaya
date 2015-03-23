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
	PreferredSubnet     bool `json:"preferred_subnet,omitempty"`
	PreferredIPv4Subnet int  `json:"preferred_ipv4_subnet,omitempty"`
	PreferredIPv6Subnet int  `json:"preferred_ipv6_subnet,omitempty"`
}

// NetConfig is the configuration used to tune networking behaviour.
type NetConfig struct {
	AllowIPSpoofing  bool   `json:"allow_ip_spoofing"`
	DualStackedPeers bool   `json:"dual_stacked_peers"`
	RealIPHeader     string `json:"real_ip_header"`
	RespectAF        bool   `json:"respect_af"`
	SubnetConfig
}

// StatsConfig is the configuration used to record runtime statistics.
type StatsConfig struct {
	BufferSize int  `json:"stats_buffer_size"`
	IncludeMem bool `json:"include_mem_stats"`
	VerboseMem bool `json:"verbose_mem_stats"`

	MemUpdateInterval Duration `json:"mem_stats_interval"`
}

// WhitelistConfig is the configuration used enable and store a whitelist of
// acceptable torrent client peer ID prefixes.
type WhitelistConfig struct {
	ClientWhitelistEnabled bool     `json:"client_whitelist_enabled"`
	ClientWhitelist        []string `json:"client_whitelist,omitempty"`
}

// TrackerConfig is the configuration for tracker functionality.
type TrackerConfig struct {
	PrivateEnabled        bool     `json:"private_enabled"`
	FreeleechEnabled      bool     `json:"freeleech_enabled"`
	PurgeInactiveTorrents bool     `json:"purge_inactive_torrents"`
	Announce              Duration `json:"announce"`
	MinAnnounce           Duration `json:"min_announce"`
	NumWantFallback       int      `json:"default_num_want"`
	TorrentMapShards      int      `json:"torrent_map_shards"`

	NetConfig
	WhitelistConfig
}

// HTTPConfig is the configuration for HTTP functionality.
type HTTPConfig struct {
	ListenAddr       string   `json:"http_listen_addr"`
	RequestTimeout   Duration `json:"http_request_timeout"`
	HTTPReadTimeout  Duration `json:"http_read_timeout"`
	HTTPWriteTimeout Duration `json:"http_write_timeout"`
	HTTPListenLimit  int      `json:"http_listen_limit"`
}

// Config is the global configuration for an instance of Chihaya.
type Config struct {
	TrackerConfig
	HTTPConfig
	DriverConfig
	StatsConfig
}

// DefaultConfig is a configuration that can be used as a fallback value.
var DefaultConfig = Config{
	TrackerConfig: TrackerConfig{
		PrivateEnabled:        false,
		FreeleechEnabled:      false,
		PurgeInactiveTorrents: true,
		Announce:              Duration{30 * time.Minute},
		MinAnnounce:           Duration{15 * time.Minute},
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

	HTTPConfig: HTTPConfig{
		ListenAddr:       ":6881",
		RequestTimeout:   Duration{10 * time.Second},
		HTTPReadTimeout:  Duration{10 * time.Second},
		HTTPWriteTimeout: Duration{10 * time.Second},
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
