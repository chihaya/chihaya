// Copyright 2016 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

package http

import (
	"net/http"

	"github.com/chihaya/chihaya"
	"github.com/chihaya/chihaya/pkg/bencode"
)

type writer struct{ http.ResponseWriter }

func (w *writer) writeError(err error) error {
	return bencode.NewEncoder(w).Encode(bencode.Dict{
		"failure reason": err.Error(),
	})
}

func (w *writer) writeAnnounceResponse(resp *chihaya.AnnounceResponse) error {
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
			IPv4CompactDict = append(IPv4CompactDict, compact(peer)...)
		}
		if len(IPv4CompactDict) > 0 {
			bdict["peers"] = IPv4CompactDict
		}

		// Add the IPv6 peers to the dictionary.
		for _, peer := range resp.IPv6Peers {
			IPv6CompactDict = append(IPv6CompactDict, compact(peer)...)
		}
		if len(IPv6CompactDict) > 0 {
			bdict["peers6"] = IPv6CompactDict
		}

		return bencode.NewEncoder(w).Encode(bdict)
	}

	// Add the peers to the dictionary.
	var peers []bencode.Dict
	for _, peer := range resp.IPv4Peers {
		peers = append(peers, dict(peer))
	}
	for _, peer := range resp.IPv6Peers {
		peers = append(peers, dict(peer))
	}
	bdict["peers"] = peers

	return bencode.NewEncoder(w).Encode(bdict)
}

func (w *writer) writeScrapeResponse(resp *chihaya.ScrapeResponse) error {
	filesDict := bencode.NewDict()
	for infohash, scrape := range resp.Files {
		filesDict[infohash] = bencode.Dict{
			"complete":   scrape.Complete,
			"incomplete": scrape.Incomplete,
			"downloaded": scrape.Downloaded,
		}
	}

	return bencode.NewEncoder(w).Encode(bencode.Dict{
		"files": filesDict,
	})
}

func compact(peer chihaya.Peer) (buf []byte) {
	buf = []byte(peer.IP)
	buf = append(buf, byte(peer.Port>>8))
	buf = append(buf, byte(peer.Port&0xff))
	return
}

func dict(peer chihaya.Peer) bencode.Dict {
	return bencode.Dict{
		"peer id": peer.ID,
		"ip":      peer.IP.String(),
		"port":    peer.Port,
	}
}
