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
	"net"
	"net/http"

	"github.com/jzelinskie/trakr/bittorrent"
)

// ParseAnnounce parses an bittorrent.AnnounceRequest from an http.Request.
//
// If allowIPSpoofing is true, IPs provided via params will be used.
// If realIPHeader is not empty string, the first value of the HTTP Header with
// that name will be used.
func ParseAnnounce(r *http.Request, realIPHeader string, allowIPSpoofing bool) (*bittorrent.AnnounceRequest, error) {
	qp, err := NewQueryParams(r.URL.RawQuery)
	if err != nil {
		return nil, err
	}

	request := &bittorrent.AnnounceRequest{Params: qp}

	eventStr, _ := qp.String("event")
	request.Event, err = bittorrent.NewEvent(eventStr)
	if err != nil {
		return nil, bittorrent.ClientError("failed to provide valid client event")
	}

	compactStr, _ := qp.String("compact")
	request.Compact = compactStr != "" && compactStr != "0"

	infoHashes := qp.InfoHashes()
	if len(infoHashes) < 1 {
		return nil, bittorrent.ClientError("no info_hash parameter supplied")
	}
	if len(infoHashes) > 1 {
		return nil, bittorrent.ClientError("multiple info_hash parameters supplied")
	}
	request.InfoHash = infoHashes[0]

	peerID, ok := qp.String("peer_id")
	if !ok {
		return nil, bittorrent.ClientError("failed to parse parameter: peer_id")
	}
	if len(peerID) != 20 {
		return nil, bittorrent.ClientError("failed to provide valid peer_id")
	}
	request.Peer.ID = bittorrent.PeerIDFromString(peerID)

	request.Left, err = qp.Uint64("left")
	if err != nil {
		return nil, bittorrent.ClientError("failed to parse parameter: left")
	}

	request.Downloaded, err = qp.Uint64("downloaded")
	if err != nil {
		return nil, bittorrent.ClientError("failed to parse parameter: downloaded")
	}

	request.Uploaded, err = qp.Uint64("uploaded")
	if err != nil {
		return nil, bittorrent.ClientError("failed to parse parameter: uploaded")
	}

	numwant, err := qp.Uint64("numwant")
	if err != nil {
		return nil, bittorrent.ClientError("failed to parse parameter: numwant")
	}
	request.NumWant = uint32(numwant)

	port, err := qp.Uint64("port")
	if err != nil {
		return nil, bittorrent.ClientError("failed to parse parameter: port")
	}
	request.Peer.Port = uint16(port)

	request.Peer.IP = requestedIP(r, qp, realIPHeader, allowIPSpoofing)
	if request.Peer.IP == nil {
		return nil, bittorrent.ClientError("failed to parse peer IP address")
	}

	return request, nil
}

// ParseScrape parses an bittorrent.ScrapeRequest from an http.Request.
func ParseScrape(r *http.Request) (*bittorrent.ScrapeRequest, error) {
	qp, err := NewQueryParams(r.URL.RawQuery)
	if err != nil {
		return nil, err
	}

	infoHashes := qp.InfoHashes()
	if len(infoHashes) < 1 {
		return nil, bittorrent.ClientError("no info_hash parameter supplied")
	}

	request := &bittorrent.ScrapeRequest{
		InfoHashes: infoHashes,
		Params:     qp,
	}

	return request, nil
}

// requestedIP determines the IP address for a BitTorrent client request.
//
// If allowIPSpoofing is true, IPs provided via params will be used.
// If realIPHeader is not empty string, the first value of the HTTP Header with
// that name will be used.
func requestedIP(r *http.Request, p bittorrent.Params, realIPHeader string, allowIPSpoofing bool) net.IP {
	if allowIPSpoofing {
		if ipstr, ok := p.String("ip"); ok {
			ip := net.ParseIP(ipstr)
			return ip
		}

		if ipstr, ok := p.String("ipv4"); ok {
			ip := net.ParseIP(ipstr)
			return ip
		}

		if ipstr, ok := p.String("ipv6"); ok {
			ip := net.ParseIP(ipstr)
			return ip
		}
	}

	if realIPHeader != "" {
		if ips, ok := r.Header[realIPHeader]; ok && len(ips) > 0 {
			ip := net.ParseIP(ips[0])
			return ip
		}
	}

	host, _, _ := net.SplitHostPort(r.RemoteAddr)
	return net.ParseIP(host)
}
