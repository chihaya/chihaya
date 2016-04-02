// Copyright 2016 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

package random

import (
	"math/rand"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAlphaNumericString(t *testing.T) {
	r := rand.NewSource(0)

	s := AlphaNumericString(r, 0)
	assert.Equal(t, 0, len(s))

	s = AlphaNumericString(r, 10)
	assert.Equal(t, 10, len(s))

	for i := 0; i < 100; i++ {
		s := AlphaNumericString(r, 10)
		for _, c := range s {
			assert.True(t, strings.Contains(AlphaNumeric, string(c)))
		}
	}
}
