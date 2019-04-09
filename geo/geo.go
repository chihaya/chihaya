package geo

import (
	"net"

	"github.com/go-redis/redis"
	"github.com/pborman/uuid"
	"go.uber.org/zap"

	"github.com/oschwald/geoip2-golang"
)

// Config ...
type Config struct {
	Address  string `yaml:"address"`
	Password string `yaml:"password"`
	Database string `yaml:"database"`
	Radius   int    `yaml:"radius"`
}

// Client ...
type Client struct {
	client   *redis.Client
	database *geoip2.Reader
	radius   float64
}

// DefaultGeoClient ...
var DefaultGeoClient *Client

// Init ...
func Init(config Config) {

	DefaultGeoClient = NewGeoClient(config)
}

// NewGeoClient ...
func NewGeoClient(config Config) *Client {

	client := redis.NewClient(&redis.Options{
		Addr:     config.Address,
		Password: config.Password,
		DB:       0,
	})

	db, err := geoip2.Open(config.Database)
	if err != nil {
		zap.S().Warnw("Cannot open geoip2 data file", zap.String("error", err.Error()))
	}

	return &Client{
		client:   client,
		database: db,
		radius:   float64(config.Radius)}
}

// SelectIPByRadius ...
func (client *Client) SelectIPByRadius(targetIP string, IPs []string) ([]string, error) {
	return selectIPByRadius(client, targetIP, IPs)
}

// IsIPInRadius ...
func (client *Client) IsIPInRadius(targetIP string, IP string) bool {

	res, err := selectIPByRadius(client, targetIP, []string{IP})
	if err != nil {
		return false
	}

	return len(res) > 0
}

func addGeo(client *redis.Client, key string, ip string, longitude, latitude float64) error {

	_, err := client.GeoAdd(key, &redis.GeoLocation{
		Name:      ip,
		Longitude: longitude,
		Latitude:  latitude,
	}).Result()

	return err
}

func selectIPByRadius(client *Client, targetIP string, IPs []string) ([]string, error) {

	var result []string

	nip := net.ParseIP(targetIP)
	record, err := client.database.City(nip)
	if err != nil {
		return result, err
	}

	targetLongitude := record.Location.Longitude
	targetLatitude := record.Location.Latitude

	redisKey := uuid.New()

	for _, ip := range IPs {

		nip = net.ParseIP(ip)
		record, err = client.database.City(nip)
		if err != nil {
			return result, err
		}

		longitude := record.Location.Longitude
		latitude := record.Location.Latitude

		err = addGeo(client.client, redisKey, ip, longitude, latitude)
		if err != nil {
			client.client.Del(redisKey)
			return result, err
		}
	}
	defer client.client.Del(redisKey)

	geolocs, err := client.client.GeoRadius(redisKey, targetLongitude, targetLatitude, &redis.GeoRadiusQuery{
		Radius:   client.radius,
		WithDist: true,
	}).Result()

	if err != nil {
		return result, err
	}

	for _, gl := range geolocs {

		result = append(result, gl.Name)
	}

	return result, nil
}
