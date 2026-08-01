package config

import (
	"os"
	"time"
)

type Config struct {
	Port         string
	DatabaseURL  string
	Bucket       string
	Region       string
	PresignTTL   time.Duration
	PollInterval time.Duration
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func Load() *Config {
	return &Config{
		Port:         getenv("PORT", "8080"),
		DatabaseURL:  os.Getenv("DATABASE_URL"),
		Bucket:       os.Getenv("MEDIA_BUCKET"),
		Region:       getenv("AWS_REGION", "eu-central-1"),
		PresignTTL:   15 * time.Minute,
		PollInterval: 5 * time.Second,
	}
}
