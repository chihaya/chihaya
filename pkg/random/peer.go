// Copyright 2016 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

package random

import (
	"math/rand"
	"net"

	"github.com/chihaya/chihaya"
)

// Peer generates a random chihaya.Peer.
//
// prefix is the prefix to use for the peer ID. If len(prefix) > 20, it will be
// truncated to 20 characters. If len(prefix) < 20, it will be padded with an
// alphanumeric random string to have 20 characters.
//
// v6 indicates whether an IPv6 address should be generated.
// Regardless of the length of the generated IP address, its bytes will have
// values in [1,254].
//
// minPort and maxPort describe the range for the randomly generated port, where
// minPort <= port < maxPort.
// minPort and maxPort will be checked and altered so that
// 1 <= minPort <= maxPort <= 65536.
// If minPort == maxPort, port will be set to minPort.
func Peer(r *rand.Rand, prefix string, v6 bool, minPort, maxPort int) chihaya.Peer {
	var (
		port uint16
		ip   net.IP
	)

	if minPort <= 0 {
		minPort = 1
	}
	if maxPort > 65536 {
		maxPort = 65536
	}
	if maxPort < minPort {
		maxPort = minPort
	}
	if len(prefix) > 20 {
		prefix = prefix[:20]
	}

	if minPort == maxPort {
		port = uint16(minPort)
	} else {
		port = uint16(r.Int63()%int64(maxPort-minPort)) + uint16(minPort)
	}

	if v6 {
		b := make([]byte, 16)
		ip = net.IP(b)
	} else {
		b := make([]byte, 4)
		ip = net.IP(b)
	}

	for i := range ip {
		b := r.Intn(254) + 1
		ip[i] = byte(b)
	}

	prefix = prefix + AlphaNumericString(r, 20-len(prefix))

	return chihaya.Peer{
		ID:   chihaya.PeerID(prefix),
		Port: port,
		IP:   ip,
	}
}
