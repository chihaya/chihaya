// Copyright 2013 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

import (
	"bytes"

	cdb "github.com/kotokoko/chihaya/database"
	"github.com/kotokoko/chihaya/util"
)

func writeScrapeInfo(torrent *cdb.Torrent, buf *bytes.Buffer) {
	buf.WriteRune('d')
	util.Bencode("complete", buf)
	util.Bencode(len(torrent.Seeders), buf)
	util.Bencode("downloaded", buf)
	util.Bencode(torrent.Snatched, buf)
	util.Bencode("incomplete", buf)
	util.Bencode(len(torrent.Leechers), buf)
	buf.WriteRune('e')
}

func scrape(params *queryParams, db *cdb.Database, buf *bytes.Buffer) {
	buf.WriteRune('d')
	util.Bencode("files", buf)
	db.TorrentsMutex.RLock()
	if params.infoHashes != nil {
		for _, infoHash := range params.infoHashes {
			torrent, exists := db.Torrents[infoHash]
			if exists {
				util.Bencode(infoHash, buf)
				writeScrapeInfo(torrent, buf)
			}
		}
	} else if infoHash, exists := params.get("info_hash"); exists {
		torrent, exists := db.Torrents[infoHash]
		if exists {
			util.Bencode(infoHash, buf)
			writeScrapeInfo(torrent, buf)
		}
	}
	db.TorrentsMutex.RUnlock()
	buf.WriteRune('e')
}
