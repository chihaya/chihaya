package cluster

import (
	"github.com/chihaya/chihaya/bittorrent"
	"github.com/google/uuid"
)

const (
	CmdPutSeeder             uint8 = 1
	CmdPutLeecher            uint8 = 2
	CmdDeleteSeeder          uint8 = 3
	CmdDeleteLeecher         uint8 = 4
	CmdGraduateLeecher       uint8 = 5
	CmdAnnouncePeersRequest  uint8 = 6
	CmdAnnouncePeersResponse uint8 = 7
	CmdScrapeSwarmRequest    uint8 = 8
	CmdScrapeSwarmResponse   uint8 = 9
)

type CmdPutSeederData struct {
	InfoHash bittorrent.InfoHash
	Peer     bittorrent.Peer
}

type CmdPutLeecherData struct {
	InfoHash bittorrent.InfoHash
	Peer     bittorrent.Peer
}

type CmdDeleteSeederData struct {
	InfoHash bittorrent.InfoHash
	Peer     bittorrent.Peer
}

type CmdDeleteLeecherData struct {
	InfoHash bittorrent.InfoHash
	Peer     bittorrent.Peer
}

type CmdGraduateLeecherData struct {
	InfoHash bittorrent.InfoHash
	Peer     bittorrent.Peer
}

type CmdAnnouncePeersRequestData struct {
	RequestID uuid.UUID
	InfoHash  bittorrent.InfoHash
	Announcer bittorrent.Peer
	NodeName  string
	Seeder    bool
	NumWant   int
}

type CmdAnnouncePeersResponseData struct {
	RequestID uuid.UUID
	Error     error
	Peers     []bittorrent.Peer
}

type CmdScrapeSwarmRequestData struct {
	RequestID     uuid.UUID
	InfoHash      bittorrent.InfoHash
	NodeName      string
	AddressFamily bittorrent.AddressFamily
}

type CmdScrapeSwarmResponseData struct {
	RequestID uuid.UUID
	Scrape    bittorrent.Scrape
}
