// Copyright 2016 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

package deniability

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v2"

	"github.com/chihaya/chihaya"
)

type configTestData struct {
	modifyProbability string
	maxNewPeers       string
	prefix            string
	minPort           string
	maxPort           string
	err               bool
	expected          Config
}

var (
	configTemplate = `
name: foo
config:
  modify_response_probability: %s
  max_random_peers: %s
  prefix: %s
  min_port: %s
  max_port: %s`

	configData = []configTestData{
		{"1.0", "5", "abc", "2000", "3000", false, Config{1.0, 5, "abc", 2000, 3000}},
		{"a", "a", "12", "a", "a", true, Config{}},
	}
)

func TestNewConfig(t *testing.T) {
	var mwconfig chihaya.MiddlewareConfig

	cfg, err := newConfig(mwconfig)
	assert.Nil(t, err)
	assert.NotNil(t, cfg)

	for _, test := range configData {
		config := fmt.Sprintf(configTemplate, test.modifyProbability, test.maxNewPeers, test.prefix, test.minPort, test.maxPort)
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
