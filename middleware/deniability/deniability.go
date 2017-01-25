package deniability

import (
	"context"
	"errors"
	"math/rand"
	"net"

	"github.com/chihaya/chihaya/bittorrent"
	"github.com/chihaya/chihaya/middleware"
	"github.com/chihaya/chihaya/pkg/prand"
)

// ErrInvalidModifyResponseProbability is returned for a config with an invalid
// modify_response_probability.
var ErrInvalidModifyResponseProbability = errors.New("invalid modify_response_probability")

// ErrInvalidMaxRandomPeers is returned for a config with an invalid
// max_random_peers.
var ErrInvalidMaxRandomPeers = errors.New("invalid max_random_peers")

// ErrInvalidPrefix is returned for a config with an invalid prefix.
var ErrInvalidPrefix = errors.New("invalid prefix")

// ErrInvalidMaxPort is returned for a config with an invalid max_port.
var ErrInvalidMaxPort = errors.New("invalid max_port")

// ErrInvalidMinPort is returned for a config with an invalid min_port.
var ErrInvalidMinPort = errors.New("invalid min_port")

// Config represents the configuration for the deniability middleware.
type Config struct {
	// ModifyResponseProbability is the probability by which a response will
	// be augmented with random peers.
	ModifyResponseProbability float32 `yaml:"modify_response_probability"`

	// MaxRandomPeers is the amount of peers that will be added at most.
	MaxRandomPeers int `yaml:"max_random_peers"`

	// Prefix is the prefix to be used for peer IDs.
	Prefix string `yaml:"prefix"`

	// MinPort is the minimum port (inclusive) for the generated peer.
	MinPort uint16 `yaml:"min_port"`

	// MaxPort is the maximum port (exclusive) for the generated peer.
	MaxPort int `yaml:"max_port"`
}

type hook struct {
	cfg Config
	pr  *prand.Container
}

func checkConfig(cfg Config) error {
	if cfg.ModifyResponseProbability > 1 || cfg.ModifyResponseProbability <= 0 {
		return ErrInvalidModifyResponseProbability
	}

	if cfg.MaxRandomPeers <= 0 {
		return ErrInvalidMaxRandomPeers
	}

	if len(cfg.Prefix) > 20 {
		return ErrInvalidPrefix
	}

	if cfg.MinPort == 0 {
		return ErrInvalidMinPort
	}

	if cfg.MaxPort <= int(cfg.MinPort) || cfg.MaxPort > 65536 {
		return ErrInvalidMaxPort
	}

	return nil
}

// New creates a new deniability hook from the given config.
func New(cfg Config) (middleware.Hook, error) {
	err := checkConfig(cfg)
	if err != nil {
		return nil, err
	}

	toReturn := &hook{
		cfg: cfg,
		pr:  prand.New(1024),
	}

	return toReturn, nil
}

func (h *hook) HandleAnnounce(ctx context.Context, req *bittorrent.AnnounceRequest, resp *bittorrent.AnnounceResponse) (context.Context, error) {
	var (
		peers *[]bittorrent.Peer
		v6    bool
	)

	switch req.IP.AddressFamily {
	case bittorrent.IPv4:
		peers = &resp.IPv4Peers
	case bittorrent.IPv6:
		v6 = true
		peers = &resp.IPv6Peers
	default:
		panic("Peer's IP is neither IPv4 nor IPv6")
	}

	r := h.pr.GetByInfohash(req.InfoHash)
	if h.cfg.ModifyResponseProbability == 1 || r.Float32() < h.cfg.ModifyResponseProbability {
		numNewPeers := r.Intn(h.cfg.MaxRandomPeers) + 1

		// Insert up to numNewPeers, but always leave space for at least
		// one real peer!
		for i := 0; i < numNewPeers && uint32(len(*peers)) < req.NumWant; i++ {
			*peers = h.insertPeer(*peers, v6, r)
		}
	}
	h.pr.ReturnByInfohash(req.InfoHash)

	return ctx, nil
}

// insertPeer inserts a randomly generated peer at a random position into the
// given slice and returns the new slice.
func (h *hook) insertPeer(peers []bittorrent.Peer, v6 bool, r *rand.Rand) []bittorrent.Peer {
	pos := 0
	if len(peers) > 0 {
		pos = r.Intn(len(peers))
	}
	peers = append(peers, bittorrent.Peer{})
	copy(peers[pos+1:], peers[pos:])
	peers[pos] = randomPeer(r, h.cfg.Prefix, v6, h.cfg.MinPort, h.cfg.MaxPort)

	return peers
}

func (h *hook) HandleScrape(ctx context.Context, req *bittorrent.ScrapeRequest, resp *bittorrent.ScrapeResponse) (context.Context, error) {
	// Nothing to do for scrapes.
	return ctx, nil
}

// randomPeer generates a random bittorrent.Peer.
//
// prefix is the prefix to use for the peer ID. If len(prefix) > 20, it will be
// truncated to 20 characters. If len(prefix) < 20, it will be padded with a
// numeric random string to have 20 characters. This matches the general style
// of PeerIDs: A prefix unique to a client (the ClientID) followed by a string
// of random numbers.
//
// v6 indicates whether an IPv6 address should be generated.
// Regardless of the length of the generated IP address, its bytes will have
// values in [1,254].
//
// minPort and maxPort describe the range for the randomly generated port, where
// minPort <= port < maxPort.
// minPort and maxPort will be checked and altered so that
// 1 <= minPort <= maxPort <= 65536.
// If minPort == maxPort, the port will be set to minPort.
func randomPeer(r *rand.Rand, prefix string, v6 bool, minPort uint16, maxPort int) bittorrent.Peer {
	var port uint16

	if maxPort > 65536 {
		maxPort = 65536
	}
	if maxPort < int(minPort) {
		maxPort = int(minPort)
	}
	if len(prefix) > 20 {
		prefix = prefix[:20]
	}

	if int(minPort) == maxPort {
		port = minPort
	} else {
		port = uint16(r.Intn(maxPort-int(minPort))) + minPort
	}

	bIP := bittorrent.IP{}
	if v6 {
		bIP.IP = make(net.IP, net.IPv6len)
		bIP.AddressFamily = bittorrent.IPv6
	} else {
		bIP.IP = make(net.IP, net.IPv4len)
		bIP.AddressFamily = bittorrent.IPv4
	}

	for i := range bIP.IP {
		b := r.Intn(254) + 1
		bIP.IP[i] = byte(b)
	}

	prefix = prefix + numericString(r, 20-len(prefix))

	return bittorrent.Peer{
		ID:   bittorrent.PeerIDFromString(prefix),
		Port: port,
		IP:   bIP,
	}
}

const numbers = "0123456789"

func numericString(r *rand.Rand, l int) string {
	b := make([]byte, l)
	for i := range b {
		b[i] = numbers[r.Intn(len(numbers))]
	}
	return string(b)
}
