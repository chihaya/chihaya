package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"sync"
	"time"
)

/*
READ_DB_FREQUENCY = 30
WRITE_MARSHALL_FREQUENCY = 60
WRITE_DB_FREQUENCY = 5
CLEAN_FREQUENCY = 120
MIN_INTERVAL = 900
ANNOUNCE_INTERVAL = 2700
*/

// Intervals
var (
	AnnounceInterval    = 30 * time.Minute
	MinAnnounceInterval = 15 * time.Minute

	// IMO it's best to offset these to distribute load
	DatabaseReloadInterval        = 45 * time.Second
	DatabaseSerializationInterval = 68 * time.Second
	PurgeInactiveInterval         = 83 * time.Second

	VerifyUsedSlotsInterval = 60 * time.Minute
)

// Time to sleep between flushes if the buffer is less than half full
var FlushSleepInterval = 200 * time.Millisecond

// Buffer sizes, see @Database.startFlushing()
var (
	TorrentFlushBufferSize         = 10000
	UserFlushBufferSize            = 10000
	TransferHistoryFlushBufferSize = 10000
	TransferIpsFlushBufferSize     = 10000
	SnatchFlushBufferSize          = 1000
)

const LogFlushes = true
const SlotsEnabled = true

// Config file stuff
var once sync.Once
type configMap map[string]interface{}

var config configMap

func Get(s string) string {
	once.Do(readConfig)
	return config.Get(s)
}

func Section(s string) configMap {
	once.Do(readConfig)
	return config.Section(s)
}

func GetDsn(s string) (string, error) {
	once.Do(readConfig)
	return config.GetDsn(s)
}

func (m configMap) Get(s string) string {
	result, _ := m[s].(string)
	return result
}

func (m configMap) Section(s string) configMap {
	result, _ := m[s].(map[string]interface{})
	return configMap(result)
}

func readConfig() {
	configFile := "config.json"
	f, err := os.Open(configFile)

	if err != nil {
		log.Fatalf("Error opening config file: %s", err)
		return
	}

	err = json.NewDecoder(f).Decode(&config)

	if err != nil {
		log.Fatalf("Error parsing config file: %s", err)
		return
	}
}

func (m configMap) GetDsn(s string) (string, error) {
	c := m.Section(s)

	if len(c) == 0 {
		return "", errors.New("could not find key " + s)
	}

	var dsn string
	socket, unix := c["unix"]

	if unix {
		dsn = fmt.Sprintf("%s:%s@unix(%s)/%s?charset=%s",
			c["username"],
			c["password"],
			socket,
			c["database"],
			c["encoding"],
		)
	} else {
		dsn = fmt.Sprintf("%s:%s@tcp(%s:%v)/%s?charset=%s",
			c["username"],
			c["password"],
			c["host"],
			c["port"],
			c["database"],
			c["encoding"],
		)
	}

	return dsn, nil
}
