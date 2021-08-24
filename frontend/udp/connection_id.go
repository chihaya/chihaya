package udp

import (
	"crypto/hmac"
	"encoding/binary"
	"hash"
	"time"

	"github.com/cespare/xxhash/v2"
	"github.com/rs/zerolog/log"
	"inet.af/netaddr"
)

// ttl is the duration a connection ID should be valid according to BEP 15.
const ttl = 2 * time.Minute

// NewConnectionID creates an 8-byte connection identifier for UDP packets as
// described by BEP 15.
// This is a wrapper around creating a new ConnectionIDGenerator and generating
// an ID. It is recommended to use the generator for performance.
func NewConnectionID(ip netaddr.IP, now time.Time, key string) []byte {
	return NewConnectionIDGenerator(key).Generate(ip, now)
}

// ValidConnectionID determines whether a connection identifier is legitimate.
// This is a wrapper around creating a new ConnectionIDGenerator and validating
// the ID. It is recommended to use the generator for performance.
func ValidConnectionID(connectionID []byte, ip netaddr.IP, now time.Time, maxClockSkew time.Duration, key string) bool {
	return NewConnectionIDGenerator(key).Validate(connectionID, ip, now, maxClockSkew)
}

// A ConnectionIDGenerator is a reusable generator and validator for connection
// IDs as described in BEP 15.
// It is not thread safe, but is safe to be pooled and reused by other
// goroutines. It manages its state itself, so it can be taken from and returned
// to a pool without any cleanup.
// After initial creation, it can generate connection IDs without allocating.
// See Generate and Validate for usage notes and guarantees.
type ConnectionIDGenerator struct {
	// mac is a keyed HMAC that can be reused for subsequent connection ID
	// generations.
	mac hash.Hash

	// connID is an 8-byte slice that holds the generated connection ID after a
	// call to Generate.
	// It must not be referenced after the generator is returned to a pool.
	// It will be overwritten by subsequent calls to Generate.
	connID []byte

	// scratch is a 32-byte slice that is used as a scratchpad for the generated
	// HMACs.
	scratch []byte
}

func hashfn() hash.Hash { return xxhash.New() }

// NewConnectionIDGenerator creates a new connection ID generator.
func NewConnectionIDGenerator(key string) *ConnectionIDGenerator {
	return &ConnectionIDGenerator{
		mac:     hmac.New(hashfn, []byte(key)),
		connID:  make([]byte, 8),
		scratch: make([]byte, 32),
	}
}

// reset resets the generator.
// This is called by other methods of the generator, it's not necessary to call
// it after getting a generator from a pool.
func (g *ConnectionIDGenerator) reset() {
	g.mac.Reset()
	g.connID = g.connID[:8]
	g.scratch = g.scratch[:0]
}

// Generate generates an 8-byte connection ID as described in BEP 15 for the
// given IP and the current time.
//
// The first 4 bytes of the connection identifier is a unix timestamp and the
// last 4 bytes are a truncated HMAC token created from the aforementioned
// unix timestamp and the source IP address of the UDP packet.
//
// Truncated HMAC is known to be safe for 2^(-n) where n is the size in bits
// of the truncated HMAC token. In this use case we have 32 bits, thus a
// forgery probability of approximately 1 in 4 billion.
//
// The generated ID is written to g.connID, which is also returned. g.connID
// will be reused, so it must not be referenced after returning the generator
// to a pool and will be overwritten be subsequent calls to Generate!
func (g *ConnectionIDGenerator) Generate(ip netaddr.IP, now time.Time) []byte {
	g.reset()

	binary.BigEndian.PutUint32(g.connID, uint32(now.Unix()))
	g.mac.Write(g.connID[:4])

	ipBytes, err := ip.MarshalBinary()
	if err != nil {
		panic("netaddr.IP.MarshalBinary() returned an error: " + err.Error())
	}
	g.mac.Write(ipBytes)

	g.scratch = g.mac.Sum(g.scratch)
	copy(g.connID[4:8], g.scratch[:4])

	log.Debug().
		Stringer("ip", ip).
		Stringer("now", now).
		Bytes("connID", g.connID).
		Msg("generated connection ID")
	return g.connID
}

// Validate validates the given connection ID for an IP and the current time.
func (g *ConnectionIDGenerator) Validate(connectionID []byte, ip netaddr.IP, now time.Time, maxClockSkew time.Duration) bool {
	ts := time.Unix(int64(binary.BigEndian.Uint32(connectionID[:4])), 0)
	log.Debug().
		Stringer("ip", ip).
		Stringer("now", now).
		Stringer("connTime", ts).
		Bytes("connID", connectionID).
		Msg("validating connection ID")
	if now.After(ts.Add(ttl)) || ts.After(now.Add(maxClockSkew)) {
		return false
	}

	g.reset()
	g.mac.Write(connectionID[:4])

	ipBytes, err := ip.MarshalBinary()
	if err != nil {
		panic("netaddr.IP.MarshalBinary() returned an error: " + err.Error())
	}
	g.mac.Write(ipBytes)

	g.scratch = g.mac.Sum(g.scratch)
	return hmac.Equal(g.scratch[:4], connectionID[4:])
}
