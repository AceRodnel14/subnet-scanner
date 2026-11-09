package server

import (
	"cmp"
	"fmt"
	"log/slog"

	"subnet-scanner/internal/scanner"
)

// Config comes only from environment variables (documented in .env.example).
type Config struct {
	Port          string
	BaseURL       string
	LogLevel      slog.Level
	DefaultSubnet string // validated like /api/scan input
	Icon          string // path of the favicon file; empty for none
}

// LoadConfig reads the config through getenv (os.Getenv, or a map in tests).
func LoadConfig(getenv func(string) string) (Config, error) {
	cfg := Config{
		Port:          cmp.Or(getenv("PORT"), "8080"),
		BaseURL:       getenv("BASE_URL"),
		DefaultSubnet: cmp.Or(getenv("DEFAULT_SUBNET"), "192.168.1.0/24"),
		Icon:          getenv("ICON"),
	}
	if cfg.BaseURL == "" {
		cfg.BaseURL = "http://localhost:" + cfg.Port
	}
	if err := cfg.LogLevel.UnmarshalText([]byte(cmp.Or(getenv("LOG_LEVEL"), "info"))); err != nil {
		return Config{}, fmt.Errorf("LOG_LEVEL: %w", err)
	}
	if _, err := scanner.ParseSubnet(cfg.DefaultSubnet); err != nil {
		return Config{}, fmt.Errorf("DEFAULT_SUBNET: %w", err)
	}
	return cfg, nil
}
