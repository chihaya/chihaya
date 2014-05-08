// Copyright 2013 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

package server

import (
	"errors"
	"net/http"
	"path"
	"strconv"

	"github.com/chihaya/chihaya/config"
)

// announce represents all of the data from an announce request.
type announce struct {
	Compact    bool
	Downloaded uint64
	Event      string
	IP         string
	Infohash   string
	Left       uint64
	NumWant    int
	Passkey    string
	PeerID     string
	Port       uint64
	Uploaded   uint64
}

// newAnnounce parses an HTTP request and generates an Announce.
func newAnnounce(r *http.Request, conf *config.Config) (*announce, error) {
	pq, err := parseQuery(r.URL.RawQuery)
	if err != nil {
		return nil, err
	}

	compact := pq.Params["compact"] == "1"
	downloaded, downloadedErr := pq.getUint64("downloaded")
	event, _ := pq.Params["event"]
	infohash, _ := pq.Params["info_hash"]
	ip, _ := requestedIP(r, pq)
	left, leftErr := pq.getUint64("left")
	numWant := requestedPeerCount(conf.DefaultNumWant, pq)
	passkey, _ := path.Split(r.URL.Path)
	peerID, _ := pq.Params["peer_id"]
	port, portErr := pq.getUint64("port")
	uploaded, uploadedErr := pq.getUint64("uploaded")

	if downloadedErr != nil ||
		infohash == "" ||
		leftErr != nil ||
		peerID == "" ||
		portErr != nil ||
		uploadedErr != nil ||
		ip == "" {
		return nil, errors.New("malformed request")
	}

	return &announce{
		Compact:    compact,
		Downloaded: downloaded,
		Event:      event,
		IP:         ip,
		Infohash:   infohash,
		Left:       left,
		NumWant:    numWant,
		Passkey:    passkey,
		PeerID:     peerID,
		Port:       port,
		Uploaded:   uploaded,
	}, nil
}

func requestedPeerCount(fallback int, pq *parsedQuery) int {
	if numWantStr, exists := pq.Params["numWant"]; exists {
		numWant, err := strconv.Atoi(numWantStr)
		if err != nil {
			return fallback
		}
		return numWant
	}

	return fallback
}

func requestedIP(r *http.Request, pq *parsedQuery) (string, error) {
	if ip, ok := pq.Params["ip"]; ok {
		return ip, nil
	}

	if ip, ok := pq.Params["ipv4"]; ok {
		return ip, nil
	}

	if xRealIPs, ok := pq.Params["X-Real-Ip"]; ok {
		return string(xRealIPs[0]), nil
	}

	if r.RemoteAddr == "" {
		return "127.0.0.1", nil
	}

	portIndex := len(r.RemoteAddr) - 1
	for ; portIndex >= 0; portIndex-- {
		if r.RemoteAddr[portIndex] == ':' {
			break
		}
	}

	if portIndex != -1 {
		return r.RemoteAddr[0:portIndex], nil
	}

	return "", errors.New("failed to parse IP address")
}
