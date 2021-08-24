package main

import (
	"crypto/rand"
	"fmt"
	"time"

	"github.com/anacrolix/torrent/tracker"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"

	"github.com/chihaya/chihaya/bittorrent"
)

// EndToEndRunCmdFunc implements a Cobra command that runs the end-to-end test
// suite for a Chihaya build.
func EndToEndRunCmdFunc(cmd *cobra.Command, args []string) error {
	delay, err := cmd.Flags().GetDuration("delay")
	if err != nil {
		return err
	}

	// Test the HTTP tracker
	httpAddr, err := cmd.Flags().GetString("httpaddr")
	if err != nil {
		return err
	}

	if len(httpAddr) != 0 {
		log.Info().Msg("testing HTTP...")
		err := test(httpAddr, delay)
		if err != nil {
			return err
		}
		log.Info().Msg("success")
	}

	// Test the UDP tracker.
	udpAddr, err := cmd.Flags().GetString("udpaddr")
	if err != nil {
		return err
	}

	if len(udpAddr) != 0 {
		log.Info().Msg("testing UDP...")
		err := test(udpAddr, delay)
		if err != nil {
			return err
		}
		log.Info().Msg("success")
	}

	return nil
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

func test(addr string, delay time.Duration) error {
	ih := generateInfohash()
	return testWithInfohash(ih, addr, delay)
}

func testWithInfohash(infoHash [20]byte, url string, delay time.Duration) error {
	req := tracker.AnnounceRequest{
		InfoHash:   infoHash,
		PeerId:     [20]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20},
		Downloaded: 50,
		Left:       100,
		Uploaded:   50,
		Event:      tracker.Started,
		IPAddress:  uint32(50<<24 | 10<<16 | 12<<8 | 1),
		NumWant:    50,
		Port:       10001,
	}

	resp, err := tracker.Announce{
		TrackerUrl: url,
		Request:    req,
		UserAgent:  "chihaya-e2e",
	}.Do()
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
		IPAddress:  uint32(50<<24 | 10<<16 | 12<<8 | 2),
		NumWant:    50,
		Port:       10002,
	}

	resp, err = tracker.Announce{
		TrackerUrl: url,
		Request:    req,
		UserAgent:  "chihaya-e2e",
	}.Do()
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
