// Copyright 2016 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

// Package event implements type-level constraints for dealing with the events
// communicated via BitTorrent announce.
package event

import (
	"errors"
	"strings"
)

// ErrUnknownEvent is returned when New fails to return an event.
var ErrUnknownEvent = errors.New("unknown event")

// Event represents an event done by a BitTorrent client.
type Event uint8

const (
	// None is the event when a BitTorrent client announces due to time lapsed
	// since the previous announce.
	None Event = iota

	// Started is the event sent by a BitTorrent client when it joins a swarm.
	Started

	// Stopped is the event sent by a BitTorrent client when it leaves a swarm.
	Stopped

	// Completed is the event sent by a BitTorrent client when it finishes
	// downloading all of the required chunks.
	Completed
)

var (
	eventToString = make(map[Event]string)
	stringToEvent = make(map[string]Event)
)

func init() {
	eventToString[None] = "none"
	eventToString[Started] = "started"
	eventToString[Stopped] = "stopped"
	eventToString[Completed] = "completed"

	for k, v := range eventToString {
		stringToEvent[v] = k
	}
}

// New returns the proper Event given a string.
func New(eventStr string) (Event, error) {
	if e, ok := stringToEvent[strings.ToLower(eventStr)]; ok {
		return e, nil
	}

	return None, ErrUnknownEvent
}

// String implements Stringer for an event.
func (e Event) String() string {
	if name, ok := eventToString[e]; ok {
		return name
	}

	panic("event: event has no associated name")
}
