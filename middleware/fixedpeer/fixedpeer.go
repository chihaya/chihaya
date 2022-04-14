// Package fixedpeers implements a Hook that
//appends a fixed peer to every Announce request
package fixedpeers

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"strings"

	yaml "gopkg.in/yaml.v2"

	"github.com/chihaya/chihaya/bittorrent"
	"github.com/chihaya/chihaya/middleware"
)

// Name is the name by which this middleware is registered with Chihaya.
const Name = "fixed peers"

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

type Config struct {
	FixedPeers []string `yaml:"fixed_peers"`
}

type hook struct {
	peers []bittorrent.Peer
}

// NewHook returns an instance of the torrent approval middleware.
func NewHook(cfg Config) (middleware.Hook, error) {
	var peers []bittorrent.Peer
	for _, peerString := range cfg.FixedPeers {
		parts := strings.Split(peerString, ":")
		port, err := strconv.Atoi(parts[1])
		if err != nil {
			return nil, err
		}
		ip := net.ParseIP(parts[0]).To4()
		if ip == nil {
			panic("Invalid ip4 on fixed_peers")
		}
		peers = append(peers,
			bittorrent.Peer{
				ID:   bittorrent.PeerID{0},
				Port: uint16(port),
				IP:   bittorrent.IP{IP: ip},
			})
	}
	h := &hook{
		peers: peers,
	}
	return h, nil
}

func (h *hook) HandleAnnounce(ctx context.Context, req *bittorrent.AnnounceRequest, resp *bittorrent.AnnounceResponse) (context.Context, error) {
	for _, peer := range h.peers {
		resp.IPv4Peers = append(resp.IPv4Peers, peer)
		resp.Complete += 1
	}
	return ctx, nil
}

func (h *hook) HandleScrape(ctx context.Context, req *bittorrent.ScrapeRequest, resp *bittorrent.ScrapeResponse) (context.Context, error) {
	// Scrapes don't require any protection.
	return ctx, nil
}
