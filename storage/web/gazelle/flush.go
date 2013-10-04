// Copyright 2013 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

package gazelle

import (
	"bytes"
	"log"
	"time"
)

func (c *Conn) flushTorrents() {
	var query bytes.Buffer
	c.waitGroup.Add(1)
	defer c.waitGroup.Done()
	var count int

	for {
		length := len(c.torrentChannel)
		query.Reset()

		query.WriteString("INSERT INTO torrents (ID, Snatched, Seeders, Leechers, last_action) VALUES\n")

		for count = 0; count < length; count++ {
			s := <-c.torrentChannel
			if s == "" {
				break
			}
			query.WriteString(s)

			if count != length-1 {
				query.WriteRune(',')
			}
		}

		if !c.terminate {
			log.Printf("[torrents] Flushing %d\n", count)
		}

		if count > 0 {
			query.WriteString("\nON DUPLICATE KEY UPDATE Snatched = Snatched + VALUES(Snatched), " +
				"Seeders = VALUES(Seeders), Leechers = VALUES(Leechers), " +
				"last_action = IF(last_action < VALUES(last_action), VALUES(last_action), last_action);")

			c.Exec(query.String())

			if length < cap(c.torrentChannel)/2 {
				time.Sleep(200 * time.Millisecond)
			}
		} else if c.terminate {
			break
		} else {
			time.Sleep(time.Second)
		}
	}
}

func (c *Conn) flushUsers()           {}
func (c *Conn) flushTransferHistory() {}
func (c *Conn) flushTransferIps()     {}
func (c *Conn) flushSnatches()        {}
