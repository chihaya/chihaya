package iputil

import (
	"net/netip"
)

// MustAddrFromSlice calls netip.AddrFromSlice and panics on error.
func MustAddrFromSlice(b []byte) netip.Addr {
	addr, ok := netip.AddrFromSlice(b)
	if !ok {
		panic("not ok when calling AddrFromSlice")
	}
	return addr
}
