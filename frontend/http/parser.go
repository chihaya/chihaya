package http

import (
	"net/http"

	"inet.af/netaddr"

	"github.com/chihaya/chihaya/bittorrent"
)

// ParseOptions is the configuration used to parse an Announce Request.
//
// If AllowIPSpoofing is true, IPs provided via BitTorrent params will be used.
// If RealIPHeader is not empty string, the value of the first HTTP Header with
// that name will be used.
type ParseOptions struct {
	AllowIPSpoofing     bool   `yaml:"allow_ip_spoofing"`
	RealIPHeader        string `yaml:"real_ip_header"`
	MaxNumWant          uint32 `yaml:"max_numwant"`
	DefaultNumWant      uint32 `yaml:"default_numwant"`
	MaxScrapeInfoHashes uint32 `yaml:"max_scrape_infohashes"`
}

// Default parser config constants.
const (
	defaultMaxNumWant          = 100
	defaultDefaultNumWant      = 50
	defaultMaxScrapeInfoHashes = 50
)

// ParseAnnounce parses an bittorrent.AnnounceRequest from an http.Request.
func ParseAnnounce(r *http.Request, opts ParseOptions) (*bittorrent.AnnounceRequest, error) {
	qp, err := bittorrent.ParseURLData(r.RequestURI)
	if err != nil {
		return nil, err
	}

	request := &bittorrent.AnnounceRequest{Params: qp}

	// Attempt to parse the event from the request.
	var eventStr string
	eventStr, request.EventProvided = qp.String("event")
	if request.EventProvided {
		request.Event, err = bittorrent.NewEvent(eventStr)
		if err != nil {
			return nil, bittorrent.ClientError("failed to provide valid client event")
		}
	} else {
		request.Event = bittorrent.None
	}

	// Determine if the client expects a compact response.
	compactStr, _ := qp.String("compact")
	request.Compact = compactStr != "" && compactStr != "0"

	// Parse the infohash from the request.
	infoHashes := qp.InfoHashes()
	if len(infoHashes) < 1 {
		return nil, bittorrent.ClientError("no info_hash parameter supplied")
	}
	if len(infoHashes) > 1 {
		return nil, bittorrent.ClientError("multiple info_hash parameters supplied")
	}
	request.InfoHash = infoHashes[0]

	// Parse the PeerID from the request.
	peerID, ok := qp.String("peer_id")
	if !ok {
		return nil, bittorrent.ClientError("failed to parse parameter: peer_id")
	}
	if len(peerID) != 20 {
		return nil, bittorrent.ClientError("failed to provide valid peer_id")
	}
	request.Peer.ID = bittorrent.PeerIDFromRawString(peerID)

	// Determine the number of remaining bytes for the client.
	request.Left, err = qp.Uint64("left")
	if err != nil {
		return nil, bittorrent.ClientError("failed to parse parameter: left")
	}

	// Determine the number of bytes downloaded by the client.
	request.Downloaded, err = qp.Uint64("downloaded")
	if err != nil {
		return nil, bittorrent.ClientError("failed to parse parameter: downloaded")
	}

	// Determine the number of bytes shared by the client.
	request.Uploaded, err = qp.Uint64("uploaded")
	if err != nil {
		return nil, bittorrent.ClientError("failed to parse parameter: uploaded")
	}

	// Determine the number of peers the client wants in the response.
	numwant, err := qp.Uint64("numwant")
	if err != nil && err != bittorrent.ErrKeyNotFound {
		return nil, bittorrent.ClientError("failed to parse parameter: numwant")
	}
	// If there were no errors, the user actually provided the numwant.
	request.NumWantProvided = err == nil
	request.NumWant = uint32(numwant)

	// Parse the port where the client is listening.
	port, err := qp.Uint64("port")
	if err != nil {
		return nil, bittorrent.ClientError("failed to parse parameter: port")
	}
	request.IPPort = request.IPPort.WithPort(uint16(port))

	// Parse the IP address where the client is listening.
	discoveredIP, spoofed, err := requestedIP(r, qp, opts)
	if err != nil {
		return nil, bittorrent.ClientError("failed to parse peer IP address")
	}
	request.IPPort = request.IPPort.WithIP(discoveredIP)
	request.IPProvided = spoofed

	if err := bittorrent.SanitizeAnnounce(request, opts.MaxNumWant, opts.DefaultNumWant); err != nil {
		return nil, err
	}

	return request, nil
}

// ParseScrape parses an bittorrent.ScrapeRequest from an http.Request.
func ParseScrape(r *http.Request, opts ParseOptions) (*bittorrent.ScrapeRequest, error) {
	qp, err := bittorrent.ParseURLData(r.RequestURI)
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

	if err := bittorrent.SanitizeScrape(request, opts.MaxScrapeInfoHashes); err != nil {
		return nil, err
	}

	return request, nil
}

// requestedIP determines the IP address for a BitTorrent client request.
func requestedIP(r *http.Request, p bittorrent.Params, opts ParseOptions) (_ netaddr.IP, spoofed bool, err error) {
	if opts.AllowIPSpoofing {
		if ipstr, ok := p.String("ip"); ok {
			ip, err := netaddr.ParseIP(ipstr)
			return ip, true, err
		}

		if ipstr, ok := p.String("ipv4"); ok {
			ip, err := netaddr.ParseIP(ipstr)
			return ip, true, err
		}

		if ipstr, ok := p.String("ipv6"); ok {
			ip, err := netaddr.ParseIP(ipstr)
			return ip, true, err
		}
	}

	if opts.RealIPHeader != "" {
		if ipstr := r.Header.Get(opts.RealIPHeader); ipstr != "" {
			ip, err := netaddr.ParseIP(ipstr)
			return ip, true, err
		}
	}

	ipport, err := netaddr.ParseIPPort(r.RemoteAddr)
	return ipport.IP(), false, err
}
