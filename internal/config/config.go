package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	ServerAddr      string
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	ShutdownTimeout time.Duration
	CheckTimeout    time.Duration
	DataPath        string
}

func Load() *Config {
	return &Config{
		ServerAddr:      getEnv("SERVER_ADDR", ":8080"),
		ReadTimeout:     getDurationEnv("READ_TIMEOUT", 15*time.Second),
		WriteTimeout:    getDurationEnv("WRITE_TIMEOUT", 15*time.Second),
		ShutdownTimeout: getDurationEnv("SHUTDOWN_TIMEOUT", 30*time.Second),
		CheckTimeout:    getDurationEnv("CHECK_TIMEOUT", 10*time.Second),
		DataPath:        getEnv("DATA_PATH", "data/storage.json"),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getDurationEnv(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if d, err := strconv.Atoi(value); err == nil {
			return time.Duration(d) * time.Second
		}
	}
	return defaultValue
}
