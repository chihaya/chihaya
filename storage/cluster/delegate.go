package cluster

import (
	"bytes"
	"encoding/gob"
	"sort"

	"github.com/chihaya/chihaya/bittorrent"
	"github.com/chihaya/chihaya/pkg/log"
	"github.com/hashicorp/memberlist"
)

func (ps *peerStore) NotifyJoin(n *memberlist.Node) {
	ps.mutex.Lock()
	defer ps.mutex.Unlock()

	ps.nodesByName[n.Name] = n

	nodes := make([]*memberlist.Node, 0)
	for _, v := range ps.nodesByName {
		nodes = append(nodes, v)
	}

	sort.SliceStable(nodes, func(i, j int) bool {
		return nodes[i].Name < nodes[j].Name
	})

	ps.nodes = nodes

	if ps.nodeName != n.Name {
		log.Info("A node joined the cluster", log.Fields{
			"name":  n.Name,
			"total": len(nodes),
		})
	}
}

func (ps *peerStore) NotifyLeave(n *memberlist.Node) {
	ps.mutex.Lock()
	defer ps.mutex.Unlock()

	delete(ps.nodesByName, n.Name)

	nodes := make([]*memberlist.Node, 0)
	for _, v := range ps.nodesByName {
		nodes = append(nodes, v)
	}

	sort.SliceStable(nodes, func(i, j int) bool {
		return nodes[i].Name < nodes[j].Name
	})

	ps.nodes = nodes

	log.Info("A node left the cluster", log.Fields{
		"name":  n.Name,
		"total": len(nodes),
	})
}

func (ps *peerStore) NotifyUpdate(n *memberlist.Node)            {}
func (ps *peerStore) NodeMeta(limit int) []byte                  { return nil }
func (ps *peerStore) GetBroadcasts(overhead, limit int) [][]byte { return nil }
func (ps *peerStore) LocalState(join bool) []byte                { return nil }
func (ps *peerStore) MergeRemoteState(buf []byte, join bool)     {}

func (ps *peerStore) NotifyMsg(msg []byte) {
	decoder := gob.NewDecoder(bytes.NewReader(msg))
	cmd := uint8(0)

	if err := decoder.Decode(&cmd); err != nil {
		log.Error("Failed to decode notification", log.Err(err))
		return
	}

	switch cmd {
	case CmdPutSeeder:
		data := CmdPutSeederData{}
		if err := decoder.Decode(&data); err != nil {
			log.Error("Failed to decode CmdPutSeeder", log.Err(err))
			return
		}

		ps.ms.PutSeeder(data.InfoHash, data.Peer)

	case CmdPutLeecher:
		data := CmdPutLeecherData{}
		if err := decoder.Decode(&data); err != nil {
			log.Error("Failed to decode CmdPutLeecher", log.Err(err))
			return
		}

		ps.ms.PutLeecher(data.InfoHash, data.Peer)

	case CmdDeleteSeeder:
		data := CmdDeleteSeederData{}
		if err := decoder.Decode(&data); err != nil {
			log.Error("Failed to decode CmdDeleteSeeder", log.Err(err))
			return
		}

		ps.ms.DeleteSeeder(data.InfoHash, data.Peer)

	case CmdDeleteLeecher:
		data := CmdDeleteLeecherData{}
		if err := decoder.Decode(&data); err != nil {
			log.Error("Failed to decode CmdDeleteLeecher", log.Err(err))
			return
		}

		ps.ms.DeleteLeecher(data.InfoHash, data.Peer)

	case CmdGraduateLeecher:
		data := CmdGraduateLeecherData{}
		if err := decoder.Decode(&data); err != nil {
			log.Error("Failed to decode CmdGraduateLeecher", log.Err(err))
			return
		}

		ps.ms.GraduateLeecher(data.InfoHash, data.Peer)

	case CmdAnnouncePeersRequest:
		data := CmdAnnouncePeersRequestData{}
		if err := decoder.Decode(&data); err != nil {
			log.Error("Failed to decode CmdAnnouncePeersRequest", log.Err(err))
			return
		}

		buffer := bytes.Buffer{}
		encoder := gob.NewEncoder(&buffer)
		peers, err := ps.ms.AnnouncePeers(data.InfoHash, data.Seeder, data.NumWant, data.Announcer)

		encoder.Encode(CmdAnnouncePeersResponse)
		encoder.Encode(CmdAnnouncePeersResponseData{
			RequestID: data.RequestID,
			Error:     err,
			Peers:     peers,
		})

		ps.mutex.RLock()
		node := ps.nodesByName[data.NodeName]
		ps.mutex.RUnlock()

		ps.cluster.SendReliable(node, buffer.Bytes())

	case CmdAnnouncePeersResponse:
		data := CmdAnnouncePeersResponseData{}
		if err := decoder.Decode(&data); err != nil {
			log.Error("Failed to decode CmdAnnouncePeersResponse", log.Err(err))
			return
		}

		c, ok := ps.pending.Load(data.RequestID)
		if !ok {
			log.Error("Request ID isn't in the pending map")
			return
		}

		c.(chan []bittorrent.Peer) <- data.Peers

	case CmdScrapeSwarmRequest:
		data := CmdScrapeSwarmRequestData{}
		if err := decoder.Decode(&data); err != nil {
			log.Error("Failed to decode CmdScrapeSwarmRequest", log.Err(err))
			return
		}

		buffer := bytes.Buffer{}
		encoder := gob.NewEncoder(&buffer)
		scrape := ps.ms.ScrapeSwarm(data.InfoHash, data.AddressFamily)

		encoder.Encode(CmdScrapeSwarmResponse)
		encoder.Encode(CmdScrapeSwarmResponseData{
			RequestID: data.RequestID,
			Scrape:    scrape,
		})

		ps.mutex.RLock()
		node := ps.nodesByName[data.NodeName]
		ps.mutex.RUnlock()

		ps.cluster.SendReliable(node, buffer.Bytes())

	case CmdScrapeSwarmResponse:
		data := CmdScrapeSwarmResponseData{}
		if err := decoder.Decode(&data); err != nil {
			log.Error("Failed to decode CmdScrapeSwarmResponse", log.Err(err))
			return
		}

		c, ok := ps.pending.Load(data.RequestID)
		if !ok {
			log.Error("Request ID isn't in the pending map")
			return
		}

		c.(chan bittorrent.Scrape) <- data.Scrape

	default:
		log.Error("Unknown notification command", log.Fields{
			"cmd": cmd,
		})
	}
}
