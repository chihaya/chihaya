package udp

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"time"

	"github.com/jzelinskie/trakr/bittorrent"
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
	writeHeader(w, txID, announceActionID)
	binary.Write(w, binary.BigEndian, uint32(resp.Interval/time.Second))
	binary.Write(w, binary.BigEndian, uint32(resp.Incomplete))
	binary.Write(w, binary.BigEndian, uint32(resp.Complete))

	for _, peer := range resp.IPv4Peers {
		w.Write(peer.IP)
		binary.Write(w, binary.BigEndian, peer.Port)
	}
}

// WriteScrape encodes a scrape response according to BEP 15.
func WriteScrape(w io.Writer, txID []byte, resp *bittorrent.ScrapeResponse) {
	writeHeader(w, txID, scrapeActionID)

	for _, scrape := range resp.Files {
		binary.Write(w, binary.BigEndian, scrape.Complete)
		binary.Write(w, binary.BigEndian, scrape.Snatches)
		binary.Write(w, binary.BigEndian, scrape.Incomplete)
	}
}

// WriteConnectionID encodes a new connection response according to BEP 15.
func WriteConnectionID(w io.Writer, txID, connID []byte) {
	writeHeader(w, txID, connectActionID)
	w.Write(connID)
}

// writeHeader writes the action and transaction ID to the provided response
// buffer.
func writeHeader(w io.Writer, txID []byte, action uint32) {
	binary.Write(w, binary.BigEndian, action)
	w.Write(txID)
}
