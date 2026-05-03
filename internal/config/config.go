// Package config loads runtime configuration from environment variables.
package config

import (
	"errors"
	"os"
	"strconv"
	"time"
)

// Config holds the runtime configuration for the TikTok downloader bot.
type Config struct {
	BotToken     string
	TikTokCookie string
	ProxyURL     string
	TikhubAPIKey string
	Port         int
	WorkDir      string
	MaxFileSize  int64
	HTTPTimeout  time.Duration
}

// Load reads configuration from environment variables.
// BOT_TOKEN is required; everything else is optional.
func Load() (*Config, error) {
	token := os.Getenv("BOT_TOKEN")
	if token == "" {
		return nil, errors.New("BOT_TOKEN environment variable is required")
	}

	port := 8080
	if v := os.Getenv("PORT"); v != "" {
		p, err := strconv.Atoi(v)
		if err != nil {
			return nil, errors.New("PORT must be an integer")
		}
		port = p
	}

	workDir := os.Getenv("WORK_DIR")
	if workDir == "" {
		workDir = os.TempDir()
	}

	return &Config{
		BotToken:     token,
		TikTokCookie: os.Getenv("TIKTOK_COOKIE"),
		ProxyURL:     os.Getenv("PROXY_URL"),
		TikhubAPIKey: os.Getenv("TIKHUB_API_KEY"),
		Port:         port,
		WorkDir:      workDir,
		MaxFileSize:  50 * 1024 * 1024,
		HTTPTimeout:  60 * time.Second,
	}, nil
}
