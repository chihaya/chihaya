// Copyright 2016 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

package http

import (
	"errors"
	"net/http/httptest"
	"testing"

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
		w := &writer{r}
		err := w.writeError(errors.New(tt.reason))
		assert.Nil(t, err, "writeError should not fail with test input")
		assert.Equal(t, r.Body.String(), tt.expected, "writer should write the expected value")
	}
}
