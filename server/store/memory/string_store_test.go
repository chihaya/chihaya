// Copyright 2016 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

package memory

import (
	"github.com/chihaya/chihaya/server/store"
	"github.com/stretchr/testify/assert"
	"testing"
)

var (
	driver = &stringStoreDriver{}
	s1     = "abc"
	s2     = "def"
)

func TestStringStore(t *testing.T) {
	ss, err := driver.New(&store.DriverConfig{})
	assert.Nil(t, err)
	assert.NotNil(t, ss)

	has, err := ss.HasString(s1)
	assert.Nil(t, err)
	assert.False(t, has)

	has, err = ss.HasString(s2)
	assert.Nil(t, err)
	assert.False(t, has)

	err = ss.RemoveString(s1)
	assert.Nil(t, err)

	err = ss.PutString(s1)
	assert.Nil(t, err)

	has, err = ss.HasString(s1)
	assert.Nil(t, err)
	assert.True(t, has)

	has, err = ss.HasString(s2)
	assert.Nil(t, err)
	assert.False(t, has)

	err = ss.PutString(s1)
	assert.Nil(t, err)

	err = ss.PutString(s2)
	assert.Nil(t, err)

	has, err = ss.HasString(s1)
	assert.Nil(t, err)
	assert.True(t, has)

	has, err = ss.HasString(s2)
	assert.Nil(t, err)
	assert.True(t, has)

	err = ss.RemoveString(s1)
	assert.Nil(t, err)

	err = ss.RemoveString(s2)
	assert.Nil(t, err)

	has, err = ss.HasString(s1)
	assert.Nil(t, err)
	assert.False(t, has)

	has, err = ss.HasString(s2)
	assert.Nil(t, err)
	assert.False(t, has)
}
