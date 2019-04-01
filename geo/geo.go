package geo

import (
	"encoding/json"
	"fmt"
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

func iPv4ToUint32(iPv4 string) uint32 {

	ipOctets := [4]uint64{}

	for i, v := range strings.SplitN(iPv4, ".", 4) {
		ipOctets[i], _ = strconv.ParseUint(v, 10, 32)
	}

	result := (ipOctets[0] << 24) | (ipOctets[1] << 16) | (ipOctets[2] << 8) | ipOctets[3]

	return uint32(result)
}

func uInt32ToIPv4(iPuInt32 uint32) (iP string) {
	iP = fmt.Sprintf("%d.%d.%d.%d",
		iPuInt32>>24,
		(iPuInt32&0x00FFFFFF)>>16,
		(iPuInt32&0x0000FFFF)>>8,
		iPuInt32&0x000000FF)
	return iP
}

func cidrRangeToIPv4Range(CIDRs []string) (ipStart string, ipEnd string, err error) {

	var ip uint32  // ip address
	var ipS uint32 // Start IP address range
	var ipE uint32 // End IP address range

	for _, CIDR := range CIDRs {
		cidrParts := strings.Split(CIDR, "/")

		ip = iPv4ToUint32(cidrParts[0])
		bits, _ := strconv.ParseUint(cidrParts[1], 10, 32)

		if ipS == 0 || ipS > ip {
			ipS = ip
		}

		ip = ip | (0xFFFFFFFF >> bits)

		if ipE < ip {
			ipE = ip
		}
	}

	ipStart = uInt32ToIPv4(ipS)
	ipEnd = uInt32ToIPv4(ipE)

	return ipStart, ipEnd, err
}

func isDigit(str string) bool {

	_, err := strconv.ParseInt(str, 10, 64)
	return err == nil
}

func ipToScore(ip string, cidr bool) int {

	score := 0

	if cidr {

		var err error
		ip, _, err = cidrRangeToIPv4Range([]string{ip})
		if err != nil {
			return 0
		}
	}

	for _, v := range strings.Split(ip, ".") {

		n, _ := strconv.Atoi(v)
		score = score*256 + n
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
		Max:    strconv.Itoa(IpId),
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
