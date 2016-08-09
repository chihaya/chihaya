package frontend

import (
	"golang.org/x/net/context"

	"github.com/jzelinskie/trakr/bittorrent"
)

// TrackerFuncs is the collection of callback functions provided by the Backend
// to (1) generate a response from a parsed request, and (2) observe anything
// after the response has been delivered to the client.
type TrackerFuncs interface {
	// HandleAnnounce generates a response for an Announce.
	HandleAnnounce(context.Context, *bittorrent.AnnounceRequest) (*bittorrent.AnnounceResponse, error)

	// AfterAnnounce does something with the results of an Announce after it
	// has been completed.
	AfterAnnounce(context.Context, *bittorrent.AnnounceRequest, *bittorrent.AnnounceResponse)

	// HandleScrape generates a response for a Scrape.
	HandleScrape(context.Context, *bittorrent.ScrapeRequest) (*bittorrent.ScrapeResponse, error)

	// AfterScrape does something with the results of a Scrape after it has been completed.
	AfterScrape(context.Context, *bittorrent.ScrapeRequest, *bittorrent.ScrapeResponse)
}
