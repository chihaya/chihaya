// Copyright 2016 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

package memory

import (
	"testing"

	"github.com/chihaya/chihaya/server/store"
)

var (
	driver            = &stringStoreDriver{}
	stringStoreTester = store.PrepareStringStoreTester(driver)
)

func TestStringStore(t *testing.T) {
	stringStoreTester.TestStringStore(t, &store.DriverConfig{})
}
