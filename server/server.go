package server

import (
	"bytes"
	"chihaya/config"
	cdb "chihaya/database"
	"chihaya/util"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"sync"
	"sync/atomic"
	"time"
)

type httpHandler struct {
	db         *cdb.Database
	bufferPool *util.BufferPool
	waitGroup  sync.WaitGroup
	startTime  time.Time
	terminate  bool

	// Internal stats
	deltaRequests int64
	throughput    float64
}

type queryParams struct {
	params     map[string]string
	infoHashes []string
}

func (p *queryParams) get(which string) (ret string, exists bool) {
	ret, exists = p.params[which]
	return
}

func (p *queryParams) getUint64(which string) (ret uint64, exists bool) {
	str, exists := p.params[which]
	if exists {
		var err error
		exists = false
		ret, err = strconv.ParseUint(str, 10, 64)
		if err == nil {
			exists = true
		}
	}
	return
}

func failure(err string, buf *bytes.Buffer) {
	buf.WriteString("d14:failure reason")
	buf.WriteString(strconv.Itoa(len(err)))
	buf.WriteRune(':')
	buf.WriteString(err)
	buf.WriteRune('e')
}

/*
 * URL.Query() is rather slow, so I rewrote it
 * Since the only parameter that can have multiple values is info_hash for scrapes, handle this specifically
 */
func (handler *httpHandler) parseQuery(query string) (ret *queryParams, err error) {
	ret = &queryParams{make(map[string]string), nil}
	queryLen := len(query)

	onKey := true

	var keyStart int
	var keyEnd int
	var valStart int
	var valEnd int

	hasInfoHash := false
	var firstInfoHash string

	for i := 0; i < queryLen; i++ {
		separator := query[i] == '&' || query[i] == ';'
		if separator || i == queryLen-1 { // ';' is a valid separator as per W3C spec
			if onKey {
				keyStart = i + 1
				continue
			}

			if i == queryLen-1 && !separator {
				if query[i] == '=' {
					continue
				}
				valEnd = i
			}

			keyStr, err1 := url.QueryUnescape(query[keyStart : keyEnd+1])
			if err1 != nil {
				err = err1
				return
			}
			valStr, err1 := url.QueryUnescape(query[valStart : valEnd+1])
			if err != nil {
				err = err1
				return
			}

			ret.params[keyStr] = valStr

			if keyStr == "info_hash" {
				if hasInfoHash {
					// There is more than one info_hash
					if ret.infoHashes == nil {
						ret.infoHashes = []string{firstInfoHash}
					}
					ret.infoHashes = append(ret.infoHashes, valStr)
				} else {
					firstInfoHash = valStr
					hasInfoHash = true
				}
			}
			onKey = true
			keyStart = i + 1
		} else if query[i] == '=' {
			onKey = false
			valStart = i + 1
		} else if onKey {
			keyEnd = i
		} else {
			valEnd = i
		}
	}
	return
}

func (handler *httpHandler) respond(r *http.Request, buf *bytes.Buffer) {
	dir, action := path.Split(r.URL.Path)
	if len(dir) != 34 {
		failure("Your passkey is invalid", buf)
		return
	}

	passkey := dir[1:33]

	params, err := handler.parseQuery(r.URL.RawQuery)

	if err != nil {
		failure("Error parsing query", buf)
		return
	}

	handler.db.UsersMutex.RLock()
	user, exists := handler.db.Users[passkey]
	handler.db.UsersMutex.RUnlock()
	if !exists {
		failure("Passkey not found", buf)
		return
	}

	ip, exists := params.get("ip")
	if !exists {
		ip, exists = params.get("ipv4")
		if !exists {
			ips, exists := r.Header["X-Real-Ip"]
			if exists && len(ips) > 0 {
				ip = ips[0]
			} else {
				portIndex := len(r.RemoteAddr) - 1
				for ; portIndex >= 0; portIndex-- {
					if r.RemoteAddr[portIndex] == ':' {
						break
					}
				}
				if portIndex != -1 {
					ip = r.RemoteAddr[0:portIndex]
				} else {
					failure("Failed to parse IP address", buf)
					return
				}
			}
		}
	}

	switch action {
	case "announce":
		announce(params, user, ip, handler.db, buf)
		return
	case "scrape":
		scrape(params, handler.db, buf)
		return
	}

	failure("Unknown action", buf)
}

var handler *httpHandler
var listener net.Listener

func (handler *httpHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if handler.terminate {
		return
	}
	handler.waitGroup.Add(1)
	defer handler.waitGroup.Done()

	buf := handler.bufferPool.Take()
	defer handler.bufferPool.Give(buf)

	//log.Println(r.URL)

	if r.URL.Path == "/stats" {
		db := handler.db
		peers := 0

		db.UsersMutex.RLock()
		db.TorrentsMutex.RLock()

		for _, t := range db.Torrents {
			peers += len(t.Leechers) + len(t.Seeders)
		}

		buf.WriteString(fmt.Sprintf("Uptime: %f\nUsers: %d\nTorrents: %d\nPeers: %d\nThroughput (last minute): %f req/s\n",
			time.Now().Sub(handler.startTime).Seconds(),
			len(db.Users),
			len(db.Torrents),
			peers,
			handler.throughput,
		))

		db.UsersMutex.RUnlock()
		db.TorrentsMutex.RUnlock()
	} else {
		handler.respond(r, buf)
	}

	/*
	 * Could do gzip here, but I'm not sure if it's worth it for compact responses.
	 * Also, according to the Ocelot source code:
	 *   "gzip compression actually makes announce returns larger from our testing."
	 *
	 * TODO: investigate this
	 */

	r.Close = true
	w.Header().Add("Content-Type", "text/plain")
	w.Header().Add("Connection", "close")
	w.Header().Add("Content-Length", strconv.Itoa(buf.Len()))

	// It would probably be good to use real response codes, but no common client actually cares

	w.Write(buf.Bytes())

	atomic.AddInt64(&handler.deltaRequests, 1)

	w.(http.Flusher).Flush()
}

func Start() {
	var err error

	handler = &httpHandler{db: &cdb.Database{}, startTime: time.Now()}

	bufferPool := util.NewBufferPool(500, 500)
	handler.bufferPool = bufferPool

	server := &http.Server{
		Handler:     handler,
		ReadTimeout: 20 * time.Second,
	}

	go collectStatistics()

	handler.db.Init()

	listener, err = net.Listen("tcp", config.Get("addr"))

	if err != nil {
		panic(err)
	}

	/*
	 * Behind the scenes, this works by spawning a new goroutine for each client.
	 * This is pretty fast and scalable since goroutines are nice and efficient.
	 */
	server.Serve(listener)

	// Wait for active connections to finish processing
	handler.waitGroup.Wait()

	handler.db.Terminate()

	log.Println("Shutdown complete")
}

func Stop() {
	// Closing the listener stops accepting connections and causes Serve to return
	listener.Close()
	handler.terminate = true
}

func collectStatistics() {
	lastTime := time.Now()
	for {
		time.Sleep(time.Minute)
		duration := time.Now().Sub(lastTime)
		handler.throughput = float64(handler.deltaRequests) / duration.Seconds()
		atomic.StoreInt64(&handler.deltaRequests, 0)

		log.Printf("Throughput last minute: %4f req/s\n", handler.throughput)
		lastTime = time.Now()
	}
}
