// Package torrentapproval implements a Hook that fails an Announce based on a
// whitelist or blacklist of torrent hash.
package torrentapproval

import (
	"context"
	"encoding/hex"
	"fmt"

	"gopkg.in/yaml.v2"

	"github.com/chihaya/chihaya/bittorrent"
	"github.com/chihaya/chihaya/middleware"
)

// Name is the name by which this middleware is registered with Chihaya.
const Name = "torrent approval"

func init() {
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

// ErrTorrentUnapproved is the error returned when a torrent hash is invalid.
var ErrTorrentUnapproved = bittorrent.ClientError("unapproved torrent")

// Config represents all the values required by this middleware to validate
// torrents based on their hash value.
type Config struct {
	Whitelist []string `yaml:"whitelist"`
	Blacklist []string `yaml:"blacklist"`
}

type hook struct {
	approved   map[bittorrent.InfoHash]struct{}
	unapproved map[bittorrent.InfoHash]struct{}
}

// NewHook returns an instance of the torrent approval middleware.
func NewHook(cfg Config) (middleware.Hook, error) {
	h := &hook{
		approved:   make(map[bittorrent.InfoHash]struct{}),
		unapproved: make(map[bittorrent.InfoHash]struct{}),
	}

	if len(cfg.Whitelist) > 0 && len(cfg.Blacklist) > 0 {
		return nil, fmt.Errorf("using both whitelist and blacklist is invalid")
	}

	for _, hashString := range cfg.Whitelist {
		hashinfo, err := hex.DecodeString(hashString)
		if err != nil {
			return nil, fmt.Errorf("whitelist : invalid hash %s", hashString)
		}
		if len(hashinfo) != 20 {
			return nil, fmt.Errorf("whitelist : hash %s is not 20 byes", hashString)
		}
		h.approved[bittorrent.InfoHashFromBytes(hashinfo)] = struct{}{}
	}

	for _, hashString := range cfg.Blacklist {
		hashinfo, err := hex.DecodeString(hashString)
		if err != nil {
			return nil, fmt.Errorf("blacklist : invalid hash %s", hashString)
		}
		if len(hashinfo) != 20 {
			return nil, fmt.Errorf("blacklist : hash %s is not 20 byes", hashString)
		}
		h.unapproved[bittorrent.InfoHashFromBytes(hashinfo)] = struct{}{}
	}

	return h, nil
}

func (h *hook) HandleAnnounce(ctx context.Context, req *bittorrent.AnnounceRequest, resp *bittorrent.AnnounceResponse) (context.Context, error) {
	infohash := req.InfoHash

	if len(h.approved) > 0 {
		if _, found := h.approved[infohash]; !found {
			return ctx, ErrTorrentUnapproved
		}
	}

	if len(h.unapproved) > 0 {
		if _, found := h.unapproved[infohash]; found {
			return ctx, ErrTorrentUnapproved
		}
	}

	return ctx, nil
}

func (h *hook) HandleScrape(ctx context.Context, req *bittorrent.ScrapeRequest, resp *bittorrent.ScrapeResponse) (context.Context, error) {
	// Scrapes don't require any protection.
	return ctx, nil
}
