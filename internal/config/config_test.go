package config

import (
	"testing"
	"time"
)

func TestServerConfigValidate(t *testing.T) {
	for _, test := range []struct {
		host    string
		wantErr bool
	}{
		{host: "127.0.0.1"},
		{host: "::1"},
		{host: "localhost"},
		{host: "0.0.0.0", wantErr: true},
		{host: "192.168.1.10", wantErr: true},
		{host: "example.com", wantErr: true},
		{host: "0.0.0.0", wantErr: false},
	} {
		t.Run(test.host, func(t *testing.T) {
			allowRemote := test.host == "0.0.0.0" && !test.wantErr
			err := (ServerConfig{Host: test.host, Port: "3847", AllowRemote: allowRemote}).Validate()
			if (err != nil) != test.wantErr {
				t.Fatalf("Validate() error = %v, wantErr %v", err, test.wantErr)
			}
		})
	}
}

func TestServerConfigAddr(t *testing.T) {
	for _, test := range []struct {
		host string
		want string
	}{
		{host: "127.0.0.1", want: "127.0.0.1:3847"},
		{host: "::1", want: "[::1]:3847"},
		{host: "localhost", want: "localhost:3847"},
	} {
		t.Run(test.host, func(t *testing.T) {
			if got := (ServerConfig{Host: test.host, Port: "3847"}).Addr(); got != test.want {
				t.Fatalf("Addr() = %q, want %q", got, test.want)
			}
		})
	}
}

func TestAuthConfig(t *testing.T) {
	if (AuthConfig{Mode: "disabled"}).Enabled() {
		t.Fatal("disabled auth must not be enabled")
	}
	if !(AuthConfig{Mode: "token"}).Enabled() {
		t.Fatal("token auth must be enabled")
	}
	if !(AuthConfig{Mode: "oidc"}).OIDCEnabled() {
		t.Fatal("oidc auth must be enabled")
	}
	if err := (AuthConfig{Mode: "unknown"}).Validate(); err == nil {
		t.Fatal("unknown auth mode must fail validation")
	}
}

func TestProductionConfigFailClosed(t *testing.T) {
	cfg := Config{
		Environment: "production",
		Server: ServerConfig{
			Host: "0.0.0.0", Port: "3847", AllowRemote: true,
			PublicBaseURL: "https://areaflow.example", TrustedProxyCIDRs: []string{"10.0.0.0/8"},
		},
		Database: DatabaseConfig{
			URL: "postgres://db/areaflow?sslmode=verify-full", MaxConnections: 30, MinConnections: 5,
			ConnectTimeout: 5 * time.Second, AcquireTimeout: 5 * time.Second, QueryTimeout: 10 * time.Second,
			MaxConnectionIdle: 5 * time.Minute, MaxConnectionLifetime: 30 * time.Minute,
		},
		Auth: AuthConfig{
			Mode: "oidc", OIDCIssuerURL: "https://issuer.example", OIDCClientID: "areaflow",
			OIDCClientSecretFile: "/run/secrets/oidc", OIDCRedirectURL: "https://areaflow.example/api/v1/auth/oidc/callback",
			SessionSecretFile: "/run/secrets/session", SessionIdleTTL: 8 * time.Hour,
			SessionAbsoluteTTL: 24 * time.Hour, TokenMaxTTL: 90 * 24 * time.Hour,
		},
		Artifact:      ArtifactConfig{Backend: "s3", S3Region: "us-east-1", S3Bucket: "areaflow"},
		Observability: ObservabilityConfig{OTLPEndpoint: "https://otel.example"},
	}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("valid production config rejected: %v", err)
	}
	cfg.Auth.Mode = "disabled"
	if err := cfg.Validate(); err == nil {
		t.Fatal("production without OIDC must be rejected")
	}
}
