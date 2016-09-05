package bittorrent

// ClientID represents the part of a PeerID that identifies a Peer's client
// software.
type ClientID [6]byte

// NewClientID parses a ClientID from a PeerID.
func NewClientID(pid PeerID) ClientID {
	var cid ClientID
	length := len(pid)
	if length >= 6 {
		if pid[0] == '-' {
			if length >= 7 {
				copy(cid[:], pid[1:7])
			}
		} else {
			copy(cid[:], pid[:6])
		}
	}

	return cid
}
