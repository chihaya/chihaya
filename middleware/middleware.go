// Package middleware implements the TrackerLogic interface by executing
// a series of middleware hooks.
package middleware

import (
	"context"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/chihaya/chihaya/bittorrent"
	"github.com/chihaya/chihaya/frontend"
	"github.com/chihaya/chihaya/storage"
)

type Config struct {
	AnnounceInterval time.Duration `yaml:"announce_interval"`
}

var _ frontend.TrackerLogic = &Logic{}

func NewLogic(cfg Config, peerStore storage.PeerStore, preHooks, postHooks []Hook) *Logic {
	l := &Logic{
		announceInterval: cfg.AnnounceInterval,
		peerStore:        peerStore,
		preHooks:         preHooks,
		postHooks:        postHooks,
	}

	if len(l.preHooks) == 0 {
		l.preHooks = []Hook{nopHook{}}
	}

	if len(l.postHooks) == 0 {
		l.postHooks = []Hook{nopHook{}}
	}

	return l
}

// Logic is an implementation of the TrackerLogic that functions by
// executing a series of middleware hooks.
type Logic struct {
	announceInterval time.Duration
	peerStore        storage.PeerStore
	preHooks         []Hook
	postHooks        []Hook
}

// HandleAnnounce generates a response for an Announce.
func (l *Logic) HandleAnnounce(ctx context.Context, req *bittorrent.AnnounceRequest) (*bittorrent.AnnounceResponse, error) {
	resp := &bittorrent.AnnounceResponse{
		Interval: l.announceInterval,
	}
	for _, h := range l.preHooks {
		if err := h.HandleAnnounce(ctx, req, resp); err != nil {
			return nil, err
		}
	}

	return resp, nil
}

// AfterAnnounce does something with the results of an Announce after it has
// been completed.
func (l *Logic) AfterAnnounce(ctx context.Context, req *bittorrent.AnnounceRequest, resp *bittorrent.AnnounceResponse) {
	for _, h := range l.postHooks {
		if err := h.HandleAnnounce(ctx, req, resp); err != nil {
			log.Errorln("chihaya: post-announce hooks failed:", err.Error())
			return
		}
	}
}

// HandleScrape generates a response for a Scrape.
func (l *Logic) HandleScrape(ctx context.Context, req *bittorrent.ScrapeRequest) (*bittorrent.ScrapeResponse, error) {
	resp := &bittorrent.ScrapeResponse{
		Files: make(map[bittorrent.InfoHash]bittorrent.Scrape),
	}
	for _, h := range l.preHooks {
		if err := h.HandleScrape(ctx, req, resp); err != nil {
			return nil, err
		}
	}

	return resp, nil
}

// AfterScrape does something with the results of a Scrape after it has been
// completed.
func (l *Logic) AfterScrape(ctx context.Context, req *bittorrent.ScrapeRequest, resp *bittorrent.ScrapeResponse) {
	for _, h := range l.postHooks {
		if err := h.HandleScrape(ctx, req, resp); err != nil {
			log.Errorln("chihaya: post-scrape hooks failed:", err.Error())
			return
		}
	}
}
