package bittorrent

import (
	"context"
	"log/slog"
	"net"
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

	if slog.Default().Enabled(context.TODO(), slog.LevelDebug) {
		slog.LogAttrs(
			context.TODO(),
			slog.LevelDebug,
			"sanitized announce",
			slog.Any("request", r),
			slog.Uint64("maxNumWant", uint64(maxNumWant)),
			slog.Uint64("defaultNumWant", uint64(defaultNumWant)),
		)
	}
	return nil
}

// SanitizeScrape enforces a max number of infohashes for a single scrape
// request.
func SanitizeScrape(r *ScrapeRequest, maxScrapeInfoHashes uint32) error {
	if len(r.InfoHashes) > int(maxScrapeInfoHashes) {
		r.InfoHashes = r.InfoHashes[:maxScrapeInfoHashes]
	}

	if slog.Default().Enabled(context.TODO(), slog.LevelDebug) {
		slog.LogAttrs(
			context.TODO(),
			slog.LevelDebug,
			"sanitized scrape",
			slog.Any("request", r),
			slog.Uint64("maxScrapeInfoHashes", uint64(maxScrapeInfoHashes)),
		)
	}
	return nil
}
