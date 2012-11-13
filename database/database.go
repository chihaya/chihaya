package database

import (
	"bytes"
	"chihaya/config"
	"chihaya/util"
	"github.com/ziutek/mymysql/mysql"
	_ "github.com/ziutek/mymysql/native"
	"log"
	"sync"
	"time"
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

	SlotsLastChecked int64
}

type DatabaseConnection struct {
	sqlDb mysql.Conn
	mutex sync.Mutex
}

type Database struct {
	terminate bool

	mainConn *DatabaseConnection // Used for reloading and misc queries

	loadUsersStmt       mysql.Stmt
	loadTorrentsStmt    mysql.Stmt
	loadWhitelistStmt   mysql.Stmt
	loadFreeleechStmt   mysql.Stmt
	cleanStalePeersStmt mysql.Stmt

	Users      map[string]*User // 32 bytes
	UsersMutex sync.RWMutex

	Torrents      map[string]*Torrent // SHA-1 hash (20 bytes)
	TorrentsMutex sync.RWMutex

	Whitelist      []string
	WhitelistMutex sync.RWMutex

	torrentChannel          chan *bytes.Buffer
	userChannel             chan *bytes.Buffer
	transferHistoryChannel  chan *bytes.Buffer
	transferIpsChannel      chan *bytes.Buffer
	snatchChannel           chan *bytes.Buffer
	slotVerificationChannel chan *User

	waitGroup sync.WaitGroup

	bufferPool *util.BufferPool
}

func (db *Database) Init() {
	db.terminate = false

	db.mainConn = OpenDatabaseConnection()

	maxBuffers := config.TorrentFlushBufferSize + config.UserFlushBufferSize + config.TransferHistoryFlushBufferSize +
		config.TransferIpsFlushBufferSize + config.SnatchFlushBufferSize

	// Used for recording updates, so the max required size should be < 128 bytes. See record.go for details
	db.bufferPool = util.NewBufferPool(maxBuffers, 128)

	db.loadUsersStmt = db.mainConn.prepareStatement("SELECT ID, torrent_pass, DownMultiplier, UpMultiplier, Slots FROM users_main")
	db.loadTorrentsStmt = db.mainConn.prepareStatement("SELECT ID, info_hash, DownMultiplier, UpMultiplier, Snatched FROM torrents")
	db.loadWhitelistStmt = db.mainConn.prepareStatement("SELECT peer_id FROM xbt_client_whitelist")
	db.loadFreeleechStmt = db.mainConn.prepareStatement("SELECT mod_setting FROM mod_core WHERE mod_option='global_freeleech'")
	db.cleanStalePeersStmt = db.mainConn.prepareStatement("UPDATE transfer_history SET active = '0' WHERE last_announce < ?")

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
	close(db.slotVerificationChannel)

	go func() {
		time.Sleep(10 * time.Second)
		log.Printf("Waiting for database flushing to finish. This can take a few minutes, please be patient!")
	}()

	db.waitGroup.Wait()
	db.mainConn.mutex.Lock()
	db.mainConn.Close()
	db.mainConn.mutex.Unlock()
	db.serialize()
}

func OpenDatabaseConnection() (db *DatabaseConnection) {
	db = &DatabaseConnection{}
	databaseConfig := config.Section("database")

	db.sqlDb = mysql.New(databaseConfig["proto"].(string),
		"",
		databaseConfig["addr"].(string),
		databaseConfig["username"].(string),
		databaseConfig["password"].(string),
		databaseConfig["database"].(string),
	)

	err := db.sqlDb.Connect()
	if err != nil {
		log.Fatalf("Couldn't connect to database at %s:%s - %s", databaseConfig["proto"], databaseConfig["addr"], err)
	}
	return
}

func (db *DatabaseConnection) Close() error {
	return db.sqlDb.Close()
}

func (db *DatabaseConnection) prepareStatement(sql string) mysql.Stmt {
	stmt, err := db.sqlDb.Prepare(sql)
	if err != nil {
		log.Fatalf("%s for SQL: %s", err, sql)
	}
	return stmt
}

/*
 * mymysql uses different semantics than the database/sql interface
 * For some reason (for prepared statements), mymysql's Exec is the equivalent of Query, and Run is the equivalent of Exec.
 * For the connection object, Query is still Query, but Start is Exec
 *
 * This is really confusing, which is why these wrapper functions are named as such
 */

func (db *DatabaseConnection) query(stmt mysql.Stmt, args ...interface{}) mysql.Result {
	return db.exec(stmt, args...)
}

func (db *DatabaseConnection) exec(stmt mysql.Stmt, args ...interface{}) (result mysql.Result) {
	var err error
	var tries int

	for tries = 0; tries < config.MaxDeadlockRetries; tries++ {
		result, err = stmt.Run(args...)
		if err != nil {
			if merr, isMysqlError := err.(*mysql.Error); isMysqlError {
				if merr.Code == 1213 || merr.Code == 1205 {
					log.Printf("!!! DEADLOCK !!! Retrying in %dms (%d/20)", config.DeadlockWaitTime.Nanoseconds()/1000000, tries)
					time.Sleep(config.DeadlockWaitTime)
					continue
				} else {
					log.Printf("!!! CRITICAL !!! SQL error: %v", err)
				}
			} else {
				log.Panicf("Error executing SQL: %v", err)
			}
		}
		return
	}
	log.Printf("!!! CRITICAL !!! Deadlocked %d times, giving up!", tries)
	return
}

func (db *DatabaseConnection) execBuffer(query *bytes.Buffer) (result mysql.Result) {
	var err error
	var tries int

	for tries = 0; tries < config.MaxDeadlockRetries; tries++ {
		result, err = db.sqlDb.Start(query.String())
		if err != nil {
			if merr, isMysqlError := err.(*mysql.Error); isMysqlError {
				if merr.Code == 1213 || merr.Code == 1205 {
					log.Printf("!!! DEADLOCK !!! Retrying in %dms (%d/20)", config.DeadlockWaitTime.Nanoseconds()/1000000, tries)
					time.Sleep(config.DeadlockWaitTime)
					continue
				} else {
					log.Printf("!!! CRITICAL !!! SQL error: %v", err)
				}
			} else {
				log.Panicf("Error executing SQL: %v", err)
			}
		}
		return
	}
	log.Printf("!!! CRITICAL !!! Deadlocked %d times, giving up!", tries)
	return
}
