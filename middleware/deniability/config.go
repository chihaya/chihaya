// Copyright 2016 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

package deniability

import (
	"gopkg.in/yaml.v2"

	"github.com/chihaya/chihaya"
)

// Config represents the configuration for the deniability middleware.
type Config struct {
	// ModifyResponseProbability is the probability by which a response will
	// be augmented with random peers.
	ModifyResponseProbability float32 `yaml:"modify_response_probability"`

	// MaxRandomPeers is the amount of peers that will be added at most.
	MaxRandomPeers int `yaml:"max_random_peers"`

	// Prefix is the prefix to be used for peer IDs.
	Prefix string `yaml:"prefix"`

	// MinPort is the minimum port (inclusive) for the generated peer.
	MinPort int `yaml:"min_port"`

	// MaxPort is the maximum port (exclusive) for the generated peer.
	MaxPort int `yaml:"max_port"`
}

// newConfig parses the given MiddlewareConfig as a deniability.Config.
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

	return &cfg, nil
}
