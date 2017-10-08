package bittorrent

import (
	"net"

	log "github.com/sirupsen/logrus"
)

// ErrInvalidIP indicates an invalid IP for an Announce.
var ErrInvalidIP = ClientError("invalid IP")

// RequestSanitizer is used to replace unreasonable values in requests parsed
// from a frontend into sane values.
type RequestSanitizer struct {
	MaxNumWant          uint32 `yaml:"max_numwant"`
	DefaultNumWant      uint32 `yaml:"default_numwant"`
	MaxScrapeInfoHashes uint32 `yaml:"max_scrape_infohashes"`
}

// SanitizeAnnounce enforces a max and default NumWant and coerces the peer's
// IP address into the proper format.
func (rs *RequestSanitizer) SanitizeAnnounce(r *AnnounceRequest) error {
	if !r.NumWantProvided {
		r.NumWant = rs.DefaultNumWant
	} else if r.NumWant > rs.MaxNumWant {
		r.NumWant = rs.MaxNumWant
	}

	if ip := r.Peer.IP.To4(); ip != nil {
		r.Peer.IP.IP = ip
		r.Peer.IP.AddressFamily = IPv4
	} else if len(r.Peer.IP.IP) == net.IPv6len { // implies r.Peer.IP.To4() == nil
		r.Peer.IP.AddressFamily = IPv6
	} else {
		return ErrInvalidIP
	}

	log.Debug("sanitized announce", rs, r)
	return nil
}

// SanitizeScrape enforces a max number of infohashes for a single scrape
// request.
func (rs *RequestSanitizer) SanitizeScrape(r *ScrapeRequest) error {
	if len(r.InfoHashes) > int(rs.MaxScrapeInfoHashes) {
		r.InfoHashes = r.InfoHashes[:rs.MaxScrapeInfoHashes]
	}

	log.Debug("sanitized scrape", rs, r)
	return nil
}

// LogFields renders the request sanitizer's configuration as a set of loggable
// fields.
func (rs *RequestSanitizer) LogFields() log.Fields {
	return log.Fields{
		"maxNumWant":          rs.MaxNumWant,
		"defaultNumWant":      rs.DefaultNumWant,
		"maxScrapeInfohashes": rs.MaxScrapeInfoHashes,
	}
}
