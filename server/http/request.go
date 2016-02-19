// Copyright 2016 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

package http

import (
	"net"
	"net/http"

	"github.com/chihaya/chihaya"
	"github.com/chihaya/chihaya/errors"
	"github.com/chihaya/chihaya/pkg/event"
	"github.com/chihaya/chihaya/server/http/query"
)

func announceRequest(r *http.Request, cfg *httpConfig) (*chihaya.AnnounceRequest, error) {
	q, err := query.New(r.URL.RawQuery)
	if err != nil {
		return nil, err
	}

	request := &chihaya.AnnounceRequest{Params: q}

	eventStr, err := q.String("event")
	if err != nil {
		return nil, errors.NewBadRequest("failed to parse parameter: event")
	}
	request.Event, err = event.New(eventStr)
	if err != nil {
		return nil, errors.NewBadRequest("failed to provide valid client event")
	}

	compactStr, _ := q.String("compact")
	request.Compact = compactStr != "0"

	infoHashes := q.InfoHashes()
	if len(infoHashes) < 1 {
		return nil, errors.NewBadRequest("no info_hash parameter supplied")
	}
	if len(infoHashes) > 1 {
		return nil, errors.NewBadRequest("multiple info_hash parameters supplied")
	}
	request.InfoHash = infoHashes[0]

	peerID, err := q.String("peer_id")
	if err != nil {
		return nil, errors.NewBadRequest("failed to parse parameter: peer_id")
	}
	request.PeerID = chihaya.PeerID(peerID)

	request.Left, err = q.Uint64("left")
	if err != nil {
		return nil, errors.NewBadRequest("failed to parse parameter: left")
	}

	request.Downloaded, err = q.Uint64("downloaded")
	if err != nil {
		return nil, errors.NewBadRequest("failed to parse parameter: downloaded")
	}

	request.Uploaded, err = q.Uint64("uploaded")
	if err != nil {
		return nil, errors.NewBadRequest("failed to parse parameter: uploaded")
	}

	numwant, _ := q.Uint64("numwant")
	request.NumWant = int32(numwant)

	port, err := q.Uint64("port")
	if err != nil {
		return nil, errors.NewBadRequest("failed to parse parameter: port")
	}
	request.Port = uint16(port)

	v4, v6, err := requestedIP(q, r, cfg)
	if err != nil {
		return nil, errors.NewBadRequest("failed to parse remote IP")
	}
	request.IPv4 = v4
	request.IPv6 = v6

	return request, nil
}

func scrapeRequest(r *http.Request, cfg *httpConfig) (*chihaya.ScrapeRequest, error) {
	q, err := query.New(r.URL.RawQuery)
	if err != nil {
		return nil, err
	}

	infoHashes := q.InfoHashes()
	if len(infoHashes) < 1 {
		return nil, errors.NewBadRequest("no info_hash parameter supplied")
	}

	request := &chihaya.ScrapeRequest{
		InfoHashes: infoHashes,
		Params:     q,
	}

	return request, nil
}

// requestedIP returns the IP address for a request. If there are multiple in
// the request, one IPv4 and one IPv6 will be returned.
func requestedIP(q *query.Query, r *http.Request, cfg *httpConfig) (v4, v6 net.IP, err error) {
	var done bool

	if cfg.AllowIPSpoofing {
		if str, e := q.String("ip"); e == nil {
			if v4, v6, done = getIPs(str, v4, v6, cfg); done {
				return
			}
		}

		if str, e := q.String("ipv4"); e == nil {
			if v4, v6, done = getIPs(str, v4, v6, cfg); done {
				return
			}
		}

		if str, e := q.String("ipv6"); e == nil {
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
		err = errors.NewBadRequest("failed to parse IP address")
	}

	return
}

func getIPs(ipstr string, ipv4, ipv6 net.IP, cfg *httpConfig) (net.IP, net.IP, bool) {
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
