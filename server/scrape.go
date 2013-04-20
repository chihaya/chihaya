// This file is part of Chihaya.
//
// Chihaya is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// Chihaya is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with Chihaya.  If not, see <http://www.gnu.org/licenses/>.

package server

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
