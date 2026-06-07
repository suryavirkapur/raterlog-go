package config

import (
	"os"
	"strconv"
	"strings"
)

type Config struct {
	HTTPAddr       string
	PublicURL      string
	CORSOrigins    []string
	PostgresURL    string
	ScyllaHosts    []string
	ScyllaKeyspace string
	CookieSecure   bool
	SMTPHost       string
	SMTPPort       string
	SMTPFrom       string
}

func Load() Config {
	return Config{
		HTTPAddr:       env("HTTP_ADDR", "0.0.0.0:8080"),
		PublicURL:      env("PUBLIC_URL", "http://localhost:3000"),
		CORSOrigins:    csv(env("CORS_ORIGINS", "http://localhost:3000,http://localhost:5173")),
		PostgresURL:    env("POSTGRES_URL", "postgres://postgres:postgres@localhost:5432/raterlog?sslmode=disable"),
		ScyllaHosts:    csv(env("SCYLLA_HOSTS", "localhost:9042")),
		ScyllaKeyspace: env("SCYLLA_KEYSPACE", "raterlog"),
		CookieSecure:   envBool("COOKIE_SECURE", false),
		SMTPHost:       env("SMTP_HOST", "localhost"),
		SMTPPort:       env("SMTP_PORT", "1025"),
		SMTPFrom:       env("SMTP_FROM", "Raterlog <noreply@raterlog.dev>"),
	}
}

func env(key, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}

func envBool(key string, fallback bool) bool {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func csv(value string) []string {
	parts := strings.Split(value, ",")
	items := make([]string, 0, len(parts))
	for _, part := range parts {
		item := strings.TrimSpace(part)
		if item != "" {
			items = append(items, item)
		}
	}
	return items
}
