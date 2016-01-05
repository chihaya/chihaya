// Copyright 2015 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

package http

import (
	"errors"
	"net"
	"net/http"
	"strconv"

	"github.com/julienschmidt/httprouter"

	"github.com/chihaya/chihaya/config"
	"github.com/chihaya/chihaya/http/query"
	"github.com/chihaya/chihaya/tracker/models"
)

// newAnnounce parses an HTTP request and generates a models.Announce.
func (s *Server) newAnnounce(r *http.Request, p httprouter.Params) (*models.Announce, error) {
	q, err := query.New(r.URL.RawQuery)
	if err != nil {
		return nil, err
	}

	compact := q.Params["compact"] != "0"
	event, _ := q.Params["event"]
	numWant := requestedPeerCount(q, s.config.NumWantFallback)

	infohash, exists := q.Params["info_hash"]
	if !exists {
		return nil, models.ErrMalformedRequest
	}

	peerID, exists := q.Params["peer_id"]
	if !exists {
		return nil, models.ErrMalformedRequest
	}

	port, err := q.Uint64("port")
	if err != nil {
		return nil, models.ErrMalformedRequest
	}

	left, err := q.Uint64("left")
	if err != nil {
		return nil, models.ErrMalformedRequest
	}

	ipv4, ipv6, err := requestedIP(q, r, &s.config.NetConfig)
	if err != nil {
		return nil, models.ErrMalformedRequest
	}

	ipv4Endpoint := models.Endpoint{ipv4, uint16(port)}
	ipv6Endpoint := models.Endpoint{ipv6, uint16(port)}

	downloaded, err := q.Uint64("downloaded")
	if err != nil {
		return nil, models.ErrMalformedRequest
	}

	uploaded, err := q.Uint64("uploaded")
	if err != nil {
		return nil, models.ErrMalformedRequest
	}

	return &models.Announce{
		Config:     s.config,
		Compact:    compact,
		Downloaded: downloaded,
		Event:      event,
		IPv4:       ipv4Endpoint,
		IPv6:       ipv6Endpoint,
		Infohash:   infohash,
		Left:       left,
		NumWant:    numWant,
		PeerID:     peerID,
		Uploaded:   uploaded,
	}, nil
}

// newScrape parses an HTTP request and generates a models.Scrape.
func (s *Server) newScrape(r *http.Request, p httprouter.Params) (*models.Scrape, error) {
	q, err := query.New(r.URL.RawQuery)
	if err != nil {
		return nil, err
	}

	if q.Infohashes == nil {
		if _, exists := q.Params["info_hash"]; !exists {
			// There aren't any infohashes.
			return nil, models.ErrMalformedRequest
		}
		q.Infohashes = []string{q.Params["info_hash"]}
	}

	return &models.Scrape{
		Config:     s.config,
		Infohashes: q.Infohashes,
	}, nil
}

// requestedPeerCount returns the wanted peer count or the provided fallback.
func requestedPeerCount(q *query.Query, fallback int) int {
	if numWantStr, exists := q.Params["numwant"]; exists {
		numWant, err := strconv.Atoi(numWantStr)
		if err != nil {
			return fallback
		}
		return numWant
	}

	return fallback
}

// requestedIP returns the IP address for a request. If there are multiple in
// the request, one IPv4 and one IPv6 will be returned.
func requestedIP(q *query.Query, r *http.Request, cfg *config.NetConfig) (v4, v6 net.IP, err error) {
	var done bool

	if cfg.AllowIPSpoofing {
		if str, ok := q.Params["ip"]; ok {
			if v4, v6, done = getIPs(str, v4, v6, cfg); done {
				return
			}
		}

		if str, ok := q.Params["ipv4"]; ok {
			if v4, v6, done = getIPs(str, v4, v6, cfg); done {
				return
			}
		}

		if str, ok := q.Params["ipv6"]; ok {
			if v4, v6, done = getIPs(str, v4, v6, cfg); done {
				return
			}
		}
	}

	if cfg.RealIPHeader != "" {
		if xRealIPs, ok := r.Header[cfg.RealIPHeader]; ok {
			if v4, v6, done = getIPs(string(xRealIPs[0]), v4, v6, cfg); done {
				return
			}
		}
	} else {
		if r.RemoteAddr == "" && v4 == nil {
			if v4, v6, done = getIPs("127.0.0.1", v4, v6, cfg); done {
				return
			}
		}

		if v4, v6, done = getIPs(r.RemoteAddr, v4, v6, cfg); done {
			return
		}
	}

	if v4 == nil && v6 == nil {
		err = errors.New("failed to parse IP address")
	}

	return
}

func getIPs(ipstr string, ipv4, ipv6 net.IP, cfg *config.NetConfig) (net.IP, net.IP, bool) {
	host, _, err := net.SplitHostPort(ipstr)
	if err != nil {
		host = ipstr
	}

	if ip := net.ParseIP(host); ip != nil {
		ipTo4 := ip.To4()
		if ipv4 == nil && ipTo4 != nil {
			ipv4 = ipTo4
		} else if ipv6 == nil && ipTo4 == nil {
			ipv6 = ip
		}
	}

	var done bool
	if cfg.DualStackedPeers {
		done = ipv4 != nil && ipv6 != nil
	} else {
		done = ipv4 != nil || ipv6 != nil
	}

	return ipv4, ipv6, done
}
