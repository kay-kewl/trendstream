package metrics

import (
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/kay-kewl/trendstream/internal/ingest"
)

const namespace = "trendstream"

type Metrics struct {
	registry *prometheus.Registry

	httpRequestsTotal     *prometheus.CounterVec
	httpRequestDuration   *prometheus.HistogramVec
	ingestEventsTotal     *prometheus.CounterVec
	kafkaRecordsPolled    prometheus.Counter
	kafkaRecordsCommitted prometheus.Counter
	kafkaFetchErrors      *prometheus.CounterVec
	kafkaCommitErrors     prometheus.Counter
	kafkaDecodeErrors     prometheus.Counter

	snapshotRebuildDuration prometheus.Histogram
	snapshotAgeSeconds      prometheus.Gauge
	snapshotItems           prometheus.Gauge
	currentUniqueQueries    prometheus.Gauge
	currentWindowEvents     prometheus.Gauge
	currentActorCounters    prometheus.Gauge
	stopListRules           prometheus.Gauge
}

func New() *Metrics {
	registry := prometheus.NewRegistry()

	m := &Metrics{
		registry: registry,
		httpRequestsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: "http",
				Name:      "requests_total",
				Help:      "Total number of HTTP requests handled by the service.",
			},
			[]string{"server", "method", "path", "status"},
		),
		httpRequestDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Subsystem: "http",
				Name:      "request_duration_seconds",
				Help:      "HTTP request duration in seconds.",
				Buckets:   prometheus.DefBuckets,
			},
			[]string{"server", "method", "path"},
		),
		ingestEventsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: "ingest",
				Name:      "events_total",
				Help:      "Total number of processed ingest events by result and reason.",
			},
			[]string{"result", "reason"},
		),
		kafkaRecordsPolled: prometheus.NewCounter(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: "kafka",
				Name:      "records_polled_total",
				Help:      "Total number of Kafka records polled by the consumer.",
			},
		),
		kafkaRecordsCommitted: prometheus.NewCounter(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: "kafka",
				Name:      "records_committed_total",
				Help:      "Total number of Kafka records committed after processing.",
			},
		),
		kafkaFetchErrors: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: "kafka",
				Name:      "fetch_errors_total",
				Help:      "Total number of Kafka fetch errors.",
			},
			[]string{"topic", "partition"},
		),
		kafkaCommitErrors: prometheus.NewCounter(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: "kafka",
				Name:      "commit_errors_total",
				Help:      "Total number of Kafka offset commit errors.",
			},
		),
		kafkaDecodeErrors: prometheus.NewCounter(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: "kafka",
				Name:      "decode_errors_total",
				Help:      "Total number of Kafka records rejected because their payload could not be decoded.",
			},
		),
		snapshotRebuildDuration: prometheus.NewHistogram(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Subsystem: "snapshot",
				Name:      "rebuild_duration_seconds",
				Help:      "Snapshot rebuild duration in seconds.",
				Buckets:   prometheus.DefBuckets,
			},
		),
		snapshotAgeSeconds: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Subsystem: "snapshot",
				Name:      "age_seconds",
				Help:      "Current published snapshot age in seconds.",
			},
		),
		snapshotItems: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Subsystem: "snapshot",
				Name:      "items",
				Help:      "Number of items in the current published trend snapshot.",
			},
		),
		currentUniqueQueries: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Subsystem: "aggregator",
				Name:      "current_unique_queries",
				Help:      "Number of unique queries currently tracked in the sliding window.",
			},
		),
		currentWindowEvents: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Subsystem: "aggregator",
				Name:      "current_window_events",
				Help:      "Total number of accepted events currently represented in the sliding window.",
			},
		),
		currentActorCounters: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Subsystem: "aggregator",
				Name:      "current_actor_counters",
				Help:      "Number of actor/query counters currently tracked for abuse guardrails.",
			},
		),
		stopListRules: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Subsystem: "stoplist",
				Name:      "rules",
				Help:      "Number of currently configured stop-list rules.",
			},
		),
	}

	registry.MustRegister(
		m.httpRequestsTotal,
		m.httpRequestDuration,
		m.ingestEventsTotal,
		m.kafkaRecordsPolled,
		m.kafkaRecordsCommitted,
		m.kafkaFetchErrors,
		m.kafkaCommitErrors,
		m.kafkaDecodeErrors,
		m.snapshotRebuildDuration,
		m.snapshotAgeSeconds,
		m.snapshotItems,
		m.currentUniqueQueries,
		m.currentWindowEvents,
		m.currentActorCounters,
		m.stopListRules,
	)

	return m
}

func (m *Metrics) Registry() *prometheus.Registry {
	return m.registry
}

func (m *Metrics) ObserveHTTPRequest(server string, method string, path string, statusCode int, duration time.Duration) {
	status := strconv.Itoa(statusCode)

	m.httpRequestsTotal.WithLabelValues(server, method, path, status).Inc()
	m.httpRequestDuration.WithLabelValues(server, method, path).Observe(duration.Seconds())
}

func (m *Metrics) ObserveIngestResult(result ingest.Result) {
	if result.Accepted {
		m.ingestEventsTotal.WithLabelValues("accepted", "").Inc()
		return
	}

	reason := string(result.Reason)
	if reason == "" {
		reason = "unknown"
	}

	m.ingestEventsTotal.WithLabelValues("dropped", reason).Inc()
}

func (m *Metrics) ObserveKafkaRecordsPolled(count int) {
	if count <= 0 {
		return
	}

	m.kafkaRecordsPolled.Add(float64(count))
}

func (m *Metrics) ObserveKafkaRecordsCommitted(count int) {
	if count <= 0 {
		return
	}

	m.kafkaRecordsCommitted.Add(float64(count))
}

func (m *Metrics) ObserveKafkaFetchError(topic string, partition int32) {
	m.kafkaFetchErrors.WithLabelValues(topic, strconv.Itoa(int(partition))).Inc()
}

func (m *Metrics) ObserveKafkaCommitError() {
	m.kafkaCommitErrors.Inc()
}

func (m *Metrics) ObserveKafkaDecodeError() {
	m.kafkaDecodeErrors.Inc()
}

func (m *Metrics) ObserveSnapshotRebuild(duration time.Duration, generatedAt time.Time, items int) {
	m.snapshotRebuildDuration.Observe(duration.Seconds())
	m.snapshotAgeSeconds.Set(time.Since(generatedAt).Seconds())
	m.snapshotItems.Set(float64(items))
}

func (m *Metrics) SetAggregatorStats(uniqueQueries int, windowEvents int64, actorCounters int) {
	m.currentUniqueQueries.Set(float64(uniqueQueries))
	m.currentWindowEvents.Set(float64(windowEvents))
	m.currentActorCounters.Set(float64(actorCounters))
}

func (m *Metrics) SetStopListRules(count int) {
	m.stopListRules.Set(float64(count))
}
