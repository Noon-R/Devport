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
}

func Load() *Config {
	return &Config{
		AuthToken:  getEnv("AUTH_TOKEN", ""),
		ServerPort: getEnv("SERVER_PORT", "9870"),
		WorkDir:    getEnv("WORK_DIR", "."),
		DataDir:    getEnv("DATA_DIR", ".devport"),
		DevMode:    getEnv("DEV_MODE", "false") == "true",
		LogLevel:   getEnv("LOG_LEVEL", "info"),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
