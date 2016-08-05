// Copyright 2016 Jimmy Zelinskie
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package http

import (
	"net/http"

	"github.com/jzelinskie/trakr/bittorrent"
	"github.com/jzelinskie/trakr/bittorrent/http/bencode"
)

// WriteError communicates an error to a BitTorrent client over HTTP.
func WriteError(w http.ResponseWriter, err error) error {
	message := "internal server error"
	if _, clientErr := err.(bittorrent.ClientError); clientErr {
		message = err.Error()
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

// WriteScrapeResponse communicates the results of a Scrape to a BitTorrent
// client over HTTP.
func WriteScrapeResponse(w http.ResponseWriter, resp *bittorrent.ScrapeResponse) error {
	filesDict := bencode.NewDict()
	for infohash, scrape := range resp.Files {
		filesDict[string(infohash[:])] = bencode.Dict{
			"complete":   scrape.Complete,
			"incomplete": scrape.Incomplete,
		}
	}

	return bencode.NewEncoder(w).Encode(bencode.Dict{
		"files": filesDict,
	})
}

func compact(peer bittorrent.Peer) (buf []byte) {
	buf = []byte(peer.IP)
	buf = append(buf, byte(peer.Port>>8))
	buf = append(buf, byte(peer.Port&0xff))
	return
}

func dict(peer bittorrent.Peer) bencode.Dict {
	return bencode.Dict{
		"peer id": string(peer.ID[:]),
		"ip":      peer.IP.String(),
		"port":    peer.Port,
	}
}
