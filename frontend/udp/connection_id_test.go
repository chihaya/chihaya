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
