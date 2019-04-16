package geo

import (
	"math"
	"net"

	"github.com/oschwald/geoip2-golang"
	"go.uber.org/zap"
)

type Config struct {
	Database string `yaml:"database"`
	Radius   int    `yaml:"radius"`
}

type Client struct {
	database *geoip2.Reader
	radius   float64
}

var DefaultGeoClient *Client

func Init(config Config) {

	DefaultGeoClient = NewGeoClient(config)
}

func NewGeoClient(config Config) *Client {

	db, err := geoip2.Open(config.Database)
	if err != nil {
		zap.S().Warnw("Cannot open geoip2 data file", zap.String("error", err.Error()))
	}

	return &Client{
		database: db,
		radius:   float64(config.Radius)}
}

func CalcDistance(lat1 float64, lng1 float64, lat2 float64, lng2 float64, unit string) float64 {

	const PI float64 = 3.141592653589793

	radlat1 := float64(PI * lat1 / 180)
	radlat2 := float64(PI * lat2 / 180)

	theta := float64(lng1 - lng2)
	radtheta := float64(PI * theta / 180)

	dist := math.Sin(radlat1)*math.Sin(radlat2) + math.Cos(radlat1)*math.Cos(radlat2)*math.Cos(radtheta)

	if dist > 1 {
		dist = 1
	}

	dist = math.Acos(dist)
	dist = dist * 180 / PI
	dist = dist * 60 * 1.1515

	if len(unit) > 0 {
		if unit == "K" {
			dist = dist * 1.609344
		} else if unit == "N" {
			dist = dist * 0.8684
		}
	}

	return dist
}

func (client *Client) IsInRadius(lat1 float64, lng1 float64, lat2 float64, lng2 float64) bool {

	dist := CalcDistance(lat1, lng1, lat2, lng2, "K")
	return client.radius >= dist
}

func (client *Client) GetLocation(IP net.IP) (float64, float64) {

	record, err := client.database.City(IP)
	if err != nil {
		return 0, 0
	}

	return record.Location.Latitude, record.Location.Longitude
}
