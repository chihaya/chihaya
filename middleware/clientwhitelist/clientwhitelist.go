// Package clientwhitelist implements a Hook that fails an Announce if the
// client's PeerID does not begin with any of the approved prefixes.
package clientwhitelist

import (
	"context"

	"github.com/chihaya/chihaya/bittorrent"
	"github.com/chihaya/chihaya/middleware"
)

// ClientUnapproved is the error returned when a client's PeerID fails to
// begin with an approved prefix.
var ClientUnapproved = bittorrent.ClientError("unapproved client")

type Hook struct {
	approved map[bittorrent.ClientID]struct{}
}

func NewHook(approved []string) {
	h := &hook{
		approved: make(map[bittorrent.ClientID]struct{}),
	}

	for _, clientID := range approved {
		h.approved[bittorrent.NewClientID(clientID)] = struct{}{}
	}

	return h
}

var _ middleware.Hook = &Hook{}

func (h *Hook) HandleAnnounce(context.Context, *bittorrent.AnnounceRequest, *bittorrent.AnnounceResponse) error {
	if _, found := h.approved[bittorrent.NewClientID(req.Peer.ID)]; !found {
		return ClientUnapproved
	}

	return nil
}

func (h *Hook) HandleScrape(context.Context, *bittorrent.ScrapeRequest, *bittorrent.ScrapeResponse) error {
	return nil
}
