package database

import (
	"github.com/ProtocolONE/chihaya/frontend/cord/config"
	"github.com/ProtocolONE/chihaya/frontend/cord/models"

	"go.uber.org/zap"
	mgo "gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
	"sync"
)

// DbConf ...
type DbConf struct {
	Dbs      *mgo.Session
	Database string
}

var dbConf *DbConf

// Init ...
func Init() error {

	cfg := config.Get().Database

	dbConf = &DbConf{
		Database: cfg.Database,
	}

	session, err := mgo.DialWithInfo(&mgo.DialInfo{
		Addrs:    []string{cfg.Host},
		Database: cfg.Database,
		Username: cfg.User,
		Password: cfg.Password,
	})
	if err != nil {
		session, err := mgo.Dial(cfg.Host)
		if err != nil {
			zap.S().Fatal(err)
			return err
		}

		db := session.DB(cfg.Database)
		err = db.Login(cfg.User, cfg.Password)
		if err != nil {
			zap.S().Fatal(err)
			return err
		}
	}

	dbConf.Dbs = session
	zap.S().Infof("Connected to DB: \"%s\" [u:\"%s\":p\"%s\"]", dbConf.Database, cfg.User, cfg.Password)

	return nil
}

// UserManager ...
type UserManager struct {
	collection *mgo.Collection
}

// NewUserManager ...
func NewUserManager() *UserManager {
	session := dbConf.Dbs.Copy()
	return &UserManager{collection: session.DB(dbConf.Database).C("users")}
}

// FindByName ...
func (manager *UserManager) FindByName(name string) ([]*models.User, error) {

	var dbUsers []*models.User
	err := manager.collection.Find(bson.M{"username": name}).All(&dbUsers)
	if err != nil {
		return nil, err
	}

	return dbUsers, nil
}

// RemoveByName ...
func (manager *UserManager) RemoveByName(name string) error {

	err := manager.collection.Remove(bson.M{"username": name})
	if err != nil {
		return err
	}

	return nil
}

// Insert ...
func (manager *UserManager) Insert(user *models.User) error {

	err := manager.collection.Insert(user)
	if err != nil {
		return err
	}

	return nil
}

// TorrentManager ...
type TorrentManager struct {
	collection *mgo.Collection
}

// NewTorrentManager ...
func NewTorrentManager() *TorrentManager {
	session := dbConf.Dbs.Copy()
	return &TorrentManager{collection: session.DB(dbConf.Database).C("torrents")}
}

// Insert ...
func (manager *TorrentManager) Insert(torrent *models.Torrent) error {

	err := manager.collection.Insert(torrent)
	if err != nil {
		return err
	}

	return nil
}

// RemoveByInfoHash ...
func (manager *TorrentManager) RemoveByInfoHash(infoHash string) error {

	err := manager.collection.Remove(bson.M{"info_hash": infoHash})
	if err != nil {
		return err
	}

	return nil
}

// FindByInfoHash ...
func (manager *TorrentManager) FindByInfoHash(infoHash string) ([]*models.Torrent, error) {

	var dbTorrent []*models.Torrent
	err := manager.collection.Find(bson.M{"info_hash": infoHash}).All(&dbTorrent)
	if err != nil {
		return nil, err
	}

	return dbTorrent, nil
}

// FindAll ...
func (manager *TorrentManager) FindAll() ([]*models.Torrent, error) {

	var dbTorrent []*models.Torrent
	err := manager.collection.Find(nil).All(&dbTorrent)
	if err != nil {
		return nil, err
	}

	return dbTorrent, nil
}

// MemTorrentManager ...
type MemTorrentManager struct {
	collection sync.Map
}

var memTorrentManager = &MemTorrentManager{}

// NewMemTorrentManager ...
func NewMemTorrentManager() *MemTorrentManager {
	return memTorrentManager
}

// Insert ...
func (manager *MemTorrentManager) Insert(torrent *models.Torrent) {

	manager.collection.Store(torrent.InfoHash, torrent)
}

// RemoveByInfoHash ...
func (manager *MemTorrentManager) RemoveByInfoHash(infoHash string) {

	manager.collection.Delete(infoHash)
}

// FindByInfoHash ...
func (manager *MemTorrentManager) FindByInfoHash(infoHash string) *models.Torrent {

	v, ok := manager.collection.Load(infoHash)
	if !ok {
		return nil
	}

	return v.(*models.Torrent)
}
