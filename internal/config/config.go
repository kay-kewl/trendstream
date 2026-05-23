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

	defaultKafkaEnabled  = false
	defaultKafkaBrokers  = "localhost:9092"
	defaultKafkaTopic    = "search-events"
	defaultKafkaGroupID  = "trendstream"
	defaultKafkaClientID = "trendstream-local"
)

type Config struct {
	ServiceName     string
	HTTPAddr        string
	AdminAddr       string
	LogLevel        string
	ShutdownTimeout time.Duration

	AdminToken   string
	StopListPath string

	KafkaEnabled  bool
	KafkaBrokers  []string
	KafkaTopic    string
	KafkaGroupID  string
	KafkaClientID string
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

		KafkaEnabled:  getBoolEnv("KAFKA_ENABLED", defaultKafkaEnabled),
		KafkaBrokers:  getCSVEnv("KAFKA_BROKERS", defaultKafkaBrokers),
		KafkaTopic:    getEnv("KAFKA_TOPIC", defaultKafkaTopic),
		KafkaGroupID:  getEnv("KAFKA_GROUP_ID", defaultKafkaGroupID),
		KafkaClientID: getEnv("KAFKA_CLIENT_ID", defaultKafkaClientID),
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
