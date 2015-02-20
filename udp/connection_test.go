// Copyright 2015 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

package udp

import (
	"bytes"
	"net"
	"testing"
)

func TestInitReturnsNoError(t *testing.T) {
	gen := &ConnectionIDGenerator{}
	if err := gen.Init(); err != nil {
		t.Error("Init returned", err)
	}
}

func testGenerateConnectionID(t *testing.T, ip net.IP) {
	gen := &ConnectionIDGenerator{}
	gen.Init()

	id1 := gen.Generate(ip)
	id2 := gen.Generate(ip)

	if !bytes.Equal(id1, id2) {
		t.Errorf("Connection ID mismatch: %x != %x", id1, id2)
	}

	if len(id1) != 8 {
		t.Errorf("Connection ID had length: %d != 8", len(id1))
	}

	if bytes.Count(id1, []byte{0}) == 8 {
		t.Errorf("Connection ID was 0")
	}
}

func TestGenerateConnectionIDIPv4(t *testing.T) {
	testGenerateConnectionID(t, net.ParseIP("192.168.1.123").To4())
}

func TestGenerateConnectionIDIPv6(t *testing.T) {
	testGenerateConnectionID(t, net.ParseIP("1:2:3:4::5:6"))
}

func TestMatchesWorksWithPreviousIV(t *testing.T) {
	gen := &ConnectionIDGenerator{}
	gen.Init()
	ip := net.ParseIP("192.168.1.123").To4()

	id1 := gen.Generate(ip)
	if !gen.Matches(id1, ip) {
		t.Errorf("Connection ID mismatch for current IV")
	}

	gen.NewIV()
	if !gen.Matches(id1, ip) {
		t.Errorf("Connection ID mismatch for previous IV")
	}

	id2 := gen.Generate(ip)
	gen.NewIV()

	if gen.Matches(id1, ip) {
		t.Errorf("Connection ID matched for discarded IV")
	}

	if !gen.Matches(id2, ip) {
		t.Errorf("Connection ID mismatch for previous IV")
	}
}
