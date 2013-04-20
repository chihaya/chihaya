// This file is part of Chihaya.
//
// Chihaya is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// Chihaya is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with Chihaya.  If not, see <http://www.gnu.org/licenses/>.

// Package config implements the configuration and loading of Chihaya configuration files.
package config

import (
	"encoding/json"
	"log"
	"os"
	"time"
)

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

type TrackerIntervals struct {
	Announce    TrackerDuration `json:"announce"`
	MinAnnounce TrackerDuration `json:"min_announce"`

	DatabaseReload        TrackerDuration `json:"database_reload"`
	DatabaseSerialization TrackerDuration `json:"database_serialization"`
	PurgeInactive         TrackerDuration `json:"purge_inactive"`

	VerifyUsedSlots int64 `json:"verify_used_slots"`

	FlushSleep TrackerDuration `json:"flush_sleep"`

	// Initial time to wait before retrying the query when the database deadlocks (ramps linearly)
	DeadlockWait TrackerDuration `json:"deadlock_wait"`
}

// Buffer sizes, see @Database.startFlushing()
type TrackerFlushBufferSizes struct {
	Torrent         int `json:"torrent"`
	User            int `json:"user"`
	TransferHistory int `json:"transfer_history"`
	TransferIps     int `json:"transfer_ips"`
	Snatch          int `json:"snatch"`
}

type TrackerDatabase struct {
	Username string `json:"user"`
	Password string `json:"pass"`
	Database string `json:"database"`
	Proto    string `json:"proto"`
	Addr     string `json:"addr"`
	Encoding string `json:"encoding"`
}

type TrackerConfig struct {
	Database           TrackerDatabase         `json:"database"`
	GlobalFreeleech    bool                    `json:"global_freeleach"` // Loaded from the database
	Intervals          TrackerIntervals        `json:"intervals"`
	FlushSizes         TrackerFlushBufferSizes `json:"sizes"`
	MaxDeadlockRetries int                     `json:"max_deadlock_retries"` // Maximum times to retry a deadlocked query before giving up
	LogFlushes         bool                    `json:"log_flushes"`
	SlotsEnabled       bool                    `json:"slots_enabled"`
	BindAddress        string                  `json:"addr"`
}

var Config TrackerConfig

func ReadConfig(configFile string) {
	f, err := os.Open(configFile)

	if err != nil {
		log.Fatalf("Error opening config file: %s", err)
		return
	}

	err = json.NewDecoder(f).Decode(&Config)

	if err != nil {
		log.Fatalf("Error parsing config file: %s", err)
		return
	}
}
