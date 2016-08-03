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

package udp

import (
	"encoding/binary"
	"net"

	"github.com/jzelinskie/trakr/bittorrent"
)

const (
	connectActionID uint32 = iota
	announceActionID
	scrapeActionID
	errorActionID
	announceDualStackActionID
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

	// emptyIPs are the value of an IP field that has been left blank.
	emptyIPv4 = []byte{0, 0, 0, 0}
	emptyIPv6 = []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}

	// eventIDs map values described in BEP 15 to Events.
	eventIDs = []bittorrent.Event{
		bittorrent.None,
		bittorrent.Completed,
		bittorrent.Started,
		bittorrent.Stopped,
	}

	errMalformedPacket = bittorrent.ClientError("malformed packet")
	errMalformedIP     = bittorrent.ClientError("malformed IP address")
	errMalformedEvent  = bittorrent.ClientError("malformed event ID")
	errUnknownAction   = bittorrent.ClientError("unknown action ID")
	errBadConnectionID = bittorrent.ClientError("bad connection ID")
)

// ParseAnnounce parses an AnnounceRequest from a UDP request.
//
// If allowIPSpoofing is true, IPs provided via params will be used.
func ParseAnnounce(r Request, allowIPSpoofing bool) (*bittorrent.AnnounceRequest, error) {
	if len(r.packet) < 98 {
		return nil, errMalformedPacket
	}

	infohash := r.packet[16:36]
	peerID := r.packet[36:56]
	downloaded := binary.BigEndian.Uint64(r.packet[56:64])
	left := binary.BigEndian.Uint64(r.packet[64:72])
	uploaded := binary.BigEndian.Uint64(r.packet[72:80])

	eventID := int(r.packet[83])
	if eventID >= len(eventIDs) {
		return nil, errMalformedEvent
	}

	ip := r.IP
	ipbytes := r.packet[84:88]
	if allowIPSpoofing {
		ip = net.IP(ipbytes)
	}
	if !allowIPSpoofing && r.ip == nil {
		// We have no IP address to fallback on.
		return nil, errMalformedIP
	}

	numWant := binary.BigEndian.Uint32(r.packet[92:96])
	port := binary.BigEndian.Uint16(r.packet[96:98])

	params, err := handleOptionalParameters(r.packet)
	if err != nil {
		return nil, err
	}

	return &bittorrent.AnnounceRequest{
		Event:      eventIDs[eventID],
		InfoHash:   bittorrent.InfoHashFromBytes(infohash),
		NumWant:    uint32(numWant),
		Left:       left,
		Downloaded: downloaded,
		Uploaded:   uploaded,
		Peer: bittorrent.Peer{
			ID:   bittorrent.PeerIDFromBytes(peerID),
			IP:   ip,
			Port: port,
		},
		Params: params,
	}, nil
}

// handleOptionalParameters parses the optional parameters as described in BEP
// 41 and updates an announce with the values parsed.
func handleOptionalParameters(packet []byte) (params bittorrent.Params, err error) {
	if len(packet) <= 98 {
		return
	}

	optionStartIndex := 98
	for optionStartIndex < len(packet)-1 {
		option := packet[optionStartIndex]
		switch option {
		case optionEndOfOptions:
			return

		case optionNOP:
			optionStartIndex++

		case optionURLData:
			if optionStartIndex+1 > len(packet)-1 {
				return params, errMalformedPacket
			}

			length := int(packet[optionStartIndex+1])
			if optionStartIndex+1+length > len(packet)-1 {
				return params, errMalformedPacket
			}

			// TODO(jzelinskie): Actually parse the URL Data as described in BEP 41
			// into something that fulfills the bittorrent.Params interface.

			optionStartIndex += 1 + length
		default:
			return
		}
	}

	return
}

// ParseScrape parses a ScrapeRequest from a UDP request.
func parseScrape(r Request) (*bittorrent.ScrapeRequest, error) {
	// If a scrape isn't at least 36 bytes long, it's malformed.
	if len(r.packet) < 36 {
		return nil, errMalformedPacket
	}

	// Skip past the initial headers and check that the bytes left equal the
	// length of a valid list of infohashes.
	r.packet = r.packet[16:]
	if len(r.packet)%20 != 0 {
		return nil, errMalformedPacket
	}

	// Allocate a list of infohashes and append it to the list until we're out.
	var infohashes []bittorrent.InfoHash
	for len(r.packet) >= 20 {
		infohashes = append(infohashes, bittorrent.InfoHashFromBytes(r.packet[:20]))
		r.packet = r.packet[20:]
	}

	return &bittorrent.ScrapeRequest{
		InfoHashes: infohashes,
	}, nil
}
