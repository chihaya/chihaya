// Copyright 2015 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

package udp

import (
	"bytes"
	"fmt"
	"net"
	"testing"

	"github.com/chihaya/chihaya/config"
)

func requestScrape(sock *net.UDPConn, connID []byte, hashes []string) ([]byte, error) {
	txID := makeTransactionID()
	request := []byte{}

	request = append(request, connID...)
	request = append(request, scrapeAction...)
	request = append(request, txID...)

	for _, hash := range hashes {
		request = append(request, []byte(hash)...)
	}

	response := make([]byte, 1024)
	n, err := sendRequest(sock, request, response)
	if err != nil {
		return nil, err
	}

	if !bytes.Equal(response[4:8], txID) {
		return nil, fmt.Errorf("transaction ID mismatch")
	}

	return response[:n], nil
}

func TestScrapeEmpty(t *testing.T) {
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

	scrape, err := requestScrape(sock, connID, []string{"aaaaaaaaaaaaaaaaaaaa"})
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(scrape[:4], errorAction) {
		t.Error("expected error response")
	}

	if string(scrape[8:]) != "torrent does not exist\000" {
		t.Error("expected torrent to not exist")
	}

	srv.Stop()
	<-done
}
