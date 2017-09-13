package udp

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"net"
	"sync"

	"github.com/chihaya/chihaya/bittorrent"
)

const (
	connectActionID uint32 = iota
	announceActionID
	scrapeActionID
	errorActionID
	announceV6ActionID
)

// Option-Types as described in BEP 41 and BEP 45.
const (
	optionEndOfOptions byte = 0x0
	optionNOP               = 0x1
	optionURLData           = 0x2
)

var (
	// initialConnectionID is the magic initial connection ID specified by BEP 15.
	initialConnectionID = []byte{0, 0, 0x04, 0x17, 0x27, 0x10, 0x19, 0x80}

	// eventIDs map values described in BEP 15 to Events.
	eventIDs = []bittorrent.Event{
		bittorrent.None,
		bittorrent.Completed,
		bittorrent.Started,
		bittorrent.Stopped,
	}

	errMalformedPacket   = bittorrent.ClientError("malformed packet")
	errMalformedIP       = bittorrent.ClientError("malformed IP address")
	errMalformedEvent    = bittorrent.ClientError("malformed event ID")
	errUnknownAction     = bittorrent.ClientError("unknown action ID")
	errBadConnectionID   = bittorrent.ClientError("bad connection ID")
	errUnknownOptionType = bittorrent.ClientError("unknown option type")
)

// ParseAnnounce parses an AnnounceRequest from a UDP request.
//
// If allowIPSpoofing is true, IPs provided via params will be used.
// If v6 is true the announce will be parsed as an IPv6 announce "the
// opentracker way":
// http://opentracker.blog.h3q.com/2007/12/28/the-ipv6-situation/
func ParseAnnounce(r Request, rs *bittorrent.RequestSanitizer, allowIPSpoofing, v6 bool) (*bittorrent.AnnounceRequest, error) {
	ipEnd := 84 + net.IPv4len
	if v6 {
		ipEnd = 84 + net.IPv6len
	}

	if len(r.Packet) < ipEnd+10 {
		return nil, errMalformedPacket
	}

	infohash := r.Packet[16:36]
	peerID := r.Packet[36:56]
	downloaded := binary.BigEndian.Uint64(r.Packet[56:64])
	left := binary.BigEndian.Uint64(r.Packet[64:72])
	uploaded := binary.BigEndian.Uint64(r.Packet[72:80])

	eventID := int(r.Packet[83])
	if eventID >= len(eventIDs) {
		return nil, errMalformedEvent
	}

	ip := r.IP
	ipProvided := false
	ipbytes := r.Packet[84:ipEnd]
	if allowIPSpoofing {
		// Make sure the bytes are copied to a new slice.
		copy(ip, net.IP(ipbytes))
		ipProvided = true
	}
	if !allowIPSpoofing && r.IP == nil {
		// We have no IP address to fallback on.
		return nil, errMalformedIP
	}

	numWant := binary.BigEndian.Uint32(r.Packet[ipEnd+4 : ipEnd+8])
	port := binary.BigEndian.Uint16(r.Packet[ipEnd+8 : ipEnd+10])

	params, err := handleOptionalParameters(r.Packet[ipEnd+10:])
	if err != nil {
		return nil, err
	}

	return rs.SanitizeAnnounce(&bittorrent.AnnounceRequest{
		Event:           eventIDs[eventID],
		InfoHash:        bittorrent.InfoHashFromBytes(infohash),
		NumWant:         uint32(numWant),
		Left:            left,
		Downloaded:      downloaded,
		Uploaded:        uploaded,
		IPProvided:      ipProvided,
		NumWantProvided: true,
		EventProvided:   true,
		Peer: bittorrent.Peer{
			ID:   bittorrent.PeerIDFromBytes(peerID),
			IP:   bittorrent.IP{IP: ip},
			Port: port,
		},
		Params: params,
	})
}

type buffer struct {
	bytes.Buffer
}

var bufferFree = sync.Pool{
	New: func() interface{} { return new(buffer) },
}

func newBuffer() *buffer {
	return bufferFree.Get().(*buffer)
}

func (b *buffer) free() {
	b.Reset()
	bufferFree.Put(b)
}

// handleOptionalParameters parses the optional parameters as described in BEP
// 41 and updates an announce with the values parsed.
func handleOptionalParameters(packet []byte) (bittorrent.Params, error) {
	if len(packet) == 0 {
		return bittorrent.ParseURLData("")
	}

	var buf = newBuffer()
	defer buf.free()

	for i := 0; i < len(packet); {
		option := packet[i]
		switch option {
		case optionEndOfOptions:
			return bittorrent.ParseURLData(buf.String())
		case optionNOP:
			i++
		case optionURLData:
			if i+1 >= len(packet) {
				return nil, errMalformedPacket
			}

			length := int(packet[i+1])
			if i+2+length > len(packet) {
				return nil, errMalformedPacket
			}

			n, err := buf.Write(packet[i+2 : i+2+length])
			if err != nil {
				return nil, err
			}
			if n != length {
				return nil, fmt.Errorf("expected to write %d bytes, wrote %d", length, n)
			}

			i += 2 + length
		default:
			return nil, errUnknownOptionType
		}
	}

	return bittorrent.ParseURLData(buf.String())
}

// ParseScrape parses a ScrapeRequest from a UDP request.
func ParseScrape(r Request, rs *bittorrent.RequestSanitizer) (*bittorrent.ScrapeRequest, error) {
	// If a scrape isn't at least 36 bytes long, it's malformed.
	if len(r.Packet) < 36 {
		return nil, errMalformedPacket
	}

	// Skip past the initial headers and check that the bytes left equal the
	// length of a valid list of infohashes.
	r.Packet = r.Packet[16:]
	if len(r.Packet)%20 != 0 {
		return nil, errMalformedPacket
	}

	// Allocate a list of infohashes and append it to the list until we're out.
	var infohashes []bittorrent.InfoHash
	for len(r.Packet) >= 20 {
		infohashes = append(infohashes, bittorrent.InfoHashFromBytes(r.Packet[:20]))
		r.Packet = r.Packet[20:]
	}

	return rs.SanitizeScrape(&bittorrent.ScrapeRequest{
		InfoHashes: infohashes,
	})
}
