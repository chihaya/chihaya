// Copyright 2015 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

package udp

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
)

var connectionKey, connectionIV []byte

func InitConnectionIDEncryption() error {
	connectionKey = make([]byte, 16)
	_, err := rand.Read(connectionKey)
	if err != nil {
		return err
	}

	connectionIV = make([]byte, 16)
	_, err = rand.Read(connectionIV)
	return err
}

func GenerateConnectionID(ip []byte) []byte {
	block, err := aes.NewCipher(connectionKey)
	if err != nil {
		panic(err)
	}

	if len(ip) > 16 {
		panic("IP larger than 16 bytes")
	}

	for len(ip) < 8 {
		ip = append(ip, ip...) // Not enough bits in output.
	}

	ct := make([]byte, 16)
	stream := cipher.NewCFBDecrypter(block, connectionIV)
	stream.XORKeyStream(ct, ip)

	for i := len(ip) - 1; i >= 8; i-- {
		ct[i-8] ^= ct[i]
	}

	return ct[:8]
}

func init() {
	InitConnectionIDEncryption()
}
