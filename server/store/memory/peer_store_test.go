// Copyright 2016 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

package memory

import (
	"testing"

	"github.com/chihaya/chihaya/server/store"
)

var (
	peerStoreTester     = store.PreparePeerStoreTester(&peerStoreDriver{})
	peerStoreTestConfig = &store.DriverConfig{}
)

func init() {
	unmarshalledConfig := struct {
		Shards int
	}{
		1,
	}
	peerStoreTestConfig.Config = unmarshalledConfig
}

func TestPeerStore(t *testing.T) {
	peerStoreTester.TestPeerStore(t, peerStoreTestConfig)
}
