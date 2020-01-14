package storage

import "github.com/prometheus/client_golang/prometheus"

func init() {
	// Register the metrics.
	prometheus.MustRegister(
		PromGCDurationMilliseconds,
		PromInfohashesCount,
		PromSeedersCount,
		PromLeechersCount,
	)
}

var (
	// PromGCDurationMilliseconds is a histogram used by the storage to record
	// the durations of execution time required for removing expired peers.
	PromGCDurationMilliseconds = prometheus.NewHistogram(prometheus.HistogramOpts{
		Name:    "chihaya_storage_gc_duration_milliseconds",
		Help:    "The time it takes to perform storage garbage collection",
		Buckets: prometheus.ExponentialBuckets(9.375, 2, 10),
	})

	// PromFullscrapeDurationMilliseconds is a histogram used by the storage to
	// record the execution time of fullscrapes.
	PromFullscrapeDurationMilliseconds = prometheus.NewHistogram(prometheus.HistogramOpts{
		Name:    "chihaya_storage_fullscrape_duration_milliseconds",
		Help:    "The time it takes to fullfill a fullscrape request",
		Buckets: prometheus.ExponentialBuckets(9.375, 2, 10),
	})

	// PromInfohashesCount is a gauge used to hold the current total amount of
	// unique swarms being tracked by a storage.
	PromInfohashesCount = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "chihaya_storage_infohashes_count",
		Help: "The number of Infohashes tracked",
	})

	// PromSeedersCount is a gauge used to hold the current total amount of
	// unique seeders per swarm.
	PromSeedersCount = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "chihaya_storage_seeders_count",
		Help: "The number of seeders tracked",
	})

	// PromLeechersCount is a gauge used to hold the current total amount of
	// unique leechers per swarm.
	PromLeechersCount = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "chihaya_storage_leechers_count",
		Help: "The number of leechers tracked",
	})
)
