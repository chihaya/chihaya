package varinterval

import (
	"context"
	"errors"
	"math/rand"
	"sync"
	"time"

	"github.com/chihaya/chihaya/bittorrent"
	"github.com/chihaya/chihaya/middleware"
	"github.com/chihaya/chihaya/pkg/xorshift"
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
	pr  [1024]*xorshift.LockedXORShift128Plus
	sync.Mutex
}

// New creates a middleware to randomly modify the announce interval from the
// given config.
func New(cfg Config) (middleware.Hook, error) {
	err := checkConfig(cfg)
	if err != nil {
		return nil, err
	}

	h := &hook{
		cfg: cfg,
		pr:  [1024]*xorshift.LockedXORShift128Plus{},
	}
	for i := range h.pr {
		h.pr[i] = xorshift.NewLockedXORShift128Plus(rand.Uint64(), rand.Uint64())
	}
	return h, nil
}

func (h *hook) getXORShiftByInfohash(ih *bittorrent.InfoHash) *xorshift.LockedXORShift128Plus {
	return h.pr[(int(ih[1])|int(ih[0])<<8)%len(h.pr)]
}

func (h *hook) HandleAnnounce(ctx context.Context, req *bittorrent.AnnounceRequest, resp *bittorrent.AnnounceResponse) (context.Context, error) {
	r := h.getXORShiftByInfohash(&req.InfoHash)

	if h.cfg.ModifyResponseProbability == 1 || float32(xorshift.Intn(r, 1<<24))/(1<<24) < h.cfg.ModifyResponseProbability {
		addSeconds := time.Duration(xorshift.Intn(r, h.cfg.MaxIncreaseDelta)+1) * time.Second

		resp.Interval += addSeconds

		if h.cfg.ModifyMinInterval {
			resp.MinInterval += addSeconds
		}

		return ctx, nil
	}

	return ctx, nil
}

func (h *hook) HandleScrape(ctx context.Context, req *bittorrent.ScrapeRequest, resp *bittorrent.ScrapeResponse) (context.Context, error) {
	// Scrapes are not altered.
	return ctx, nil
}
