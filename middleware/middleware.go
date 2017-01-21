// Package middleware implements the TrackerLogic interface by executing
// a series of middleware hooks.
package middleware

import (
	"context"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/chihaya/chihaya/bittorrent"
	"github.com/chihaya/chihaya/frontend"
	"github.com/chihaya/chihaya/pkg/stopper"
	"github.com/chihaya/chihaya/storage"
)

// Config holds the configuration common across all middleware.
type Config struct {
	AnnounceInterval time.Duration `yaml:"announce_interval"`
	MaxNumWant       uint32        `yaml:"max_numwant"`
	DefaultNumWant   uint32        `yaml:"default_numwant"`
}

var _ frontend.TrackerLogic = &Logic{}

// NewLogic creates a new instance of a TrackerLogic that executes the provided
// middleware hooks.
func NewLogic(cfg Config, peerStore storage.PeerStore, preHooks, postHooks []Hook) *Logic {
	l := &Logic{
		announceInterval: cfg.AnnounceInterval,
		peerStore:        peerStore,
		preHooks:         []Hook{&sanitizationHook{maxNumWant: cfg.MaxNumWant, defaultNumWant: cfg.DefaultNumWant}},
		postHooks:        append(postHooks, &swarmInteractionHook{store: peerStore}),
	}

	l.preHooks = append(l.preHooks, preHooks...)
	l.preHooks = append(l.preHooks, &responseHook{store: peerStore})

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
func (l *Logic) HandleAnnounce(ctx context.Context, req *bittorrent.AnnounceRequest) (resp *bittorrent.AnnounceResponse, err error) {
	resp = &bittorrent.AnnounceResponse{
		Interval:    l.announceInterval,
		MinInterval: l.announceInterval,
		Compact:     req.Compact,
	}
	for _, h := range l.preHooks {
		if ctx, err = h.HandleAnnounce(ctx, req, resp); err != nil {
			return nil, err
		}
	}

	return resp, nil
}

// AfterAnnounce does something with the results of an Announce after it has
// been completed.
func (l *Logic) AfterAnnounce(ctx context.Context, req *bittorrent.AnnounceRequest, resp *bittorrent.AnnounceResponse) {
	var err error
	for _, h := range l.postHooks {
		if ctx, err = h.HandleAnnounce(ctx, req, resp); err != nil {
			log.Errorln("chihaya: post-announce hooks failed:", err.Error())
			return
		}
	}
}

// HandleScrape generates a response for a Scrape.
func (l *Logic) HandleScrape(ctx context.Context, req *bittorrent.ScrapeRequest) (resp *bittorrent.ScrapeResponse, err error) {
	resp = &bittorrent.ScrapeResponse{
		Files: make(map[bittorrent.InfoHash]bittorrent.Scrape),
	}
	for _, h := range l.preHooks {
		if ctx, err = h.HandleScrape(ctx, req, resp); err != nil {
			return nil, err
		}
	}

	return resp, nil
}

// AfterScrape does something with the results of a Scrape after it has been
// completed.
func (l *Logic) AfterScrape(ctx context.Context, req *bittorrent.ScrapeRequest, resp *bittorrent.ScrapeResponse) {
	var err error
	for _, h := range l.postHooks {
		if ctx, err = h.HandleScrape(ctx, req, resp); err != nil {
			log.Errorln("chihaya: post-scrape hooks failed:", err.Error())
			return
		}
	}
}

// Stop stops the Logic.
//
// This stops any hooks that implement stopper.Stopper.
func (l *Logic) Stop() []error {
	stopGroup := stopper.NewStopGroup()
	for _, hook := range l.preHooks {
		stoppable, ok := hook.(stopper.Stopper)
		if ok {
			stopGroup.Add(stoppable)
		}
	}

	for _, hook := range l.postHooks {
		stoppable, ok := hook.(stopper.Stopper)
		if ok {
			stopGroup.Add(stoppable)
		}
	}

	return stopGroup.Stop()
}
