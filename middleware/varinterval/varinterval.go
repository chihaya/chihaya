package varinterval

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	yaml "gopkg.in/yaml.v2"

	"github.com/chihaya/chihaya/bittorrent"
	"github.com/chihaya/chihaya/middleware"
	"github.com/chihaya/chihaya/middleware/pkg/random"
)

// Name is the name by which this middleware is registered with Chihaya.
const Name = "interval variation"

func init() {
	middleware.RegisterDriver(Name, driver{})
}

var _ middleware.Driver = driver{}

type driver struct{}

func (d driver) NewHook(optionBytes []byte) (middleware.Hook, error) {
	var cfg Config
	err := yaml.Unmarshal(optionBytes, &cfg)
	if err != nil {
		return nil, fmt.Errorf("invalid options for middleware %s: %w", Name, err)
	}

	return NewHook(cfg)
}

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
	sync.Mutex
}

// NewHook creates a middleware to randomly modify the announce interval from
// the given config.
func NewHook(cfg Config) (middleware.Hook, error) {
	if err := checkConfig(cfg); err != nil {
		return nil, err
	}

	h := &hook{
		cfg: cfg,
	}
	return h, nil
}

func (h *hook) HandleAnnounce(ctx context.Context, req *bittorrent.AnnounceRequest, resp *bittorrent.AnnounceResponse) (context.Context, error) {
	s0, s1 := random.DeriveEntropyFromRequest(req)
	// Generate a probability p < 1.0.
	v, s0, s1 := random.Intn(s0, s1, 1<<24)
	p := float32(v) / (1 << 24)
	if h.cfg.ModifyResponseProbability == 1 || p < h.cfg.ModifyResponseProbability {
		// Generate the increase delta.
		v, _, _ = random.Intn(s0, s1, h.cfg.MaxIncreaseDelta)
		deltaDuration := time.Duration(v+1) * time.Second

		resp.Interval += deltaDuration

		if h.cfg.ModifyMinInterval {
			resp.MinInterval += deltaDuration
		}

		return ctx, nil
	}

	return ctx, nil
}

func (h *hook) HandleScrape(ctx context.Context, req *bittorrent.ScrapeRequest, resp *bittorrent.ScrapeResponse) (context.Context, error) {
	// Scrapes are not altered.
	return ctx, nil
}
