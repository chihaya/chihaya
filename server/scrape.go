package server

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"path"
	"strconv"
	"time"

	"github.com/pushrax/chihaya/storage"
)

func (s *Server) serveScrape(w http.ResponseWriter, r *http.Request) {
	passkey, _ := path.Split(r.URL.Path)
	user, err := validatePasskey(passkey, s.storage)
	if err != nil {
		fail(err, w, r)
		return
	}

	pq, err := parseQuery(r.URL.RawQuery)
	if err != nil {
		fail(errors.New("Error parsing query"), w, r)
		return
	}

	io.WriteString(w, "d")
	bencode(w, "files")
	if pq.infohashes != nil {
		for _, infohash := range pq.infohashes {
			torrent, exists, err := s.storage.FindTorrent(infohash)
			if err != nil {
				panic("server: failed to find torrent")
			}
			if exists {
				bencode(w, infohash)
				writeScrapeInfo(w, torrent)
			}
		}
	} else if infohash, exists := pq.params["info_hash"]; exists {
		torrent, exists, err := s.storage.FindTorrent(infohash)
		if err != nil {
			panic("server: failed to find torrent")
		}
		if exists {
			bencode(w, infohash)
			writeScrapeInfo(w, torrent)
		}
	}
	io.WriteString(w, "e")
	finalizeResponse(w, r)
}

func writeScrapeInfo(w io.Writer, torrent *storage.Torrent) {
	io.WriteString(w, "d")
	bencode(w, "complete")
	bencode(w, len(torrent.Seeders))
	bencode(w, "downloaded")
	bencode(w, torrent.Snatched)
	bencode(w, "incomplete")
	bencode(w, len(torrent.Leechers))
	io.WriteString(w, "e")
}

func bencode(w io.Writer, data interface{}) {
	switch v := data.(type) {
	case string:
		str := fmt.Sprintf("%s:%s", strconv.Itoa(len(v)), v)
		io.WriteString(w, str)
	case int:
		str := fmt.Sprintf("i%se", strconv.Itoa(v))
		io.WriteString(w, str)
	case uint:
		str := fmt.Sprintf("i%se", strconv.FormatUint(uint64(v), 10))
		io.WriteString(w, str)
	case int64:
		str := fmt.Sprintf("i%se", strconv.FormatInt(v, 10))
		io.WriteString(w, str)
	case uint64:
		str := fmt.Sprintf("i%se", strconv.FormatUint(v, 10))
		io.WriteString(w, str)
	case time.Duration: // Assume seconds
		str := fmt.Sprintf("i%se", strconv.FormatInt(int64(v/time.Second), 10))
		io.WriteString(w, str)
	case map[string]interface{}:
		io.WriteString(w, "d")
		for key, val := range v {
			str := fmt.Sprintf("%s:%s", strconv.Itoa(len(key)), key)
			io.WriteString(w, str)
			bencode(w, val)
		}
		io.WriteString(w, "e")
	case []string:
		io.WriteString(w, "l")
		for _, val := range v {
			bencode(w, val)
		}
		io.WriteString(w, "e")
	default:
		// Although not currently necessary,
		// should handle []interface{} manually; Go can't do it implicitly
		panic("Tried to bencode an unsupported type!")
	}
}
