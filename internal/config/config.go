package config

import (
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	defaultServiceName     = "trendstream"
	defaultHTTPAddr        = ":8080"
	defaultAdminAddr       = ":9090"
	defaultLogLevel        = "info"
	defaultShutdownTimeout = 5 * time.Second
	defaultAdminToken      = "dev-token"
	defaultStopListPath    = "data/stoplist.json"
	defaultPPROFEnabled    = false

	defaultKafkaEnabled  = false
	defaultKafkaBrokers  = "localhost:9092"
	defaultKafkaTopic    = "search-events"
	defaultKafkaGroupID  = "trendstream"
	defaultKafkaClientID = "trendstream-local"

	defaultShardCount                = 32
	defaultMaxUniqueQueries          = 1_000_000
	defaultMaxUniqueQueriesPerBucket = 100_000
	defaultPerActorQueryLimit        = 3
)

type Config struct {
	ServiceName     string
	HTTPAddr        string
	AdminAddr       string
	LogLevel        string
	ShutdownTimeout time.Duration

	AdminToken   string
	StopListPath string
	PPROFEnabled bool

	KafkaEnabled  bool
	KafkaBrokers  []string
	KafkaTopic    string
	KafkaGroupID  string
	KafkaClientID string

	ShardCount                int
	MaxUniqueQueries          int
	MaxUniqueQueriesPerBucket int
	PerActorQueryLimit        int64
}

func Load() Config {
	return Config{
		ServiceName:     getEnv("SERVICE_NAME", defaultServiceName),
		HTTPAddr:        getEnv("HTTP_ADDR", defaultHTTPAddr),
		AdminAddr:       getEnv("ADMIN_ADDR", defaultAdminAddr),
		LogLevel:        getEnv("LOG_LEVEL", defaultLogLevel),
		ShutdownTimeout: getDurationEnv("SHUTDOWN_TIMEOUT", defaultShutdownTimeout),

		AdminToken:   getEnv("ADMIN_TOKEN", defaultAdminToken),
		StopListPath: getEnv("STOPLIST_PATH", defaultStopListPath),
		PPROFEnabled: getBoolEnv("PPROF_ENABLED", defaultPPROFEnabled),

		KafkaEnabled:  getBoolEnv("KAFKA_ENABLED", defaultKafkaEnabled),
		KafkaBrokers:  getCSVEnv("KAFKA_BROKERS", defaultKafkaBrokers),
		KafkaTopic:    getEnv("KAFKA_TOPIC", defaultKafkaTopic),
		KafkaGroupID:  getEnv("KAFKA_GROUP_ID", defaultKafkaGroupID),
		KafkaClientID: getEnv("KAFKA_CLIENT_ID", defaultKafkaClientID),

		ShardCount:                getIntEnv("SHARD_COUNT", defaultShardCount),
		MaxUniqueQueries:          getIntEnv("MAX_UNIQUE_QUERIES", defaultMaxUniqueQueries),
		MaxUniqueQueriesPerBucket: getIntEnv("MAX_UNIQUE_QUERIES_PER_BUCKET", defaultMaxUniqueQueriesPerBucket),
		PerActorQueryLimit:        int64(getIntEnv("PER_ACTOR_QUERY_LIMIT", defaultPerActorQueryLimit)),
	}
}

func getEnv(key string, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}

	return value
}

func getCSVEnv(key string, fallback string) []string {
	raw := getEnv(key, fallback)

	parts := strings.Split(raw, ",")
	values := make([]string, 0, len(parts))

	for _, part := range parts {
		value := strings.TrimSpace(part)
		if value == "" {
			continue
		}

		values = append(values, value)
	}

	return values
}

func getBoolEnv(key string, fallback bool) bool {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}

	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return fallback
	}

	return parsed
}

func getIntEnv(key string, fallback int) int {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}

	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}

	return parsed
}

func getDurationEnv(key string, fallback time.Duration) time.Duration {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}

	duration, err := time.ParseDuration(value)
	if err == nil {
		return duration
	}

	seconds, err := strconv.Atoi(value)
	if err == nil && seconds > 0 {
		return time.Duration(seconds) * time.Second
	}

	return fallback
}
