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
	DatabaseURL                   string
	HTTPPort                      string
	PollInterval                  time.Duration
	APIKey                        string
	InsightsRateLimitPerSecond    float64
	InsightsRateLimitBurst        int
	HistoryDBPath                 string
	HistoryRetention              time.Duration
	HistoryRecordInterval         time.Duration
	HistoryMaxSessionsPerSnapshot int
	HistoryMaxDBSizeBytes         int64
}

const (
	defaultPollIntervalSeconds           = 1
	defaultInsightsRateLimitPerSecond    = 5
	defaultInsightsRateLimitBurst        = 20
	defaultHistoryDBPath                 = "./data/pgscope.db"
	defaultHistoryRetentionDays          = 3
	defaultHistoryRecordIntervalSeconds  = 15
	defaultHistoryMaxSessionsPerSnapshot = 50
	defaultHistoryMaxDBSizeMB            = 500
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

	historyDBPath := os.Getenv("PGSCOPE_HISTORY_DB_PATH")
	if historyDBPath == "" {
		historyDBPath = defaultHistoryDBPath
	}

	historyRetention, err := loadHistoryRetention()
	if err != nil {
		return Config{}, err
	}

	historyRecordInterval, err := loadHistoryRecordInterval()
	if err != nil {
		return Config{}, err
	}

	historyMaxSessions, err := loadHistoryMaxSessionsPerSnapshot()
	if err != nil {
		return Config{}, err
	}

	historyMaxDBSizeBytes, err := loadHistoryMaxDBSizeBytes()
	if err != nil {
		return Config{}, err
	}

	return Config{
		DatabaseURL:                   databaseURL,
		HTTPPort:                      httpPort,
		PollInterval:                  pollInterval,
		APIKey:                        apiKey,
		InsightsRateLimitPerSecond:    insightsRateLimit,
		InsightsRateLimitBurst:        insightsBurst,
		HistoryDBPath:                 historyDBPath,
		HistoryRetention:              historyRetention,
		HistoryRecordInterval:         historyRecordInterval,
		HistoryMaxSessionsPerSnapshot: historyMaxSessions,
		HistoryMaxDBSizeBytes:         historyMaxDBSizeBytes,
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

// loadHistoryRetention controls how long periodic snapshots are kept in the
// SQLite history store before being pruned. Incident snapshots (a blocking
// chain appearing, etc.) are never pruned by age regardless of this value,
// since they're rare and stay valuable far longer than routine samples.
func loadHistoryRetention() (time.Duration, error) {
	raw := os.Getenv("PGSCOPE_HISTORY_RETENTION_DAYS")
	if raw == "" {
		return defaultHistoryRetentionDays * 24 * time.Hour, nil
	}

	days, err := strconv.Atoi(raw)
	if err != nil || days <= 0 {
		return 0, fmt.Errorf("PGSCOPE_HISTORY_RETENTION_DAYS must be a positive integer, got %q", raw)
	}

	return time.Duration(days) * 24 * time.Hour, nil
}

// loadHistoryRecordInterval controls how often a periodic snapshot is
// persisted to the history store, decoupled from PollInterval (which
// governs the live SSE dashboard). A slower record cadence keeps disk
// usage bounded independent of poll frequency; incident snapshots are
// still recorded immediately regardless of this interval.
func loadHistoryRecordInterval() (time.Duration, error) {
	raw := os.Getenv("PGSCOPE_HISTORY_RECORD_INTERVAL_SECONDS")
	if raw == "" {
		return defaultHistoryRecordIntervalSeconds * time.Second, nil
	}

	seconds, err := strconv.Atoi(raw)
	if err != nil || seconds <= 0 {
		return 0, fmt.Errorf("PGSCOPE_HISTORY_RECORD_INTERVAL_SECONDS must be a positive integer, got %q", raw)
	}

	return time.Duration(seconds) * time.Second, nil
}

// loadHistoryMaxSessionsPerSnapshot bounds how many sessions a periodic
// snapshot stores, regardless of how many are actually active — this keeps
// per-snapshot size (and therefore disk growth) independent of how busy the
// monitored database is. Incident snapshots are never trimmed this way,
// since understanding a blocking chain requires every session involved.
func loadHistoryMaxSessionsPerSnapshot() (int, error) {
	raw := os.Getenv("PGSCOPE_HISTORY_MAX_SESSIONS_PER_SNAPSHOT")
	if raw == "" {
		return defaultHistoryMaxSessionsPerSnapshot, nil
	}

	maxSessions, err := strconv.Atoi(raw)
	if err != nil || maxSessions <= 0 {
		return 0, fmt.Errorf("PGSCOPE_HISTORY_MAX_SESSIONS_PER_SNAPSHOT must be a positive integer, got %q", raw)
	}

	return maxSessions, nil
}

// loadHistoryMaxDBSizeBytes is a backstop on top of the age-based retention
// (loadHistoryRetention) — if the history file somehow grows past this size
// before the retention window catches up (an unusually busy monitored
// database, a misconfigured record interval, ...), older periodic snapshots
// are pruned more aggressively to bring it back under the cap.
func loadHistoryMaxDBSizeBytes() (int64, error) {
	raw := os.Getenv("PGSCOPE_HISTORY_MAX_DB_SIZE_MB")
	if raw == "" {
		return defaultHistoryMaxDBSizeMB * 1024 * 1024, nil
	}

	megabytes, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || megabytes <= 0 {
		return 0, fmt.Errorf("PGSCOPE_HISTORY_MAX_DB_SIZE_MB must be a positive integer, got %q", raw)
	}

	return megabytes * 1024 * 1024, nil
}
