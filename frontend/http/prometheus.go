package http

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"inet.af/netaddr"

	"github.com/chihaya/chihaya/bittorrent"
)

func init() {
	prometheus.MustRegister(promResponseDurationMilliseconds)
}

var promResponseDurationMilliseconds = prometheus.NewHistogramVec(
	prometheus.HistogramOpts{
		Name:    "chihaya_http_response_duration_milliseconds",
		Help:    "The duration of time it takes to receive and write a response to an API request",
		Buckets: prometheus.ExponentialBuckets(9.375, 2, 10),
	},
	[]string{"action", "address_family", "error"},
)

// recordResponseDuration records the duration of time to respond to a Request
// in milliseconds.
func recordResponseDuration(action string, ip netaddr.IP, err error, duration time.Duration) {
	var errString string
	if err != nil {
		if _, ok := err.(bittorrent.ClientError); ok {
			errString = err.Error()
		} else {
			errString = "internal error"
		}
	}

	var addressFamily string
	switch {
	case ip.IsZero(), ip.IsUnspecified():
		addressFamily = "Unknown"
	case ip.Is4(), ip.Is4in6():
		addressFamily = "IPv4"
	case ip.Is6():
		addressFamily = "IPv6"
	default:
		addressFamily = "Unknown"
	}

	promResponseDurationMilliseconds.
		WithLabelValues(action, addressFamily, errString).
		Observe(float64(duration.Nanoseconds()) / float64(time.Millisecond))
}
