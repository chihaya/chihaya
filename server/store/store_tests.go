// Copyright 2016 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

package store

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// StringStoreTester is a collection of tests for a StringStore driver.
// Every benchmark expects a new, clean storage. Every benchmark should be
// called with a DriverConfig that ensures this.
type StringStoreTester interface {
	TestStringStore(*testing.T, *DriverConfig)
}

var _ StringStoreTester = &stringStoreTester{}

type stringStoreTester struct {
	s1, s2 string
	driver StringStoreDriver
}

// PrepareStringStoreTester prepares a reusable suite for StringStore driver
// tests.
func PrepareStringStoreTester(driver StringStoreDriver) StringStoreTester {
	return &stringStoreTester{
		s1:     "abc",
		s2:     "def",
		driver: driver,
	}
}

func (s *stringStoreTester) TestStringStore(t *testing.T, cfg *DriverConfig) {
	ss, err := s.driver.New(cfg)
	require.Nil(t, err)
	require.NotNil(t, ss)

	has, err := ss.HasString(s.s1)
	require.Nil(t, err)
	require.False(t, has)

	has, err = ss.HasString(s.s2)
	require.Nil(t, err)
	require.False(t, has)

	err = ss.RemoveString(s.s1)
	require.NotNil(t, err)

	err = ss.PutString(s.s1)
	require.Nil(t, err)

	has, err = ss.HasString(s.s1)
	require.Nil(t, err)
	require.True(t, has)

	has, err = ss.HasString(s.s2)
	require.Nil(t, err)
	require.False(t, has)

	err = ss.PutString(s.s1)
	require.Nil(t, err)

	err = ss.PutString(s.s2)
	require.Nil(t, err)

	has, err = ss.HasString(s.s1)
	require.Nil(t, err)
	require.True(t, has)

	has, err = ss.HasString(s.s2)
	require.Nil(t, err)
	require.True(t, has)

	err = ss.RemoveString(s.s1)
	require.Nil(t, err)

	err = ss.RemoveString(s.s2)
	require.Nil(t, err)

	has, err = ss.HasString(s.s1)
	require.Nil(t, err)
	require.False(t, has)

	has, err = ss.HasString(s.s2)
	require.Nil(t, err)
	require.False(t, has)

	errChan := ss.Stop()
	err = <-errChan
	require.Nil(t, err, "StringStore shutdown must not fail")
}
