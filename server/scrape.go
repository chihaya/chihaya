package server

import (
	"bytes"
	cdb "chihaya/database"
	"chihaya/util"
)

func scrape(params *queryParams, db *cdb.Database, buf *bytes.Buffer) {
	buf.WriteRune('d')
	util.Bencode("files", buf)
	db.TorrentsMutex.RLock()
	if params.infoHashes != nil {
		for _, infoHash := range params.infoHashes {
			torrent, exists := db.Torrents[infoHash]
			if exists {
				util.Bencode(infoHash, buf)
				buf.WriteRune('d')
				util.Bencode("complete", buf)
				util.Bencode(len(torrent.Seeders), buf)
				util.Bencode("downloaded", buf)
				util.Bencode(torrent.Snatched, buf)
				util.Bencode("incomplete", buf)
				util.Bencode(len(torrent.Leechers), buf)
				buf.WriteRune('e')
			}
		}
	} else if infoHash, exists := params.get("info_hash"); exists {
		torrent, exists := db.Torrents[infoHash]
		if exists {
			util.Bencode(infoHash, buf)
			buf.WriteRune('d')
			util.Bencode("complete", buf)
			util.Bencode(len(torrent.Seeders), buf)
			util.Bencode("downloaded", buf)
			util.Bencode(torrent.Snatched, buf)
			util.Bencode("incomplete", buf)
			util.Bencode(len(torrent.Leechers), buf)
			buf.WriteRune('e')
		}
	}
	db.TorrentsMutex.RUnlock()
	buf.WriteRune('e')
}
