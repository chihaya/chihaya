package database

import (
	"bytes"
	"chihaya/config"
	"chihaya/util"
	_ "code.google.com/p/go-mysql-driver/mysql"
	"database/sql"
	"log"
	"sync"
)

type Peer struct {
	Id        string
	UserId    uint64
	TorrentId uint64

	Port uint
	Ip   string
	Addr []byte

	Uploaded   uint64
	Downloaded uint64
	Left       uint64
	Seeding    bool

	StartTime    int64 // unix time
	LastAnnounce int64
}

type Torrent struct {
	Id             uint64
	UpMultiplier   float64
	DownMultiplier float64

	Seeders  map[string]*Peer
	Leechers map[string]*Peer

	Snatched uint
}

type User struct {
	Id             uint64
	UpMultiplier   float64
	DownMultiplier float64
	Slots          int64
	UsedSlots      int64
}

type Database struct {
	terminate bool

	sqlDb *sql.DB

	loadUsersStmt       *sql.Stmt
	loadTorrentsStmt    *sql.Stmt
	loadWhitelistStmt   *sql.Stmt
	cleanStalePeersStmt *sql.Stmt

	Users      map[string]*User // 32 bytes
	UsersMutex sync.RWMutex

	Torrents      map[string]*Torrent // SHA-1 hash (20 bytes)
	TorrentsMutex sync.RWMutex

	Whitelist      []string
	WhitelistMutex sync.RWMutex

	torrentChannel         chan *bytes.Buffer
	userChannel            chan *bytes.Buffer
	transferHistoryChannel chan *bytes.Buffer
	transferIpsChannel     chan *bytes.Buffer
	snatchChannel          chan *bytes.Buffer

	waitGroup sync.WaitGroup

	bufferPool *util.BufferPool
}

func (db *Database) Init() {
	var err error
	db.terminate = false
	dsn, _ := config.GetDsn("database")
	db.sqlDb, err = sql.Open("mysql", dsn)
	if err != nil {
		log.Fatalf("Couldn't connect to database (%s): %s", dsn, err)
	}

	maxBuffers := config.TorrentFlushBufferSize + config.UserFlushBufferSize + config.TransferHistoryFlushBufferSize +
		config.TransferIpsFlushBufferSize + config.SnatchFlushBufferSize

	// Used for recording updates, so the max required size should be < 128 bytes. See record.go for details
	db.bufferPool = util.NewBufferPool(maxBuffers, 128)

	db.loadUsersStmt = db.prepareStatement("SELECT ID, torrent_pass, DownMultiplier, UpMultiplier, Slots FROM users_main")
	db.loadTorrentsStmt = db.prepareStatement("SELECT ID, info_hash, DownMultiplier, UpMultiplier, Snatched FROM torrents")
	db.loadWhitelistStmt = db.prepareStatement("SELECT peer_id FROM xbt_client_whitelist")
	db.cleanStalePeersStmt = db.prepareStatement("UPDATE transfer_history SET active = '0' WHERE last_announce < ?")

	db.Users = make(map[string]*User)
	db.Torrents = make(map[string]*Torrent)
	db.Whitelist = make([]string, 0, 100)

	db.deserialize()

	db.startReloading()
	db.startSerializing()
	db.startFlushing()
}

func (db *Database) Terminate() {
	db.terminate = true

	close(db.torrentChannel)
	close(db.userChannel)
	close(db.transferHistoryChannel)
	close(db.transferIpsChannel)
	close(db.snatchChannel)

	db.waitGroup.Wait()

	db.serialize()
}

func (db *Database) prepareStatement(sql string) *sql.Stmt {
	stmt, err := db.sqlDb.Prepare(sql)
	if err != nil {
		log.Fatalf("%s for SQL: %s", err, sql)
	}
	return stmt
}

func (db *Database) query(stmt *sql.Stmt, args ...interface{}) *sql.Rows {
	rows, err := stmt.Query(args...)
	if err != nil {
		log.Panicf("SQL error: %v", err)
	}
	return rows
}

func (db *Database) exec(stmt *sql.Stmt, args ...interface{}) sql.Result {
	result, err := stmt.Exec(args...)
	if err != nil {
		log.Panicf("SQL error: %v", err)
	}
	return result
}

func (db *Database) execBuffer(query *bytes.Buffer) sql.Result {
	result, err := db.sqlDb.Exec(query.String())
	if err != nil {
		log.Panicf("SQL error: %v", err)
	}
	return result
}

func (db *Database) execString(query string) sql.Result {
	//log.Println(query)
	result, err := db.sqlDb.Exec(query)
	if err != nil {
		log.Panicf("SQL error: %v", err)
	}
	return result
}
