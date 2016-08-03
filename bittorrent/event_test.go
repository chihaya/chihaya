// Copyright 2016 Jimmy Zelinskie
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package bittorrent

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNew(t *testing.T) {
	var table = []struct {
		data        string
		expected    Event
		expectedErr error
	}{
		{"", None, nil},
		{"NONE", None, nil},
		{"none", None, nil},
		{"started", Started, nil},
		{"stopped", Stopped, nil},
		{"completed", Completed, nil},
		{"notAnEvent", None, ErrUnknownEvent},
	}

	for _, tt := range table {
		got, err := NewEvent(tt.data)
		assert.Equal(t, err, tt.expectedErr, "errors should equal the expected value")
		assert.Equal(t, got, tt.expected, "events should equal the expected value")
	}
}
