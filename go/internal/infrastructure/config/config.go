package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// Config holds all runtime configuration for pgscope, sourced from
// environment variables. No defaults are silently assumed for secrets.
type Config struct {
	DatabaseURL                string
	HTTPPort                   string
	PollInterval               time.Duration
	APIKey                     string
	InsightsRateLimitPerSecond float64
	InsightsRateLimitBurst     int
}

const (
	defaultPollIntervalSeconds        = 1
	defaultInsightsRateLimitPerSecond = 5
	defaultInsightsRateLimitBurst     = 20
)

func Load() (Config, error) {
	databaseURL := os.Getenv("PGSCOPE_DATABASE_URL")
	if databaseURL == "" {
		return Config{}, fmt.Errorf("PGSCOPE_DATABASE_URL is required")
	}

	apiKey := os.Getenv("PGSCOPE_API_KEY")
	if apiKey == "" {
		return Config{}, fmt.Errorf("PGSCOPE_API_KEY is required")
	}

	httpPort := os.Getenv("PGSCOPE_HTTP_PORT")
	if httpPort == "" {
		httpPort = "8090"
	}

	pollInterval, err := loadPollInterval()
	if err != nil {
		return Config{}, err
	}

	insightsRateLimit, err := loadInsightsRateLimitPerSecond()
	if err != nil {
		return Config{}, err
	}

	insightsBurst, err := loadInsightsRateLimitBurst()
	if err != nil {
		return Config{}, err
	}

	return Config{
		DatabaseURL:                databaseURL,
		HTTPPort:                   httpPort,
		PollInterval:               pollInterval,
		APIKey:                     apiKey,
		InsightsRateLimitPerSecond: insightsRateLimit,
		InsightsRateLimitBurst:     insightsBurst,
	}, nil
}

func loadPollInterval() (time.Duration, error) {
	raw := os.Getenv("PGSCOPE_POLL_INTERVAL_SECONDS")
	if raw == "" {
		return defaultPollIntervalSeconds * time.Second, nil
	}

	seconds, err := strconv.Atoi(raw)
	if err != nil || seconds <= 0 {
		return 0, fmt.Errorf("PGSCOPE_POLL_INTERVAL_SECONDS must be a positive integer, got %q", raw)
	}

	return time.Duration(seconds) * time.Second, nil
}

// loadInsightsRateLimitPerSecond controls how often /api/v1/insights can be
// called per IP, sustained rate. Defaults to a generous value since this is
// a single-operator tool by default — tighten it via env var if pgscope is
// ever exposed somewhere with untrusted traffic, since each call triggers
// 15+ sequential queries against a small connection pool.
func loadInsightsRateLimitPerSecond() (float64, error) {
	raw := os.Getenv("PGSCOPE_INSIGHTS_RATE_LIMIT")
	if raw == "" {
		return defaultInsightsRateLimitPerSecond, nil
	}

	rate, err := strconv.ParseFloat(raw, 64)
	if err != nil || rate <= 0 {
		return 0, fmt.Errorf("PGSCOPE_INSIGHTS_RATE_LIMIT must be a positive number, got %q", raw)
	}

	return rate, nil
}

func loadInsightsRateLimitBurst() (int, error) {
	raw := os.Getenv("PGSCOPE_INSIGHTS_RATE_BURST")
	if raw == "" {
		return defaultInsightsRateLimitBurst, nil
	}

	burst, err := strconv.Atoi(raw)
	if err != nil || burst <= 0 {
		return 0, fmt.Errorf("PGSCOPE_INSIGHTS_RATE_BURST must be a positive integer, got %q", raw)
	}

	return burst, nil
}
