// Copyright 2016 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

package varinterval

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/chihaya/chihaya"
	"github.com/chihaya/chihaya/tracker"
)

type constructorTestData struct {
	cfg   Config
	error bool
}

var constructorData = []constructorTestData{
	{Config{1.0, 10, false}, false},
	{Config{1.1, 10, false}, true},
	{Config{0, 10, true}, true},
	{Config{1.0, 0, false}, true},
}

func TestConstructor(t *testing.T) {
	for _, tt := range constructorData {
		_, err := constructor(chihaya.MiddlewareConfig{
			Config: tt.cfg,
		})

		if tt.error {
			assert.NotNil(t, err, fmt.Sprintf("error expected for %+v", tt.cfg))
		} else {
			assert.Nil(t, err, fmt.Sprintf("no error expected for %+v", tt.cfg))
		}
	}
}

func TestModifyResponse(t *testing.T) {
	var (
		achain tracker.AnnounceChain
		req chihaya.AnnounceRequest
		resp chihaya.AnnounceResponse
	)

	mw, err := constructor(chihaya.MiddlewareConfig{
		Config: Config{
			ModifyResponseProbability: 1.0,
			MaxIncreaseDelta:          10,
			ModifyMinInterval:         true,
		},
	})
	assert.Nil(t, err)

	achain.Append(mw)
	handler := achain.Handler()

	err = handler(nil, &req, &resp)
	assert.Nil(t, err)
	assert.True(t, resp.Interval > 0, "interval should have been increased")
	assert.True(t, resp.MinInterval > 0, "min_interval should have been increased")
}
