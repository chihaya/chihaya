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
	"net"
	"testing"
	"time"
)

var golden = []struct {
	createdAt int64
	now       int64
	ip        string
	key       string
	valid     bool
}{
	{0, 1, "127.0.0.1", "", true},
	{0, 420420, "127.0.0.1", "", false},
	{0, 0, "[::]", "", true},
}

func TestVerification(t *testing.T) {
	for _, tt := range golden {
		cid := NewConnectionID(net.ParseIP(tt.ip), time.Unix(tt.createdAt, 0), tt.key)
		got := ValidConnectionID(cid, net.ParseIP(tt.ip), time.Unix(tt.now, 0), time.Minute, tt.key)
		if got != tt.valid {
			t.Errorf("expected validity: %t got validity: %t", tt.valid, got)
		}
	}
}
