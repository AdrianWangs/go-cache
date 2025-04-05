// Package config handles the configuration for the cache system
package config

import (
	"encoding/json"
	"os"
	"strconv"
	"strings"
)

// Config represents the application configuration
type Config struct {
	// Cache settings
	MaxCacheBytes      int64 `json:"max_cache_bytes"`
	DefaultCacheExpiry int   `json:"default_cache_expiry_seconds"`

	// Server settings
	APIPort       int      `json:"api_port"`
	CachePort     int      `json:"cache_port"`
	Host          string   `json:"host"`
	BasePath      string   `json:"base_path"`
	PeerAddresses []string `json:"peer_addresses"`

	// Logging settings
	LogLevel  string `json:"log_level"`
	LogFormat string `json:"log_format"`
}

// DefaultConfig returns the default configuration
func DefaultConfig() *Config {
	return &Config{
		MaxCacheBytes:      1024 * 1024 * 100, // 100MB
		DefaultCacheExpiry: 3600,              // 1 hour
		APIPort:            9999,
		CachePort:          8001,
		Host:               "localhost",
		BasePath:           "/_gocache/",
		PeerAddresses:      []string{"http://localhost:8001", "http://localhost:8002", "http://localhost:8003"},
		LogLevel:           "info",
		LogFormat:          "text",
	}
}

// LoadFromFile loads configuration from a JSON file
func LoadFromFile(filepath string) (*Config, error) {
	config := DefaultConfig()

	file, err := os.ReadFile(filepath)
	if err != nil {
		return config, err
	}

	err = json.Unmarshal(file, config)
	if err != nil {
		return config, err
	}

	return config, nil
}

// LoadFromEnv loads configuration from environment variables
func LoadFromEnv() *Config {
	config := DefaultConfig()

	// Cache settings
	if val := os.Getenv("GOCACHE_MAX_BYTES"); val != "" {
		if parsed, err := strconv.ParseInt(val, 10, 64); err == nil {
			config.MaxCacheBytes = parsed
		}
	}

	if val := os.Getenv("GOCACHE_EXPIRY_SECONDS"); val != "" {
		if parsed, err := strconv.Atoi(val); err == nil {
			config.DefaultCacheExpiry = parsed
		}
	}

	// Server settings
	if val := os.Getenv("GOCACHE_API_PORT"); val != "" {
		if parsed, err := strconv.Atoi(val); err == nil {
			config.APIPort = parsed
		}
	}

	if val := os.Getenv("GOCACHE_PORT"); val != "" {
		if parsed, err := strconv.Atoi(val); err == nil {
			config.CachePort = parsed
		}
	}

	if val := os.Getenv("GOCACHE_HOST"); val != "" {
		config.Host = val
	}

	if val := os.Getenv("GOCACHE_BASE_PATH"); val != "" {
		config.BasePath = val
	}

	if val := os.Getenv("GOCACHE_PEERS"); val != "" {
		config.PeerAddresses = strings.Split(val, ",")
	}

	// Logging settings
	if val := os.Getenv("GOCACHE_LOG_LEVEL"); val != "" {
		config.LogLevel = val
	}

	if val := os.Getenv("GOCACHE_LOG_FORMAT"); val != "" {
		config.LogFormat = val
	}

	return config
}

// SaveToFile saves configuration to a JSON file
func (c *Config) SaveToFile(filepath string) error {
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(filepath, data, 0644)
}
