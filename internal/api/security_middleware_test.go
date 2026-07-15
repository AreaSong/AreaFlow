package api

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/areasong/areaflow/internal/config"
)

func TestSecurityMiddlewareLimitsOIDCRequests(t *testing.T) {
	server := Server{loginLimiter: newIPRateLimiter(1, time.Minute)}
	handler := server.securityMiddleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	first := httptest.NewRecorder()
	handler.ServeHTTP(first, httptest.NewRequest(http.MethodGet, "/api/auth/oidc/login", nil))
	second := httptest.NewRecorder()
	handler.ServeHTTP(second, httptest.NewRequest(http.MethodGet, "/api/auth/oidc/login", nil))
	if first.Code != http.StatusNoContent || second.Code != http.StatusTooManyRequests || second.Header().Get("Retry-After") == "" {
		t.Fatalf("rate limit responses = %d/%d headers=%v", first.Code, second.Code, second.Header())
	}
}

func TestIPRateLimiterBoundsHighCardinalityState(t *testing.T) {
	limiter := newIPRateLimiter(1, time.Minute)
	limiter.maxEntries = 1
	now := time.Now().UTC()
	if allowed, _ := limiter.Allow("first", now); !allowed {
		t.Fatal("first key rejected")
	}
	if allowed, _ := limiter.Allow("second", now); allowed {
		t.Fatal("new key must be rejected when limiter state is full")
	}
	if allowed, _ := limiter.Allow("second", now.Add(2*time.Minute)); !allowed {
		t.Fatal("expired limiter entry was not reclaimed")
	}
}

func TestSecurityMiddlewareUsesTrustedForwardedClientIP(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/api/auth/oidc/login", nil)
	request.RemoteAddr = "10.0.0.5:1234"
	request.Header.Set("X-Forwarded-For", "203.0.113.9, 10.0.0.5")
	if got := clientIP(request, []string{"10.0.0.0/8"}); got != "203.0.113.9" {
		t.Fatalf("clientIP = %q", got)
	}
}

func TestSecurityMiddlewareCapsRequestBody(t *testing.T) {
	server := Server{}
	called := false
	handler := server.securityMiddleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		called = true
		w.WriteHeader(http.StatusNoContent)
	}))
	request := httptest.NewRequest(http.MethodPost, "/api/projects", strings.NewReader(strings.Repeat("x", maxRequestBodyBytes+1)))
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusRequestEntityTooLarge || called || response.Header().Get("Content-Type") != "application/problem+json" {
		t.Fatalf("status = %d called=%t headers=%v body=%s", response.Code, called, response.Header(), response.Body.String())
	}
}

func TestSecurityMiddlewareRestoresAcceptedRequestBody(t *testing.T) {
	server := Server{}
	handler := server.securityMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		content, err := io.ReadAll(r.Body)
		if err != nil || string(content) != `{"ok":true}` {
			t.Fatalf("restored body = %q err=%v", content, err)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/api/projects", strings.NewReader(`{"ok":true}`)))
	if response.Code != http.StatusNoContent {
		t.Fatalf("status = %d", response.Code)
	}
}

func TestDeprecatedAPIAliasReturnsSuccessorHeaders(t *testing.T) {
	handler := apiVersionAlias(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/health", nil))
	if response.Header().Get("Deprecation") != "true" || !strings.Contains(response.Header().Get("Link"), "/api/v1") {
		t.Fatalf("missing API deprecation headers: %v", response.Header())
	}
}

func TestUntrustedForwardedHeadersFailClosed(t *testing.T) {
	server := Server{serverConfig: config.ServerConfig{TrustedProxyCIDRs: []string{"10.0.0.0/8"}}}
	handler := server.securityMiddleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	request := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	request.RemoteAddr = "192.0.2.5:1234"
	request.Header.Set("X-Forwarded-Proto", "https")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", response.Code)
	}
}
