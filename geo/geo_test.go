package geo

import (
	"testing"
)

func TestIsIPInRadius(t *testing.T) {

	radius := 1000

	client := NewGeoClient(Config{
		Host:        "localhost",
		Port:        6379,
		DB:          0,
		Password:    "",
		KeyIPv4:     "ipv4_location",
		KeyIPv6:     "ipv6_location",
		KeyIPv6Info: "ipv6_location_info",
		Radius:      radius,
	})

	ip := "2001:0db8:1111:000a:00b0:0000:9000:0200"

	res := client.IsIPInRadius(ip, "2001:0db8:0000:0000:abcd:0000:0000:1234") // true
	t.Logf("target IP: %s, IP: %s, radius: %d, result %t", ip, "2001:0db8:0000:0000:abcd:0000:0000:1234", radius, res)
	res = client.IsIPInRadius(ip, "2001:0db8:cafe:0001:0000:0000:0000:0100") // false
	t.Logf("target IP: %s, IP: %s, radius: %d, result %t", ip, "2001:0db8:cafe:0001:0000:0000:0000:0100", radius, res)
	res = client.IsIPInRadius(ip, "2001:0db8:cafe:0001:0000:0000:0000:0200") // true
	t.Logf("target IP: %s, IP: %s, radius: %d, result %t", ip, "2001:0db8:cafe:0001:0000:0000:0000:0200", radius, res)
	res = client.IsIPInRadius(ip, "2001:0238:0000:0000:0000:0000:0000:0000") // true
	t.Logf("target IP: %s, IP: %s, radius: %d, result %t", ip, "2001:0238:0000:0000:0000:0000:0000:0000", radius, res)

	radius = 50.0
	ip = "91.78.43.197"

	res = client.IsIPInRadius(ip, "91.78.144.24") // true
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
