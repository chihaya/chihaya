// Copyright 2015 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

package udp

import (
	"bytes"
	"encoding/binary"

	"github.com/chihaya/chihaya/tracker/models"
)

type Writer struct {
	buf *bytes.Buffer

	connectionID  []byte
	transactionID []byte
}

func (w *Writer) WriteError(err error) error {
	w.writeHeader(3)
	w.buf.WriteString(err.Error())
	w.buf.WriteRune('\000')
	return nil
}

func (w *Writer) WriteAnnounce(res *models.AnnounceResponse) error {
	w.writeHeader(1)
	binary.Write(w.buf, binary.BigEndian, uint32(res.Interval))
	binary.Write(w.buf, binary.BigEndian, uint32(res.Incomplete))
	binary.Write(w.buf, binary.BigEndian, uint32(res.Complete))

	for _, peer := range res.IPv4Peers {
		w.buf.Write(peer.IP)
		binary.Write(w.buf, binary.BigEndian, peer.Port)
	}

	return nil
}

func (w *Writer) WriteScrape(res *models.ScrapeResponse) error {
	w.writeHeader(2)

	for _, torrent := range res.Files {
		binary.Write(w.buf, binary.BigEndian, uint32(torrent.Seeders.Len()))
		binary.Write(w.buf, binary.BigEndian, uint32(torrent.Snatches))
		binary.Write(w.buf, binary.BigEndian, uint32(torrent.Leechers.Len()))
	}

	return nil
}

func (w *Writer) writeHeader(action uint32) {
	binary.Write(w.buf, binary.BigEndian, action)
	w.buf.Write(w.transactionID)
}
