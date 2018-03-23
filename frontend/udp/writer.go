package udp

import (
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

	buf := newBuffer()
	writeHeader(buf, txID, errorActionID)
	buf.WriteString(err.Error())
	buf.WriteRune('\000')
	w.Write(buf.Bytes())
	buf.free()
}

// WriteAnnounce encodes an announce response according to BEP 15.
// The peers returned will be resp.IPv6Peers or resp.IPv4Peers, depending on
// whether v6Peers is set.
// If v6Action is set, the action will be 4, according to
// https://web.archive.org/web/20170503181830/http://opentracker.blog.h3q.com/2007/12/28/the-ipv6-situation/
func WriteAnnounce(w io.Writer, txID []byte, resp *bittorrent.AnnounceResponse, v6Action, v6Peers bool) {
	buf := newBuffer()

	if v6Action {
		writeHeader(buf, txID, announceV6ActionID)
	} else {
		writeHeader(buf, txID, announceActionID)
	}
	binary.Write(buf, binary.BigEndian, uint32(resp.Interval/time.Second))
	binary.Write(buf, binary.BigEndian, resp.Incomplete)
	binary.Write(buf, binary.BigEndian, resp.Complete)

	peers := resp.IPv4Peers
	if v6Peers {
		peers = resp.IPv6Peers
	}

	for _, peer := range peers {
		buf.Write(peer.IP.IP)
		binary.Write(buf, binary.BigEndian, peer.Port)
	}

	w.Write(buf.Bytes())
	buf.free()
}

// WriteScrape encodes a scrape response according to BEP 15.
func WriteScrape(w io.Writer, txID []byte, resp *bittorrent.ScrapeResponse) {
	buf := newBuffer()

	writeHeader(buf, txID, scrapeActionID)

	for _, scrape := range resp.Files {
		binary.Write(buf, binary.BigEndian, scrape.Complete)
		binary.Write(buf, binary.BigEndian, scrape.Snatches)
		binary.Write(buf, binary.BigEndian, scrape.Incomplete)
	}

	w.Write(buf.Bytes())
	buf.free()
}

// WriteConnectionID encodes a new connection response according to BEP 15.
func WriteConnectionID(w io.Writer, txID, connID []byte) {
	buf := newBuffer()

	writeHeader(buf, txID, connectActionID)
	buf.Write(connID)

	w.Write(buf.Bytes())
	buf.free()
}

// writeHeader writes the action and transaction ID to the provided response
// buffer.
func writeHeader(w io.Writer, txID []byte, action uint32) {
	binary.Write(w, binary.BigEndian, action)
	w.Write(txID)
}
