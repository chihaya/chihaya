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

package http

import (
	"net/http/httptest"
	"testing"

	"github.com/jzelinskie/trakr/bittorrent"
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
		err := writeError(r, bittorrent.ClientError(tt.reason))
		assert.Nil(t, err)
		assert.Equal(t, r.Body.String(), tt.expected)
	}
}

func TestWriteStatus(t *testing.T) {
	r := httptest.NewRecorder()
	err := writeError(r, bittorrent.ClientError("something is missing"))
	assert.Nil(t, err)
	assert.Equal(t, r.Body.String(), "d14:failure reason20:something is missinge")
}
