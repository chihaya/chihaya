// Copyright 2016 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

// Package server implements an abstraction over servers meant to be run       .
// alongside a tracker.
//
// Servers may be implementations of different transport protocols or have their
// own custom behavior.
package server

import (
	"fmt"

	"github.com/chihaya/chihaya"
	"github.com/chihaya/chihaya/tracker"
)

var constructors = make(map[string]Constructor)

// Constructor is a function that creates a new Server.
type Constructor func(*chihaya.ServerConfig, *tracker.Tracker) (Server, error)

// Register makes a Constructor available by the provided name.
//
// If this function is called twice with the same name or if the Constructor is
// nil, it panics.
func Register(name string, con Constructor) {
	if con == nil {
		panic("server: could not register nil Constructor")
	}
	if _, dup := constructors[name]; dup {
		panic("server: could not register duplicate Constructor: " + name)
	}
	constructors[name] = con
}

// New creates a Server specified by a configuration.
func New(cfg *chihaya.ServerConfig, tkr *tracker.Tracker) (Server, error) {
	con, ok := constructors[cfg.Name]
	if !ok {
		return nil, fmt.Errorf("server: unknown Constructor %q (forgotten import?)", cfg.Name)
	}
	return con(cfg, tkr)
}

// Server represents one instance of a server accessing the tracker.
type Server interface {
	Start()
	Stop()
}
