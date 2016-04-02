// Copyright 2016 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

package varinterval

import (
	"gopkg.in/yaml.v2"

	"github.com/chihaya/chihaya"
)

// Config represents the configuration for the varinterval middleware.
type Config struct {
	// ModifyResponseProbability is the probability by which a response will
	// be modified.
	ModifyResponseProbability float32 `yaml:"modify_response_probability"`

	// MaxIncreaseDelta is the amount of seconds that will be added at most.
	MaxIncreaseDelta int `yaml:"max_increase_delta"`

	// ModifyMinInterval specifies whether min_interval should be increased
	// as well.
	ModifyMinInterval bool `yaml:"modify_min_interval"`
}

// newConfig parses the given MiddlewareConfig as a varinterval.Config.
//
// The contents of the config are not checked.
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
