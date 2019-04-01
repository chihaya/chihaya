package geo

import (
	"testing"
)

func TestIsIPInRadius(t *testing.T) {

	ip := "91.78.43.197"
	radius := 50

	client := NewGeoClient(Config{
		Host:   "localhost",
		Port:   6379,
		DB:     0,
		Key:    "ip2location",
		Radius: radius,
	})

	res := client.IsIPInRadius(ip, "91.78.144.24") // true
	t.Logf("target IP: %s, IP: %s, radius: %d, result %t", ip, "91.78.144.24", radius, res)
	res = client.IsIPInRadius(ip, "91.78.40.24") // false
	t.Logf("target IP: %s, IP: %s, radius: %d, result %t", ip, "91.78.40.24", radius, res)
	res = client.IsIPInRadius(ip, "91.78.80.18") // true
	t.Logf("target IP: %s, IP: %s, radius: %d, result %t", ip, "91.78.80.18", radius, res)
	res = client.IsIPInRadius(ip, "91.78.224.24") // true
	t.Logf("target IP: %s, IP: %s, radius: %d, result %t", ip, "91.78.224.24", radius, res)
	res = client.IsIPInRadius(ip, "91.75.168.21") // false
	t.Logf("target IP: %s, IP: %s, radius: %d, result %t", ip, "91.75.168.21", radius, res)
}
