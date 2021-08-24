package redis

import (
	"errors"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/go-redsync/redsync"
	"github.com/gomodule/redigo/redis"
)

// redisBackend represents a redis handler.
type redisBackend struct {
	pool    *redis.Pool
	redsync *redsync.Redsync
}

// newRedisBackend creates a redisBackend instance.
func newRedisBackend(cfg *Config, u *redisURL, socketPath string) *redisBackend {
	rc := &redisConnector{
		URL:            u,
		SocketPath:     socketPath,
		ReadTimeout:    cfg.RedisReadTimeout,
		WriteTimeout:   cfg.RedisWriteTimeout,
		ConnectTimeout: cfg.RedisConnectTimeout,
	}
	pool := rc.NewPool()
	redsync := redsync.New([]redsync.Pool{pool})
	return &redisBackend{
		pool:    pool,
		redsync: redsync,
	}
}

// open returns or creates instance of Redis connection.
func (rb *redisBackend) open() redis.Conn {
	return rb.pool.Get()
}

type redisConnector struct {
	URL            *redisURL
	SocketPath     string
	ReadTimeout    time.Duration
	WriteTimeout   time.Duration
	ConnectTimeout time.Duration
}

// NewPool returns a new pool of Redis connections
func (rc *redisConnector) NewPool() *redis.Pool {
	return &redis.Pool{
		MaxIdle:     3,
		IdleTimeout: 240 * time.Second,
		Dial: func() (redis.Conn, error) {
			c, err := rc.open()
			if err != nil {
				return nil, err
			}

			if rc.URL.DB != 0 {
				_, err = c.Do("SELECT", rc.URL.DB)
				if err != nil {
					return nil, err
				}
			}

			return c, err
		},
		// PINGs connections that have been idle more than 10 seconds
		TestOnBorrow: func(c redis.Conn, t time.Time) error {
			if time.Since(t) < time.Duration(10*time.Second) {
				return nil
			}
			_, err := c.Do("PING")
			return err
		},
	}
}

// Open a new Redis connection
func (rc *redisConnector) open() (redis.Conn, error) {
	opts := []redis.DialOption{
		redis.DialDatabase(rc.URL.DB),
		redis.DialReadTimeout(rc.ReadTimeout),
		redis.DialWriteTimeout(rc.WriteTimeout),
		redis.DialConnectTimeout(rc.ConnectTimeout),
	}

	if rc.URL.Password != "" {
		opts = append(opts, redis.DialPassword(rc.URL.Password))
	}

	if rc.SocketPath != "" {
		return redis.Dial("unix", rc.SocketPath, opts...)
	}

	return redis.Dial("tcp", rc.URL.Host, opts...)
}

// A redisURL represents a parsed redisURL
// The general form represented is:
//
//	redis://[password@]host][/][db]
type redisURL struct {
	Host     string
	Password string
	DB       int
}

// parseRedisURL parse rawurl into redisURL
func parseRedisURL(target string) (*redisURL, error) {
	var u *url.URL
	u, err := url.Parse(target)
	if err != nil {
		return nil, err
	}
	if u.Scheme != "redis" {
		return nil, errors.New("no redis scheme found")
	}

	db := 0 // default redis db
	parts := strings.Split(u.Path, "/")
	if len(parts) != 1 {
		db, err = strconv.Atoi(parts[1])
		if err != nil {
			return nil, err
		}
	}
	return &redisURL{
		Host:     u.Host,
		Password: u.User.String(),
		DB:       db,
	}, nil
}
