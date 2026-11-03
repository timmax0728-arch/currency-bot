package config

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	TelegramToken string
	HTTPTimeout   time.Duration
	CacheTTL      time.Duration
}

func Load() (*Config, error) {
	_ = godotenv.Load()

	token := os.Getenv("TELEGRAM_TOKEN")
	if token == "" {
		return nil, errors.New("TELEGRAM_TOKEN is not set")
	}

	httpTimeout, err := getDuration("HTTP_TIMEOUT", 10*time.Second)
	if err != nil {
		return nil, err
	}

	cacheTTL, err := getDuration("CACHE_TTL", 10*time.Minute)
	if err != nil {
		return nil, err
	}

	return &Config{
		TelegramToken: token,
		HTTPTimeout:   httpTimeout,
		CacheTTL:      cacheTTL,
	}, nil
}

func getDuration(key string, def time.Duration) (time.Duration, error) {
	raw := os.Getenv(key)
	if raw == "" {
		return def, nil
	}

	d, err := time.ParseDuration(raw)
	if err != nil {
		return 0, fmt.Errorf("parse %s: %w", key, err)
	}

	if d <= 0 {
		return 0, fmt.Errorf("%s must be positive, got %s", key, raw)
	}
	return d, nil
}
