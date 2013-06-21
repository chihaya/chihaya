package storage

import (
	"fmt"

	"github.com/jzelinskie/chihaya/config"
)

var drivers = make(map[string]StorageDriver)

type StorageDriver interface {
	New(*config.StorageConfig) (Storage, error)
}

func Register(name string, driver StorageDriver) {
	if driver == nil {
		panic("storage: Register driver is nil")
	}
	if _, dup := drivers[name]; dup {
		panic("storage: Register called twice for driver " + name)
	}
	drivers[name] = driver
}

func New(name string, conf *config.Storage) (Storage, error) {
	driver, ok := drivers[name]
	if !ok {
		return nil, fmt.Errorf(
			"storage: unknown driver %q (forgotten import?)",
			name,
		)
	}
	store, err := driver.New(conf)
	if err != nil {
		return nil, err
	}
	return store, nil
}

type Storage interface {
	Shutdown() error

	FindUser(passkey []byte) (*User, bool, error)
	FindTorrent(infohash []byte) (*Torrent, bool, error)
	UnpruneTorrent(torrent *Torrent) error

	RecordUser(
		user *User,
		rawDeltaUpload int64,
		rawDeltaDownload int64,
		deltaUpload int64,
		deltaDownload int64,
	) error
	RecordSnatch(peer *Peer, now int64) error
	RecordTorrent(torrent *Torrent, deltaSnatch uint64) error
	RecordTransferIP(peer *Peer) error
	RecordTransferHistory(
		peer *Peer,
		rawDeltaUpload int64,
		rawDeltaDownload int64,
		deltaTime int64,
		deltaSnatch uint64,
		active bool,
	) error
}
