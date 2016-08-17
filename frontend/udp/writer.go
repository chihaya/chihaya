package udp

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"time"

	"github.com/chihaya/chihaya/bittorrent"
)

// WriteError writes the failure reason as a null-terminated string.
func WriteError(w io.Writer, txID []byte, err error) {
	// If the client wasn't at fault, acknowledge it.
	if _, ok := err.(bittorrent.ClientError); !ok {
		err = fmt.Errorf("internal error occurred: %s", err.Error())
	}

	var buf bytes.Buffer
	writeHeader(&buf, txID, errorActionID)
	buf.WriteString(err.Error())
	buf.WriteRune('\000')
	w.Write(buf.Bytes())
}

// WriteAnnounce encodes an announce response according to BEP 15.
func WriteAnnounce(w io.Writer, txID []byte, resp *bittorrent.AnnounceResponse) {
	var buf bytes.Buffer

	writeHeader(&buf, txID, announceActionID)
	binary.Write(&buf, binary.BigEndian, uint32(resp.Interval/time.Second))
	binary.Write(&buf, binary.BigEndian, uint32(resp.Incomplete))
	binary.Write(&buf, binary.BigEndian, uint32(resp.Complete))

	for _, peer := range resp.IPv4Peers {
		buf.Write(peer.IP)
		binary.Write(&buf, binary.BigEndian, peer.Port)
	}

	w.Write(buf.Bytes())
}

// WriteScrape encodes a scrape response according to BEP 15.
func WriteScrape(w io.Writer, txID []byte, resp *bittorrent.ScrapeResponse) {
	var buf bytes.Buffer

	writeHeader(&buf, txID, scrapeActionID)

	for _, scrape := range resp.Files {
		binary.Write(&buf, binary.BigEndian, scrape.Complete)
		binary.Write(&buf, binary.BigEndian, scrape.Snatches)
		binary.Write(&buf, binary.BigEndian, scrape.Incomplete)
	}

	w.Write(buf.Bytes())
}

// WriteConnectionID encodes a new connection response according to BEP 15.
func WriteConnectionID(w io.Writer, txID, connID []byte) {
	var buf bytes.Buffer

	writeHeader(&buf, txID, connectActionID)
	buf.Write(connID)

	w.Write(buf.Bytes())
}

// writeHeader writes the action and transaction ID to the provided response
// buffer.
func writeHeader(w io.Writer, txID []byte, action uint32) {
	binary.Write(w, binary.BigEndian, action)
	w.Write(txID)
}
