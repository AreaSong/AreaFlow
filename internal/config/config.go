package config

import (
	"fmt"
	"net"
	"os"
	"strings"
)

type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	Auth     AuthConfig
}

type ServerConfig struct {
	Host string
	Port string
}

type DatabaseConfig struct {
	URL string
}

type AuthConfig struct {
	Mode string
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
		Auth: AuthConfig{Mode: envOrDefault("AREAFLOW_AUTH_MODE", "disabled")},
	}
}

func (cfg AuthConfig) Enabled() bool {
	return strings.EqualFold(strings.TrimSpace(cfg.Mode), "token")
}

func (cfg AuthConfig) Validate() error {
	switch strings.ToLower(strings.TrimSpace(cfg.Mode)) {
	case "", "disabled", "token":
		return nil
	default:
		return fmt.Errorf("AREAFLOW_AUTH_MODE must be disabled or token")
	}
}

func (cfg ServerConfig) Addr() string {
	return net.JoinHostPort(cfg.Host, cfg.Port)
}

func (cfg ServerConfig) Validate() error {
	host := strings.TrimSpace(cfg.Host)
	if strings.EqualFold(host, "localhost") {
		return nil
	}
	ip := net.ParseIP(host)
	if ip == nil || !ip.IsLoopback() {
		return fmt.Errorf("AREAFLOW_HOST must be a loopback address until remote authentication and TLS are enabled")
	}
	return nil
}

func envOrDefault(key string, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}
