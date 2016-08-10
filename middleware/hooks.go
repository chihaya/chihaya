package middleware

import (
	"golang.org/x/net/context"

	"github.com/jzelinskie/trakr/bittorrent"
)

// Hook abstracts the concept of anything that needs to interact with a
// BitTorrent client's request and response to a BitTorrent tracker.
type Hook interface {
	HandleAnnounce(context.Context, *bittorrent.AnnounceRequest, *bittorrent.AnnounceResponse) error
	HandleScrape(context.Context, *bittorrent.ScrapeRequest, *bittorrent.ScrapeResponse) error
}

type nopHook struct{}

func (nopHook) HandleAnnounce(context.Context, *bittorrent.AnnounceRequest, *bittorrent.AnnounceResponse) error {
	return nil
}

func (nopHook) HandleScrape(context.Context, *bittorrent.ScrapeRequest, *bittorrent.ScrapeResponse) error {
	return nil
}
