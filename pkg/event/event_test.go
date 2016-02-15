// Copyright 2016 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

package event

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNew(t *testing.T) {
	var table = []struct {
		data        string
		expected    event
		expectedErr error
	}{
		{"", None, ErrUnknownEvent},
		{"NONE", None, nil},
		{"none", None, nil},
		{"started", Started, nil},
		{"stopped", Stopped, nil},
		{"completed", Completed, nil},
		{"notAnEvent", None, ErrUnknownEvent},
	}

	for _, tt := range table {
		got, err := New(tt.data)
		assert.Equal(t, err, tt.expectedErr, "errors should equal the expected value")
		assert.Equal(t, got, tt.expected, "events should equal the expected value")
	}
}
