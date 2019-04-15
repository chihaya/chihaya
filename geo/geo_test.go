package geo

import (
	"testing"
	"net"
)

func TestIsIPInRadius(t *testing.T) {

	radius := 1000

	client := NewGeoClient(Config{
		Database: "D:\\Projects\\Syncopate\\sources\\ProtocolONE\\chihaya\\geo\\data\\GeoLite2-City.mmdb",
		Radius:   radius,
	})

	ip := "91.78.43.197"
	latitude1, longitude1, _ := client.GetLocation(net.ParseIP(ip))

	latitude2, longitude2, _ := client.GetLocation(net.ParseIP("91.78.144.24"))
	res := client.IsInRadius(latitude1, longitude1, latitude2, longitude2)
	t.Logf("target IP: %s, IP: %s, radius: %d, result %t", ip, "91.78.144.24", radius, res)

	latitude2, longitude2, _ = client.GetLocation(net.ParseIP("92.127.155.231"))
	res = client.IsInRadius(latitude1, longitude1, latitude2, longitude2)
	t.Logf("target IP: %s, IP: %s, radius: %d, result %t", ip, "92.127.155.231", radius, res)

	latitude2, longitude2, _ = client.GetLocation(net.ParseIP("91.78.80.18"))
	res = client.IsInRadius(latitude1, longitude1, latitude2, longitude2)
	t.Logf("target IP: %s, IP: %s, radius: %d, result %t", ip, "91.78.80.18", radius, res)

	latitude2, longitude2, _ = client.GetLocation(net.ParseIP("91.78.224.24"))
	res = client.IsInRadius(latitude1, longitude1, latitude2, longitude2)
	t.Logf("target IP: %s, IP: %s, radius: %d, result %t", ip, "91.78.224.24", radius, res)

	latitude2, longitude2, _ = client.GetLocation(net.ParseIP("91.75.168.21"))
	res = client.IsInRadius(latitude1, longitude1, latitude2, longitude2)
	t.Logf("target IP: %s, IP: %s, radius: %d, result %t", ip, "91.75.168.21", radius, res)
}
