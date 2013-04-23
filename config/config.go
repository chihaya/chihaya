// Copyright 2013 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

// Package config implements the configuration and loading of Chihaya configuration files.
package config

import (
	"encoding/json"
	"log"
	"os"
	"time"
)

const ConfigFileName = "config.json"

var (
	// TrackerDatabase represents the database object in a config file.
	Database trackerDatabase

	// TrackerIntervals represents the intervals object in a config file.
	Intervals trackerIntervals

	// TrackerFlushBufferSizes represents the buffer_sizes object in a config file.
	// See github.com/kotokoko/chihaya/database/Database.startFlushing() for more info.
	FlushSizes trackerFlushBufferSizes

	LogFlushes bool

	SlotsEnabled bool

	BindAddress string

	// When true disregards download. This value is loaded from the database.
	GlobalFreeleech bool

	// Maximum times to retry a deadlocked query before giving up.
	MaxDeadlockRetries int
)

type trackerDuration struct {
	time.Duration
}

func (d *trackerDuration) MarshalJSON() ([]byte, error) {
	return json.Marshal(d.String())
}

func (d *trackerDuration) UnmarshalJSON(b []byte) error {
	var str string
	err := json.Unmarshal(b, &str)
	d.Duration, err = time.ParseDuration(str)
	return err
}

// TrackerIntervals represents the intervals object in a config file.
type trackerIntervals struct {
	Announce    trackerDuration `json:"announce"`
	MinAnnounce trackerDuration `json:"min_announce"`

	DatabaseReload        trackerDuration `json:"database_reload"`
	DatabaseSerialization trackerDuration `json:"database_serialization"`
	PurgeInactive         trackerDuration `json:"purge_inactive"`

	VerifyUsedSlots int64 `json:"verify_used_slots"`

	FlushSleep trackerDuration `json:"flush_sleep"`

	// Initial wait time before retrying a query when the db deadlocks (ramps linearly)
	DeadlockWait trackerDuration `json:"deadlock_wait"`
}

// TrackerFlushBufferSizes represents the buffer_sizes object in a config file.
// See github.com/kotokoko/chihaya/database/Database.startFlushing() for more info.
type trackerFlushBufferSizes struct {
	Torrent         int `json:"torrent"`
	User            int `json:"user"`
	TransferHistory int `json:"transfer_history"`
	TransferIps     int `json:"transfer_ips"`
	Snatch          int `json:"snatch"`
}

// TrackerDatabase represents the database object in a config file.
type trackerDatabase struct {
	Username string `json:"user"`
	Password string `json:"pass"`
	Database string `json:"database"`
	Proto    string `json:"proto"`
	Addr     string `json:"addr"`
	Encoding string `json:"encoding"`
}

// TrackerConfig represents a whole Chihaya config file.
type trackerConfig struct {
	Database     trackerDatabase         `json:"database"`
	Intervals    trackerIntervals        `json:"intervals"`
	FlushSizes   trackerFlushBufferSizes `json:"sizes"`
	LogFlushes   bool                    `json:"log_flushes"`
	SlotsEnabled bool                    `json:"slots_enabled"`
	BindAddress  string                  `json:"addr"`

	// When true disregards download. This value is loaded from the database.
	GlobalFreeleech bool `json:"global_freeleach"`

	// Maximum times to retry a deadlocked query before giving up.
	MaxDeadlockRetries int `json:"max_deadlock_retries"`
}

// loadConfig loads a configuration and exits if there is a failure.
func loadConfig() {
	f, err := os.Open(ConfigFileName)
	if err != nil {
		log.Fatalf("Error opening config file: %s", err)
		return
	}
	defer f.Close()

	config := &trackerConfig{}
	err = json.NewDecoder(f).Decode(&config)
	if err != nil {
		log.Fatalf("Error parsing config file: %s", err)
		return
	}

	Database = config.Database
	Intervals = config.Intervals
	FlushSizes = config.FlushSizes
	LogFlushes = config.LogFlushes
	SlotsEnabled = config.SlotsEnabled
	BindAddress = config.BindAddress
	GlobalFreeleech = config.GlobalFreeleech
	MaxDeadlockRetries = config.MaxDeadlockRetries
}

func init() {
	loadConfig()
}
