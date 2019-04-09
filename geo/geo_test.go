package geo

import (
	"testing"
)

func TestIsIPInRadius(t *testing.T) {

	radius := 30

	client := NewGeoClient(Config{
		Address:  "localhost:6379",
		Password: "",
		Database: "D:\\Projects\\Syncopate\\sources\\ProtocolONE\\chihaya\\geo\\data\\GeoLite2-City.mmdb",
		Radius:   radius,
	})

	ip := "91.78.43.197"

	res := client.IsIPInRadius(ip, "91.78.144.24") // true
	t.Logf("target IP: %s, IP: %s, radius: %d, result %t", ip, "91.78.144.24", radius, res)
	res = client.IsIPInRadius(ip, "92.127.155.231") // false
	t.Logf("target IP: %s, IP: %s, radius: %d, result %t", ip, "92.127.155.231", radius, res)
	res = client.IsIPInRadius(ip, "91.78.80.18") // true
	t.Logf("target IP: %s, IP: %s, radius: %d, result %t", ip, "91.78.80.18", radius, res)
	res = client.IsIPInRadius(ip, "91.78.224.24") // true
	t.Logf("target IP: %s, IP: %s, radius: %d, result %t", ip, "91.78.224.24", radius, res)
	res = client.IsIPInRadius(ip, "91.75.168.21") // false
	t.Logf("target IP: %s, IP: %s, radius: %d, result %t", ip, "91.75.168.21", radius, res)
}
