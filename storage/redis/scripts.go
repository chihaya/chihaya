package redis

import (
	"github.com/gomodule/redigo/redis"

	"github.com/chihaya/chihaya/bittorrent"
)

// scripts holds all of the scripts necessary.
type scripts struct {
	putPeer      *putPeerScript
	deletePeer   *deletePeerScript
	graduatePeer *graduatePeerScript
	gcPeer       *gcPeerScript
}

// The putPeerScript is a script that adds or updates a peer to a swarm.
// It handles incrementing counters as well, if necessary.
//
// In pseudocode, this is what this script does:
//
// ```
// new_peer = HSET <swarm_key> <peer_key> <mtime>
// if new_peer != 0
//   HINCRBY <address_family_swarm_counts> <infohash> 1
//   INCR <address_family_{seeder|leecher}_key>
// ```
type putPeerScript struct {
	script *redis.Script
}

func newPutPeerScript() *putPeerScript {
	src := `
	local peer_key = ARGV[1]
	local mtime = ARGV[2]
	local info_hash = ARGV[3]
	local new_peer = redis.call('HSET', KEYS[1], peer_key, mtime)
	if new_peer == 1 then
	  redis.call('HINCRBY', KEYS[2], info_hash, 1)
	  redis.call('INCR', KEYS[3])
	end`

	return &putPeerScript{script: redis.NewScript(3, src)}
}

func (s *putPeerScript) execute(conn redis.Conn, peer bittorrent.Peer, mtime int64, infoHash bittorrent.InfoHash, seeder bool) error {
	var swarmKey, countKey string
	if seeder {
		swarmKey = seederSwarmKey(peer.IP.AddressFamily, infoHash)
		countKey = seederCountKey(peer.IP.AddressFamily)
	} else {
		swarmKey = leecherSwarmKey(peer.IP.AddressFamily, infoHash)
		countKey = leecherCountKey(peer.IP.AddressFamily)
	}

	_, err := s.script.Do(conn,
		swarmKey,
		swarmCountsKey(peer.IP.AddressFamily),
		countKey,
		makePeerKey(peer), mtime, infoHash.RawString())
	return err
}

// The deletePeerScript removes a peer from a swarm.
// It also handles decrementing counters and removing hash keys, if necessary.
// It returns whether a peer was actually removed.
//
// In pseudocode, this is what the script does:
//
// ```
// removed = HDEL <swarm_key> <peer_key>
// if removed != 0
//   peers_in_swarm = HINCRBY <address_family_swarm_counts> <infohash> -1
//   if peers_in_swarm == 0
//     HDEL <address_family_swarm_counts> <infohash>
//   DECR <address_family_{seeder|leecher}_key>
// return removed
// ```
type deletePeerScript struct {
	script *redis.Script
}

func newDeletePeerScript() *deletePeerScript {
	src := `
	local peer_key = ARGV[1]
	local info_hash = ARGV[2]
	local removed = redis.call('HDEL', KEYS[1], peer_key)
	if removed == 1 then
	  local peers_in_swarm = redis.call('HINCRBY', KEYS[2], info_hash, -1)
	    if peers_in_swarm == 0 then
	      redis.call('HDEL', KEYS[2], info_hash)
	    end
	  redis.call('DECR', KEYS[3])
	end
	return removed`

	return &deletePeerScript{script: redis.NewScript(3, src)}
}

func (s *deletePeerScript) execute(conn redis.Conn, peer bittorrent.Peer, infoHash bittorrent.InfoHash, seeder bool) (int, error) {
	var swarmKey, countKey string
	if seeder {
		swarmKey = seederSwarmKey(peer.IP.AddressFamily, infoHash)
		countKey = seederCountKey(peer.IP.AddressFamily)
	} else {
		swarmKey = leecherSwarmKey(peer.IP.AddressFamily, infoHash)
		countKey = leecherCountKey(peer.IP.AddressFamily)
	}

	return redis.Int(s.script.Do(conn,
		swarmKey,
		swarmCountsKey(peer.IP.AddressFamily),
		countKey,
		makePeerKey(peer), infoHash.RawString()))
}

// The graduatePeerScript handles graduating a peer from a leecher to a seeder.
// It updates the necessary counters and deletes keys as necessary.
//
// In pseudocode, this is what the script does:
//
// ```
// new_seeder = HSET <seeder_swarm_key> <peer_key> <mtime>
// leecher_removed = HDEL <leecher_swarm_key> <peer_key>
// if new_seeder != 0
//   HINCRBY <address_family_swarm_counts> <infohash> 1
//   INCR <address_family_seeder_key>
// if leecher_removed != 0
//   peers_in_swarm = HINCRBY <address_family_swarm_counts> <infohash> -1
//   if peers_in_swarm == 0 // this should never happen
//     error("swarm empty after graduation")
//   DECR <address_family_{seeder|leecher}_key>
// ```
type graduatePeerScript struct {
	script *redis.Script
}

func newGraduatePeerScript() *graduatePeerScript {
	src := `
	local peer_key = ARGV[1]
	local mtime = ARGV[2]
	local info_hash = ARGV[3]
	local new_seeder = redis.call('HSET', KEYS[1], peer_key, mtime)
	local leecher_removed = redis.call('HDEL', KEYS[2], peer_key)
	if new_seeder == 1 then
	  redis.call('HINCRBY', KEYS[3], info_hash, 1)
	  redis.call('INCR', KEYS[4])
	end
	if leecher_removed == 1 then
	  local peers_in_swarm = redis.call('HINCRBY', KEYS[3], info_hash, -1)
	    if peers_in_swarm == 0 then -- This should never happen
	      error('swarm empty after graduation')
	    end
	  redis.call('DECR', KEYS[5])
	end`

	return &graduatePeerScript{script: redis.NewScript(5, src)}
}

func (s *graduatePeerScript) execute(conn redis.Conn, peer bittorrent.Peer, mtime int64, infoHash bittorrent.InfoHash) error {
	_, err := s.script.Do(conn,
		seederSwarmKey(peer.IP.AddressFamily, infoHash),
		leecherSwarmKey(peer.IP.AddressFamily, infoHash),
		swarmCountsKey(peer.IP.AddressFamily),
		seederCountKey(peer.IP.AddressFamily),
		leecherCountKey(peer.IP.AddressFamily),
		makePeerKey(peer), mtime, infoHash.RawString())
	return err
}

// The gcPeerScript handles garbage collection of a single peer.
// It updates counters and deletes hash keys as necessary.
// It returns the number of actually removed peers.
// The script compares the mtime of the peer in redis with the mtime provided to
// it via arguments and only removes the peer if they match.
// This should guarantee that we operate race-free :)
//
// In pseudocode, this is what this script does:
//
// ```
// mtime = HGET <swarm_key> <peer_key>
// if mtime != <mtime arg>
//   return 0
// removed = HDEL <swarm_key> <peer_key>
// if removed != 0
//   peers_in_swarm = HINCRBY <address_family_swarm_counts> <infohash> -1
//   if peers_in_swarm == 0
//     HDEL <address_family_swarm_counts> <infohash>
//   DECR <address_family_{seeder|leecher}_key>
// return removed
// ```
type gcPeerScript struct {
	script *redis.Script
}

func newGCPeerScript() *gcPeerScript {
	src := `
	local peer_key = ARGV[1]
	local expected_mtime = ARGV[2]
	local info_hash = ARGV[3]
	local mtime = redis.call('HGET', KEYS[1], peer_key)
	if mtime ~= expected_mtime then
	  return 0
	end
	
	local removed = redis.call('HDEL', KEYS[1], peer_key)
	if removed == 1 then
	  local peers_in_swarm = redis.call('HINCRBY', KEYS[2], info_hash, -1)
	    if peers_in_swarm == 0 then
	      redis.call('HDEL', KEYS[2], info_hash)
	    end
	  redis.call('DECR', KEYS[3])
	end
	
	return removed`

	return &gcPeerScript{script: redis.NewScript(3, src)}
}

func (s *gcPeerScript) execute(conn redis.Conn, peer bittorrent.Peer, mtime int64, infoHash bittorrent.InfoHash, seeder bool) (int, error) {
	var swarmKey, countKey string
	if seeder {
		swarmKey = seederSwarmKey(peer.IP.AddressFamily, infoHash)
		countKey = seederCountKey(peer.IP.AddressFamily)
	} else {
		swarmKey = leecherSwarmKey(peer.IP.AddressFamily, infoHash)
		countKey = leecherCountKey(peer.IP.AddressFamily)
	}

	return redis.Int(s.script.Do(conn,
		swarmKey,
		swarmCountsKey(peer.IP.AddressFamily),
		countKey,
		makePeerKey(peer), mtime, infoHash.RawString()))
}
