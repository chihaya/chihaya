package cord

import (
	"context"
	"fmt"

	yaml "gopkg.in/yaml.v2"

	"github.com/ProtocolONE/chihaya/bittorrent"
	"github.com/ProtocolONE/chihaya/middleware"
	"github.com/ProtocolONE/chihaya/pkg/log"
	"github.com/ProtocolONE/chihaya/frontend/cord/database"
	"github.com/ProtocolONE/chihaya/frontend/cord/core"

	"go.uber.org/zap"
)

// Name is the name by which this middleware is registered with Chihaya.
const Name = "cord"

func Init() {
	middleware.RegisterDriver(Name, driver{})
}

var _ middleware.Driver = driver{}

type driver struct{}

func (d driver) NewHook(optionBytes []byte) (middleware.Hook, error) {
	var cfg Config
	err := yaml.Unmarshal(optionBytes, &cfg)
	if err != nil {
		return nil, fmt.Errorf("invalid options for middleware %s: %s", Name, err)
	}

	return NewHook(cfg)
}

var ErrTorrentUnapproved = bittorrent.ClientError("unapproved torrent")

type Config struct {
	Storage            string        `yaml:"storage"`
}

func (cfg Config) LogFields() log.Fields {
	
	return log.Fields{
		"storage":            cfg.Storage,
	}
}

type hook struct {
	cfg     Config
	manager *database.MemTorrentManager
}

func NewHook(cfg Config) (middleware.Hook, error) {

	err := core.InitCord()
	if err != nil {
		return nil, err
	}

	h := &hook{
		cfg: cfg,
		manager: database.NewMemTorrentManager(),
	}

	return h, nil
}

func (h *hook) HandleAnnounce(ctx context.Context, req *bittorrent.AnnounceRequest, resp *bittorrent.AnnounceResponse) (context.Context, error) {

	torrent := h.manager.FindByInfoHash(req.InfoHash.String())
	if torrent == nil {
		zap.S().Debugw("Torrent is not approved")
		return ctx, ErrTorrentUnapproved
	}

	zap.S().Debugw("Torrent is approved")
	return ctx, nil
}

func (h *hook) HandleScrape(ctx context.Context, req *bittorrent.ScrapeRequest, resp *bittorrent.ScrapeResponse) (context.Context, error) {
	// Scrapes don't require any protection.
	return ctx, nil
}
