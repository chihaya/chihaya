// Copyright 2016 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

package infohash

import (
	"fmt"
	"testing"

	"gopkg.in/yaml.v2"

	"github.com/chihaya/chihaya"
	"github.com/stretchr/testify/assert"
)

var (
	configTemplate = `name: foo
config:
  %s: %s`

	data = []testData{
		{"mode", "block", false, ModeBlock},
		{"mode", "filter", false, ModeFilter},
		{"some", "stuff", true, ModeBlock},
	}
)

type testData struct {
	key      string
	value    string
	err      bool
	expected Mode
}

func TestNewConfig(t *testing.T) {
	var mwconfig chihaya.MiddlewareConfig

	cfg, err := newConfig(mwconfig)
	assert.NotNil(t, err)
	assert.Nil(t, cfg)

	for _, test := range data {
		config := fmt.Sprintf(configTemplate, test.key, test.value)
		err = yaml.Unmarshal([]byte(config), &mwconfig)
		assert.Nil(t, err)

		cfg, err = newConfig(mwconfig)
		if test.err {
			assert.NotNil(t, err)
			continue
		}
		assert.Nil(t, err)
		assert.Equal(t, test.expected, cfg.Mode)
	}
}
