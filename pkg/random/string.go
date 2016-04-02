// Copyright 2016 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

package random

import "math/rand"

// AlphaNumeric is an alphabet with all lower- and uppercase letters and
// numbers.
const AlphaNumeric = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

// AlphaNumericString is a shorthand for String(r, l, AlphaNumeric).
func AlphaNumericString(r rand.Source, l int) string {
	return String(r, l, AlphaNumeric)
}

// String generates a random string of length l, containing only runes from
// the alphabet using the random source r.
func String(r rand.Source, l int, alphabet string) string {
	b := make([]byte, l)
	for i := range b {
		b[i] = alphabet[r.Int63()%int64(len(alphabet))]
	}
	return string(b)
}
