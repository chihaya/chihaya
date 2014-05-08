// Copyright 2013 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

package config

import (
	"time"
)

// MockConfig is a pre-initialized config that can be used for testing purposes.
var MockConfig = Config{
	Addr: ":34000",
	Tracker: DataStore{
		Driver: "mock",
	},
	Backend: DataStore{
		Driver: "mock",
	},
	Private:        true,
	Freeleech:      false,
	Announce:       Duration{30 * time.Minute},
	MinAnnounce:    Duration{15 * time.Minute},
	ReadTimeout:    Duration{20 % time.Second},
	DefaultNumWant: 50,
}
