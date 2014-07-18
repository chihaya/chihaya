// Copyright 2014 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

package http

import (
	"bytes"
	"net/http"

	"github.com/chihaya/bencode"
	"github.com/chihaya/chihaya/tracker/models"
)

// Writer implements the tracker.Writer interface for the HTTP protocol.
type Writer struct {
	http.ResponseWriter
}

// WriteError writes a bencode dict with a failure reason.
func (w *Writer) WriteError(err error) error {
	bencoder := bencode.NewEncoder(w)

	return bencoder.Encode(bencode.Dict{
		"failure reason": err.Error(),
	})
}

// WriteAnnounce writes a bencode dict representation of an AnnounceResponse.
func (w *Writer) WriteAnnounce(res *models.AnnounceResponse) error {
	dict := bencode.Dict{
		"complete":     res.Complete,
		"incomplete":   res.Incomplete,
		"interval":     res.Interval,
		"min interval": res.MinInterval,
	}

	if res.IPv4Peers != nil || res.IPv4Peers != nil {
		if res.Compact {
			dict["peers"] = compactPeers(false, res.IPv4Peers)
			dict["peers6"] = compactPeers(true, res.IPv6Peers)
		} else {
			dict["peers"] = peersList(res.IPv6Peers, res.IPv4Peers)
		}
	}

	bencoder := bencode.NewEncoder(w)
	return bencoder.Encode(dict)
}

// WriteScrape writes a bencode dict representation of a ScrapeResponse.
func (w *Writer) WriteScrape(res *models.ScrapeResponse) error {
	dict := bencode.Dict{
		"files": filesDict(res.Files),
	}

	bencoder := bencode.NewEncoder(w)
	return bencoder.Encode(dict)
}

func compactPeers(ipv6 bool, peers models.PeerList) []byte {
	var compactPeers bytes.Buffer

	if ipv6 {
		for _, peer := range peers {
			if ip := peer.IP.To16(); ip != nil {
				compactPeers.Write(ip)
				compactPeers.Write([]byte{byte(peer.Port >> 8), byte(peer.Port & 0xff)})
			}
		}
	} else {
		for _, peer := range peers {
			if ip := peer.IP.To4(); ip != nil {
				compactPeers.Write(ip)
				compactPeers.Write([]byte{byte(peer.Port >> 8), byte(peer.Port & 0xff)})
			}
		}
	}

	return compactPeers.Bytes()
}

func peersList(ipv4s, ipv6s models.PeerList) (peers []bencode.Dict) {
	for _, peer := range ipv4s {
		peers = append(peers, peerDict(&peer))
	}
	for _, peer := range ipv6s {
		peers = append(peers, peerDict(&peer))
	}
	return peers
}

func peerDict(peer *models.Peer) bencode.Dict {
	return bencode.Dict{
		"ip":      peer.IP.String(),
		"peer id": peer.ID,
		"port":    peer.Port,
	}
}

func filesDict(torrents []*models.Torrent) bencode.Dict {
	d := bencode.NewDict()
	for _, torrent := range torrents {
		d[torrent.Infohash] = torrentDict(torrent)
	}
	return d
}

func torrentDict(torrent *models.Torrent) bencode.Dict {
	return bencode.Dict{
		"complete":   len(torrent.Seeders),
		"incomplete": len(torrent.Leechers),
		"downloaded": torrent.Snatches,
	}
}
