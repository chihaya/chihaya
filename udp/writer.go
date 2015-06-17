// Copyright 2015 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

package udp

import (
	"bytes"
	"encoding/binary"
	"time"

	"github.com/chihaya/chihaya/tracker/models"
)

// Writer implements the tracker.Writer interface for the UDP protocol.
type Writer struct {
	buf *bytes.Buffer

	connectionID  []byte
	transactionID []byte
}

// WriteError writes the failure reason as a null-terminated string.
func (w *Writer) WriteError(err error) error {
	w.writeHeader(errorActionID)
	w.buf.WriteString(err.Error())
	w.buf.WriteRune('\000')
	return nil
}

// WriteAnnounce encodes an announce response by selecting the proper announce
// format based on the BitTorrent spec.
func (w *Writer) WriteAnnounce(resp *models.AnnounceResponse) (err error) {
	if resp.Announce.HasIPv6() {
		err = w.WriteAnnounceIPv6(resp)
	} else {
		err = w.WriteAnnounceIPv4(resp)
	}

	return
}

// WriteAnnounceIPv6 encodes an announce response according to BEP45.
func (w *Writer) WriteAnnounceIPv6(resp *models.AnnounceResponse) error {
	w.writeHeader(announceDualStackActionID)
	binary.Write(w.buf, binary.BigEndian, uint32(resp.Interval/time.Second))
	binary.Write(w.buf, binary.BigEndian, uint32(resp.Incomplete))
	binary.Write(w.buf, binary.BigEndian, uint32(resp.Complete))
	binary.Write(w.buf, binary.BigEndian, uint32(len(resp.IPv4Peers)))
	binary.Write(w.buf, binary.BigEndian, uint32(len(resp.IPv6Peers)))

	for _, peer := range resp.IPv4Peers {
		w.buf.Write(peer.IP)
		binary.Write(w.buf, binary.BigEndian, peer.Port)
	}

	for _, peer := range resp.IPv6Peers {
		w.buf.Write(peer.IP)
		binary.Write(w.buf, binary.BigEndian, peer.Port)
	}

	return nil
}

// WriteAnnounceIPv4 encodes an announce response according to BEP15.
func (w *Writer) WriteAnnounceIPv4(resp *models.AnnounceResponse) error {
	w.writeHeader(announceActionID)
	binary.Write(w.buf, binary.BigEndian, uint32(resp.Interval/time.Second))
	binary.Write(w.buf, binary.BigEndian, uint32(resp.Incomplete))
	binary.Write(w.buf, binary.BigEndian, uint32(resp.Complete))

	for _, peer := range resp.IPv4Peers {
		w.buf.Write(peer.IP)
		binary.Write(w.buf, binary.BigEndian, peer.Port)
	}

	return nil
}

// WriteScrape encodes a scrape response according to BEP15.
func (w *Writer) WriteScrape(resp *models.ScrapeResponse) error {
	w.writeHeader(scrapeActionID)

	for _, torrent := range resp.Files {
		binary.Write(w.buf, binary.BigEndian, uint32(torrent.Seeders.Len()))
		binary.Write(w.buf, binary.BigEndian, uint32(torrent.Snatches))
		binary.Write(w.buf, binary.BigEndian, uint32(torrent.Leechers.Len()))
	}

	return nil
}

// writeHeader writes the action and transaction ID to the response.
func (w *Writer) writeHeader(action uint32) {
	binary.Write(w.buf, binary.BigEndian, action)
	w.buf.Write(w.transactionID)
}
