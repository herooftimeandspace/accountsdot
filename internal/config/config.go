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

func Load() (Config, error) {
	return Config{
		AppEnv:            getEnv("APP_ENV", "development"),
		AppPort:           getEnv("APP_PORT", "8080"),
		ZoomSLGMaxMembers: getEnvInt("ZOOM_SLG_MAX_MEMBERS", 10),
		ReplayEventLimit:  getEnvInt("REPLAY_EVENT_LIMIT", 100),
	}, nil
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

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
