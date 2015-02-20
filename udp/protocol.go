// Copyright 2015 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

package udp

import (
	"bytes"
	"encoding/binary"
	"net"
)

var initialConnectionID = []byte{0x04, 0x17, 0x27, 0x10, 0x19, 0x80}

func (srv *Server) handlePacket(packet []byte, addr *net.UDPAddr) (response []byte) {
	if len(packet) < 16 {
		return nil // Malformed, no client packets are less than 16 bytes.
	}

	connID := packet[0:8]
	action := binary.BigEndian.Uint32(packet[8:12])
	transactionID := packet[12:16]
	generatedConnID := GenerateConnectionID(addr.IP)

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
	}
	return
}

func writeHeader(response []byte, action uint32, transactionID []byte) {
	binary.BigEndian.PutUint32(response, action)
	copy(response[4:], transactionID)
}
