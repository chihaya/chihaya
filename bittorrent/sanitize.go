package bittorrent

import (
	"net"

	"github.com/chihaya/chihaya/pkg/log"
)

// ErrInvalidIP indicates an invalid IP for an Announce.
var ErrInvalidIP = ClientError("invalid IP")

// ErrInvalidPort indicates an invalid Port for an Announce.
var ErrInvalidPort = ClientError("invalid port")

// SanitizeAnnounce enforces a max and default NumWant and coerces the peer's
// IP address into the proper format.
func SanitizeAnnounce(r *AnnounceRequest, maxNumWant, defaultNumWant uint32) error {
	if r.Port == 0 {
		return ErrInvalidPort
	}

	if !r.NumWantProvided {
		r.NumWant = defaultNumWant
	} else if r.NumWant > maxNumWant {
		r.NumWant = maxNumWant
	}

	if ip := r.IP.To4(); ip != nil {
		r.IP.IP = ip
		r.IP.AddressFamily = IPv4
	} else if len(r.IP.IP) == net.IPv6len { // implies r.IP.To4() == nil
		r.IP.AddressFamily = IPv6
	} else {
		return ErrInvalidIP
	}

	log.Debug("sanitized announce", r, log.Fields{
		"maxNumWant":     maxNumWant,
		"defaultNumWant": defaultNumWant,
	})
	return nil
}

// SanitizeScrape enforces a max number of infohashes for a single scrape
// request.
func SanitizeScrape(r *ScrapeRequest, maxScrapeInfoHashes uint32) error {
	if len(r.InfoHashes) > int(maxScrapeInfoHashes) {
		r.InfoHashes = r.InfoHashes[:maxScrapeInfoHashes]
	}

	log.Debug("sanitized scrape", r, log.Fields{
		"maxScrapeInfoHashes": maxScrapeInfoHashes,
	})
	return nil
}
