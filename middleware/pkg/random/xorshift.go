// Package random implements the XORShift PRNG and a way to derive random state
// from an AnnounceRequest.
package random

// GenerateAndAdvance applies XORShift128Plus on s0 and s1, returning
// the new states newS0, newS1 and a pseudo-random number v.
func GenerateAndAdvance(s0, s1 uint64) (v, newS0, newS1 uint64) {
	v = s0 + s1
	newS0 = s1
	s0 ^= (s0 << 23)
	newS1 = s0 ^ s1 ^ (s0 >> 18) ^ (s1 >> 5)
	return
}

// Intn generates an int k that satisfies k >= 0 && k < n.
// n must be > 0.
// It returns the generated k and the new state of the generator.
func Intn(s0, s1 uint64, n int) (int, uint64, uint64) {
	if n <= 0 {
		panic("invalid n <= 0")
	}
	v, newS0, newS1 := GenerateAndAdvance(s0, s1)
	k := int(v)
	if k < 0 {
		k = -k
	}
	return k % n, newS0, newS1
}
