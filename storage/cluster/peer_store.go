// Package memory implements the storage interface for a Chihaya
// BitTorrent tracker keeping peer data in memory.
package cluster

import (
	"bytes"
	"encoding/gob"
	"errors"
	"fmt"
	"hash/fnv"
	"math"
	"sync"
	"time"

	yaml "gopkg.in/yaml.v2"

	"github.com/chihaya/chihaya/bittorrent"
	"github.com/chihaya/chihaya/pkg/log"
	"github.com/chihaya/chihaya/pkg/stop"
	"github.com/chihaya/chihaya/storage"
	"github.com/chihaya/chihaya/storage/memory"
	"github.com/google/uuid"
	"github.com/hashicorp/memberlist"
)

// Name is the name by which this peer store is registered with Chihaya.
const Name = "cluster"

// Default config constants.
const (
	defaultShardCount                  = 1024
	defaultPrometheusReportingInterval = time.Second * 1
	defaultGarbageCollectionInterval   = time.Minute * 3
	defaultPeerLifetime                = time.Minute * 30
	defaultRelayTimeout                = time.Second * 5
	defaultJoinAddr                    = "localhost"
	defaultJoinPort                    = 7946
	defaultBindAddr                    = "0.0.0.0"
	defaultBindPort                    = 7946
	defaultAdvertiseAddr               = ""
	defaultAdvertisePort               = 7946
)

func init() {
	// Register the storage driver.
	storage.RegisterDriver(Name, driver{})
	gob.Register(bittorrent.ClientError(""))
}

type driver struct{}

func (d driver) NewPeerStore(icfg interface{}) (storage.PeerStore, error) {
	// Marshal the config back into bytes.
	bytes, err := yaml.Marshal(icfg)
	if err != nil {
		return nil, err
	}

	// Unmarshal the bytes into the proper config type.
	var cfg Config
	err = yaml.Unmarshal(bytes, &cfg)
	if err != nil {
		return nil, err
	}

	return New(cfg)
}

// Config holds the configuration of a memory PeerStore.
type Config struct {
	GarbageCollectionInterval   time.Duration `yaml:"gc_interval"`
	PrometheusReportingInterval time.Duration `yaml:"prometheus_reporting_interval"`
	PeerLifetime                time.Duration `yaml:"peer_lifetime"`
	RelayTimeout                time.Duration `yaml:"relay_timeout"`
	ShardCount                  int           `yaml:"shard_count"`
	JoinAddr                    string        `yaml:"join_addr"`
	JoinPort                    int           `yaml:"join_port"`
	BindAddr                    string        `yaml:"bind_addr"`
	BindPort                    int           `yaml:"bind_port"`
	AdvertiseAddr               string        `yaml:"advertise_addr"`
	AdvertisePort               int           `yaml:"advertise_port"`
}

// LogFields renders the current config as a set of Logrus fields.
func (cfg Config) LogFields() log.Fields {
	return log.Fields{
		"name":               Name,
		"gcInterval":         cfg.GarbageCollectionInterval,
		"promReportInterval": cfg.PrometheusReportingInterval,
		"peerLifetime":       cfg.PeerLifetime,
		"relayTimeout":       cfg.RelayTimeout,
		"shardCount":         cfg.ShardCount,
		"joinAddr":           cfg.JoinAddr,
		"joinPort":           cfg.JoinPort,
		"bindAddr":           cfg.BindAddr,
		"bindPort":           cfg.BindPort,
		"advertiseAddr":      cfg.AdvertiseAddr,
		"advertisePort":      cfg.AdvertisePort,
	}
}

// Validate sanity checks values set in a config and returns a new config with
// default values replacing anything that is invalid.
//
// This function warns to the logger when a value is changed.
func (cfg Config) Validate() Config {
	validcfg := cfg

	if cfg.RelayTimeout == 0 {
		validcfg.RelayTimeout = defaultRelayTimeout
		log.Warn("falling back to default configuration", log.Fields{
			"name":     Name + ".RelayTimeout",
			"provided": cfg.RelayTimeout,
			"default":  validcfg.RelayTimeout,
		})
	}

	if cfg.AdvertisePort == 0 {
		validcfg.AdvertisePort = defaultAdvertisePort
		log.Warn("falling back to default configuration", log.Fields{
			"name":     Name + ".AdvertisePort",
			"provided": cfg.AdvertisePort,
			"default":  validcfg.AdvertisePort,
		})
	}

	if cfg.BindAddr == "" {
		validcfg.BindAddr = defaultBindAddr
		log.Warn("falling back to default configuration", log.Fields{
			"name":     Name + ".BindAddr",
			"provided": cfg.BindAddr,
			"default":  validcfg.BindAddr,
		})
	}

	if cfg.BindPort == 0 {
		validcfg.BindPort = defaultBindPort
		log.Warn("falling back to default configuration", log.Fields{
			"name":     Name + ".BindPort",
			"provided": cfg.BindPort,
			"default":  validcfg.BindPort,
		})
	}

	if cfg.JoinAddr == "" {
		validcfg.JoinAddr = defaultJoinAddr
		log.Warn("falling back to default configuration", log.Fields{
			"name":     Name + ".JoinAddr",
			"provided": cfg.JoinAddr,
			"default":  validcfg.JoinAddr,
		})
	}

	if cfg.JoinPort == 0 {
		validcfg.JoinPort = defaultJoinPort
		log.Warn("falling back to default configuration", log.Fields{
			"name":     Name + ".JoinPort",
			"provided": cfg.JoinPort,
			"default":  validcfg.JoinPort,
		})
	}

	if cfg.ShardCount <= 0 || cfg.ShardCount > (math.MaxInt/2) {
		validcfg.ShardCount = defaultShardCount
		log.Warn("falling back to default configuration", log.Fields{
			"name":     Name + ".ShardCount",
			"provided": cfg.ShardCount,
			"default":  validcfg.ShardCount,
		})
	}

	if cfg.GarbageCollectionInterval <= 0 {
		validcfg.GarbageCollectionInterval = defaultGarbageCollectionInterval
		log.Warn("falling back to default configuration", log.Fields{
			"name":     Name + ".GarbageCollectionInterval",
			"provided": cfg.GarbageCollectionInterval,
			"default":  validcfg.GarbageCollectionInterval,
		})
	}

	if cfg.PrometheusReportingInterval <= 0 {
		validcfg.PrometheusReportingInterval = defaultPrometheusReportingInterval
		log.Warn("falling back to default configuration", log.Fields{
			"name":     Name + ".PrometheusReportingInterval",
			"provided": cfg.PrometheusReportingInterval,
			"default":  validcfg.PrometheusReportingInterval,
		})
	}

	if cfg.PeerLifetime <= 0 {
		validcfg.PeerLifetime = defaultPeerLifetime
		log.Warn("falling back to default configuration", log.Fields{
			"name":     Name + ".PeerLifetime",
			"provided": cfg.PeerLifetime,
			"default":  validcfg.PeerLifetime,
		})
	}

	return validcfg
}

// New creates a new PeerStore backed by a cluster.
func New(provided Config) (storage.PeerStore, error) {
	cfg := provided.Validate()

	// nodeName, _ := os.Hostname()
	nodeName := uuid.New().String()

	ps := &peerStore{
		cfg:         cfg,
		closed:      make(chan struct{}),
		nodeName:    nodeName,
		nodes:       make([]*memberlist.Node, 0),
		nodesByName: make(map[string]*memberlist.Node),
	}

	ms, err := memory.New(memory.Config{
		GarbageCollectionInterval:   cfg.GarbageCollectionInterval,
		PrometheusReportingInterval: cfg.PrometheusReportingInterval,
		PeerLifetime:                cfg.PeerLifetime,
		ShardCount:                  cfg.ShardCount,
	})

	if err != nil {
		return nil, err
	}

	ps.ms = ms

	clusterCfg := memberlist.DefaultLANConfig()
	clusterCfg.Name = ps.nodeName
	clusterCfg.BindAddr = cfg.BindAddr
	clusterCfg.BindPort = cfg.BindPort
	clusterCfg.AdvertiseAddr = cfg.AdvertiseAddr
	clusterCfg.AdvertisePort = cfg.AdvertisePort
	clusterCfg.Delegate = ps
	clusterCfg.Events = ps

	cluster, err := memberlist.Create(clusterCfg)

	if err != nil {
		return nil, err
	}

	ps.cluster = cluster

	// Start a goroutine for joining the cluster.
	ps.wg.Add(1)
	go func() {
		defer ps.wg.Done()
		ps.joinCluster()
	}()

	return ps, nil
}

type peerStore struct {
	cfg    Config
	ms     storage.PeerStore
	closed chan struct{}
	wg     sync.WaitGroup

	nodeName    string
	nodes       []*memberlist.Node
	nodesByName map[string]*memberlist.Node
	mutex       sync.RWMutex

	cluster *memberlist.Memberlist
	pending sync.Map
}

var _ storage.PeerStore = &peerStore{}

func (ps *peerStore) responsibleNode(ih bittorrent.InfoHash) *memberlist.Node {
	h := fnv.New32a()
	h.Write(ih[:])

	ps.mutex.RLock()
	idx := h.Sum32() % uint32(len(ps.nodes))
	node := ps.nodes[idx]
	ps.mutex.RUnlock()

	return node
}

func (ps *peerStore) joinCluster() {
	log.Info("Trying to join the cluster", log.Fields{
		"addr": ps.cfg.JoinAddr,
		"port": ps.cfg.JoinPort,
	})

	for {
		select {
		case <-ps.closed:
			return
		default:
		}

		_, err := ps.cluster.Join([]string{fmt.Sprintf("%s:%d", ps.cfg.JoinAddr, ps.cfg.JoinPort)})
		if err != nil {
			log.Error("Failed to join cluster, retrying...")
			time.Sleep(time.Second) // Barely noticeable, just enough to avoid spinning.
			continue
		}

		log.Info(fmt.Sprintf("Joined cluster (%d members)", ps.cluster.NumMembers()))
		break
	}
}

func (ps *peerStore) PutSeeder(ih bittorrent.InfoHash, p bittorrent.Peer) error {
	select {
	case <-ps.closed:
		panic("attempted to interact with stopped memory store")
	default:
	}

	buffer := bytes.Buffer{}
	encoder := gob.NewEncoder(&buffer)

	encoder.Encode(CmdPutSeeder)

	if err := encoder.Encode(&CmdPutSeederData{InfoHash: ih, Peer: p}); err != nil {
		return err
	}

	return ps.cluster.SendReliable(ps.responsibleNode(ih), buffer.Bytes())
}

func (ps *peerStore) DeleteSeeder(ih bittorrent.InfoHash, p bittorrent.Peer) error {
	select {
	case <-ps.closed:
		panic("attempted to interact with stopped memory store")
	default:
	}

	buffer := bytes.Buffer{}
	encoder := gob.NewEncoder(&buffer)

	encoder.Encode(CmdDeleteSeeder)

	if err := encoder.Encode(&CmdDeleteSeederData{InfoHash: ih, Peer: p}); err != nil {
		return err
	}

	return ps.cluster.SendReliable(ps.responsibleNode(ih), buffer.Bytes())
}

func (ps *peerStore) PutLeecher(ih bittorrent.InfoHash, p bittorrent.Peer) error {
	select {
	case <-ps.closed:
		panic("attempted to interact with stopped memory store")
	default:
	}

	buffer := bytes.Buffer{}
	encoder := gob.NewEncoder(&buffer)

	encoder.Encode(CmdPutLeecher)

	if err := encoder.Encode(&CmdPutLeecherData{InfoHash: ih, Peer: p}); err != nil {
		return err
	}

	return ps.cluster.SendReliable(ps.responsibleNode(ih), buffer.Bytes())
}

func (ps *peerStore) DeleteLeecher(ih bittorrent.InfoHash, p bittorrent.Peer) error {
	select {
	case <-ps.closed:
		panic("attempted to interact with stopped memory store")
	default:
	}

	buffer := bytes.Buffer{}
	encoder := gob.NewEncoder(&buffer)

	encoder.Encode(CmdDeleteLeecher)

	if err := encoder.Encode(&CmdDeleteLeecherData{InfoHash: ih, Peer: p}); err != nil {
		return err
	}

	return ps.cluster.SendReliable(ps.responsibleNode(ih), buffer.Bytes())
}

func (ps *peerStore) GraduateLeecher(ih bittorrent.InfoHash, p bittorrent.Peer) error {
	select {
	case <-ps.closed:
		panic("attempted to interact with stopped memory store")
	default:
	}

	buffer := bytes.Buffer{}
	encoder := gob.NewEncoder(&buffer)

	encoder.Encode(CmdGraduateLeecher)

	if err := encoder.Encode(&CmdGraduateLeecherData{InfoHash: ih, Peer: p}); err != nil {
		return err
	}

	return ps.cluster.SendReliable(ps.responsibleNode(ih), buffer.Bytes())
}

func (ps *peerStore) AnnouncePeers(ih bittorrent.InfoHash, seeder bool, numWant int, announcer bittorrent.Peer) (peers []bittorrent.Peer, err error) {
	select {
	case <-ps.closed:
		panic("attempted to interact with stopped memory store")
	default:
	}

	requestID := uuid.New()
	buffer := bytes.Buffer{}
	encoder := gob.NewEncoder(&buffer)
	c := make(chan []bittorrent.Peer)

	if err := encoder.Encode(CmdAnnouncePeersRequest); err != nil {
		return nil, err
	}

	data := CmdAnnouncePeersRequestData{
		RequestID: requestID,
		NodeName:  ps.nodeName,
		InfoHash:  ih,
		Seeder:    seeder,
		NumWant:   numWant,
		Announcer: announcer,
	}

	if err := encoder.Encode(data); err != nil {
		return nil, err
	}

	ps.pending.Store(requestID, c)
	ps.cluster.SendReliable(ps.responsibleNode(ih), buffer.Bytes())

	select {
	case peers := <-c:
		close(c)
		ps.pending.Delete(requestID)

		return peers, nil
	case <-time.After(ps.cfg.RelayTimeout):
		log.Error("Timed out waiting on announce request response channel")

		close(c)
		ps.pending.Delete(requestID)

		return nil, errors.New("no response from responsible node")
	}
}

func (ps *peerStore) ScrapeSwarm(ih bittorrent.InfoHash, addressFamily bittorrent.AddressFamily) (resp bittorrent.Scrape) {
	select {
	case <-ps.closed:
		panic("attempted to interact with stopped memory store")
	default:
	}

	resp.InfoHash = ih

	requestID := uuid.New()
	buffer := bytes.Buffer{}
	encoder := gob.NewEncoder(&buffer)

	c := make(chan bittorrent.Scrape)

	if err := encoder.Encode(CmdScrapeSwarmRequest); err != nil {
		return
	}

	data := CmdScrapeSwarmRequestData{
		RequestID:     requestID,
		NodeName:      ps.nodeName,
		InfoHash:      ih,
		AddressFamily: addressFamily,
	}

	if err := encoder.Encode(data); err != nil {
		return
	}

	ps.pending.Store(requestID, c)
	ps.cluster.SendReliable(ps.responsibleNode(ih), buffer.Bytes())

	select {
	case scrape := <-c:
		close(c)
		ps.pending.Delete(requestID)

		resp = scrape

		return
	case <-time.After(ps.cfg.RelayTimeout):
		log.Error("Timed out waiting on scrape request response channel")

		close(c)
		ps.pending.Delete(requestID)

		return
	}
}

func (ps *peerStore) Stop() stop.Result {
	c := make(stop.Channel)

	go func() {
		// Attempt to gracefully leave the cluster.
		ps.cluster.Leave(10 * time.Second)

		// Stop everything else we were doing in other goroutines.
		close(ps.closed)
		ps.wg.Wait()

		// Explicitly stop the memory store.
		ps.ms.Stop()

		// Forcefully quit the cluster.
		ps.cluster.Shutdown()

		c.Done()
	}()

	return c.Result()
}

func (ps *peerStore) LogFields() log.Fields {
	return ps.cfg.LogFields()
}
