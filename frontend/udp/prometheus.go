package udp

import (
	"errors"
	"net/netip"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/chihaya/chihaya/bittorrent"
	"github.com/chihaya/chihaya/pkg/metrics"
)

func init() {
	prometheus.MustRegister(promResponseDurationMilliseconds)
}

var promResponseDurationMilliseconds = prometheus.NewHistogramVec(
	prometheus.HistogramOpts{
		Name:    "chihaya_udp_response_duration_milliseconds",
		Help:    "The duration of time it takes to receive and write a response to an API request",
		Buckets: prometheus.ExponentialBuckets(9.375, 2, 10),
	},
	[]string{"action", "address_family", "error"},
)

// recordResponseDuration records the duration of time to respond to a UDP
// Request in milliseconds.
func recordResponseDuration(action string, ip netip.Addr, err error, duration time.Duration) {
	var errString string
	if err != nil {
		var clientErr bittorrent.ClientError
		if errors.As(err, &clientErr) {
			errString = clientErr.Error()
		} else {
			errString = "internal error"
		}
	}

	promResponseDurationMilliseconds.
		WithLabelValues(action, metrics.AddressFamily(ip), errString).
		Observe(float64(duration.Nanoseconds()) / float64(time.Millisecond))
}
