package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"
)

var ErrInvalidSession = errors.New("invalid web session")

type CreatedSession struct {
	RawSession string
	RawCSRF    string
	Principal  Principal
	ExpiresAt  time.Time
}

type CreateSessionOptions struct {
	User        UserRecord
	Issuer      string
	AuthTime    time.Time
	IdleTTL     time.Duration
	AbsoluteTTL time.Duration
}

func (s Service) CreateSession(ctx context.Context, options CreateSessionOptions) (CreatedSession, error) {
	if options.User.ID == 0 || options.Issuer == "" || options.IdleTTL <= 0 || options.AbsoluteTTL <= 0 || options.IdleTTL > options.AbsoluteTTL {
		return CreatedSession{}, fmt.Errorf("valid user, issuer and session TTLs are required")
	}
	key, secret, err := generateOpaqueParts()
	if err != nil {
		return CreatedSession{}, err
	}
	csrf, err := randomURLToken(32)
	if err != nil {
		return CreatedSession{}, err
	}
	rawSession := "afs_" + key + "_" + secret
	now := time.Now().UTC()
	if options.AuthTime.IsZero() {
		options.AuthTime = now
	}
	idleExpiresAt := now.Add(options.IdleTTL)
	absoluteExpiresAt := now.Add(options.AbsoluteTTL)
	var sessionID int64
	err = s.pool.QueryRow(ctx, `
INSERT INTO web_sessions (
    user_id, actor_id, session_key, session_hash, csrf_hash, issuer, auth_time,
    idle_ttl_seconds, idle_expires_at, absolute_expires_at, last_seen_at, status
)
VALUES ($1, NULLIF($2, 0), $3, $4, $5, $6, $7, $8, $9, $10, $11, 'active')
RETURNING id`, options.User.ID, options.User.ActorID, key, opaqueHash(rawSession), opaqueHash(csrf),
		options.Issuer, options.AuthTime, int64(options.IdleTTL/time.Second), idleExpiresAt, absoluteExpiresAt, now).Scan(&sessionID)
	if err != nil {
		return CreatedSession{}, fmt.Errorf("create web session: %w", err)
	}
	principal, err := s.PrincipalForUser(ctx, options.User)
	if err != nil {
		return CreatedSession{}, err
	}
	principal.SessionID = sessionID
	principal.CSRFHash = opaqueHash(csrf)
	return CreatedSession{RawSession: rawSession, RawCSRF: csrf, Principal: principal, ExpiresAt: absoluteExpiresAt}, nil
}

func (s Service) AuthenticateSession(ctx context.Context, rawSession string) (Principal, error) {
	key, err := opaqueKey(rawSession, "afs")
	if err != nil {
		return Principal{}, ErrInvalidSession
	}
	var user UserRecord
	var sessionID int64
	var idleTTLSeconds int64
	var storedHash, csrfHash, status string
	var idleExpiresAt, absoluteExpiresAt time.Time
	err = s.pool.QueryRow(ctx, `
SELECT ws.id, ws.session_hash, ws.csrf_hash, ws.status, ws.idle_ttl_seconds, ws.idle_expires_at, ws.absolute_expires_at,
       u.id, COALESCE(u.actor_id, 0), COALESCE(a.display_name, u.display_name), COALESCE(u.email, ''), u.display_name
FROM web_sessions ws
JOIN users u ON u.id = ws.user_id
LEFT JOIN actors a ON a.id = u.actor_id
WHERE ws.session_key = $1`, key).Scan(
		&sessionID, &storedHash, &csrfHash, &status, &idleTTLSeconds, &idleExpiresAt, &absoluteExpiresAt,
		&user.ID, &user.ActorID, &user.Actor, &user.Email, &user.DisplayName,
	)
	if err != nil || status != "active" || time.Now().UTC().After(idleExpiresAt) || time.Now().UTC().After(absoluteExpiresAt) ||
		subtle.ConstantTimeCompare([]byte(storedHash), []byte(opaqueHash(rawSession))) != 1 {
		return Principal{}, ErrInvalidSession
	}
	principal, err := s.PrincipalForUser(ctx, user)
	if err != nil {
		return Principal{}, err
	}
	principal.SessionID = sessionID
	principal.CSRFHash = csrfHash
	_, _ = s.pool.Exec(ctx, `
UPDATE web_sessions
SET last_seen_at = now(),
    idle_expires_at = LEAST(absolute_expires_at, now() + make_interval(secs => idle_ttl_seconds::double precision))
WHERE id = $1`, sessionID)
	return principal, nil
}

func (s Service) ValidateCSRF(principal Principal, rawCSRF string) bool {
	if principal.SessionID == 0 || principal.CSRFHash == "" || rawCSRF == "" {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(principal.CSRFHash), []byte(opaqueHash(rawCSRF))) == 1
}

func (s Service) RevokeSession(ctx context.Context, sessionID int64, actor, reason string) error {
	if sessionID == 0 || strings.TrimSpace(actor) == "" || strings.TrimSpace(reason) == "" {
		return fmt.Errorf("session, actor and reason are required")
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	actorID, err := ensureActor(ctx, tx, actor)
	if err != nil {
		return err
	}
	result, err := tx.Exec(ctx, `UPDATE web_sessions SET status = 'revoked', revoked_at = now() WHERE id = $1 AND status = 'active'`, sessionID)
	if err != nil {
		return fmt.Errorf("revoke session: %w", err)
	}
	if result.RowsAffected() != 1 {
		return ErrInvalidSession
	}
	if _, err := tx.Exec(ctx, `
INSERT INTO audit_events (actor_id, action, capability, resource_type, resource, decision, reason, metadata)
VALUES ($1, 'auth.session.revoke', 'auth.session.revoke', 'web_session', $2, 'allowed', $3, '{}'::jsonb)`,
		actorID, fmt.Sprint(sessionID), reason); err != nil {
		return fmt.Errorf("audit session revocation: %w", err)
	}
	return tx.Commit(ctx)
}

func generateOpaqueParts() (string, string, error) {
	keyBytes := make([]byte, 12)
	if _, err := rand.Read(keyBytes); err != nil {
		return "", "", err
	}
	secret, err := randomURLToken(32)
	return hex.EncodeToString(keyBytes), secret, err
}

func randomURLToken(size int) (string, error) {
	value := make([]byte, size)
	if _, err := rand.Read(value); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(value), nil
}

func opaqueKey(raw, prefix string) (string, error) {
	parts := strings.SplitN(raw, "_", 3)
	if len(parts) != 3 || parts[0] != prefix || len(parts[1]) != 24 || len(parts[2]) < 40 {
		return "", ErrInvalidSession
	}
	if _, err := hex.DecodeString(parts[1]); err != nil {
		return "", ErrInvalidSession
	}
	if _, err := base64.RawURLEncoding.DecodeString(parts[2]); err != nil {
		return "", ErrInvalidSession
	}
	return parts[1], nil
}

func opaqueHash(raw string) string {
	digest := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(digest[:])
}
