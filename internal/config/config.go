package config

import (
	"fmt"
	"os"
	"strconv"
)

// Config holds all application configuration.
type Config struct {
	Port        int
	DataDir     string
	MaxUploadMB int64

	MySQLDSN string

	CacheMaxMB int64

	NeutronAPIURL   string
	VanaranaBaseURL string
}

// Load reads configuration from environment variables with sensible defaults.
func Load() (*Config, error) {
	cfg := &Config{
		Port:            envInt("VANARANA_PORT", 8080),
		DataDir:         envStr("VANARANA_DATA_DIR", "./data"),
		MaxUploadMB:     envInt64("VANARANA_MAX_UPLOAD_MB", 100),
		MySQLDSN:        envStr("VANARANA_MYSQL_DSN", "root:@tcp(127.0.0.1:3306)/vanarana?parseTime=true&charset=utf8mb4"),
		CacheMaxMB:      envInt64("VANARANA_CACHE_MAX_MB", 2048),
		NeutronAPIURL:   envStr("VANARANA_NEUTRON_URL", ""),
		VanaranaBaseURL: envStr("VANARANA_BASE_URL", "http://localhost:8080"),
	}

	if cfg.MySQLDSN == "" {
		return nil, fmt.Errorf("VANARANA_MYSQL_DSN is required")
	}

	return cfg, nil
}

// ArchiveDir returns the directory for storing uploaded tar.gz archives.
func (c *Config) ArchiveDir() string {
	return c.DataDir + "/archives"
}

// CacheDir returns the directory for extracted report cache.
func (c *Config) CacheDir() string {
	return c.DataDir + "/cache"
}

func envStr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func envInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		n, err := strconv.Atoi(v)
		if err == nil {
			return n
		}
	}
	return fallback
}

func envInt64(key string, fallback int64) int64 {
	if v := os.Getenv(key); v != "" {
		n, err := strconv.ParseInt(v, 10, 64)
		if err == nil {
			return n
		}
	}
	return fallback
}
