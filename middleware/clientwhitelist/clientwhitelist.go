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

type hook struct {
	approved map[bittorrent.ClientID]struct{}
}

func NewHook(approved []string) middleware.Hook {
	h := &hook{
		approved: make(map[bittorrent.ClientID]struct{}),
	}

	for _, clientID := range approved {
		h.approved[bittorrent.NewClientID(clientID)] = struct{}{}
	}

	return h
}

func (h *hook) HandleAnnounce(ctx context.Context, req *bittorrent.AnnounceRequest, resp *bittorrent.AnnounceResponse) error {
	if _, found := h.approved[bittorrent.NewClientID(req.Peer.ID)]; !found {
		return ClientUnapproved
	}

	return nil
}

func (h *hook) HandleScrape(ctx context.Context, req *bittorrent.ScrapeRequest, resp *bittorrent.ScrapeResponse) error {
	return nil
}
