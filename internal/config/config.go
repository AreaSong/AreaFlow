package config

import "os"

type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
}

type ServerConfig struct {
	Host string
	Port string
}

type DatabaseConfig struct {
	URL string
}

func FromEnv() Config {
	return Config{
		Server: ServerConfig{
			Host: envOrDefault("AREAFLOW_HOST", "127.0.0.1"),
			Port: envOrDefault("AREAFLOW_PORT", "3847"),
		},
		Database: DatabaseConfig{
			URL: envOrDefault("AREAFLOW_DATABASE_URL", "postgres://areaflow:areaflow@localhost:54329/areaflow?sslmode=disable"),
		},
	}
}

func (cfg ServerConfig) Addr() string {
	return cfg.Host + ":" + cfg.Port
}

func envOrDefault(key string, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}
