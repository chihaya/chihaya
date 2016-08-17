// Package middleware implements the TrackerLogic interface by executing
// a series of middleware hooks.
package middleware

import (
	"log"
	"time"

	"golang.org/x/net/context"

	"github.com/chihaya/chihaya/bittorrent"
	"github.com/chihaya/chihaya/frontend"
	"github.com/chihaya/chihaya/storage"
)

type Config struct {
	AnnounceInterval time.Duration `yaml:"announce_interval"`
}

var _ frontend.TrackerLogic = &Logic{}

func NewLogic(config Config, peerStore storage.PeerStore, announcePreHooks, announcePostHooks, scrapePreHooks, scrapePostHooks []Hook) *Logic {
	l := &Logic{
		announceInterval:  config.AnnounceInterval,
		peerStore:         peerStore,
		announcePreHooks:  announcePreHooks,
		announcePostHooks: announcePostHooks,
		scrapePreHooks:    scrapePreHooks,
		scrapePostHooks:   scrapePostHooks,
	}

	if len(l.announcePreHooks) == 0 {
		l.announcePreHooks = []Hook{nopHook{}}
	}

	if len(l.announcePostHooks) == 0 {
		l.announcePostHooks = []Hook{nopHook{}}
	}

	if len(l.scrapePreHooks) == 0 {
		l.scrapePreHooks = []Hook{nopHook{}}
	}

	if len(l.scrapePostHooks) == 0 {
		l.scrapePostHooks = []Hook{nopHook{}}
	}

	return l
}

// Logic is an implementation of the TrackerLogic that functions by
// executing a series of middleware hooks.
type Logic struct {
	announceInterval  time.Duration
	peerStore         storage.PeerStore
	announcePreHooks  []Hook
	announcePostHooks []Hook
	scrapePreHooks    []Hook
	scrapePostHooks   []Hook
}

// HandleAnnounce generates a response for an Announce.
func (l *Logic) HandleAnnounce(ctx context.Context, req *bittorrent.AnnounceRequest) (*bittorrent.AnnounceResponse, error) {
	resp := &bittorrent.AnnounceResponse{
		Interval: l.announceInterval,
	}
	for _, h := range l.announcePreHooks {
		if err := h.HandleAnnounce(ctx, req, resp); err != nil {
			return nil, err
		}
	}

	return resp, nil
}

// AfterAnnounce does something with the results of an Announce after it has
// been completed.
func (l *Logic) AfterAnnounce(ctx context.Context, req *bittorrent.AnnounceRequest, resp *bittorrent.AnnounceResponse) {
	for _, h := range l.announcePostHooks {
		if err := h.HandleAnnounce(ctx, req, resp); err != nil {
			log.Println("chihaya: post-announce hooks failed:", err.Error())
			return
		}
	}
}

// HandleScrape generates a response for a Scrape.
func (l *Logic) HandleScrape(ctx context.Context, req *bittorrent.ScrapeRequest) (*bittorrent.ScrapeResponse, error) {
	resp := &bittorrent.ScrapeResponse{
		Files: make(map[bittorrent.InfoHash]bittorrent.Scrape),
	}
	for _, h := range l.scrapePreHooks {
		if err := h.HandleScrape(ctx, req, resp); err != nil {
			return nil, err
		}
	}

	return resp, nil
}

// AfterScrape does something with the results of a Scrape after it has been
// completed.
func (l *Logic) AfterScrape(ctx context.Context, req *bittorrent.ScrapeRequest, resp *bittorrent.ScrapeResponse) {
	for _, h := range l.scrapePostHooks {
		if err := h.HandleScrape(ctx, req, resp); err != nil {
			log.Println("chihaya: post-scrape hooks failed:", err.Error())
			return
		}
	}
}
