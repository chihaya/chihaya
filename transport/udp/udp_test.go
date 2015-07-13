// Copyright 2015 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

package udp

import (
	"bytes"
	"crypto/rand"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/chihaya/chihaya/config"
	"github.com/chihaya/chihaya/tracker"

	_ "github.com/chihaya/chihaya/event/consumer/nop"
	_ "github.com/chihaya/chihaya/event/producer/nop"
	_ "github.com/chihaya/chihaya/store/memory"
)

var (
	testPort       = "34137"
	connectAction  = []byte{0, 0, 0, byte(connectActionID)}
	announceAction = []byte{0, 0, 0, byte(announceActionID)}
	scrapeAction   = []byte{0, 0, 0, byte(scrapeActionID)}
	errorAction    = []byte{0, 0, 0, byte(errorActionID)}
)

func setupTracker(cfg *config.Config) (*Server, chan struct{}, error) {
	tkr, err := tracker.New(cfg)
	if err != nil {
		return nil, nil, err
	}

	srv := NewServer(cfg, tkr)
	done := make(chan struct{})

	go func() {
		if err := srv.serve(":" + testPort); err != nil {
			panic(err)
		}
		close(done)
	}()

	<-srv.booting
	return srv, done, nil
}

func setupSocket() (*net.UDPAddr, *net.UDPConn, error) {
	srvAddr, err := net.ResolveUDPAddr("udp", "localhost:"+testPort)
	if err != nil {
		return nil, nil, err
	}

	sock, err := net.DialUDP("udp", nil, srvAddr)
	if err != nil {
		return nil, nil, err
	}

	return srvAddr, sock, err
}

func makeTransactionID() []byte {
	out := make([]byte, 4)
	rand.Read(out)
	return out
}

func sendRequest(sock *net.UDPConn, request, response []byte) (int, error) {
	if _, err := sock.Write(request); err != nil {
		return 0, err
	}

	sock.SetReadDeadline(time.Now().Add(time.Second))
	n, err := sock.Read(response)

	if err != nil {
		if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
			return 0, fmt.Errorf("no response from tracker: %s", err)
		}
	}

	return n, err
}

func requestConnectionID(sock *net.UDPConn) ([]byte, error) {
	txID := makeTransactionID()
	request := []byte{}

	request = append(request, initialConnectionID...)
	request = append(request, connectAction...)
	request = append(request, txID...)

	response := make([]byte, 1024)
	n, err := sendRequest(sock, request, response)
	if err != nil {
		return nil, err
	}

	if n != 16 {
		return nil, fmt.Errorf("packet length mismatch: %d != 16", n)
	}

	if !bytes.Equal(response[4:8], txID) {
		return nil, fmt.Errorf("transaction ID mismatch")
	}

	if !bytes.Equal(response[0:4], connectAction) {
		return nil, fmt.Errorf("action mismatch")
	}

	return response[8:16], nil
}

func TestRequestConnectionID(t *testing.T) {
	srv, done, err := setupTracker(&config.DefaultConfig)
	if err != nil {
		t.Fatal(err)
	}

	_, sock, err := setupSocket()
	if err != nil {
		t.Fatal(err)
	}

	if _, err = requestConnectionID(sock); err != nil {
		t.Fatal(err)
	}

	srv.Stop()
	<-done
}
