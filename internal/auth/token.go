package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrInvalidToken = errors.New("invalid API token")

type Principal struct {
	TokenID      int64
	TokenKey     string
	ActorID      int64
	Actor        string
	Projects     []string
	Capabilities []string
	ScopeHash    string
}

func (p Principal) AllowsProject(projectKey string) bool {
	return contains(p.Projects, "*") || contains(p.Projects, projectKey)
}

func (p Principal) AllowsCapability(capability string) bool {
	return contains(p.Capabilities, "*") || contains(p.Capabilities, capability)
}

type TokenRecord struct {
	ID           int64
	TokenKey     string
	Actor        string
	Projects     []string
	Capabilities []string
	Status       string
	ExpiresAt    *time.Time
	CreatedAt    time.Time
	RevokedAt    *time.Time
}

type CreateTokenOptions struct {
	Actor        string
	Reason       string
	Projects     []string
	Capabilities []string
	ExpiresAt    *time.Time
}

type CreatedToken struct {
	Token  string
	Record TokenRecord
}

type Service struct {
	pool *pgxpool.Pool
}

func NewService(pool *pgxpool.Pool) Service {
	return Service{pool: pool}
}

func (s Service) CreateToken(ctx context.Context, options CreateTokenOptions) (CreatedToken, error) {
	options.Actor = strings.TrimSpace(options.Actor)
	options.Reason = strings.TrimSpace(options.Reason)
	options.Projects = normalize(options.Projects)
	options.Capabilities = normalize(options.Capabilities)
	if options.Actor == "" || options.Reason == "" {
		return CreatedToken{}, fmt.Errorf("actor and reason are required")
	}
	if len(options.Projects) == 0 || len(options.Capabilities) == 0 {
		return CreatedToken{}, fmt.Errorf("at least one project and capability are required")
	}
	tokenKey, secret, err := generateTokenParts()
	if err != nil {
		return CreatedToken{}, err
	}
	rawToken := "af_" + tokenKey + "_" + secret
	scope := tokenScope{Projects: options.Projects, Capabilities: options.Capabilities}
	scopeJSON, err := json.Marshal(scope)
	if err != nil {
		return CreatedToken{}, fmt.Errorf("marshal token scope: %w", err)
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return CreatedToken{}, fmt.Errorf("begin token creation: %w", err)
	}
	defer tx.Rollback(ctx)
	actorID, err := ensureActor(ctx, tx, options.Actor)
	if err != nil {
		return CreatedToken{}, err
	}
	record := TokenRecord{TokenKey: tokenKey, Actor: options.Actor, Projects: options.Projects, Capabilities: options.Capabilities, Status: "active", ExpiresAt: options.ExpiresAt}
	err = tx.QueryRow(ctx, `
INSERT INTO api_tokens (actor_id, token_key, token_hash, scope, status, expires_at)
VALUES ($1, $2, $3, $4::jsonb, 'active', $5)
RETURNING id, created_at`, actorID, tokenKey, tokenHash(rawToken), string(scopeJSON), options.ExpiresAt).Scan(&record.ID, &record.CreatedAt)
	if err != nil {
		return CreatedToken{}, fmt.Errorf("insert API token: %w", err)
	}
	auditMetadata, _ := json.Marshal(map[string]any{
		"token_key": tokenKey, "scope_hash": scopeHash(scopeJSON), "projects": options.Projects, "capabilities": options.Capabilities,
	})
	if _, err := tx.Exec(ctx, `
INSERT INTO audit_events (actor_id, action, capability, resource_type, resource, decision, reason, metadata)
VALUES ($1, 'auth.token.create', 'manage_tokens', 'api_token', $2, 'allowed', $3, $4::jsonb)`, actorID, tokenKey, options.Reason, string(auditMetadata)); err != nil {
		return CreatedToken{}, fmt.Errorf("audit API token creation: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return CreatedToken{}, fmt.Errorf("commit token creation: %w", err)
	}
	return CreatedToken{Token: rawToken, Record: record}, nil
}

func (s Service) Authenticate(ctx context.Context, rawToken string) (Principal, error) {
	tokenKey, err := tokenKeyFromRaw(rawToken)
	if err != nil {
		return Principal{}, ErrInvalidToken
	}
	var principal Principal
	var storedHash string
	var scopeRaw []byte
	var status string
	var expiresAt *time.Time
	err = s.pool.QueryRow(ctx, `
SELECT t.id, t.token_key, COALESCE(t.actor_id, 0), COALESCE(a.display_name, 'token-user'),
       t.token_hash, t.scope, t.status, t.expires_at
FROM api_tokens t
LEFT JOIN actors a ON a.id = t.actor_id
WHERE t.token_key = $1`, tokenKey).Scan(
		&principal.TokenID, &principal.TokenKey, &principal.ActorID, &principal.Actor,
		&storedHash, &scopeRaw, &status, &expiresAt,
	)
	if err != nil {
		return Principal{}, ErrInvalidToken
	}
	providedHash := tokenHash(rawToken)
	if subtle.ConstantTimeCompare([]byte(storedHash), []byte(providedHash)) != 1 || status != "active" || (expiresAt != nil && !expiresAt.After(time.Now().UTC())) {
		return Principal{}, ErrInvalidToken
	}
	var scope tokenScope
	if err := json.Unmarshal(scopeRaw, &scope); err != nil {
		return Principal{}, ErrInvalidToken
	}
	principal.Projects = normalize(scope.Projects)
	principal.Capabilities = normalize(scope.Capabilities)
	principal.ScopeHash = scopeHash(scopeRaw)
	if len(principal.Projects) == 0 || len(principal.Capabilities) == 0 {
		return Principal{}, ErrInvalidToken
	}
	return principal, nil
}

func (s Service) ListTokens(ctx context.Context) ([]TokenRecord, error) {
	rows, err := s.pool.Query(ctx, `
SELECT t.id, t.token_key, COALESCE(a.display_name, 'token-user'), t.scope, t.status,
       t.expires_at, t.created_at, t.revoked_at
FROM api_tokens t
LEFT JOIN actors a ON a.id = t.actor_id
ORDER BY t.created_at DESC, t.id DESC`)
	if err != nil {
		return nil, fmt.Errorf("list API tokens: %w", err)
	}
	defer rows.Close()
	records := []TokenRecord{}
	for rows.Next() {
		var record TokenRecord
		var scopeRaw []byte
		if err := rows.Scan(&record.ID, &record.TokenKey, &record.Actor, &scopeRaw, &record.Status, &record.ExpiresAt, &record.CreatedAt, &record.RevokedAt); err != nil {
			return nil, fmt.Errorf("scan API token: %w", err)
		}
		var scope tokenScope
		if err := json.Unmarshal(scopeRaw, &scope); err != nil {
			return nil, fmt.Errorf("decode API token scope: %w", err)
		}
		record.Projects = normalize(scope.Projects)
		record.Capabilities = normalize(scope.Capabilities)
		records = append(records, record)
	}
	return records, rows.Err()
}

func (s Service) RevokeToken(ctx context.Context, tokenKey string, actor string, reason string) error {
	tokenKey = strings.TrimSpace(tokenKey)
	actor = strings.TrimSpace(actor)
	reason = strings.TrimSpace(reason)
	if tokenKey == "" || actor == "" || reason == "" {
		return fmt.Errorf("token key, actor and reason are required")
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin token revocation: %w", err)
	}
	defer tx.Rollback(ctx)
	actorID, err := ensureActor(ctx, tx, actor)
	if err != nil {
		return err
	}
	result, err := tx.Exec(ctx, `UPDATE api_tokens SET status = 'revoked', revoked_at = now() WHERE token_key = $1 AND status = 'active'`, tokenKey)
	if err != nil {
		return fmt.Errorf("revoke API token: %w", err)
	}
	if result.RowsAffected() != 1 {
		return fmt.Errorf("active API token not found: %s", tokenKey)
	}
	if _, err := tx.Exec(ctx, `
INSERT INTO audit_events (actor_id, action, capability, resource_type, resource, decision, reason, metadata)
VALUES ($1, 'auth.token.revoke', 'manage_tokens', 'api_token', $2, 'allowed', $3, '{}'::jsonb)`, actorID, tokenKey, reason); err != nil {
		return fmt.Errorf("audit API token revocation: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit token revocation: %w", err)
	}
	return nil
}

type tokenScope struct {
	Projects     []string `json:"projects"`
	Capabilities []string `json:"capabilities"`
}

func ensureActor(ctx context.Context, tx pgx.Tx, actor string) (int64, error) {
	externalKey := "auth:" + actor
	var actorID int64
	err := tx.QueryRow(ctx, `
INSERT INTO actors (kind, display_name, external_key)
VALUES ('user', $1, $2)
ON CONFLICT (external_key) WHERE external_key IS NOT NULL DO UPDATE SET display_name = EXCLUDED.display_name
RETURNING id`, actor, externalKey).Scan(&actorID)
	if err != nil {
		return 0, fmt.Errorf("ensure auth actor: %w", err)
	}
	return actorID, nil
}

func generateTokenParts() (string, string, error) {
	keyBytes := make([]byte, 12)
	secretBytes := make([]byte, 32)
	if _, err := rand.Read(keyBytes); err != nil {
		return "", "", fmt.Errorf("generate token key: %w", err)
	}
	if _, err := rand.Read(secretBytes); err != nil {
		return "", "", fmt.Errorf("generate token secret: %w", err)
	}
	return hex.EncodeToString(keyBytes), base64.RawURLEncoding.EncodeToString(secretBytes), nil
}

func tokenKeyFromRaw(raw string) (string, error) {
	parts := strings.SplitN(raw, "_", 3)
	if len(parts) != 3 || parts[0] != "af" || len(parts[1]) != 24 || len(parts[2]) < 40 {
		return "", ErrInvalidToken
	}
	if _, err := hex.DecodeString(parts[1]); err != nil {
		return "", ErrInvalidToken
	}
	secret, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil || len(secret) != 32 {
		return "", ErrInvalidToken
	}
	return parts[1], nil
}

func tokenHash(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

func scopeHash(scope []byte) string {
	sum := sha256.Sum256(scope)
	return hex.EncodeToString(sum[:])
}

func normalize(values []string) []string {
	seen := map[string]bool{}
	result := []string{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}

func contains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
