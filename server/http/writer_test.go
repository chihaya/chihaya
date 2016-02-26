// Copyright 2016 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

package http

import (
	"net/http/httptest"
	"testing"

	"github.com/chihaya/chihaya/tracker"
	"github.com/stretchr/testify/assert"
)

func TestWriteError(t *testing.T) {
	var table = []struct {
		reason, expected string
	}{
		{"hello world", "d14:failure reason11:hello worlde"},
		{"what's up", "d14:failure reason9:what's upe"},
	}

	for _, tt := range table {
		r := httptest.NewRecorder()
		err := writeError(r, tracker.ClientError(tt.reason))
		assert.Nil(t, err)
		assert.Equal(t, r.Body.String(), tt.expected)
	}
}

func TestWriteStatus(t *testing.T) {
	r := httptest.NewRecorder()
	err := writeError(r, tracker.ClientError("something is missing"))
	assert.Nil(t, err)
	assert.Equal(t, r.Body.String(), "d14:failure reason20:something is missinge")
}
