package bittorrent

import (
	"errors"
	"net"

	log "github.com/Sirupsen/logrus"
)

// ErrInvalidIP indicates an invalid IP for an Announce.
var ErrInvalidIP = errors.New("invalid IP")

// RequestSanitizer is used to replace unreasonable values in requests parsed
// from a frontend into sane values.
type RequestSanitizer struct {
	MaxNumWant          uint32 `yaml:"max_numwant"`
	DefaultNumWant      uint32 `yaml:"default_numwant"`
	MaxScrapeInfoHashes uint32 `yaml:"max_scrape_infohashes"`
}

// SanitizeAnnounce enforces a max and default NumWant and coerces the a peer's
// IP address into the proper format.
func (rs *RequestSanitizer) SanitizeAnnounce(r *AnnounceRequest) (*AnnounceRequest, error) {
	if r.NumWant > rs.MaxNumWant {
		r.NumWant = rs.MaxNumWant
	}

	if r.NumWant == 0 {
		r.NumWant = rs.DefaultNumWant
	}

	if ip := r.Peer.IP.To4(); ip != nil {
		r.Peer.IP.IP = ip
		r.Peer.IP.AddressFamily = IPv4
	} else if len(r.Peer.IP.IP) == net.IPv6len { // implies r.Peer.IP.To4() == nil
		r.Peer.IP.AddressFamily = IPv6
	} else {
		return r, ErrInvalidIP
	}

	log.Debugf("sanitized request: %#v", r)
	return r, nil
}

// SanitizeScrape enforces a max number of infohashes for a single scrape
// request.
func (rs *RequestSanitizer) SanitizeScrape(r *ScrapeRequest) (*ScrapeRequest, error) {
	if len(r.InfoHashes) > int(rs.MaxScrapeInfoHashes) {
		r.InfoHashes = r.InfoHashes[:rs.MaxScrapeInfoHashes]
	}

	return r, nil
}
