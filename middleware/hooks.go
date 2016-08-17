package middleware

import (
	"golang.org/x/net/context"

	"github.com/chihaya/chihaya/bittorrent"
)

// Hook abstracts the concept of anything that needs to interact with a
// BitTorrent client's request and response to a BitTorrent tracker.
type Hook interface {
	HandleAnnounce(context.Context, *bittorrent.AnnounceRequest, *bittorrent.AnnounceResponse) error
	HandleScrape(context.Context, *bittorrent.ScrapeRequest, *bittorrent.ScrapeResponse) error
}

type nopHook struct{}

func (nopHook) HandleAnnounce(ctx context.Context, req *bittorrent.AnnounceRequest, resp *bittorrent.AnnounceResponse) error {
	return nil
}

func (nopHook) HandleScrape(ctx context.Context, req *bittorrent.ScrapeRequest, resp *bittorrent.ScrapeResponse) error {
	return nil
}
