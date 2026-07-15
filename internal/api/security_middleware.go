package api

import (
	"bytes"
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

const maxRequestBodyBytes = 1 << 20

type rateWindow struct {
	started time.Time
	count   int
}

type ipRateLimiter struct {
	mu         sync.Mutex
	limit      int
	window     time.Duration
	entries    map[string]rateWindow
	maxEntries int
}

func newIPRateLimiter(limit int, window time.Duration) *ipRateLimiter {
	return &ipRateLimiter{limit: limit, window: window, entries: map[string]rateWindow{}, maxEntries: 10000}
}

func (l *ipRateLimiter) Allow(key string, now time.Time) (bool, time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()
	entry := l.entries[key]
	if entry.started.IsZero() || now.Sub(entry.started) >= l.window {
		if entry.started.IsZero() && len(l.entries) >= l.maxEntries {
			for candidate, candidateEntry := range l.entries {
				if now.Sub(candidateEntry.started) >= l.window {
					delete(l.entries, candidate)
				}
			}
			if len(l.entries) >= l.maxEntries {
				return false, l.window
			}
		}
		l.entries[key] = rateWindow{started: now, count: 1}
		return true, 0
	}
	if entry.count >= l.limit {
		return false, l.window - now.Sub(entry.started)
	}
	entry.count++
	l.entries[key] = entry
	return true, 0
}

func (s Server) securityMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if hasForwardedHeaders(r) && !remoteAddressAllowed(r.RemoteAddr, s.serverConfig.TrustedProxyCIDRs) {
			writeError(w, http.StatusBadRequest, "forwarded headers require a trusted proxy")
			return
		}
		if r.Method != http.MethodGet && r.Method != http.MethodHead && r.Method != http.MethodOptions {
			content, err := io.ReadAll(io.LimitReader(r.Body, maxRequestBodyBytes+1))
			if err != nil {
				writeError(w, http.StatusBadRequest, "read request body failed")
				return
			}
			if len(content) > maxRequestBodyBytes {
				writeError(w, http.StatusRequestEntityTooLarge, "request body exceeds 1 MiB limit")
				return
			}
			r.Body = io.NopCloser(bytes.NewReader(content))
		}
		if isOIDCRateLimitedPath(r.URL.Path) && s.loginLimiter != nil {
			allowed, retryAfter := s.loginLimiter.Allow(clientIP(r, s.serverConfig.TrustedProxyCIDRs), time.Now().UTC())
			if !allowed {
				w.Header().Set("Retry-After", strconv.Itoa(max(1, int(retryAfter.Seconds()))))
				writeError(w, http.StatusTooManyRequests, "authentication rate limit exceeded")
				return
			}
			w.Header().Set("Cache-Control", "no-store")
		}
		next.ServeHTTP(w, r)
	})
}

func isOIDCRateLimitedPath(path string) bool {
	return path == "/api/auth/oidc/login" || path == "/api/auth/oidc/callback"
}

func hasForwardedHeaders(r *http.Request) bool {
	return strings.TrimSpace(r.Header.Get("Forwarded")) != "" ||
		strings.TrimSpace(r.Header.Get("X-Forwarded-For")) != "" ||
		strings.TrimSpace(r.Header.Get("X-Forwarded-Host")) != "" ||
		strings.TrimSpace(r.Header.Get("X-Forwarded-Proto")) != ""
}

func remoteAddressAllowed(remoteAddress string, trustedCIDRs []string) bool {
	ip := remoteIP(remoteAddress)
	if ip == nil {
		return false
	}
	for _, value := range trustedCIDRs {
		_, network, err := net.ParseCIDR(value)
		if err == nil && network.Contains(ip) {
			return true
		}
	}
	return false
}

func clientIP(r *http.Request, trustedCIDRs []string) string {
	remote := remoteIP(r.RemoteAddr)
	if remote != nil && remoteAddressAllowed(r.RemoteAddr, trustedCIDRs) {
		for _, value := range strings.Split(r.Header.Get("X-Forwarded-For"), ",") {
			if parsed := net.ParseIP(strings.TrimSpace(value)); parsed != nil {
				return parsed.String()
			}
		}
	}
	if remote == nil {
		return "unknown"
	}
	return remote.String()
}

func remoteIP(remoteAddress string) net.IP {
	host, _, err := net.SplitHostPort(strings.TrimSpace(remoteAddress))
	if err != nil {
		host = strings.TrimSpace(remoteAddress)
	}
	return net.ParseIP(host)
}
