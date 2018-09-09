package middleware

import (
	"context"
	"time"

	"github.com/chihaya/chihaya/bittorrent"
	"github.com/chihaya/chihaya/frontend"
	"github.com/chihaya/chihaya/pkg/log"
	"github.com/chihaya/chihaya/pkg/stop"
	"github.com/chihaya/chihaya/storage"
)

// ResponseConfig holds the configuration used for the actual response.
//
// TODO(jzelinskie): Evaluate whether we would like to make this optional.
// We can make Chihaya extensible enough that you can program a new response
// generator at the cost of making it possible for users to create config that
// won't compose a functional tracker.
type ResponseConfig struct {
	AnnounceInterval    time.Duration `yaml:"announce_interval"`
	MinAnnounceInterval time.Duration `yaml:"min_announce_interval"`
}

var _ frontend.TrackerLogic = &Logic{}

// NewLogic creates a new instance of a TrackerLogic that executes the provided
// middleware hooks.
func NewLogic(cfg ResponseConfig, peerStore storage.PeerStore, preHooks, postHooks []Hook) *Logic {
	return &Logic{
		announceInterval:    cfg.AnnounceInterval,
		minAnnounceInterval: cfg.MinAnnounceInterval,
		peerStore:           peerStore,
		preHooks:            append(preHooks, &responseHook{store: peerStore}),
		postHooks:           append(postHooks, &swarmInteractionHook{store: peerStore}),
	}
}

// Logic is an implementation of the TrackerLogic that functions by
// executing a series of middleware hooks.
type Logic struct {
	announceInterval    time.Duration
	minAnnounceInterval time.Duration
	peerStore           storage.PeerStore
	preHooks            []Hook
	postHooks           []Hook
}

// HandleAnnounce generates a response for an Announce.
func (l *Logic) HandleAnnounce(ctx context.Context, req *bittorrent.AnnounceRequest) (_ context.Context, resp *bittorrent.AnnounceResponse, err error) {
	resp = &bittorrent.AnnounceResponse{
		Interval:    l.announceInterval,
		MinInterval: l.minAnnounceInterval,
		Compact:     req.Compact,
	}
	for _, h := range l.preHooks {
		if ctx, err = h.HandleAnnounce(ctx, req, resp); err != nil {
			return nil, nil, err
		}
	}

	log.Debug("generated announce response", resp)
	return ctx, resp, nil
}

// AfterAnnounce does something with the results of an Announce after it has
// been completed.
func (l *Logic) AfterAnnounce(ctx context.Context, req *bittorrent.AnnounceRequest, resp *bittorrent.AnnounceResponse) {
	var err error
	for _, h := range l.postHooks {
		if ctx, err = h.HandleAnnounce(ctx, req, resp); err != nil {
			log.Error("post-announce hooks failed", log.Err(err))
			return
		}
	}
}

// HandleScrape generates a response for a Scrape.
func (l *Logic) HandleScrape(ctx context.Context, req *bittorrent.ScrapeRequest) (_ context.Context, resp *bittorrent.ScrapeResponse, err error) {
	resp = &bittorrent.ScrapeResponse{
		Files: make([]bittorrent.Scrape, 0, len(req.InfoHashes)),
	}
	for _, h := range l.preHooks {
		if ctx, err = h.HandleScrape(ctx, req, resp); err != nil {
			return nil, nil, err
		}
	}

	log.Debug("generated scrape response", resp)
	return ctx, resp, nil
}

// AfterScrape does something with the results of a Scrape after it has been
// completed.
func (l *Logic) AfterScrape(ctx context.Context, req *bittorrent.ScrapeRequest, resp *bittorrent.ScrapeResponse) {
	var err error
	for _, h := range l.postHooks {
		if ctx, err = h.HandleScrape(ctx, req, resp); err != nil {
			log.Error("post-scrape hooks failed", log.Err(err))
			return
		}
	}
}

// Stop stops the Logic.
//
// This stops any hooks that implement stop.Stopper.
func (l *Logic) Stop() stop.Result {
	stopGroup := stop.NewGroup()
	for _, hook := range l.preHooks {
		stoppable, ok := hook.(stop.Stopper)
		if ok {
			stopGroup.Add(stoppable)
		}
	}

	for _, hook := range l.postHooks {
		stoppable, ok := hook.(stop.Stopper)
		if ok {
			stopGroup.Add(stoppable)
		}
	}

	return stopGroup.Stop()
}
