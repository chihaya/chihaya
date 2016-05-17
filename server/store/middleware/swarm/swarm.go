// Copyright 2016 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

package response

import (
	"github.com/chihaya/chihaya"
	"github.com/chihaya/chihaya/pkg/event"
	"github.com/chihaya/chihaya/server/store"
	"github.com/chihaya/chihaya/tracker"
)

func init() {
	tracker.RegisterAnnounceMiddleware("store_swarm_interaction", announceSwarmInteraction)
}

// FailedSwarmInteraction represents an error that indicates that the
// interaction of a peer with a swarm failed.
type FailedSwarmInteraction string

// Error satisfies the error interface for FailedSwarmInteraction.
func (f FailedSwarmInteraction) Error() string { return string(f) }

// announceSwarmInteraction provides a middleware that manages swarm
// interactions for a peer based on the announce.
func announceSwarmInteraction(next tracker.AnnounceHandler) tracker.AnnounceHandler {
	return func(cfg *chihaya.TrackerConfig, req *chihaya.AnnounceRequest, resp *chihaya.AnnounceResponse) (err error) {
		if req.IPv4 != nil {
			err = updatePeerStore(req, req.Peer4())
			if err != nil {
				return FailedSwarmInteraction(err.Error())
			}
		}

		if req.IPv6 != nil {
			err = updatePeerStore(req, req.Peer6())
			if err != nil {
				return FailedSwarmInteraction(err.Error())
			}
		}

		return next(cfg, req, resp)
	}
}

func updatePeerStore(req *chihaya.AnnounceRequest, peer chihaya.Peer) (err error) {
	storage := store.MustGetStore()

	switch {
	case req.Event == event.Stopped:
		err = storage.DeleteSeeder(req.InfoHash, peer)
		if err != nil && err != store.ErrResourceDoesNotExist {
			return err
		}

		err = storage.DeleteLeecher(req.InfoHash, peer)
		if err != nil && err != store.ErrResourceDoesNotExist {
			return err
		}

	case req.Event == event.Completed || req.Left == 0:
		err = storage.GraduateLeecher(req.InfoHash, peer)
		if err != nil {
			return err
		}
	default:
		err = storage.PutLeecher(req.InfoHash, peer)
		if err != nil {
			return err
		}
	}

	return nil
}
