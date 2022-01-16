package http

import (
	"errors"
	"time"

	"github.com/prometheus/client_golang/prometheus"

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
func recordResponseDuration(action string, af *bittorrent.AddressFamily, err error, duration time.Duration) {
	var errString string
	if err != nil {
		var clientErr bittorrent.ClientError
		if errors.As(err, &clientErr) {
			errString = clientErr.Error()
		} else {
			errString = "internal error"
		}
	}

	var afString string
	if af == nil {
		afString = "Unknown"
	} else if *af == bittorrent.IPv4 {
		afString = "IPv4"
	} else if *af == bittorrent.IPv6 {
		afString = "IPv6"
	}

	promResponseDurationMilliseconds.
		WithLabelValues(action, afString, errString).
		Observe(float64(duration.Nanoseconds()) / float64(time.Millisecond))
}
