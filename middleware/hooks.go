package middleware

import (
	"context"

	"github.com/chihaya/chihaya/bittorrent"
	"github.com/chihaya/chihaya/storage"
)

// Hook abstracts the concept of anything that needs to interact with a
// BitTorrent client's request and response to a BitTorrent tracker.
type Hook interface {
	HandleAnnounce(context.Context, *bittorrent.AnnounceRequest, *bittorrent.AnnounceResponse) (context.Context, error)
	HandleScrape(context.Context, *bittorrent.ScrapeRequest, *bittorrent.ScrapeResponse) (context.Context, error)
}

type skipSwarmInteraction struct{}

// SkipSwarmInteractionKey is a key for the context of an Announce to control
// whether the swarm interaction middleware should run.
// Any non-nil value set for this key will cause the swarm interaction
// middleware to skip.
var SkipSwarmInteractionKey = skipSwarmInteraction{}

type swarmInteractionHook struct {
	store storage.PeerStore
}

func (h *swarmInteractionHook) HandleAnnounce(ctx context.Context, req *bittorrent.AnnounceRequest, resp *bittorrent.AnnounceResponse) (_ context.Context, err error) {
	if ctx.Value(SkipSwarmInteractionKey) != nil {
		return ctx, nil
	}

	switch {
	case req.Event == bittorrent.Stopped:
		err = h.store.DeleteSeeder(req.InfoHash, req.Peer)
		if err != nil && err != storage.ErrResourceDoesNotExist {
			return ctx, err
		}

		err = h.store.DeleteLeecher(req.InfoHash, req.Peer)
		if err != nil && err != storage.ErrResourceDoesNotExist {
			return ctx, err
		}
	case req.Event == bittorrent.Completed:
		err = h.store.GraduateLeecher(req.InfoHash, req.Peer)
		return ctx, err
	case req.Left == 0:
		// Completed events will also have Left == 0, but by making this
		// an extra case we can treat "old" seeders differently from
		// graduating leechers. (Calling PutSeeder is probably faster
		// than calling GraduateLeecher.)
		err = h.store.PutSeeder(req.InfoHash, req.Peer)
		return ctx, err
	default:
		err = h.store.PutLeecher(req.InfoHash, req.Peer)
		return ctx, err
	}

	return ctx, nil
}

func (h *swarmInteractionHook) HandleScrape(ctx context.Context, _ *bittorrent.ScrapeRequest, _ *bittorrent.ScrapeResponse) (context.Context, error) {
	// Scrapes have no effect on the swarm.
	return ctx, nil
}
