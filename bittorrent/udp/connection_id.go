// Copyright 2016 Jimmy Zelinskie
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package udp

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/binary"
	"net"
	"time"
)

// ttl is the number of seconds a connection ID should be valid according to
// BEP 15.
const ttl = 2 * time.Minute

// NewConnectionID creates a new 8 byte connection identifier for UDP packets
// as described by BEP 15.
//
// The first 4 bytes of the connection identifier is a unix timestamp and the
// last 4 bytes are a truncated HMAC token created from the aforementioned
// unix timestamp and the source IP address of the UDP packet.
//
// Truncated HMAC is known to be safe for 2^(-n) where n is the size in bits
// of the truncated HMAC token. In this use case we have 32 bits, thus a
// forgery probability of approximately 1 in 4 billion.
func NewConnectionID(ip net.IP, now time.Time, key string) []byte {
	buf := make([]byte, 8)
	binary.BigEndian.PutUint32(buf, uint32(now.UTC().Unix()))

	mac := hmac.New(sha256.New, []byte(key))
	mac.Write(buf[:4])
	mac.Write(ip)
	macBytes := mac.Sum(nil)[:4]
	copy(buf[4:], macBytes)

	return buf
}

// ValidConnectionID determines whether a connection identifier is legitimate.
func ValidConnectionID(connectionID []byte, ip net.IP, now time.Time, maxClockSkew time.Duration, key string) bool {
	ts := time.Unix(int64(binary.BigEndian.Uint32(connectionID[:4])), 0)
	if now.After(ts.Add(ttl)) || ts.After(now.Add(maxClockSkew)) {
		return false
	}

	mac := hmac.New(sha256.New, []byte(key))
	mac.Write(connectionID[:4])
	mac.Write(ip)
	expectedMAC := mac.Sum(nil)[:4]
	return hmac.Equal(expectedMAC, connectionID[4:])
}
