package config

import (
	"os"
	"strconv"
	"time"
)

const (
	defaultServiceName     = "trendstream"
	defaultHTTPAddr        = ":8080"
	defaultAdminAddr       = ":9090"
	defaultLogLevel        = "info"
	defaultShutdownTimeout = 5 * time.Second
)

type Config struct {
	ServiceName     string
	HTTPAddr        string
	AdminAddr       string
	LogLevel        string
	ShutdownTimeout time.Duration
}

func Load() Config {
	return Config{
		ServiceName:     getEnv("SERVICE_NAME", defaultServiceName),
		HTTPAddr:        getEnv("HTTP_ADDR", defaultHTTPAddr),
		AdminAddr:       getEnv("ADMIN_ADDR", defaultAdminAddr),
		LogLevel:        getEnv("LOG_LEVEL", defaultLogLevel),
		ShutdownTimeout: getDurationEnv("SHUTDOWN_TIMEOUT", defaultShutdownTimeout),
	}
}

func getEnv(key string, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}

	return value
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
