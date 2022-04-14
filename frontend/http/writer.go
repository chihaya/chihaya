package http

import (
	"errors"
	"net/http"

	"github.com/chihaya/chihaya/bittorrent"
	"github.com/chihaya/chihaya/frontend/http/bencode"
	"github.com/chihaya/chihaya/pkg/log"
)

// WriteError communicates an error to a BitTorrent client over HTTP.
func WriteError(w http.ResponseWriter, err error) error {
	message := "internal server error"
	var clientErr bittorrent.ClientError
	if errors.As(err, &clientErr) {
		message = clientErr.Error()
	} else {
		log.Error("http: internal error", log.Err(err))
	}

	w.WriteHeader(http.StatusOK)
	return bencode.NewEncoder(w).Encode(bencode.Dict{
		"failure reason": message,
	})
}

// WriteAnnounceResponse communicates the results of an Announce to a
// BitTorrent client over HTTP.
func WriteAnnounceResponse(w http.ResponseWriter, resp *bittorrent.AnnounceResponse) error {
	bdict := bencode.Dict{
		"complete":     resp.Complete,
		"incomplete":   resp.Incomplete,
		"interval":     resp.Interval,
		"min interval": resp.MinInterval,
	}

	// Add the peers to the dictionary in the compact format.
	if resp.Compact {
		var IPv4CompactDict, IPv6CompactDict []byte

		// Add the IPv4 peers to the dictionary.
		for _, peer := range resp.IPv4Peers {
			IPv4CompactDict = append(IPv4CompactDict, compact4(peer)...)
		}
		if len(IPv4CompactDict) > 0 {
			bdict["peers"] = IPv4CompactDict
		}

		// Add the IPv6 peers to the dictionary.
		for _, peer := range resp.IPv6Peers {
			IPv6CompactDict = append(IPv6CompactDict, compact6(peer)...)
		}
		if len(IPv6CompactDict) > 0 {
			bdict["peers6"] = IPv6CompactDict
		}

		return bencode.NewEncoder(w).Encode(bdict)
	}

	// Add the peers to the dictionary.
	peers := make([]bencode.Dict, 0, len(resp.IPv4Peers)+len(resp.IPv6Peers))
	for _, peer := range resp.IPv4Peers {
		peers = append(peers, dict(peer))
	}
	for _, peer := range resp.IPv6Peers {
		peers = append(peers, dict(peer))
	}
	bdict["peers"] = peers

	return bencode.NewEncoder(w).Encode(bdict)
}

// WriteScrapeResponse communicates the results of a Scrape to a BitTorrent
// client over HTTP.
func WriteScrapeResponse(w http.ResponseWriter, resp *bittorrent.ScrapeResponse) error {
	filesDict := bencode.NewDict()
	for _, scrape := range resp.Files {
		filesDict[string(scrape.InfoHash[:])] = bencode.Dict{
			"complete":   scrape.Complete,
			"incomplete": scrape.Incomplete,
		}
	}

	return bencode.NewEncoder(w).Encode(bencode.Dict{
		"files": filesDict,
	})
}

func compact4(peer bittorrent.Peer) (buf []byte) {
	ip := peer.AddrPort.Addr().As4()
	buf = append(buf, ip[:]...)
	port := peer.AddrPort.Port()
	buf = append(buf, byte(port>>8))
	buf = append(buf, byte(port&0xff))
	return
}

func compact6(peer bittorrent.Peer) (buf []byte) {
	ip := peer.AddrPort.Addr().As16()
	buf = append(buf, ip[:]...)
	port := peer.AddrPort.Port()
	buf = append(buf, byte(port>>8))
	buf = append(buf, byte(port&0xff))
	return
}

func dict(peer bittorrent.Peer) bencode.Dict {
	return bencode.Dict{
		"peer id": string(peer.ID[:]),
		"ip":      peer.AddrPort.Addr().String(),
		"port":    peer.AddrPort.Port(),
	}
}
