// Package clientapproval implements a Hook that fails an Announce based on a
// whitelist or blacklist of BitTorrent client IDs.
package clientapproval

import (
	"context"
	"errors"
	"fmt"

	yaml "gopkg.in/yaml.v2"

	"github.com/chihaya/chihaya/bittorrent"
	"github.com/chihaya/chihaya/middleware"
)

// Name is the name by which this middleware is registered with Chihaya.
const Name = "client approval"

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

// ErrClientUnapproved is the error returned when a client's PeerID is invalid.
var ErrClientUnapproved = bittorrent.ClientError("unapproved client")

// Config represents all the values required by this middleware to validate
// peers based on their BitTorrent client ID.
type Config struct {
	Whitelist []string `yaml:"whitelist"`
	Blacklist []string `yaml:"blacklist"`
}

type hook struct {
	approved   map[bittorrent.ClientID]struct{}
	unapproved map[bittorrent.ClientID]struct{}
}

// NewHook returns an instance of the client approval middleware.
func NewHook(cfg Config) (middleware.Hook, error) {
	h := &hook{
		approved:   make(map[bittorrent.ClientID]struct{}),
		unapproved: make(map[bittorrent.ClientID]struct{}),
	}

	if len(cfg.Whitelist) > 0 && len(cfg.Blacklist) > 0 {
		return nil, fmt.Errorf("using both whitelist and blacklist is invalid")
	}

	for _, cidString := range cfg.Whitelist {
		cidBytes := []byte(cidString)
		if len(cidBytes) != 6 {
			return nil, errors.New("client ID " + cidString + " must be 6 bytes")
		}
		var cid bittorrent.ClientID
		copy(cid[:], cidBytes)
		h.approved[cid] = struct{}{}
	}

	for _, cidString := range cfg.Blacklist {
		cidBytes := []byte(cidString)
		if len(cidBytes) != 6 {
			return nil, errors.New("client ID " + cidString + " must be 6 bytes")
		}
		var cid bittorrent.ClientID
		copy(cid[:], cidBytes)
		h.unapproved[cid] = struct{}{}
	}

	return h, nil
}

func (h *hook) HandleAnnounce(ctx context.Context, req *bittorrent.AnnounceRequest, resp *bittorrent.AnnounceResponse) (context.Context, error) {
	clientID := req.Peer.ID.ClientID()

	if len(h.approved) > 0 {
		if _, found := h.approved[clientID]; !found {
			return ctx, ErrClientUnapproved
		}
	}

	if len(h.unapproved) > 0 {
		if _, found := h.unapproved[clientID]; found {
			return ctx, ErrClientUnapproved
		}
	}

	return ctx, nil
}

func (h *hook) HandleScrape(ctx context.Context, req *bittorrent.ScrapeRequest, resp *bittorrent.ScrapeResponse) (context.Context, error) {
	// Scrapes don't require any protection.
	return ctx, nil
}
