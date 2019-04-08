package geo

import (
	"encoding/json"
	"fmt"
	"net"
	"strconv"
	"strings"

	"github.com/go-redis/redis"
	"github.com/pborman/uuid"
)

// Config ...
type Config struct {
	Host        string `yaml:"host"`
	Port        int    `yaml:"port"`
	Password    string `yaml:"password"`
	DB          int    `yaml:"db"`
	KeyIPv4     string `yaml:"key_ipv4"`
	KeyIPv6     string `yaml:"ipv6_location"`
	KeyIPv6Info string `yaml:"key_ipv6_info"`
	Radius      int    `yaml:"radius"`
}

// Client ...
type Client struct {
	client      *redis.Client
	keyIPv4     string
	keyIPv6     string
	keyIPv6Info string
	radius      float64
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
		Addr:     config.Host + ":" + strconv.Itoa(config.Port),
		Password: config.Password,
		DB:       config.DB,
	})

	return &Client{
		client:      client,
		keyIPv4:     config.KeyIPv4,
		keyIPv6:     config.KeyIPv6,
		keyIPv6Info: config.KeyIPv6Info,
		radius:      float64(config.Radius)}
}

// SelectIPByRadius ...
func (client *Client) SelectIPByRadius(targetIP string, IPs []string) ([]string, error) {
	return selectIPByRadius(client, targetIP, IPs, client.radius)
}

// IsIPInRadius ...
func (client *Client) IsIPInRadius(targetIP string, IP string) bool {

	res, err := selectIPByRadius(client, targetIP, []string{IP}, client.radius)
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

func iPv6ToString(ip net.IP) string {

	const IPv6len = 16
	var part [8]uint16

	for i := 0; i < IPv6len; i += 2 {

		part[i / 2] = uint16(ip[i])
		part[i / 2] = part[i / 2] << 8
		part[i / 2] = part[i / 2] | uint16(ip[i+1])
	}


	result := fmt.Sprintf("%04X:%04X:%04X:%04X:%04X:%04X:%04X:%04X", 
		part[0], part[1], part[2], part[3], part[4], part[5], part[6], part[7])

	return result
}

func iPv6ToValue(ip string, cidr bool) string {

	ipv6 := ""

	if cidr {

		startIP, _, err := ipRange(ip)
		if err != nil {
			return ""
		}

		ipv6 = iPv6ToString(startIP)

	} else {

		ipv6 = strings.ToUpper(ip)
	}

	return ipv6
}

func iPv4ToScore(ip string, cidr bool) uint64 {

	var score uint64
	score = 0

	if cidr {

		startIP, _, err := ipRange(ip)
		if err != nil {
			return 0
		}

		ip = startIP.String()
	}

	for _, v := range strings.Split(ip, ".") {

		n, _ := strconv.Atoi(v)
		score = score*256 + uint64(n)
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

func getCord(client *redis.Client, keyV4 string, keyV6 string, keyV6Info string, ip string) (float64, float64, error) {

	var longitude, latitude float64
	longitude = 0
	latitude = 0

	var err error
	var vals []string

	var IPv4ID uint64
	var IPv6ID string

	if strings.Index(ip, ".") != -1 {
		IPv4ID = iPv4ToScore(ip, false)
	} else if strings.Index(ip, ":") != -1 {
		IPv6ID = iPv6ToValue(ip, false)
	} else {
		return longitude, latitude, nil
	}

	if len(IPv6ID) > 0 {

		vals, err = client.ZRevRangeByLex(keyV6, redis.ZRangeBy{
			Max:    "[" + IPv6ID,
			Min:    "-",
			Offset: 0,
			Count:  1,
		}).Result()

		if err != nil {
			return longitude, latitude, err
		}

		if len(vals) > 0 {

			val, err := client.HGet(keyV6Info, vals[0]).Result()
			if err != nil {
				return longitude, latitude, err
			}

			vals[0] = val
		}

	} else {

		vals, err = client.ZRevRangeByScore(keyV4, redis.ZRangeBy{
			Min:    "0",
			Max:    strconv.FormatUint(IPv4ID, 10),
			Offset: 0,
			Count:  1,
		}).Result()

		if err != nil {
			return longitude, latitude, err
		}
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

func selectIPByRadius(client *Client, targetIP string, IPs []string, radius float64) ([]string, error) {

	var result []string

	targetLongitude, targetLatitude, err := getCord(client.client, client.keyIPv4, client.keyIPv6, client.keyIPv6Info, targetIP)
	if err != nil {
		return result, err
	}

	redisKey := uuid.New()

	for _, ip := range IPs {

		longitude, latitude, err := getCord(client.client, client.keyIPv4, client.keyIPv6, client.keyIPv6Info, ip)
		if err != nil {
			client.client.Del(redisKey)
			return result, err
		}

		err = addGeo(client.client, redisKey, ip, longitude, latitude)
		if err != nil {
			client.client.Del(redisKey)
			return result, err
		}
	}

	defer client.client.Del(redisKey)

	geolocs, err := client.client.GeoRadius(redisKey, targetLongitude, targetLatitude, &redis.GeoRadiusQuery{
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
