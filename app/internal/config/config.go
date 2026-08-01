package config

import (
	"fmt"
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
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = buildDatabaseURL()
	}
	return &Config{
		Port:         getenv("PORT", "8080"),
		DatabaseURL:  dbURL,
		Bucket:       os.Getenv("MEDIA_BUCKET"),
		Region:       getenv("AWS_REGION", "eu-central-1"),
		PresignTTL:   15 * time.Minute,
		PollInterval: 5 * time.Second,
	}
}

func buildDatabaseURL() string {
	host := os.Getenv("DB_HOST")
	port := os.Getenv("DB_PORT")
	name := os.Getenv("DB_NAME")
	user := os.Getenv("DB_USERNAME")
	pass := os.Getenv("DB_PASSWORD")
	if host == "" {
		return ""
	}
	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s", user, pass, host, port, name)
}
