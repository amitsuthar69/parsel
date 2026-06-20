package config

import (
	"crypto/rand"
	"fmt"
	"os"
)

type Config struct {
	RedisAddr   string
	LogDir      string
	WSAddr      string
	NodeName    string
	StreamName  string
	DatabaseURL string
}

func Load() Config {
	return Config{
		RedisAddr:   getenv("REDIS_ADDR", "localhost:6379"),
		LogDir:      getenv("LOG_DIR", "/var/log/containers"),
		WSAddr:      getenv("WS_ADDR", ":8080"),
		NodeName:    getenv("NODE_NAME", hostnameOrFallback()),
		StreamName:  getenv("STREAM_NAME", "parsel:logs"),
		DatabaseURL: getenv("DATABASE_URL", "postgres://parsel:parsel@postgres:5432/parsel"),
	}
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func hostnameOrFallback() string {
	h, err := os.Hostname()
	if err == nil && h != "" {
		return h
	}
	return randomID()
}

func randomID() string {
	var b [4]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "local"
	}
	return fmt.Sprintf("%x", b[:])
}
