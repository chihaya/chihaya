package varinterval

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/chihaya/chihaya/bittorrent"
	"github.com/chihaya/chihaya/middleware"
	"github.com/chihaya/chihaya/pkg/prand"
)

// ErrInvalidModifyResponseProbability is returned for a config with an invalid
// ModifyResponseProbability.
var ErrInvalidModifyResponseProbability = errors.New("invalid modify_response_probability")

// ErrInvalidMaxIncreaseDelta is returned for a config with an invalid
// MaxIncreaseDelta.
var ErrInvalidMaxIncreaseDelta = errors.New("invalid max_increase_delta")

// Config represents the configuration for the varinterval middleware.
type Config struct {
	// ModifyResponseProbability is the probability by which a response will
	// be modified.
	ModifyResponseProbability float32 `yaml:"modify_response_probability"`

	// MaxIncreaseDelta is the amount of seconds that will be added at most.
	MaxIncreaseDelta int `yaml:"max_increase_delta"`

	// ModifyMinInterval specifies whether min_interval should be increased
	// as well.
	ModifyMinInterval bool `yaml:"modify_min_interval"`
}

func checkConfig(cfg Config) error {
	if cfg.ModifyResponseProbability <= 0 || cfg.ModifyResponseProbability > 1 {
		return ErrInvalidModifyResponseProbability
	}

	if cfg.MaxIncreaseDelta <= 0 {
		return ErrInvalidMaxIncreaseDelta
	}

	return nil
}

type hook struct {
	cfg Config
	pr  *prand.Container
	sync.Mutex
}

// New creates a middleware to randomly modify the announce interval from the
// given config.
func New(cfg Config) (middleware.Hook, error) {
	err := checkConfig(cfg)
	if err != nil {
		return nil, err
	}

	return &hook{
		cfg: cfg,
		pr:  prand.New(1024),
	}, nil
}

func (h *hook) HandleAnnounce(ctx context.Context, req *bittorrent.AnnounceRequest, resp *bittorrent.AnnounceResponse) (context.Context, error) {
	r := h.pr.GetByInfohash(req.InfoHash)

	if h.cfg.ModifyResponseProbability == 1 || r.Float32() < h.cfg.ModifyResponseProbability {
		addSeconds := time.Duration(r.Intn(h.cfg.MaxIncreaseDelta)+1) * time.Second
		h.pr.ReturnByInfohash(req.InfoHash)

		resp.Interval += addSeconds

		if h.cfg.ModifyMinInterval {
			resp.MinInterval += addSeconds
		}

		return ctx, nil
	}

	h.pr.ReturnByInfohash(req.InfoHash)
	return ctx, nil
}

func (h *hook) HandleScrape(ctx context.Context, req *bittorrent.ScrapeRequest, resp *bittorrent.ScrapeResponse) (context.Context, error) {
	// Scrapes are not altered.
	return ctx, nil
}
