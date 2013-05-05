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


const ConfigFileName = "$GOPATH/src/github.com/kotokoko/chihaya/config.json"

var Loaded TrackerConfig

type TrackerDuration struct {
	time.Duration
}

func (d *TrackerDuration) MarshalJSON() ([]byte, error) {
	return json.Marshal(d.String())
}

func (d *TrackerDuration) UnmarshalJSON(b []byte) error {
	var str string
	err := json.Unmarshal(b, &str)
	d.Duration, err = time.ParseDuration(str)
	return err
}

// TrackerIntervals represents the intervals object in a config file.
type TrackerIntervals struct {
	Announce    TrackerDuration `json:"announce"`
	MinAnnounce TrackerDuration `json:"min_announce"`

	DatabaseReload        TrackerDuration `json:"database_reload"`
	DatabaseSerialization TrackerDuration `json:"database_serialization"`
	PurgeInactive         TrackerDuration `json:"purge_inactive"`

	VerifyUsedSlots int64 `json:"verify_used_slots"`

	FlushSleep TrackerDuration `json:"flush_sleep"`

	// Initial wait time before retrying a query when the db deadlocks (ramps linearly)
	DeadlockWait TrackerDuration `json:"deadlock_wait"`
}

// TrackerFlushBufferSizes represents the buffer_sizes object in a config file.
// See github.com/kotokoko/chihaya/database/Database.startFlushing() for more info.
type TrackerFlushBufferSizes struct {
	Torrent         int `json:"torrent"`
	User            int `json:"user"`
	TransferHistory int `json:"transfer_history"`
	TransferIps     int `json:"transfer_ips"`
	Snatch          int `json:"snatch"`
}

// TrackerDatabase represents the database object in a config file.
type TrackerDatabase struct {
	Username string `json:"user"`
	Password string `json:"pass"`
	Database string `json:"database"`
	Proto    string `json:"proto"`
	Addr     string `json:"addr"`
	Encoding string `json:"encoding"`
}

// TrackerConfig represents a whole Chihaya config file.
type TrackerConfig struct {
	Database     TrackerDatabase         `json:"database"`
	Intervals    TrackerIntervals        `json:"intervals"`
	FlushSizes   TrackerFlushBufferSizes `json:"sizes"`
	LogFlushes   bool                    `json:"log_flushes"`
	SlotsEnabled bool                    `json:"slots_enabled"`
	BindAddress  string                  `json:"addr"`

	// When true disregards download. This value is loaded from the database.
	GlobalFreeleech bool `json:"global_freeleach"`

	// Maximum times to retry a deadlocked query before giving up.
	MaxDeadlockRetries int `json:"max_deadlock_retries"`
}

// ReloadConfig loads the config file from the CWD.
func ReloadConfig() (err error) {
	expandedFileName := os.ExpandEnv(ConfigFileName)
	f, err := os.Open(expandedFileName)
	if err != nil {
		return
	}
	defer f.Close()

	err = json.NewDecoder(f).Decode(&Loaded)
	if err != nil {
		return
	}
	return
}

func init() {
	err := ReloadConfig()
	if err != nil {
		log.Fatalf(err.Error())
	}
}
