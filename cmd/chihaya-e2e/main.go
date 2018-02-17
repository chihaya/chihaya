package main

import (
	"crypto/rand"
	"flag"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/anacrolix/torrent/tracker"
	"github.com/pkg/errors"

	"github.com/chihaya/chihaya/bittorrent"
)

func init() {
	flag.StringVar(&httpTrackerURL, "http", "http://127.0.0.1:6969/announce", "the address of the HTTP tracker")
	flag.StringVar(&udpTrackerURL, "udp", "udp://127.0.0.1:6969", "the address of the UDP tracker")
	flag.DurationVar(&delay, "delay", 1*time.Second, "the delay between announces")
}

var (
	httpTrackerURL string
	udpTrackerURL  string
	delay          time.Duration
)

func main() {
	flag.Parse()

	if len(httpTrackerURL) != 0 {
		fmt.Println("testing HTTP...")
		err := testHTTP()
		if err != nil {
			fmt.Println("failed:", err)
			os.Exit(1)
		}
		fmt.Println("success")
	}

	if len(udpTrackerURL) != 0 {
		fmt.Println("testing UDP...")
		err := testUDP()
		if err != nil {
			fmt.Println("failed:", err)
			os.Exit(1)
		}
		fmt.Println("success")
	}
}

func generateInfohash() [20]byte {
	b := make([]byte, 20)

	n, err := rand.Read(b)
	if err != nil {
		panic(err)
	}
	if n != 20 {
		panic(fmt.Errorf("not enough randomness? Got %d bytes", n))
	}

	return [20]byte(bittorrent.InfoHashFromBytes(b))
}

func testUDP() error {
	ih := generateInfohash()
	return testWithInfohash(ih, udpTrackerURL)
}

func testHTTP() error {
	ih := generateInfohash()
	return testWithInfohash(ih, httpTrackerURL)
}

func testWithInfohash(infoHash [20]byte, url string) error {
	req := tracker.AnnounceRequest{
		InfoHash:   infoHash,
		PeerId:     [20]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20},
		Downloaded: 50,
		Left:       100,
		Uploaded:   50,
		Event:      tracker.Started,
		IPAddress:  int32(50<<24 | 10<<16 | 12<<8 | 1),
		NumWant:    50,
		Port:       10001,
	}

	resp, err := tracker.Announce(&http.Client{}, "ekop", url, &req)
	if err != nil {
		return errors.Wrap(err, "announce failed")
	}

	if len(resp.Peers) != 1 {
		return fmt.Errorf("expected one peer, got %d", len(resp.Peers))
	}

	time.Sleep(delay)

	req = tracker.AnnounceRequest{
		InfoHash:   infoHash,
		PeerId:     [20]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 21},
		Downloaded: 50,
		Left:       100,
		Uploaded:   50,
		Event:      tracker.Started,
		IPAddress:  int32(50<<24 | 10<<16 | 12<<8 | 2),
		NumWant:    50,
		Port:       10002,
	}

	resp, err = tracker.Announce(&http.Client{}, "ekop", url, &req)
	if err != nil {
		return errors.Wrap(err, "announce failed")
	}

	if len(resp.Peers) != 1 {
		return fmt.Errorf("expected 1 peers, got %d", len(resp.Peers))
	}

	if resp.Peers[0].Port != 10001 {
		return fmt.Errorf("expected port 10001, got %d ", resp.Peers[0].Port)
	}

	return nil
}
