package udp

import (
	"fmt"
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
		t.Run(fmt.Sprintf("%s created at %d verified at %d", tt.ip, tt.createdAt, tt.now), func(t *testing.T) {
			cid := NewConnectionID(net.ParseIP(tt.ip), time.Unix(tt.createdAt, 0), tt.key)
			got := ValidConnectionID(cid, net.ParseIP(tt.ip), time.Unix(tt.now, 0), time.Minute, tt.key)
			if got != tt.valid {
				t.Errorf("expected validity: %t got validity: %t", tt.valid, got)
			}
		})
	}
}

func BenchmarkNewConnectionID(b *testing.B) {
	ip := net.ParseIP("127.0.0.1")
	key := "some random string that is hopefully at least this long"
	createdAt := time.Now()
	sum := int64(0)

	for i := 0; i < b.N; i++ {
		cid := NewConnectionID(ip, createdAt, key)
		sum += int64(cid[7])
	}

	_ = sum
}

func BenchmarkValidConnectionID(b *testing.B) {
	ip := net.ParseIP("127.0.0.1")
	key := "some random string that is hopefully at least this long"
	createdAt := time.Now()
	cid := NewConnectionID(ip, createdAt, key)

	for i := 0; i < b.N; i++ {
		if !ValidConnectionID(cid, ip, createdAt, 10*time.Second, key) {
			b.FailNow()
		}
	}
}
