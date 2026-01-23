package config

import (
	"os"
)

type Config struct {
	AuthToken  string
	ServerPort string
	WorkDir    string
	DataDir    string
	DevMode    bool
	LogLevel   string

	// Relay settings
	RelayEnabled bool
	RelayURL     string
	RelayToken   string
	Subdomain    string
}

func Load() *Config {
	return &Config{
		AuthToken:  getEnv("AUTH_TOKEN", ""),
		ServerPort: getEnv("SERVER_PORT", "9870"),
		WorkDir:    getEnv("WORK_DIR", "."),
		DataDir:    getEnv("DATA_DIR", ".devport"),
		DevMode:    getEnv("DEV_MODE", "false") == "true",
		LogLevel:   getEnv("LOG_LEVEL", "info"),

		// Relay settings
		RelayEnabled: getEnv("RELAY_ENABLED", "true") == "true",
		RelayURL:     getEnv("RELAY_URL", "https://cloud.devport.app"),
		RelayToken:   getEnv("RELAY_TOKEN", ""),
		Subdomain:    getEnv("SUBDOMAIN", ""),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
