// Copyright 2016 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

package deniability

import (
	"errors"
	"math/rand"
	"time"

	"github.com/chihaya/chihaya"
	"github.com/chihaya/chihaya/pkg/random"
	"github.com/chihaya/chihaya/tracker"
)

func init() {
	tracker.RegisterAnnounceMiddlewareConstructor("deniability", constructor)
}

type deniabilityMiddleware struct {
	cfg *Config
	r   *rand.Rand
}

// constructor provides a middleware constructor that returns a middleware to
// insert peers into the peer lists returned as a response to an announce.
//
// It returns an error if the config provided is either syntactically or
// semantically incorrect.
func constructor(c chihaya.MiddlewareConfig) (tracker.AnnounceMiddleware, error) {
	cfg, err := newConfig(c)
	if err != nil {
		return nil, err
	}

	if cfg.ModifyResponseProbability <= 0 || cfg.ModifyResponseProbability > 1 {
		return nil, errors.New("modify_response_probability must be in [0,1)")
	}

	if cfg.MaxRandomPeers <= 0 {
		return nil, errors.New("max_random_peers must be > 0")
	}

	if cfg.MinPort <= 0 {
		return nil, errors.New("min_port must not be <= 0")
	}

	if cfg.MaxPort > 65536 {
		return nil, errors.New("max_port must not be > 65536")
	}

	if cfg.MinPort >= cfg.MaxPort {
		return nil, errors.New("max_port must not be <= min_port")
	}

	if len(cfg.Prefix) > 20 {
		return nil, errors.New("prefix must not be longer than 20 bytes")
	}

	mw := deniabilityMiddleware{
		cfg: cfg,
		r:   rand.New(rand.NewSource(time.Now().UnixNano())),
	}

	return mw.modifyResponse, nil
}

func (mw *deniabilityMiddleware) modifyResponse(next tracker.AnnounceHandler) tracker.AnnounceHandler {
	return func(cfg *chihaya.TrackerConfig, req *chihaya.AnnounceRequest, resp *chihaya.AnnounceResponse) error {
		err := next(cfg, req, resp)
		if err != nil {
			return err
		}

		if mw.cfg.ModifyResponseProbability == 1 || mw.r.Float32() < mw.cfg.ModifyResponseProbability {
			numNewPeers := mw.r.Intn(mw.cfg.MaxRandomPeers) + 1
			for i := 0; i < numNewPeers; i++ {
				if len(resp.IPv6Peers) > 0 {
					if len(resp.IPv6Peers) >= int(req.NumWant) {
						mw.replacePeer(resp.IPv6Peers, true)
					} else {
						resp.IPv6Peers = mw.insertPeer(resp.IPv6Peers, true)
					}
				}

				if len(resp.IPv4Peers) > 0 {
					if len(resp.IPv4Peers) >= int(req.NumWant) {
						mw.replacePeer(resp.IPv4Peers, false)
					} else {
						resp.IPv4Peers = mw.insertPeer(resp.IPv4Peers, false)
					}
				}
			}
		}

		return nil
	}
}

// replacePeer replaces a peer from a random position within the given slice
// of peers with a randomly generated one.
//
// replacePeer panics if len(peers) == 0.
func (mw *deniabilityMiddleware) replacePeer(peers []chihaya.Peer, v6 bool) {
	peers[mw.r.Intn(len(peers))] = random.Peer(mw.r, mw.cfg.Prefix, v6, mw.cfg.MinPort, mw.cfg.MaxPort)
}

// insertPeer inserts a randomly generated peer at a random position into the
// given slice and returns the new slice.
func (mw *deniabilityMiddleware) insertPeer(peers []chihaya.Peer, v6 bool) []chihaya.Peer {
	pos := 0
	if len(peers) > 0 {
		pos = mw.r.Intn(len(peers))
	}
	peers = append(peers, chihaya.Peer{})
	copy(peers[pos+1:], peers[pos:])
	peers[pos] = random.Peer(mw.r, mw.cfg.Prefix, v6, mw.cfg.MinPort, mw.cfg.MaxPort)

	return peers
}
