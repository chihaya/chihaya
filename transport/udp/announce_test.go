// Copyright 2015 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

package udp

import (
	"bytes"
	"encoding/binary"
	"net"
	"testing"

	"github.com/chihaya/chihaya/config"
)

func requestAnnounce(sock *net.UDPConn, connID []byte, hash string) ([]byte, error) {
	txID := makeTransactionID()
	peerID := []byte("-UT2210-b4a2h9a9f5c4")

	var request []byte
	request = append(request, connID...)
	request = append(request, announceAction...)
	request = append(request, txID...)
	request = append(request, []byte(hash)...)
	request = append(request, peerID...)
	request = append(request, make([]byte, 8)...) // Downloaded
	request = append(request, make([]byte, 8)...) // Left
	request = append(request, make([]byte, 8)...) // Uploaded
	request = append(request, make([]byte, 4)...) // Event
	request = append(request, make([]byte, 4)...) // IP
	request = append(request, make([]byte, 4)...) // Key
	request = append(request, make([]byte, 4)...) // NumWant
	request = append(request, make([]byte, 2)...) // Port

	return doRequest(sock, request, txID)
}

func TestAnnounce(t *testing.T) {
	srv, done, err := setupTracker(&config.DefaultConfig)
	if err != nil {
		t.Fatal(err)
	}

	_, sock, err := setupSocket()
	if err != nil {
		t.Fatal(err)
	}

	connID, err := requestConnectionID(sock)
	if err != nil {
		t.Fatal(err)
	}

	announce, err := requestAnnounce(sock, connID, "aaaaaaaaaaaaaaaaaaaa")
	if err != nil {
		t.Fatal(err)
	}

	// Parse the response.
	var action, txID, interval, leechers, seeders uint32
	buf := bytes.NewReader(announce)
	binary.Read(buf, binary.BigEndian, &action)
	binary.Read(buf, binary.BigEndian, &txID)
	binary.Read(buf, binary.BigEndian, &interval)
	binary.Read(buf, binary.BigEndian, &leechers)
	binary.Read(buf, binary.BigEndian, &seeders)

	if action != uint32(announceActionID) {
		t.Fatal("expected announce action")
	}

	if interval != uint32(config.DefaultConfig.Announce.Seconds()) {
		t.Fatal("incorrect interval")
	}

	if leechers != uint32(0) {
		t.Fatal("incorrect leecher count")
	}

	// We're the only seeder.
	if seeders != uint32(1) {
		t.Fatal("incorrect seeder count")
	}

	srv.Stop()
	<-done
}
