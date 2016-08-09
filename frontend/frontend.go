package frontend

import (
	"github.com/jzelinskie/trakr/bittorrent"
	"golang.org/x/net/context"
)

// TrackerFuncs is the collection of callback functions provided by the Backend
// to (1) generate a response from a parsed request, and (2) observe anything
// after the response has been delivered to the client.
type TrackerFuncs struct {
	HandleAnnounce AnnounceHandler
	HandleScrape   ScrapeHandler
	AfterAnnounce  AnnounceCallback
	AfterScrape    ScrapeCallback
}

// AnnounceHandler is a function that generates a response for an Announce.
type AnnounceHandler func(context.Context, *bittorrent.AnnounceRequest) (*bittorrent.AnnounceResponse, error)

// AnnounceCallback is a function that does something with the results of an
// Announce after it has been completed.
type AnnounceCallback func(*bittorrent.AnnounceRequest, *bittorrent.AnnounceResponse)

// ScrapeHandler is a function that generates a response for a Scrape.
type ScrapeHandler func(context.Context, *bittorrent.ScrapeRequest) (*bittorrent.ScrapeResponse, error)

// ScrapeCallback is a function that does something with the results of a
// Scrape after it has been completed.
type ScrapeCallback func(*bittorrent.ScrapeRequest, *bittorrent.ScrapeResponse)
