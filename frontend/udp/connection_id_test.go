package udp

import (
	"crypto/hmac"
	"encoding/binary"
	"fmt"
	"net"
	"sync"
	"testing"
	"time"

	sha256 "github.com/minio/sha256-simd"
	"github.com/stretchr/testify/require"

	"github.com/chihaya/chihaya/pkg/log"
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

// simpleNewConnectionID generates a new connection ID the explicit way.
// This is used to verify correct behaviour of the generator.
func simpleNewConnectionID(ip net.IP, now time.Time, key string) []byte {
	buf := make([]byte, 8)
	binary.BigEndian.PutUint32(buf, uint32(now.Unix()))

	mac := hmac.New(sha256.New, []byte(key))
	mac.Write(buf[:4])
	mac.Write(ip)
	macBytes := mac.Sum(nil)[:4]
	copy(buf[4:], macBytes)

	// this is just in here because logging impacts performance and we benchmark
	// this version too.
	log.Debug("manually generated connection ID", log.Fields{"ip": ip, "now": now, "connID": buf})
	return buf
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

func TestGeneration(t *testing.T) {
	for _, tt := range golden {
		t.Run(fmt.Sprintf("%s created at %d", tt.ip, tt.createdAt), func(t *testing.T) {
			want := simpleNewConnectionID(net.ParseIP(tt.ip), time.Unix(tt.createdAt, 0), tt.key)
			got := NewConnectionID(net.ParseIP(tt.ip), time.Unix(tt.createdAt, 0), tt.key)
			require.Equal(t, want, got)
		})
	}
}

func TestReuseGeneratorGenerate(t *testing.T) {
	for _, tt := range golden {
		t.Run(fmt.Sprintf("%s created at %d", tt.ip, tt.createdAt), func(t *testing.T) {
			cid := NewConnectionID(net.ParseIP(tt.ip), time.Unix(tt.createdAt, 0), tt.key)
			require.Len(t, cid, 8)

			gen := NewConnectionIDGenerator(tt.key)

			for i := 0; i < 3; i++ {
				connID := gen.Generate(net.ParseIP(tt.ip), time.Unix(tt.createdAt, 0))
				require.Equal(t, cid, connID)
			}
		})
	}
}

func TestReuseGeneratorValidate(t *testing.T) {
	for _, tt := range golden {
		t.Run(fmt.Sprintf("%s created at %d verified at %d", tt.ip, tt.createdAt, tt.now), func(t *testing.T) {
			gen := NewConnectionIDGenerator(tt.key)
			cid := gen.Generate(net.ParseIP(tt.ip), time.Unix(tt.createdAt, 0))
			for i := 0; i < 3; i++ {
				got := gen.Validate(cid, net.ParseIP(tt.ip), time.Unix(tt.now, 0), time.Minute)
				if got != tt.valid {
					t.Errorf("expected validity: %t got validity: %t", tt.valid, got)
				}
			}
		})
	}
}

func BenchmarkSimpleNewConnectionID(b *testing.B) {
	ip := net.ParseIP("127.0.0.1")
	key := "some random string that is hopefully at least this long"
	createdAt := time.Now()

	b.RunParallel(func(pb *testing.PB) {
		sum := int64(0)

		for pb.Next() {
			cid := simpleNewConnectionID(ip, createdAt, key)
			sum += int64(cid[7])
		}

		_ = sum
	})
}

func BenchmarkNewConnectionID(b *testing.B) {
	ip := net.ParseIP("127.0.0.1")
	key := "some random string that is hopefully at least this long"
	createdAt := time.Now()

	b.RunParallel(func(pb *testing.PB) {
		sum := int64(0)

		for pb.Next() {
			cid := NewConnectionID(ip, createdAt, key)
			sum += int64(cid[7])
		}

		_ = sum
	})
}

func BenchmarkConnectionIDGenerator_Generate(b *testing.B) {
	ip := net.ParseIP("127.0.0.1")
	key := "some random string that is hopefully at least this long"
	createdAt := time.Now()

	pool := &sync.Pool{
		New: func() interface{} {
			return NewConnectionIDGenerator(key)
		},
	}

	b.RunParallel(func(pb *testing.PB) {
		sum := int64(0)
		for pb.Next() {
			gen := pool.Get().(*ConnectionIDGenerator)
			cid := gen.Generate(ip, createdAt)
			sum += int64(cid[7])
			pool.Put(gen)
		}
	})
}

func BenchmarkValidConnectionID(b *testing.B) {
	ip := net.ParseIP("127.0.0.1")
	key := "some random string that is hopefully at least this long"
	createdAt := time.Now()
	cid := NewConnectionID(ip, createdAt, key)

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			if !ValidConnectionID(cid, ip, createdAt, 10*time.Second, key) {
				b.FailNow()
			}
		}
	})
}

func BenchmarkConnectionIDGenerator_Validate(b *testing.B) {
	ip := net.ParseIP("127.0.0.1")
	key := "some random string that is hopefully at least this long"
	createdAt := time.Now()
	cid := NewConnectionID(ip, createdAt, key)

	pool := &sync.Pool{
		New: func() interface{} {
			return NewConnectionIDGenerator(key)
		},
	}

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			gen := pool.Get().(*ConnectionIDGenerator)
			if !gen.Validate(cid, ip, createdAt, 10*time.Second) {
				b.FailNow()
			}
			pool.Put(gen)
		}
	})
}
