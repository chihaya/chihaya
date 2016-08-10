package bittorrent

// ClientID represents the part of a PeerID that identifies a Peer's client
// software.
type ClientID string

// NewClientID parses a ClientID from a PeerID.
func NewClientID(peerID string) ClientID {
	var clientID string
	length := len(peerID)
	if length >= 6 {
		if peerID[0] == '-' {
			if length >= 7 {
				clientID = peerID[1:7]
			}
		} else {
			clientID = peerID[:6]
		}
	}

	return ClientID(clientID)
}
