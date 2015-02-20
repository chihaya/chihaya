// Copyright 2015 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

package udp

import (
	"bytes"
	"encoding/binary"
	"errors"
	"net"

	"github.com/chihaya/chihaya/stats"
	"github.com/chihaya/chihaya/tracker/models"
)

var initialConnectionID = []byte{0, 0, 0x04, 0x17, 0x27, 0x10, 0x19, 0x80}

var eventIDs = []string{"", "completed", "started", "stopped"}

var (
	errMalformedPacket = errors.New("malformed packet")
	errMalformedIP     = errors.New("malformed IP address")
	errMalformedEvent  = errors.New("malformed event ID")
)

func writeHeader(response []byte, action uint32, transactionID []byte) {
	binary.BigEndian.PutUint32(response, action)
	copy(response[4:], transactionID)
}

func handleTorrentError(err error, w *Writer) {
	if err == nil {
		return
	}

	if _, ok := err.(models.ClientError); ok {
		w.WriteError(err)
		stats.RecordEvent(stats.ClientError)
	}
}

func (s *Server) handlePacket(packet []byte, addr *net.UDPAddr) (response []byte) {
	if len(packet) < 16 {
		return nil // Malformed, no client packets are less than 16 bytes.
	}

	connID := packet[0:8]
	action := binary.BigEndian.Uint32(packet[8:12])
	transactionID := packet[12:16]
	generatedConnID := GenerateConnectionID(addr.IP)

	writer := &Writer{transactionID: transactionID}

	switch action {
	case 0:
		// Connect request.
		if !bytes.Equal(connID, initialConnectionID) {
			return nil // Malformed packet.
		}

		response = make([]byte, 16)
		writeHeader(response, action, transactionID)
		copy(response[8:], generatedConnID)

	case 1:
		// Announce request.
		writer.buf = new(bytes.Buffer)
		ann, err := s.newAnnounce(packet, addr.IP)

		if err == nil {
			err = s.tracker.HandleAnnounce(ann, writer)
		}

		handleTorrentError(err, writer)

	case 2:
		// Scrape request.
		writer.buf = new(bytes.Buffer)
		// handleTorrentError(s.tracker.HandleScrape(scrape, writer), writer)
	}

	if writer.buf != nil {
		response = writer.buf.Bytes()
	}
	return
}

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

	ipbuf := packet[84:88]
	if !bytes.Equal(ipbuf, []byte{0, 0, 0, 0}) {
		ip = net.ParseIP(string(ipbuf))
	}
	if ip == nil {
		return nil, errMalformedIP
	}
	if ipv4 := ip.To4(); ipv4 != nil {
		ip = ipv4
	}

	// TODO(pushrax): what exactly is the key "key" used for?

	numWant := binary.BigEndian.Uint32(packet[92:96])
	port := binary.BigEndian.Uint16(packet[96:98])

	return &models.Announce{
		Config:     s.config,
		Downloaded: downloaded,
		Event:      eventIDs[eventID],
		IPv4:       ip,
		Infohash:   string(infohash),
		Left:       left,
		NumWant:    int(numWant),
		PeerID:     string(peerID),
		Port:       port,
		Uploaded:   uploaded,
	}, nil
}
