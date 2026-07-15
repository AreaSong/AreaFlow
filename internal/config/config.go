package config

import (
	"fmt"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Environment   string
	Server        ServerConfig
	Database      DatabaseConfig
	Auth          AuthConfig
	Artifact      ArtifactConfig
	Observability ObservabilityConfig
}

type ServerConfig struct {
	Host              string
	Port              string
	PublicBaseURL     string
	TrustedProxyCIDRs []string
	AllowRemote       bool
}

type DatabaseConfig struct {
	URL                   string
	MaxConnections        int32
	MinConnections        int32
	ConnectTimeout        time.Duration
	AcquireTimeout        time.Duration
	QueryTimeout          time.Duration
	MaxConnectionIdle     time.Duration
	MaxConnectionLifetime time.Duration
}

type AuthConfig struct {
	Mode                 string
	OIDCIssuerURL        string
	OIDCClientID         string
	OIDCClientSecretFile string
	OIDCRedirectURL      string
	OIDCGroupsClaim      string
	SessionSecretFile    string
	SessionCookieName    string
	SessionIdleTTL       time.Duration
	SessionAbsoluteTTL   time.Duration
	TokenMaxTTL          time.Duration
	BootstrapSubjects    []string
}

type ArtifactConfig struct {
	Backend        string
	LocalRoot      string
	S3Endpoint     string
	S3Region       string
	S3Bucket       string
	S3UsePathStyle bool
}

type ObservabilityConfig struct {
	MetricsHost  string
	MetricsPort  string
	OTLPEndpoint string
	ServiceName  string
}

func FromEnv() Config {
	environment := strings.ToLower(envOrDefault("AREAFLOW_ENV", "development"))
	authMode := envOrDefault("AREAFLOW_AUTH_MODE", "disabled")
	return Config{
		Environment: environment,
		Server: ServerConfig{
			Host:              envOrDefault("AREAFLOW_HOST", "127.0.0.1"),
			Port:              envOrDefault("AREAFLOW_PORT", "3847"),
			PublicBaseURL:     strings.TrimSpace(os.Getenv("AREAFLOW_PUBLIC_BASE_URL")),
			TrustedProxyCIDRs: csvList(os.Getenv("AREAFLOW_TRUSTED_PROXY_CIDRS")),
			AllowRemote:       environment == "production" && strings.EqualFold(authMode, "oidc"),
		},
		Database: DatabaseConfig{
			URL:                   envOrDefault("AREAFLOW_DATABASE_URL", "postgres://areaflow:areaflow@localhost:54329/areaflow?sslmode=disable"),
			MaxConnections:        int32(envInt("AREAFLOW_DB_MAX_CONNECTIONS", 30)),
			MinConnections:        int32(envInt("AREAFLOW_DB_MIN_CONNECTIONS", 5)),
			ConnectTimeout:        envDuration("AREAFLOW_DB_CONNECT_TIMEOUT", 5*time.Second),
			AcquireTimeout:        envDuration("AREAFLOW_DB_ACQUIRE_TIMEOUT", 5*time.Second),
			QueryTimeout:          envDuration("AREAFLOW_DB_QUERY_TIMEOUT", 10*time.Second),
			MaxConnectionIdle:     envDuration("AREAFLOW_DB_MAX_CONNECTION_IDLE", 5*time.Minute),
			MaxConnectionLifetime: envDuration("AREAFLOW_DB_MAX_CONNECTION_LIFETIME", 30*time.Minute),
		},
		Auth: AuthConfig{
			Mode:                 authMode,
			OIDCIssuerURL:        strings.TrimSpace(os.Getenv("AREAFLOW_OIDC_ISSUER_URL")),
			OIDCClientID:         strings.TrimSpace(os.Getenv("AREAFLOW_OIDC_CLIENT_ID")),
			OIDCClientSecretFile: strings.TrimSpace(os.Getenv("AREAFLOW_OIDC_CLIENT_SECRET_FILE")),
			OIDCRedirectURL:      strings.TrimSpace(os.Getenv("AREAFLOW_OIDC_REDIRECT_URL")),
			OIDCGroupsClaim:      envOrDefault("AREAFLOW_OIDC_GROUPS_CLAIM", "groups"),
			SessionSecretFile:    strings.TrimSpace(os.Getenv("AREAFLOW_SESSION_SECRET_FILE")),
			SessionCookieName:    envOrDefault("AREAFLOW_SESSION_COOKIE_NAME", "areaflow_session"),
			SessionIdleTTL:       envDuration("AREAFLOW_SESSION_IDLE_TTL", 8*time.Hour),
			SessionAbsoluteTTL:   envDuration("AREAFLOW_SESSION_ABSOLUTE_TTL", 24*time.Hour),
			TokenMaxTTL:          envDuration("AREAFLOW_TOKEN_MAX_TTL", 90*24*time.Hour),
			BootstrapSubjects:    csvList(os.Getenv("AREAFLOW_OIDC_BOOTSTRAP_SUBJECTS")),
		},
		Artifact: ArtifactConfig{
			Backend:        envOrDefault("AREAFLOW_ARTIFACT_BACKEND", "local"),
			LocalRoot:      envOrDefault("AREAFLOW_ARTIFACT_ROOT", ".areaflow/artifacts"),
			S3Endpoint:     strings.TrimSpace(os.Getenv("AREAFLOW_S3_ENDPOINT")),
			S3Region:       strings.TrimSpace(os.Getenv("AREAFLOW_S3_REGION")),
			S3Bucket:       strings.TrimSpace(os.Getenv("AREAFLOW_S3_BUCKET")),
			S3UsePathStyle: envBool("AREAFLOW_S3_USE_PATH_STYLE", false),
		},
		Observability: ObservabilityConfig{
			MetricsHost:  envOrDefault("AREAFLOW_METRICS_HOST", "127.0.0.1"),
			MetricsPort:  envOrDefault("AREAFLOW_METRICS_PORT", "9090"),
			OTLPEndpoint: strings.TrimSpace(os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")),
			ServiceName:  envOrDefault("OTEL_SERVICE_NAME", "areaflow"),
		},
	}
}

func (cfg AuthConfig) Enabled() bool {
	mode := strings.ToLower(strings.TrimSpace(cfg.Mode))
	return mode == "token" || mode == "oidc"
}

func (cfg AuthConfig) OIDCEnabled() bool {
	return strings.EqualFold(strings.TrimSpace(cfg.Mode), "oidc")
}

func (cfg AuthConfig) Validate() error {
	switch strings.ToLower(strings.TrimSpace(cfg.Mode)) {
	case "", "disabled":
		return nil
	case "token":
		return validateTokenMaxTTL(cfg.TokenMaxTTL)
	case "oidc":
		if cfg.OIDCIssuerURL == "" || cfg.OIDCClientID == "" || cfg.OIDCClientSecretFile == "" || cfg.OIDCRedirectURL == "" || cfg.SessionSecretFile == "" {
			return fmt.Errorf("OIDC auth requires issuer, client id, client secret file, redirect URL and session secret file")
		}
		if cfg.SessionIdleTTL <= 0 || cfg.SessionAbsoluteTTL <= 0 || cfg.SessionIdleTTL > cfg.SessionAbsoluteTTL {
			return fmt.Errorf("OIDC session TTLs must be positive and idle TTL must not exceed absolute TTL")
		}
		return validateTokenMaxTTL(cfg.TokenMaxTTL)
	default:
		return fmt.Errorf("AREAFLOW_AUTH_MODE must be disabled, token or oidc")
	}
}

func validateTokenMaxTTL(value time.Duration) error {
	if value <= 0 || value > 90*24*time.Hour {
		return fmt.Errorf("AREAFLOW_TOKEN_MAX_TTL must be between 1ns and 2160h")
	}
	return nil
}

func (cfg Config) Validate() error {
	if cfg.Environment != "development" && cfg.Environment != "production" {
		return fmt.Errorf("AREAFLOW_ENV must be development or production")
	}
	if err := cfg.Auth.Validate(); err != nil {
		return err
	}
	if err := cfg.Server.Validate(); err != nil {
		return err
	}
	if err := cfg.Database.Validate(); err != nil {
		return err
	}
	if cfg.Environment != "production" {
		return nil
	}
	if !cfg.Auth.OIDCEnabled() {
		return fmt.Errorf("production mode requires OIDC authentication")
	}
	publicURL, err := url.Parse(cfg.Server.PublicBaseURL)
	if err != nil || publicURL.Scheme != "https" || publicURL.Host == "" {
		return fmt.Errorf("production mode requires an https AREAFLOW_PUBLIC_BASE_URL")
	}
	if len(cfg.Server.TrustedProxyCIDRs) == 0 {
		return fmt.Errorf("production mode requires AREAFLOW_TRUSTED_PROXY_CIDRS")
	}
	for _, cidr := range cfg.Server.TrustedProxyCIDRs {
		if _, _, err := net.ParseCIDR(cidr); err != nil {
			return fmt.Errorf("invalid trusted proxy CIDR %q", cidr)
		}
	}
	databaseURL, err := url.Parse(cfg.Database.URL)
	if err != nil || strings.EqualFold(databaseURL.Query().Get("sslmode"), "disable") || databaseURL.Query().Get("sslmode") == "" {
		return fmt.Errorf("production database URL must enable PostgreSQL TLS")
	}
	if strings.ToLower(cfg.Artifact.Backend) != "s3" || cfg.Artifact.S3Region == "" || cfg.Artifact.S3Bucket == "" {
		return fmt.Errorf("production mode requires an S3 artifact backend with region and bucket")
	}
	if cfg.Observability.OTLPEndpoint == "" {
		return fmt.Errorf("production mode requires OTEL_EXPORTER_OTLP_ENDPOINT")
	}
	return nil
}

func (cfg DatabaseConfig) Validate() error {
	if strings.TrimSpace(cfg.URL) == "" {
		return fmt.Errorf("AREAFLOW_DATABASE_URL is required")
	}
	if cfg.MaxConnections <= 0 || cfg.MinConnections < 0 || cfg.MinConnections > cfg.MaxConnections {
		return fmt.Errorf("database connection limits must satisfy 0 <= min <= max")
	}
	if cfg.ConnectTimeout <= 0 || cfg.AcquireTimeout <= 0 || cfg.QueryTimeout <= 0 || cfg.MaxConnectionIdle <= 0 || cfg.MaxConnectionLifetime <= 0 {
		return fmt.Errorf("database timeouts and connection lifetimes must be positive")
	}
	return nil
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
	if (ip == nil || !ip.IsLoopback()) && !cfg.AllowRemote {
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

func envDuration(key string, fallback time.Duration) time.Duration {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parsed, err := time.ParseDuration(value)
	if err != nil {
		return -1
	}
	return parsed
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

func envInt(key string, fallback int) int {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return -1
	}
	return parsed
}

func csvList(value string) []string {
	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}
