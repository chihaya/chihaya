// Copyright 2016 Jimmy Zelinskie
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package trakr implements a BitTorrent Tracker that supports multiple
// protocols and configurable Hooks that execute before and after a Response
// has been delievered to a BitTorrent client.
package trakr

// MultiTracker is a multi-protocol, customizable BitTorrent Tracker.
type MultiTracker struct {
	HTTPConfig       http.Config
	UDPConfig        udp.Config
	AnnounceInterval time.Duration
	GCInterval       time.Duration
	GCExpiration     time.Duration
	PreHooks         []Hook
	PostHooks        []Hook

	httpTracker http.Tracker
	udpTracker  udp.Tracker
}

// ListenAndServe listens on the protocols and addresses specified in the
// HTTPConfig and UDPConfig then blocks serving BitTorrent requests until
// t.Stop() is called or an error is returned.
func (t *MultiTracker) ListenAndServe() error {
}
