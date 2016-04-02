// Copyright 2016 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

package varinterval

import (
	"errors"
	"math/rand"
	"time"

	"github.com/chihaya/chihaya"
	"github.com/chihaya/chihaya/tracker"
)

func init() {
	tracker.RegisterAnnounceMiddlewareConstructor("varinterval", constructor)
}

type varintervalMiddleware struct {
	cfg *Config
	r   *rand.Rand
}

// constructor provides a middleware constructor that returns a middleware to
// insert a variation into announce intervals.
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

	if cfg.MaxIncreaseDelta <= 0 {
		return nil, errors.New("max_increase_delta must be > 0")
	}

	mw := varintervalMiddleware{
		cfg: cfg,
		r:   rand.New(rand.NewSource(time.Now().UnixNano())),
	}

	return mw.modifyResponse, nil
}

func (mw *varintervalMiddleware) modifyResponse(next tracker.AnnounceHandler) tracker.AnnounceHandler {
	return func(cfg *chihaya.TrackerConfig, req *chihaya.AnnounceRequest, resp *chihaya.AnnounceResponse) error {
		err := next(cfg, req, resp)
		if err != nil {
			return err
		}

		if mw.cfg.ModifyResponseProbability == 1 || mw.r.Float32() < mw.cfg.ModifyResponseProbability {
			addSeconds := time.Duration(mw.r.Intn(mw.cfg.MaxIncreaseDelta)+1) * time.Second
			resp.Interval += addSeconds

			if mw.cfg.ModifyMinInterval {
				resp.MinInterval += addSeconds
			}
		}

		return nil
	}
}
