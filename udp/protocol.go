// Copyright 2015 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

package udp

import (
	"bytes"
	"encoding/binary"
	"net"

	"github.com/chihaya/chihaya/stats"
	"github.com/chihaya/chihaya/tracker/models"
)

const (
	connectActionID uint32 = iota
	announceActionID
	scrapeActionID
	errorActionID
	announceDualStackActionID
)

var (
	// initialConnectionID is the magic initial connection ID specified by BEP 15.
	initialConnectionID = []byte{0, 0, 0x04, 0x17, 0x27, 0x10, 0x19, 0x80}

	// emptyIPs are the value of an IP field that has been left blank.
	emptyIPv4 = []byte{0, 0, 0, 0}
	emptyIPv6 = []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}

	// Option-Types described in BEP41 and BEP45.
	optionEndOfOptions = byte(0x0)
	optionNOP          = byte(0x1)
	optionURLData      = byte(0x2)
	optionIPv6         = byte(0x3)

	// eventIDs map IDs to event names.
	eventIDs = []string{
		"",
		"completed",
		"started",
		"stopped",
	}

	errMalformedPacket = models.ProtocolError("malformed packet")
	errMalformedIP     = models.ProtocolError("malformed IP address")
	errMalformedEvent  = models.ProtocolError("malformed event ID")
	errBadConnectionID = models.ProtocolError("bad connection ID")
)

// handleTorrentError writes err to w if err is a models.ClientError.
func handleTorrentError(err error, w *Writer) {
	if err == nil {
		return
	}

	if models.IsPublicError(err) {
		w.WriteError(err)
		stats.RecordEvent(stats.ClientError)
	}
}

// handlePacket decodes and processes one UDP request, returning the response.
func (s *Server) handlePacket(packet []byte, addr *net.UDPAddr) (response []byte, actionName string) {
	if len(packet) < 16 {
		return // Malformed, no client packets are less than 16 bytes.
	}

	connID := packet[0:8]
	action := binary.BigEndian.Uint32(packet[8:12])
	transactionID := packet[12:16]

	writer := &Writer{
		buf: new(bytes.Buffer),

		connectionID:  connID,
		transactionID: transactionID,
	}
	defer func() { response = writer.buf.Bytes() }()

	if action != 0 && !s.connIDGen.Matches(connID, addr.IP) {
		writer.WriteError(errBadConnectionID)
		return
	}

	switch action {
	case connectActionID:
		actionName = "connect"
		if !bytes.Equal(connID, initialConnectionID) {
			return // Malformed packet.
		}

		writer.writeHeader(0)
		writer.buf.Write(s.connIDGen.Generate(addr.IP))

	case announceActionID:
		actionName = "announce"
		ann, err := s.newAnnounce(packet, addr.IP)

		if err == nil {
			err = s.tracker.HandleAnnounce(ann, writer)
		}

		handleTorrentError(err, writer)

	case scrapeActionID:
		actionName = "scrape"
		scrape, err := s.newScrape(packet)

		if err == nil {
			err = s.tracker.HandleScrape(scrape, writer)
		}

		handleTorrentError(err, writer)
	}

	return
}

// newAnnounce decodes one announce packet, returning a models.Announce.
func (s *Server) newAnnounce(packet []byte, ip net.IP) (*models.Announce, error) {
	if len(packet) < 98 {
		return nil, errMalformedPacket
	}

	infohash := packet[16:36]
	peerID := packet[36:56]

	downloaded := binary.BigEndian.Uint64(packet[56:64])
	left := binary.BigEndian.Uint64(packet[64:72])
	uploaded := binary.BigEndian.Uint64(packet[72:80])

	eventID := packet[83]
	if eventID > 3 {
		return nil, errMalformedEvent
	}

	ipv4bytes := packet[84:88]
	if s.config.AllowIPSpoofing && !bytes.Equal(ipv4bytes, emptyIPv4) {
		ip = net.ParseIP(string(ipv4bytes))
	}

	if ip == nil {
		return nil, errMalformedIP
	} else if ipv4 := ip.To4(); ipv4 != nil {
		ip = ipv4
	}

	numWant := binary.BigEndian.Uint32(packet[92:96])
	port := binary.BigEndian.Uint16(packet[96:98])

	announce := &models.Announce{
		Config:     s.config,
		Downloaded: downloaded,
		Event:      eventIDs[eventID],
		IPv4: models.Endpoint{
			IP:   ip,
			Port: port,
		},
		Infohash: string(infohash),
		Left:     left,
		NumWant:  int(numWant),
		PeerID:   string(peerID),
		Uploaded: uploaded,
	}

	if err := s.handleOptionalParameters(packet, announce); err != nil {
		return nil, err
	}

	return announce, nil
}

// handleOptionalParameters parses the optional parameters as described in BEP41
// and updates an announce with the values parsed.
func (s *Server) handleOptionalParameters(packet []byte, announce *models.Announce) error {
	if len(packet) > 98 {
		optionStartIndex := 98
		for optionStartIndex < len(packet)-1 {
			option := packet[optionStartIndex]
			switch option {
			case optionEndOfOptions:
				return nil

			case optionNOP:
				optionStartIndex++

			case optionURLData:
				if optionStartIndex+1 > len(packet)-1 {
					return errMalformedPacket
				}

				length := int(packet[optionStartIndex+1])
				if optionStartIndex+1+length > len(packet)-1 {
					return errMalformedPacket
				}

				// TODO: Actually parse the URL Data as described in BEP41.

				optionStartIndex += 1 + length

			case optionIPv6:
				if optionStartIndex+19 > len(packet)-1 {
					return errMalformedPacket
				}

				ipv6bytes := packet[optionStartIndex+1 : optionStartIndex+17]
				if s.config.AllowIPSpoofing && !bytes.Equal(ipv6bytes, emptyIPv6) {
					announce.IPv6.IP = net.ParseIP(string(ipv6bytes)).To16()
					announce.IPv6.Port = binary.BigEndian.Uint16(packet[optionStartIndex+17 : optionStartIndex+19])
					if announce.IPv6.IP == nil {
						return errMalformedIP
					}
				}

				optionStartIndex += 19

			default:
				return nil
			}
		}
	}

	// There was no optional parameters to parse.
	return nil
}

// newScrape decodes one announce packet, returning a models.Scrape.
func (s *Server) newScrape(packet []byte) (*models.Scrape, error) {
	if len(packet) < 36 {
		return nil, errMalformedPacket
	}

	var infohashes []string
	packet = packet[16:]

	if len(packet)%20 != 0 {
		return nil, errMalformedPacket
	}

	for len(packet) >= 20 {
		infohash := packet[:20]
		infohashes = append(infohashes, string(infohash))
		packet = packet[20:]
	}

	return &models.Scrape{
		Config:     s.config,
		Infohashes: infohashes,
	}, nil
}
