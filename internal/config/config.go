package config

import (
	"os"
	"strconv"
)

type Config struct {
	AppEnv            string
	AppPort           string
	ZoomSLGMaxMembers int
	ReplayEventLimit  int
}

// Load documents the data flow for internal/config/config.go. Startup and configuration tests reach this function; debug it by checking environment variables, defaults, and fallback parsing. It accepts the parameters in its signature, returns the declared result values, and the expected output is the behavior asserted by nearby tests or consumed by direct callers.
func Load() (Config, error) {
	return Config{
		AppEnv:            getEnv("APP_ENV", "development"),
		AppPort:           getEnv("APP_PORT", "8080"),
		ZoomSLGMaxMembers: getEnvInt("ZOOM_SLG_MAX_MEMBERS", 10),
		ReplayEventLimit:  getEnvInt("REPLAY_EVENT_LIMIT", 100),
	}, nil
}

// getEnv documents the data flow for internal/config/config.go. Startup and configuration tests reach this function; debug it by checking environment variables, defaults, and fallback parsing. It accepts the parameters in its signature, returns the declared result values, and the expected output is the behavior asserted by nearby tests or consumed by direct callers.
func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

// getEnvInt documents the data flow for internal/config/config.go. Startup and configuration tests reach this function; debug it by checking environment variables, defaults, and fallback parsing. It accepts the parameters in its signature, returns the declared result values, and the expected output is the behavior asserted by nearby tests or consumed by direct callers.
func getEnvInt(key string, fallback int) int {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}
