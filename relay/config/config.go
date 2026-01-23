package config

import (
	"os"
)

type Config struct {
	ServerPort  string
	ServerHost  string
	BaseURL     string
	Domain      string
	DevMode     bool
}

func Load() *Config {
	cfg := &Config{
		ServerPort: getEnv("SERVER_PORT", "8080"),
		ServerHost: getEnv("SERVER_HOST", "0.0.0.0"),
		BaseURL:    getEnv("BASE_URL", "http://localhost:8080"),
		Domain:     getEnv("DOMAIN", "cloud.devport.app"),
		DevMode:    getEnv("DEV_MODE", "") == "true",
	}
	return cfg
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
