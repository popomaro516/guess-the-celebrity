package config

import "os"

type Config struct {
	HTTPAddr string
	BaseURL  string
}

func Load() Config {
	addr := getenv("HTTP_ADDR", ":8080")
	baseURL := getenv("BASE_URL", "http://localhost:8080")
	return Config{HTTPAddr: addr, BaseURL: baseURL}
}

func getenv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
