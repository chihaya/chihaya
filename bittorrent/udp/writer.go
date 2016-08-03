// Copyright 2016 Jimmy Zelinskie
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package udp

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"time"

	"github.com/jzelinskie/trakr/bittorrent"
)

// WriteError writes the failure reason as a null-terminated string.
func WriteError(writer io.Writer, txID []byte, err error) {
	// If the client wasn't at fault, acknowledge it.
	if _, ok := err.(bittorrent.ClientError); !ok {
		err = fmt.Errorf("internal error occurred: %s", err.Error())
	}

	var buf bytes.Buffer
	writeHeader(buf, txID, errorActionID)
	buf.WriteString(err.Error())
	buf.WriteRune('\000')
	writer.Write(buf.Bytes())
}

// WriteAnnounce encodes an announce response according to BEP 15.
func WriteAnnounce(respBuf *bytes.Buffer, txID []byte, resp *bittorrent.AnnounceResponse) {
	writeHeader(respBuf, txID, announceActionID)
	binary.Write(respBuf, binary.BigEndian, uint32(resp.Interval/time.Second))
	binary.Write(respBuf, binary.BigEndian, uint32(resp.Incomplete))
	binary.Write(respBuf, binary.BigEndian, uint32(resp.Complete))

	for _, peer := range resp.IPv4Peers {
		respBuf.Write(peer.IP)
		binary.Write(respBuf, binary.BigEndian, peer.Port)
	}
}

// WriteScrape encodes a scrape response according to BEP 15.
func WriteScrape(respBuf *bytes.Buffer, txID []byte, resp *bittorrent.ScrapeResponse) {
	writeHeader(respBuf, txID, scrapeActionID)

	for _, scrape := range resp.Files {
		binary.Write(respBuf, binary.BigEndian, scrape.Complete)
		binary.Write(respBuf, binary.BigEndian, scrape.Snatches)
		binary.Write(respBuf, binary.BigEndian, scrape.Incomplete)
	}
}

// WriteConnectionID encodes a new connection response according to BEP 15.
func WriteConnectionID(respBuf *bytes.Buffer, txID, connID []byte) {
	writeHeader(respBuf, txID, connectActionID)
	respBuf.Write(connID)
}

// writeHeader writes the action and transaction ID to the provided response
// buffer.
func writeHeader(respBuf *bytes.Buffer, txID []byte, action uint32) {
	binary.Write(respBuf, binary.BigEndian, action)
	respBuf.Write(txID)
}
