// Copyright 2016 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

package random

import (
	"math/rand"
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPeer(t *testing.T) {
	r := rand.New(rand.NewSource(0))

	for i := 0; i < 100; i++ {
		minPort := 2000
		maxPort := 2010
		p := Peer(r, "", false, minPort, maxPort)
		assert.Equal(t, 20, len(p.ID))
		assert.True(t, p.Port >= uint16(minPort) && p.Port < uint16(maxPort))
		assert.NotNil(t, p.IP.To4())
	}

	for i := 0; i < 100; i++ {
		minPort := 2000
		maxPort := 2010
		p := Peer(r, "", true, minPort, maxPort)
		assert.Equal(t, 20, len(p.ID))
		assert.True(t, p.Port >= uint16(minPort) && p.Port < uint16(maxPort))
		assert.True(t, len(p.IP) == net.IPv6len)
	}

	p := Peer(r, "abcdefghijklmnopqrst", false, 2000, 2000)
	assert.Equal(t, "abcdefghijklmnopqrst", string(p.ID[:]))
	assert.Equal(t, uint16(2000), p.Port)

	p = Peer(r, "abcdefghijklmnopqrstUVWXYZ", true, -10, -5)
	assert.Equal(t, "abcdefghijklmnopqrst", string(p.ID[:]))
	assert.True(t, p.Port >= uint16(1) && p.Port <= uint16(65535))
}
