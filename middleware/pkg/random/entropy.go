package random

import (
	"encoding/binary"

	"github.com/ProtocolONE/chihaya/bittorrent"
)

// DeriveEntropyFromRequest generates 2*64 bits of pseudo random state from an
// AnnounceRequest.
//
// Calling DeriveEntropyFromRequest multiple times yields the same values.
func DeriveEntropyFromRequest(req *bittorrent.AnnounceRequest) (uint64, uint64) {
	v0 := binary.BigEndian.Uint64([]byte(req.InfoHash[:8])) + binary.BigEndian.Uint64([]byte(req.InfoHash[8:16]))
	v1 := binary.BigEndian.Uint64([]byte(req.Peer.ID[:8])) + binary.BigEndian.Uint64([]byte(req.Peer.ID[8:16]))
	return v0, v1
}
