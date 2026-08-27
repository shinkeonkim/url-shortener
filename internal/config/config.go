package config

import (
	"errors"
	"os"
	"time"
)

type Config struct {
	Address      string
	DatabasePath string
	BaseDomain   string
	BaseURL      string
	AdminUser    string
	AdminHash    string
	AdminToken   string
	SessionKey   string
	CookieSecure bool
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
}

func Load() (Config, error) {
	c := Config{
		Address: env("ADDRESS", ":8080"), DatabasePath: env("DATABASE_PATH", "url-shortener.db"),
		BaseDomain: env("BASE_DOMAIN", "url.shinkeonkim.com"), BaseURL: env("BASE_URL", "https://url.shinkeonkim.com"),
		AdminUser: os.Getenv("ADMIN_USERNAME"), AdminHash: os.Getenv("ADMIN_PASSWORD_HASH"),
		AdminToken: os.Getenv("ADMIN_TOKEN"), SessionKey: os.Getenv("SESSION_KEY"),
		CookieSecure: env("COOKIE_SECURE", "true") == "true", ReadTimeout: 5 * time.Second, WriteTimeout: 10 * time.Second,
	}
	if c.DatabasePath == "" || c.BaseDomain == "" {
		return Config{}, errors.New("DATABASE_PATH and BASE_DOMAIN must not be empty")
	}
	return c, nil
}

func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
