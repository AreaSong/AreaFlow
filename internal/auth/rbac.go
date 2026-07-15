package auth

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

const (
	RolePlatformAdmin = "platform_admin"
	RoleProjectAdmin  = "project_admin"
	RoleOperator      = "operator"
	RoleApprover      = "approver"
	RoleAuditor       = "auditor"
	RoleViewer        = "viewer"
)

var roleCapabilities = map[string][]string{
	RolePlatformAdmin: {"*"},
	RoleProjectAdmin:  {"read", "admin", "project.manage", "auth.role.manage", "auth.token.manage", "workflow.approval.record", "run.control", "worker.control"},
	RoleOperator:      {"read", "run.control", "worker.control"},
	RoleApprover:      {"read", "workflow.approval.record"},
	RoleAuditor:       {"read", "audit.read"},
	RoleViewer:        {"read"},
}

type OIDCIdentity struct {
	Issuer      string
	Subject     string
	Email       string
	DisplayName string
	Groups      []string
	Claims      map[string]any
}

type UserRecord struct {
	ID          int64
	ActorID     int64
	Actor       string
	Email       string
	DisplayName string
}

type RoleBinding struct {
	ID         int64
	ProjectID  int64
	ProjectKey string
	UserID     int64
	TeamID     int64
	Role       string
	Status     string
	Reason     string
	ExpiresAt  *time.Time
	CreatedAt  time.Time
}

type GrantRoleOptions struct {
	ProjectKey string
	UserID     int64
	TeamID     int64
	Role       string
	Actor      string
	Reason     string
	ExpiresAt  *time.Time
}

func CapabilitiesForRoles(roles []string) []string {
	set := map[string]struct{}{}
	for _, role := range roles {
		for _, capability := range roleCapabilities[role] {
			set[capability] = struct{}{}
		}
	}
	result := make([]string, 0, len(set))
	for capability := range set {
		result = append(result, capability)
	}
	sort.Strings(result)
	return result
}

func ValidRole(role string) bool {
	_, ok := roleCapabilities[strings.TrimSpace(role)]
	return ok
}

func (s Service) UpsertOIDCIdentity(ctx context.Context, identity OIDCIdentity) (UserRecord, error) {
	identity.Issuer = strings.TrimSpace(identity.Issuer)
	identity.Subject = strings.TrimSpace(identity.Subject)
	identity.Email = strings.TrimSpace(identity.Email)
	identity.DisplayName = strings.TrimSpace(identity.DisplayName)
	if identity.Issuer == "" || identity.Subject == "" {
		return UserRecord{}, fmt.Errorf("OIDC issuer and subject are required")
	}
	if identity.DisplayName == "" {
		identity.DisplayName = identity.Email
	}
	if identity.DisplayName == "" {
		identity.DisplayName = identity.Subject
	}
	claimsJSON, err := json.Marshal(identity.Claims)
	if err != nil {
		return UserRecord{}, fmt.Errorf("marshal OIDC claims: %w", err)
	}
	claimsDigest := sha256.Sum256(claimsJSON)
	externalKey := "oidc:" + sha256Hex(identity.Issuer+"\x00"+identity.Subject)

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return UserRecord{}, fmt.Errorf("begin OIDC identity upsert: %w", err)
	}
	defer tx.Rollback(ctx)

	var user UserRecord
	err = tx.QueryRow(ctx, `
SELECT u.id, COALESCE(u.actor_id, 0), COALESCE(a.display_name, ''), COALESCE(u.email, ''), u.display_name
FROM user_identities i
JOIN users u ON u.id = i.user_id
LEFT JOIN actors a ON a.id = u.actor_id
WHERE i.issuer = $1 AND i.subject = $2 AND i.status = 'active'`, identity.Issuer, identity.Subject).Scan(
		&user.ID, &user.ActorID, &user.Actor, &user.Email, &user.DisplayName,
	)
	if err != nil && err != pgx.ErrNoRows {
		return UserRecord{}, fmt.Errorf("lookup OIDC identity: %w", err)
	}
	if err == pgx.ErrNoRows {
		err = tx.QueryRow(ctx, `
INSERT INTO actors (kind, display_name, external_key)
VALUES ('oidc_user', $1, $2)
ON CONFLICT (external_key) WHERE external_key IS NOT NULL
DO UPDATE SET display_name = EXCLUDED.display_name
RETURNING id`, identity.DisplayName, externalKey).Scan(&user.ActorID)
		if err != nil {
			return UserRecord{}, fmt.Errorf("upsert OIDC actor: %w", err)
		}
		user.Actor = identity.DisplayName
		err = tx.QueryRow(ctx, `
INSERT INTO users (actor_id, email, display_name, status, external_key, metadata)
VALUES ($1, NULLIF($2, ''), $3, 'active', $4, jsonb_build_object('oidc_groups', $5::jsonb))
ON CONFLICT (actor_id) WHERE actor_id IS NOT NULL
DO UPDATE SET email = EXCLUDED.email, display_name = EXCLUDED.display_name, status = 'active', metadata = EXCLUDED.metadata, updated_at = now()
RETURNING id, COALESCE(email, ''), display_name`, user.ActorID, identity.Email, identity.DisplayName, externalKey, mustJSON(identity.Groups)).Scan(
			&user.ID, &user.Email, &user.DisplayName,
		)
		if err != nil {
			return UserRecord{}, fmt.Errorf("upsert OIDC user: %w", err)
		}
	}
	_, err = tx.Exec(ctx, `
INSERT INTO user_identities (user_id, issuer, subject, email_snapshot, claims_hash, status, last_authenticated_at)
VALUES ($1, $2, $3, NULLIF($4, ''), $5, 'active', now())
ON CONFLICT (issuer, subject)
DO UPDATE SET email_snapshot = EXCLUDED.email_snapshot, claims_hash = EXCLUDED.claims_hash,
              status = 'active', last_authenticated_at = now(), updated_at = now()`,
		user.ID, identity.Issuer, identity.Subject, identity.Email, hex.EncodeToString(claimsDigest[:]))
	if err != nil {
		return UserRecord{}, fmt.Errorf("persist OIDC identity: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return UserRecord{}, fmt.Errorf("commit OIDC identity: %w", err)
	}
	return user, nil
}

func (s Service) PrincipalForUser(ctx context.Context, user UserRecord) (Principal, error) {
	rows, err := s.pool.Query(ctx, `
SELECT b.role, COALESCE(p.project_key, '')
FROM project_role_bindings b
LEFT JOIN projects p ON p.id = b.project_id
WHERE b.user_id = $1 AND b.status = 'active'
  AND (b.expires_at IS NULL OR b.expires_at > now())
UNION
SELECT b.role, COALESCE(p.project_key, '')
FROM memberships m
JOIN project_role_bindings b ON b.team_id = m.team_id
LEFT JOIN projects p ON p.id = b.project_id
WHERE m.user_id = $1 AND m.status = 'active' AND b.status = 'active'
  AND (b.expires_at IS NULL OR b.expires_at > now())`, user.ID)
	if err != nil {
		return Principal{}, fmt.Errorf("load user roles: %w", err)
	}
	defer rows.Close()
	roleSet := map[string]struct{}{}
	projectSet := map[string]struct{}{}
	for rows.Next() {
		var role, projectKey string
		if err := rows.Scan(&role, &projectKey); err != nil {
			return Principal{}, fmt.Errorf("scan user role: %w", err)
		}
		roleSet[role] = struct{}{}
		if role == RolePlatformAdmin {
			projectSet["*"] = struct{}{}
		} else if projectKey != "" {
			projectSet[projectKey] = struct{}{}
		}
	}
	roles := sortedKeys(roleSet)
	projects := sortedKeys(projectSet)
	return Principal{
		UserID: user.ID, ActorID: user.ActorID, Actor: user.Actor, AuthMode: "oidc",
		Roles: roles, Projects: projects, Capabilities: CapabilitiesForRoles(roles),
	}, rows.Err()
}

func (s Service) EnsureBootstrapPlatformAdmin(ctx context.Context, user UserRecord, subject string, allowedSubjects []string) error {
	if !contains(normalize(allowedSubjects), strings.TrimSpace(subject)) {
		return nil
	}
	principal, err := s.PrincipalForUser(ctx, user)
	if err != nil {
		return err
	}
	if contains(principal.Roles, RolePlatformAdmin) {
		return nil
	}
	_, err = s.GrantRole(ctx, GrantRoleOptions{
		UserID: user.ID, Role: RolePlatformAdmin, Actor: user.Actor,
		Reason: "OIDC bootstrap subject allowlist",
	})
	return err
}

func (s Service) ListRoleBindings(ctx context.Context, projectKey string) ([]RoleBinding, error) {
	rows, err := s.pool.Query(ctx, `
SELECT b.id, COALESCE(b.project_id, 0), COALESCE(p.project_key, ''), COALESCE(b.user_id, 0),
       COALESCE(b.team_id, 0), b.role, b.status, b.reason, b.expires_at, b.created_at
FROM project_role_bindings b
LEFT JOIN projects p ON p.id = b.project_id
WHERE ($1 = '' AND b.project_id IS NULL) OR p.project_key = $1
ORDER BY b.created_at DESC, b.id DESC`, strings.TrimSpace(projectKey))
	if err != nil {
		return nil, fmt.Errorf("list role bindings: %w", err)
	}
	defer rows.Close()
	bindings := []RoleBinding{}
	for rows.Next() {
		var binding RoleBinding
		if err := rows.Scan(&binding.ID, &binding.ProjectID, &binding.ProjectKey, &binding.UserID, &binding.TeamID,
			&binding.Role, &binding.Status, &binding.Reason, &binding.ExpiresAt, &binding.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan role binding: %w", err)
		}
		bindings = append(bindings, binding)
	}
	return bindings, rows.Err()
}

func (s Service) GrantRole(ctx context.Context, options GrantRoleOptions) (RoleBinding, error) {
	options.ProjectKey = strings.TrimSpace(options.ProjectKey)
	options.Role = strings.TrimSpace(options.Role)
	options.Actor = strings.TrimSpace(options.Actor)
	options.Reason = strings.TrimSpace(options.Reason)
	if !ValidRole(options.Role) || options.Actor == "" || options.Reason == "" {
		return RoleBinding{}, fmt.Errorf("valid role, actor and reason are required")
	}
	if (options.UserID == 0) == (options.TeamID == 0) {
		return RoleBinding{}, fmt.Errorf("exactly one user or team is required")
	}
	if options.Role == RolePlatformAdmin && options.ProjectKey != "" {
		return RoleBinding{}, fmt.Errorf("platform_admin cannot be project scoped")
	}
	if options.Role != RolePlatformAdmin && options.ProjectKey == "" {
		return RoleBinding{}, fmt.Errorf("project role requires project key")
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return RoleBinding{}, err
	}
	defer tx.Rollback(ctx)
	actorID, err := ensureActor(ctx, tx, options.Actor)
	if err != nil {
		return RoleBinding{}, err
	}
	var projectID int64
	if options.ProjectKey != "" {
		if err := tx.QueryRow(ctx, `SELECT id FROM projects WHERE project_key = $1 AND archived_at IS NULL`, options.ProjectKey).Scan(&projectID); err != nil {
			return RoleBinding{}, fmt.Errorf("load role project: %w", err)
		}
	}
	var binding RoleBinding
	err = tx.QueryRow(ctx, `
INSERT INTO project_role_bindings (
    project_id, user_id, team_id, role, status, reason, assigned_by_actor_id, expires_at
)
VALUES (NULLIF($1, 0), NULLIF($2, 0), NULLIF($3, 0), $4, 'active', $5, $6, $7)
RETURNING id, COALESCE(project_id, 0), COALESCE(user_id, 0), COALESCE(team_id, 0), role, status, reason, expires_at, created_at`,
		projectID, options.UserID, options.TeamID, options.Role, options.Reason, actorID, options.ExpiresAt).Scan(
		&binding.ID, &binding.ProjectID, &binding.UserID, &binding.TeamID, &binding.Role,
		&binding.Status, &binding.Reason, &binding.ExpiresAt, &binding.CreatedAt,
	)
	if err != nil {
		return RoleBinding{}, fmt.Errorf("insert role binding: %w", err)
	}
	binding.ProjectKey = options.ProjectKey
	metadata := mustJSON(map[string]any{"binding_id": binding.ID, "role": binding.Role, "user_id": binding.UserID, "team_id": binding.TeamID})
	if _, err := tx.Exec(ctx, `
INSERT INTO audit_events (project_id, actor_id, action, capability, resource_type, resource, decision, reason, metadata)
VALUES (NULLIF($1, 0), $2, 'auth.role.grant', 'auth.role.manage', 'project_role_binding', $3, 'allowed', $4, $5::jsonb)`,
		projectID, actorID, fmt.Sprint(binding.ID), options.Reason, metadata); err != nil {
		return RoleBinding{}, fmt.Errorf("audit role grant: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return RoleBinding{}, err
	}
	return binding, nil
}

func (s Service) RevokeRole(ctx context.Context, bindingID int64, actor, reason string) error {
	actor = strings.TrimSpace(actor)
	reason = strings.TrimSpace(reason)
	if bindingID == 0 || actor == "" || reason == "" {
		return fmt.Errorf("binding id, actor and reason are required")
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
	var projectID int64
	err = tx.QueryRow(ctx, `
UPDATE project_role_bindings
SET status = 'revoked', revoked_at = now(), updated_at = now()
WHERE id = $1 AND status = 'active'
RETURNING COALESCE(project_id, 0)`, bindingID).Scan(&projectID)
	if err != nil {
		return fmt.Errorf("revoke role binding: %w", err)
	}
	if _, err := tx.Exec(ctx, `
INSERT INTO audit_events (project_id, actor_id, action, capability, resource_type, resource, decision, reason, metadata)
VALUES (NULLIF($1, 0), $2, 'auth.role.revoke', 'auth.role.manage', 'project_role_binding', $3, 'allowed', $4, '{}'::jsonb)`,
		projectID, actorID, fmt.Sprint(bindingID), reason); err != nil {
		return fmt.Errorf("audit role revoke: %w", err)
	}
	return tx.Commit(ctx)
}

func sha256Hex(value string) string {
	digest := sha256.Sum256([]byte(value))
	return hex.EncodeToString(digest[:])
}

func mustJSON(value any) string {
	encoded, _ := json.Marshal(value)
	return string(encoded)
}

func sortedKeys(values map[string]struct{}) []string {
	result := make([]string, 0, len(values))
	for value := range values {
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}
