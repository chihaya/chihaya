// Copyright 2015 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

package udp

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
)

// ConnectionIDGenerator represents the logic to generate 64-bit UDP
// connection IDs from peer IP addresses.
type ConnectionIDGenerator struct {
	iv, iv2 []byte
	block   cipher.Block
}

// NewConnectionIDGenerator creates a ConnectionIDGenerator and generates its
// AES key and first initialization vector.
func NewConnectionIDGenerator() (gen *ConnectionIDGenerator, err error) {
	gen = &ConnectionIDGenerator{}
	key := make([]byte, 16)

	_, err = rand.Read(key)
	if err != nil {
		return
	}

	gen.block, err = aes.NewCipher(key)
	if err != nil {
		return
	}

	err = gen.NewIV()
	return
}

// Generate returns the 64-bit connection ID for an IP
func (g *ConnectionIDGenerator) Generate(ip []byte) []byte {
	return g.generate(ip, g.iv)
}

func (g *ConnectionIDGenerator) generate(ip []byte, iv []byte) []byte {
	if len(ip) > 16 {
		panic("IP larger than 16 bytes")
	}

	for len(ip) < 8 {
		ip = append(ip, ip...) // Not enough bits in output.
	}

	ct := make([]byte, 16)
	stream := cipher.NewCFBDecrypter(g.block, iv)
	stream.XORKeyStream(ct, ip)

	for i := len(ip) - 1; i >= 8; i-- {
		ct[i-8] ^= ct[i]
	}

	return ct[:8]
}

// Matches checks if the given connection ID matches an IP with the current or
// previous initialization vectors.
func (g *ConnectionIDGenerator) Matches(id []byte, ip []byte) bool {
	if expected := g.generate(ip, g.iv); bytes.Equal(id, expected) {
		return true
	}

	if iv2 := g.iv2; iv2 != nil {
		if expected := g.generate(ip, iv2); bytes.Equal(id, expected) {
			return true
		}
	}

	return false
}

// NewIV generates a new initialization vector and rotates the current one.
func (g *ConnectionIDGenerator) NewIV() error {
	newiv := make([]byte, 16)
	if _, err := rand.Read(newiv); err != nil {
		return err
	}

	g.iv2 = g.iv
	g.iv = newiv

	return nil
}
