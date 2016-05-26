// Copyright 2016 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

package bolt

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/chihaya/chihaya/server/store"
)

func TestNewBoltConfig(t *testing.T) {
	cfg, err := newBoltConfig(&store.DriverConfig{Config: boltConfig{File: "a"}})
	require.Nil(t, err)
	require.NotNil(t, cfg)

	cfg, err = newBoltConfig(nil)
	require.Equal(t, ErrMissingConfig, err)
	require.Nil(t, cfg)

	cfg, err = newBoltConfig(&store.DriverConfig{})
	require.Equal(t, ErrMissingConfig, err)
	require.Nil(t, cfg)

	cfg, err = newBoltConfig(&store.DriverConfig{Config: nil})
	require.Equal(t, ErrMissingConfig, err)
	require.Nil(t, cfg)

	cfg, err = newBoltConfig(&store.DriverConfig{Config: boltConfig{}})
	require.Equal(t, ErrMissingFile, err)
	require.Nil(t, cfg)

}
