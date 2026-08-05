// Package config loads application configuration from environment variables.
//
// Go has no built-in equivalent of Laravel's automatic .env loading — the
// godotenv.Load call below is what reads .env into the process environment
// before we read from it with os.Getenv.
package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	ServerPort          string
	ElasticsearchURL    string
	ElasticsearchAPIKey string
}

func Load() (*Config, error) {
	// Ignore a missing file: in production the real environment may be set
	// directly rather than via .env.
	_ = godotenv.Load()

	cfg := &Config{
		ServerPort:          getEnv("SERVER_PORT", "8080"),
		ElasticsearchURL:    getEnv("ES_URL", "http://localhost:9200"),
		ElasticsearchAPIKey: os.Getenv("API_KEY"),
	}

	if cfg.ElasticsearchAPIKey == "" {
		return nil, fmt.Errorf("API_KEY is not set (expected in elastic.env or the environment)")
	}

	return cfg, nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
