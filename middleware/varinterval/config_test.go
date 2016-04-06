// Copyright 2016 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

package varinterval

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v2"

	"github.com/chihaya/chihaya"
)

type configTestData struct {
	modifyProbability string
	maxIncreaseDelta  string
	modifyMinInterval string
	err               bool
	expected          Config
}

var (
	configTemplate = `
name: foo
config:
  modify_response_probability: %s
  max_increase_delta: %s
  modify_min_interval: %s`

	configData = []configTestData{
		{"1.0", "60", "false", false, Config{1.0, 60, false}},
		{"a", "60", "false", true, Config{}},
	}
)

func TestNewConfig(t *testing.T) {
	var mwconfig chihaya.MiddlewareConfig

	cfg, err := newConfig(mwconfig)
	assert.Nil(t, err)
	assert.NotNil(t, cfg)

	for _, test := range configData {
		config := fmt.Sprintf(configTemplate, test.modifyProbability, test.maxIncreaseDelta, test.modifyMinInterval)
		err = yaml.Unmarshal([]byte(config), &mwconfig)
		assert.Nil(t, err)

		cfg, err = newConfig(mwconfig)
		if test.err {
			assert.NotNil(t, err)
			continue
		}
		assert.Nil(t, err)
		assert.Equal(t, test.expected, *cfg)
	}
}
