package database

import (
	"bytes"
	"chihaya/config"
	"chihaya/util"
	"github.com/ziutek/mymysql/mysql"
	_ "github.com/ziutek/mymysql/thrsafe"
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

	sqlDb mysql.Conn

	loadUsersStmt       mysql.Stmt
	loadTorrentsStmt    mysql.Stmt
	loadWhitelistStmt   mysql.Stmt
	cleanStalePeersStmt mysql.Stmt

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

	databaseConfig := config.Section("database")

	db.sqlDb = mysql.New(databaseConfig["proto"].(string),
		"",
		databaseConfig["addr"].(string),
		databaseConfig["username"].(string),
		databaseConfig["password"].(string),
		databaseConfig["database"].(string),
	)

	err = db.sqlDb.Connect()
	if err != nil {
		log.Fatalf("Couldn't connect to database at %s:%s - %s", databaseConfig["proto"], databaseConfig["addr"], err)
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

func (db *Database) prepareStatement(sql string) mysql.Stmt {
	stmt, err := db.sqlDb.Prepare(sql)
	if err != nil {
		log.Fatalf("%s for SQL: %s", err, sql)
	}
	return stmt
}

/*
 * mymysql uses different semantics than the database/sql interface
 * For some reason (for prepared statements), mymysql's Exec is the equivalent of Query, and Run is the equivalent of Exec.
 * However, you still use Query for string queries on the database object..
 */

func (db *Database) query(stmt mysql.Stmt, args ...interface{}) mysql.Result {
	result, err := stmt.Run(args...)
	if err != nil {
		log.Panicf("SQL error: %v", err)
	}
	return result
}

func (db *Database) exec(stmt mysql.Stmt, args ...interface{}) mysql.Result {
	result, err := stmt.Run(args...)
	if err != nil {
		log.Panicf("SQL error: %v", err)
	}
	return result
}

func (db *Database) execBuffer(query *bytes.Buffer) mysql.Result {
	_, result, err := db.sqlDb.Query(query.String())
	if err != nil {
		log.Panicf("SQL error: %v", err)
	}
	return result
}
