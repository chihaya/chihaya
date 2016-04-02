// Copyright 2016 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

package infohash

import (
	"errors"

	"gopkg.in/yaml.v2"

	"github.com/chihaya/chihaya"
)

// ErrUnknownMode is returned by a MiddlewareConstructor if the Mode specified
// in the configuration is unknown.
var ErrUnknownMode = errors.New("unknown mode")

// Mode represents the mode of operation for an infohash scrape middleware.
type Mode string

const (
	// ModeFilter makes the middleware filter disallowed infohashes from a
	// scrape request.
	ModeFilter = Mode("filter")

	// ModeBlock makes the middleware block a scrape request if it contains
	// at least one disallowed infohash.
	ModeBlock = Mode("block")
)

// Config represents the configuration for an infohash scrape middleware.
type Config struct {
	Mode Mode `yaml:"mode"`
}

// newConfig parses the given MiddlewareConfig as an infohash.Config.
// ErrUnknownMode is returned if the mode is unknown.
func newConfig(mwcfg chihaya.MiddlewareConfig) (*Config, error) {
	bytes, err := yaml.Marshal(mwcfg.Config)
	if err != nil {
		return nil, err
	}

	var cfg Config
	err = yaml.Unmarshal(bytes, &cfg)
	if err != nil {
		return nil, err
	}

	if cfg.Mode != ModeBlock && cfg.Mode != ModeFilter {
		return nil, ErrUnknownMode
	}

	return &cfg, nil
}
