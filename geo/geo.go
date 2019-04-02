package geo

import (
	"encoding/json"
	"net"
	"strconv"
	"strings"

	"github.com/go-redis/redis"
	"github.com/pborman/uuid"
)

type Config struct {
	Host   string `yaml:"host"`
	Port   int    `yaml:"port"`
	DB     int    `yaml:"db"`
	Key    string `yaml:"key"`
	Radius int    `yaml:"radius"`
}

// GeoClient ...
type GeoClient struct {
	client *redis.Client
	geoKey string
	radius float64
}

// DefaultGeoClient ...
var DefaultGeoClient *GeoClient

// Init ...
func Init(config Config) {

	DefaultGeoClient = NewGeoClient(config)
}

// NewGeoClient ...
func NewGeoClient(config Config) *GeoClient {

	client := redis.NewClient(&redis.Options{
		Addr:     config.Host + ":" + strconv.Itoa(config.Port),
		Password: "",
		DB:       config.DB,
	})

	return &GeoClient{client: client, geoKey: config.Key, radius: float64(config.Radius)}
}

// SelectIPByRadius ...
func (client *GeoClient) SelectIPByRadius(targetIP string, IPs []string) ([]string, error) {
	return selectIPByRadius(client.client, targetIP, IPs, client.radius)
}

// IsIPInRadius ...
func (client *GeoClient) IsIPInRadius(targetIP string, IP string) bool {

	res, err := selectIPByRadius(client.client, targetIP, []string{IP}, client.radius)
	if err != nil {
		return false
	}

	return len(res) > 0
}

func isDigit(str string) bool {

	_, err := strconv.ParseInt(str, 10, 64)
	return err == nil
}

func ipRange(str string) (net.IP, net.IP, error) {

	_, mask, err := net.ParseCIDR(str)
	if err != nil {
		return nil, nil, err
	}

	first := mask.IP.Mask(mask.Mask).To16()
	second := make(net.IP, len(first))
	copy(second, first)
	ones, _ := mask.Mask.Size()

	if first.To4() != nil {
		ones += 96
	}

	lastBytes := (8*16 - ones) / 8
	lastBits := 8 - ones%8
	or := 0

	for x := 0; x < lastBits; x++ {
		or = or*2 + 1
	}

	for x := 16 - lastBytes; x < 16; x++ {
		second[x] = 0xff
	}

	if lastBits < 8 {
		second[16-lastBytes-1] |= byte(or)
	}

	return first, second, nil
}

func ipToScore(ip string, cidr bool) uint64 {

	var score uint64
	score = 0

	if cidr {

		startIP, _, err := ipRange(ip)
		if err != nil {
			return 0
		}

		ip = startIP.String()
	}

	if strings.Index(ip, ".") != -1 {

		// IPv4
		for _, v := range strings.Split(ip, ".") {

			n, _ := strconv.Atoi(v)
			score = score*256 + uint64(n)
		}

	} else if strings.Index(ip, ":") != -1 {

		//IPv6 is not supported
	}

	return score
}

func addGeo(client *redis.Client, key string, ip string, longitude, latitude float64) error {

	_, err := client.GeoAdd(key, &redis.GeoLocation{
		Name:      ip,
		Longitude: longitude,
		Latitude:  latitude,
	}).Result()

	return err
}

func getCord(client *redis.Client, key string, ip string) (float64, float64, error) {

	var longitude, latitude float64
	longitude = 0
	latitude = 0

	IpId := ipToScore(ip, false)

	vals, err := client.ZRevRangeByScore(key, redis.ZRangeBy{
		Min:    "0",
		Max:    strconv.FormatUint(IpId, 10),
		Offset: 0,
		Count:  1,
	}).Result()

	if err != nil {
		return longitude, latitude, err
	}

	var cord []string

	if len(vals) != 0 {

		err = json.Unmarshal([]byte(vals[0]), &cord)
		if err != nil {
			return longitude, latitude, err
		}

		longitude, err = strconv.ParseFloat(cord[1], 64)
		if err != nil || longitude == 0 {
			return longitude, latitude, err
		}

		latitude, err = strconv.ParseFloat(cord[2], 64)
		if err != nil || latitude == 0 {
			return longitude, latitude, err
		}
	}

	return longitude, latitude, nil
}

func selectIPByRadius(client *redis.Client, targetIP string, IPs []string, radius float64) ([]string, error) {

	var result []string

	targetLongitude, targetLatitude, err := getCord(client, "ip2location", targetIP)
	if err != nil {
		return result, err
	}

	redisKey := uuid.New()

	for _, ip := range IPs {

		longitude, latitude, err := getCord(client, "ip2location", ip)
		if err != nil {
			client.Del(redisKey)
			return result, err
		}

		err = addGeo(client, redisKey, ip, longitude, latitude)
		if err != nil {
			client.Del(redisKey)
			return result, err
		}
	}

	defer client.Del(redisKey)

	geolocs, err := client.GeoRadius(redisKey, targetLongitude, targetLatitude, &redis.GeoRadiusQuery{
		Radius:   radius,
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
