package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/areasong/areaflow/internal/auth"
	"github.com/areasong/areaflow/internal/config"
	"github.com/areasong/areaflow/internal/migrate"
	"github.com/areasong/areaflow/internal/project"
	"github.com/jackc/pgx/v5/pgxpool"
)

type smokeOIDCProvider struct{}

func (smokeOIDCProvider) Begin(string) (auth.OIDCLogin, error) {
	return auth.OIDCLogin{}, nil
}

func (smokeOIDCProvider) Exchange(context.Context, string, string, string) (auth.OIDCIdentity, string, error) {
	return auth.OIDCIdentity{
		Issuer: "https://issuer.example", Subject: "smoke-admin", Email: "admin@example.com",
		DisplayName: "Smoke Admin", Groups: []string{"areaflow-admins"}, Claims: map[string]any{"sub": "smoke-admin"},
	}, "/", nil
}

func TestOIDCSessionAndRBACPostgresSmoke(t *testing.T) {
	if os.Getenv("AREAFLOW_AUTH_DB_SMOKE") != "1" {
		t.Skip("set AREAFLOW_AUTH_DB_SMOKE=1 to run the PostgreSQL auth smoke")
	}
	databaseURL := os.Getenv("AREAFLOW_DATABASE_URL")
	pool, err := pgxpool.New(context.Background(), databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	if _, err := migrate.Up(context.Background(), pool); err != nil {
		t.Fatal(err)
	}
	projectKey := "auth-smoke-" + time.Now().UTC().Format("20060102150405.000000000")
	var projectID int64
	if err := pool.QueryRow(context.Background(), `
INSERT INTO projects (project_key, name, kind, adapter, workflow_profile)
VALUES ($1, 'Auth smoke', 'fixture', 'fixture', 'fixture') RETURNING id`, projectKey).Scan(&projectID); err != nil {
		t.Fatal(err)
	}

	service := auth.NewService(pool)
	authConfig := config.AuthConfig{
		Mode: "oidc", SessionCookieName: "areaflow_session", SessionIdleTTL: time.Hour,
		SessionAbsoluteTTL: 8 * time.Hour, BootstrapSubjects: []string{"smoke-admin"},
	}
	handler := NewHandlerWithProductionAuth(
		fakeProjectStore{record: project.Record{ID: projectID, Key: projectKey}}, nil,
		config.ServerConfig{}, authConfig, service, smokeOIDCProvider{},
	)

	callback := httptest.NewRequest(http.MethodGet, "/api/v1/auth/oidc/callback?code=code&state=state", nil)
	callback.AddCookie(&http.Cookie{Name: oidcStateCookieName, Value: "state-cookie"})
	callbackResponse := httptest.NewRecorder()
	handler.ServeHTTP(callbackResponse, callback)
	if callbackResponse.Code != http.StatusFound {
		t.Fatalf("callback status = %d body=%s", callbackResponse.Code, callbackResponse.Body.String())
	}

	var sessionCookie, csrfCookie *http.Cookie
	for _, cookie := range callbackResponse.Result().Cookies() {
		switch cookie.Name {
		case authConfig.SessionCookieName:
			sessionCookie = cookie
		case csrfCookieName:
			csrfCookie = cookie
		}
	}
	if sessionCookie == nil || csrfCookie == nil {
		t.Fatalf("callback did not issue session and CSRF cookies: %v", callbackResponse.Result().Cookies())
	}

	me := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	me.AddCookie(sessionCookie)
	meResponse := httptest.NewRecorder()
	handler.ServeHTTP(meResponse, me)
	if meResponse.Code != http.StatusOK {
		t.Fatalf("me status = %d body=%s", meResponse.Code, meResponse.Body.String())
	}
	var principal struct {
		UserID int64    `json:"user_id"`
		Roles  []string `json:"roles"`
	}
	if err := json.NewDecoder(meResponse.Body).Decode(&principal); err != nil {
		t.Fatal(err)
	}
	if principal.UserID == 0 || !containsString(principal.Roles, auth.RolePlatformAdmin) {
		t.Fatalf("unexpected principal: %+v", principal)
	}

	grant := httptest.NewRequest(http.MethodPost, "/api/v1/projects/"+projectKey+"/role-bindings", strings.NewReader(
		`{"user_id":`+jsonNumber(principal.UserID)+`,"team_id":0,"role":"viewer","reason":"auth smoke"}`,
	))
	grant.AddCookie(sessionCookie)
	grant.AddCookie(csrfCookie)
	grant.Header.Set("X-CSRF-Token", csrfCookie.Value)
	grantResponse := httptest.NewRecorder()
	handler.ServeHTTP(grantResponse, grant)
	if grantResponse.Code != http.StatusCreated {
		t.Fatalf("grant status = %d body=%s", grantResponse.Code, grantResponse.Body.String())
	}
	var binding map[string]any
	if err := json.NewDecoder(grantResponse.Body).Decode(&binding); err != nil {
		t.Fatal(err)
	}
	if binding["id"] == nil || binding["role"] != "viewer" || binding["ID"] != nil {
		t.Fatalf("role binding contract is not snake_case: %+v", binding)
	}

	createdToken, err := service.WithTokenMaxTTL(24*time.Hour).CreateToken(context.Background(), auth.CreateTokenOptions{
		Actor: "smoke-service", CreatedBy: "Smoke Admin", Reason: "auth smoke token",
		Projects: []string{projectKey}, Capabilities: []string{"read"}, ExpiresAt: timePointer(time.Now().UTC().Add(time.Hour)),
	})
	if err != nil || createdToken.Token == "" {
		t.Fatalf("create scoped token: %+v %v", createdToken, err)
	}
	var auditActor string
	if err := pool.QueryRow(context.Background(), `
SELECT a.display_name
FROM audit_events e JOIN actors a ON a.id = e.actor_id
WHERE e.action = 'auth.token.create' AND e.resource = $1`, createdToken.Record.TokenKey).Scan(&auditActor); err != nil {
		t.Fatal(err)
	}
	if auditActor != "Smoke Admin" {
		t.Fatalf("token creation audit actor = %q, want creator", auditActor)
	}
	if _, err := service.WithTokenMaxTTL(time.Hour).CreateToken(context.Background(), auth.CreateTokenOptions{
		Actor: "smoke-service", CreatedBy: "Smoke Admin", Reason: "too long",
		Projects: []string{projectKey}, Capabilities: []string{"read"}, ExpiresAt: timePointer(time.Now().UTC().Add(2 * time.Hour)),
	}); err == nil {
		t.Fatal("configured token max TTL must be enforced")
	}
}

func jsonNumber(value int64) string {
	return strconv.FormatInt(value, 10)
}

func timePointer(value time.Time) *time.Time {
	return &value
}
