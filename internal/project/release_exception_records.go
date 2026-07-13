package project

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/areasong/areaflow/internal/migrate"
	"github.com/jackc/pgx/v5"
)

const (
	releaseExceptionRequestCommandType = "release.exception.request"
	releaseExceptionApproveCommandType = "release.exception.approve"
	releaseExceptionRevokeCommandType  = "release.exception.revoke"
)

type ReleaseExceptionRecord struct {
	ID               int64
	ProjectID        int64
	ProjectKey       string
	ExceptionKey     string
	SourceGateItem   string
	SourceDecision   string
	AcceptanceType   string
	Status           string
	Owner            string
	Reason           string
	RequiredEvidence []string
	RollbackPlan     string
	ReviewRequired   bool
	ReviewAt         *time.Time
	ExpiresAt        *time.Time
	RequestedBy      string
	ApprovedBy       string
	RevokedBy        string
	DecisionReason   string
	AuditEventID     int64
	Metadata         map[string]any
	CreatedAt        time.Time
	UpdatedAt        time.Time
	ApprovedAt       *time.Time
	RevokedAt        *time.Time
	IdempotencyKey   string
	Created          bool
}

type RequestReleaseExceptionOptions struct {
	ExceptionKey   string
	Actor          string
	Reason         string
	Owner          string
	ReviewAt       *time.Time
	ExpiresAt      *time.Time
	IdempotencyKey string
}

type DecideReleaseExceptionOptions struct {
	ExceptionKey   string
	Actor          string
	Reason         string
	IdempotencyKey string
}

func (s Store) RequestReleaseException(ctx context.Context, record Record, options RequestReleaseExceptionOptions) (ReleaseExceptionRecord, error) {
	options.ExceptionKey = strings.TrimSpace(options.ExceptionKey)
	options.Actor = strings.TrimSpace(options.Actor)
	options.Reason = strings.TrimSpace(options.Reason)
	options.Owner = strings.TrimSpace(options.Owner)
	if options.ExceptionKey == "" || options.Actor == "" || options.Reason == "" {
		return ReleaseExceptionRecord{}, fmt.Errorf("exception key, actor and reason are required")
	}
	if options.ExpiresAt != nil && !options.ExpiresAt.After(time.Now().UTC()) {
		return ReleaseExceptionRecord{}, fmt.Errorf("release exception expiry must be in the future")
	}
	state, err := migrate.Approval(ctx, s.pool, migrate.ReleaseExceptionMigrationName)
	if err != nil {
		return ReleaseExceptionRecord{}, err
	}
	effective, err := migrate.ApprovalEffective(migrate.ReleaseExceptionMigrationName, state)
	if err != nil {
		return ReleaseExceptionRecord{}, err
	}
	if !effective || !state.Applied {
		return ReleaseExceptionRecord{}, fmt.Errorf("release exception migration must be applied with an effective approval")
	}
	preview, err := s.ReleaseExceptionRecordPreview(ctx, ReleaseExceptionRecordPreviewOptions{ProjectID: record.ID, ProjectKey: record.Key})
	if err != nil {
		return ReleaseExceptionRecord{}, err
	}
	draft, found := releaseExceptionDraftByKey(preview.Drafts, options.ExceptionKey)
	if !found || draft.Status != "draft" {
		return ReleaseExceptionRecord{}, fmt.Errorf("requestable release exception draft not found: %s", options.ExceptionKey)
	}
	if options.Owner == "" {
		options.Owner = draft.Owner
	}
	requestHash, err := releaseExceptionRequestHash(record, options, draft)
	if err != nil {
		return ReleaseExceptionRecord{}, err
	}
	if options.IdempotencyKey == "" {
		options.IdempotencyKey = releaseExceptionIdempotencyKey(releaseExceptionRequestCommandType, record.Key, options.ExceptionKey, requestHash)
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return ReleaseExceptionRecord{}, fmt.Errorf("begin release exception request: %w", err)
	}
	defer tx.Rollback(ctx)
	created, err := reserveCommandRequest(ctx, tx, record.ID, releaseExceptionRequestCommandType, options.IdempotencyKey, requestHash)
	if err != nil {
		return ReleaseExceptionRecord{}, err
	}
	if !created {
		return loadReleaseExceptionCommandResult(ctx, tx, record, releaseExceptionRequestCommandType, options.IdempotencyKey)
	}

	result, err := upsertRequestedReleaseException(ctx, tx, record, options, draft)
	if err != nil {
		return ReleaseExceptionRecord{}, err
	}
	auditID, err := insertReleaseExceptionAuditEvent(ctx, tx, result, options.Actor, options.Reason, releaseExceptionRequestCommandType)
	if err != nil {
		return ReleaseExceptionRecord{}, err
	}
	result.AuditEventID = auditID
	result.IdempotencyKey = options.IdempotencyKey
	result.Created = true
	if _, err := tx.Exec(ctx, `UPDATE release_exceptions SET audit_event_id = $2 WHERE id = $1`, result.ID, auditID); err != nil {
		return ReleaseExceptionRecord{}, fmt.Errorf("bind release exception audit event: %w", err)
	}
	if err := completeCommandRequestResponse(ctx, tx, record.ID, releaseExceptionRequestCommandType, options.IdempotencyKey, releaseExceptionCommandResponse(result)); err != nil {
		return ReleaseExceptionRecord{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return ReleaseExceptionRecord{}, fmt.Errorf("commit release exception request: %w", err)
	}
	return result, nil
}

func (s Store) ApproveReleaseException(ctx context.Context, record Record, options DecideReleaseExceptionOptions) (ReleaseExceptionRecord, error) {
	return s.decideReleaseException(ctx, record, options, "approved", releaseExceptionApproveCommandType)
}

func (s Store) RevokeReleaseException(ctx context.Context, record Record, options DecideReleaseExceptionOptions) (ReleaseExceptionRecord, error) {
	return s.decideReleaseException(ctx, record, options, "revoked", releaseExceptionRevokeCommandType)
}

func (s Store) decideReleaseException(ctx context.Context, record Record, options DecideReleaseExceptionOptions, targetStatus string, commandType string) (ReleaseExceptionRecord, error) {
	options.ExceptionKey = strings.TrimSpace(options.ExceptionKey)
	options.Actor = strings.TrimSpace(options.Actor)
	options.Reason = strings.TrimSpace(options.Reason)
	if options.ExceptionKey == "" || options.Actor == "" || options.Reason == "" {
		return ReleaseExceptionRecord{}, fmt.Errorf("exception key, actor and reason are required")
	}
	state, err := migrate.Approval(ctx, s.pool, migrate.ReleaseExceptionMigrationName)
	if err != nil {
		return ReleaseExceptionRecord{}, err
	}
	effective, err := migrate.ApprovalEffective(migrate.ReleaseExceptionMigrationName, state)
	if err != nil {
		return ReleaseExceptionRecord{}, err
	}
	if !state.Applied || (targetStatus != "revoked" && !effective) {
		return ReleaseExceptionRecord{}, fmt.Errorf("release exception writes are disabled without an effective applied migration approval")
	}
	requestHash := releaseExceptionDecisionHash(record, options, targetStatus)
	if options.IdempotencyKey == "" {
		options.IdempotencyKey = releaseExceptionIdempotencyKey(commandType, record.Key, options.ExceptionKey, requestHash)
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return ReleaseExceptionRecord{}, fmt.Errorf("begin release exception %s: %w", targetStatus, err)
	}
	defer tx.Rollback(ctx)
	created, err := reserveCommandRequest(ctx, tx, record.ID, commandType, options.IdempotencyKey, requestHash)
	if err != nil {
		return ReleaseExceptionRecord{}, err
	}
	if !created {
		return loadReleaseExceptionCommandResult(ctx, tx, record, commandType, options.IdempotencyKey)
	}
	current, err := loadReleaseExceptionForUpdate(ctx, tx, record, options.ExceptionKey)
	if err != nil {
		return ReleaseExceptionRecord{}, err
	}
	if err := validateReleaseExceptionTransition(current.Status, targetStatus); err != nil {
		return ReleaseExceptionRecord{}, err
	}
	result, err := updateReleaseExceptionDecision(ctx, tx, current, options, targetStatus)
	if err != nil {
		return ReleaseExceptionRecord{}, err
	}
	auditID, err := insertReleaseExceptionAuditEvent(ctx, tx, result, options.Actor, options.Reason, commandType)
	if err != nil {
		return ReleaseExceptionRecord{}, err
	}
	result.AuditEventID = auditID
	result.IdempotencyKey = options.IdempotencyKey
	result.Created = true
	if _, err := tx.Exec(ctx, `UPDATE release_exceptions SET audit_event_id = $2 WHERE id = $1`, result.ID, auditID); err != nil {
		return ReleaseExceptionRecord{}, fmt.Errorf("bind release exception audit event: %w", err)
	}
	if err := completeCommandRequestResponse(ctx, tx, record.ID, commandType, options.IdempotencyKey, releaseExceptionCommandResponse(result)); err != nil {
		return ReleaseExceptionRecord{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return ReleaseExceptionRecord{}, fmt.Errorf("commit release exception %s: %w", targetStatus, err)
	}
	return result, nil
}

func releaseExceptionDraftByKey(drafts []ReleaseExceptionRecordDraft, key string) (ReleaseExceptionRecordDraft, bool) {
	for _, draft := range drafts {
		if draft.Key == key {
			return draft, true
		}
	}
	return ReleaseExceptionRecordDraft{}, false
}

func validateReleaseExceptionTransition(current string, target string) error {
	if target == "approved" && current == "requested" {
		return nil
	}
	if target == "revoked" && (current == "requested" || current == "approved") {
		return nil
	}
	return fmt.Errorf("invalid release exception transition %s -> %s", current, target)
}

func (s Store) EffectiveReleaseExceptions(ctx context.Context, projectID int64, at time.Time) ([]ReleaseExceptionRecord, error) {
	if at.IsZero() {
		at = time.Now().UTC()
	}
	var tableExists bool
	if err := s.pool.QueryRow(ctx, `SELECT to_regclass('public.release_exceptions') IS NOT NULL`).Scan(&tableExists); err != nil {
		return nil, fmt.Errorf("check release exceptions table: %w", err)
	}
	if !tableExists {
		return []ReleaseExceptionRecord{}, nil
	}
	rows, err := s.pool.Query(ctx, `
SELECT id, project_id, exception_key, source_gate_item, source_decision, acceptance_type,
       status, owner, reason, required_evidence, rollback_plan, review_required,
       review_at, expires_at, requested_by, COALESCE(approved_by, ''), COALESCE(revoked_by, ''),
       COALESCE(decision_reason, ''), COALESCE(audit_event_id, 0), metadata,
       created_at, updated_at, approved_at, revoked_at
FROM release_exceptions
WHERE project_id = $1 AND status = 'approved' AND (expires_at IS NULL OR expires_at > $2)
ORDER BY created_at, id`, projectID, at)
	if err != nil {
		return nil, fmt.Errorf("list effective release exceptions: %w", err)
	}
	defer rows.Close()
	results := []ReleaseExceptionRecord{}
	for rows.Next() {
		record, err := scanReleaseException(rows, "")
		if err != nil {
			return nil, err
		}
		results = append(results, record)
	}
	return results, rows.Err()
}

func upsertRequestedReleaseException(ctx context.Context, tx pgx.Tx, project Record, options RequestReleaseExceptionOptions, draft ReleaseExceptionRecordDraft) (ReleaseExceptionRecord, error) {
	evidence, _ := json.Marshal(draft.RequiredEvidence)
	actions, _ := json.Marshal(draft.AuditActions)
	metadata, _ := json.Marshal(draft.Metadata)
	row := tx.QueryRow(ctx, `
INSERT INTO release_exceptions (
    project_id, exception_key, source_gate_item, source_decision, acceptance_type, status,
    owner, reason, required_evidence, audit_actions, rollback_plan, review_required,
    review_at, expires_at, requested_by, metadata
)
VALUES ($1, $2, $3, $4, $5, 'requested', $6, $7, $8::jsonb, $9::jsonb, $10, $11, $12, $13, $14, $15::jsonb)
ON CONFLICT (project_id, exception_key) DO UPDATE
SET source_gate_item = EXCLUDED.source_gate_item,
    source_decision = EXCLUDED.source_decision,
    acceptance_type = EXCLUDED.acceptance_type,
    status = 'requested',
    owner = EXCLUDED.owner,
    reason = EXCLUDED.reason,
    required_evidence = EXCLUDED.required_evidence,
    audit_actions = EXCLUDED.audit_actions,
    rollback_plan = EXCLUDED.rollback_plan,
    review_required = EXCLUDED.review_required,
    review_at = EXCLUDED.review_at,
    expires_at = EXCLUDED.expires_at,
    requested_by = EXCLUDED.requested_by,
    approved_by = NULL,
    revoked_by = NULL,
    decision_reason = NULL,
    approved_at = NULL,
    revoked_at = NULL,
    metadata = EXCLUDED.metadata,
    updated_at = now()
WHERE release_exceptions.status IN ('revoked', 'rejected', 'expired')
RETURNING id, project_id, exception_key, source_gate_item, source_decision, acceptance_type,
          status, owner, reason, required_evidence, rollback_plan, review_required,
          review_at, expires_at, requested_by, COALESCE(approved_by, ''), COALESCE(revoked_by, ''),
          COALESCE(decision_reason, ''), COALESCE(audit_event_id, 0), metadata,
          created_at, updated_at, approved_at, revoked_at`, project.ID, draft.Key, draft.SourceGateItem,
		draft.SourceDecision, draft.AcceptanceType, options.Owner, options.Reason, string(evidence), string(actions),
		draft.RollbackPlan, draft.ReviewRequired, options.ReviewAt, options.ExpiresAt, options.Actor, string(metadata))
	result, err := scanReleaseException(row, project.Key)
	if err == pgx.ErrNoRows {
		return ReleaseExceptionRecord{}, fmt.Errorf("release exception %s is already active", draft.Key)
	}
	if err != nil {
		return ReleaseExceptionRecord{}, fmt.Errorf("request release exception: %w", err)
	}
	return result, nil
}

func loadReleaseExceptionForUpdate(ctx context.Context, tx pgx.Tx, project Record, key string) (ReleaseExceptionRecord, error) {
	result, err := scanReleaseException(tx.QueryRow(ctx, `
SELECT id, project_id, exception_key, source_gate_item, source_decision, acceptance_type,
       status, owner, reason, required_evidence, rollback_plan, review_required,
       review_at, expires_at, requested_by, COALESCE(approved_by, ''), COALESCE(revoked_by, ''),
       COALESCE(decision_reason, ''), COALESCE(audit_event_id, 0), metadata,
       created_at, updated_at, approved_at, revoked_at
FROM release_exceptions WHERE project_id = $1 AND exception_key = $2 FOR UPDATE`, project.ID, key), project.Key)
	if err == pgx.ErrNoRows {
		return ReleaseExceptionRecord{}, fmt.Errorf("release exception not found: %s", key)
	}
	return result, err
}

func updateReleaseExceptionDecision(ctx context.Context, tx pgx.Tx, current ReleaseExceptionRecord, options DecideReleaseExceptionOptions, status string) (ReleaseExceptionRecord, error) {
	query := `
UPDATE release_exceptions
SET status = $2, decision_reason = $3, updated_at = now(),
    approved_by = CASE WHEN $2 = 'approved' THEN $4 ELSE approved_by END,
    approved_at = CASE WHEN $2 = 'approved' THEN now() ELSE approved_at END,
    revoked_by = CASE WHEN $2 = 'revoked' THEN $4 ELSE revoked_by END,
    revoked_at = CASE WHEN $2 = 'revoked' THEN now() ELSE revoked_at END
WHERE id = $1
RETURNING id, project_id, exception_key, source_gate_item, source_decision, acceptance_type,
          status, owner, reason, required_evidence, rollback_plan, review_required,
          review_at, expires_at, requested_by, COALESCE(approved_by, ''), COALESCE(revoked_by, ''),
          COALESCE(decision_reason, ''), COALESCE(audit_event_id, 0), metadata,
          created_at, updated_at, approved_at, revoked_at`
	result, err := scanReleaseException(tx.QueryRow(ctx, query, current.ID, status, options.Reason, options.Actor), current.ProjectKey)
	if err != nil {
		return ReleaseExceptionRecord{}, fmt.Errorf("update release exception decision: %w", err)
	}
	return result, nil
}

type releaseExceptionScanner interface{ Scan(...any) error }

func scanReleaseException(row releaseExceptionScanner, projectKey string) (ReleaseExceptionRecord, error) {
	var result ReleaseExceptionRecord
	var evidenceRaw, metadataRaw []byte
	if err := row.Scan(&result.ID, &result.ProjectID, &result.ExceptionKey, &result.SourceGateItem,
		&result.SourceDecision, &result.AcceptanceType, &result.Status, &result.Owner, &result.Reason,
		&evidenceRaw, &result.RollbackPlan, &result.ReviewRequired, &result.ReviewAt, &result.ExpiresAt,
		&result.RequestedBy, &result.ApprovedBy, &result.RevokedBy, &result.DecisionReason,
		&result.AuditEventID, &metadataRaw, &result.CreatedAt, &result.UpdatedAt, &result.ApprovedAt,
		&result.RevokedAt); err != nil {
		return ReleaseExceptionRecord{}, err
	}
	result.ProjectKey = projectKey
	if err := json.Unmarshal(evidenceRaw, &result.RequiredEvidence); err != nil {
		return ReleaseExceptionRecord{}, fmt.Errorf("decode release exception evidence: %w", err)
	}
	if err := json.Unmarshal(metadataRaw, &result.Metadata); err != nil {
		return ReleaseExceptionRecord{}, fmt.Errorf("decode release exception metadata: %w", err)
	}
	return result, nil
}

func insertReleaseExceptionAuditEvent(ctx context.Context, tx pgx.Tx, record ReleaseExceptionRecord, actor string, reason string, action string) (int64, error) {
	metadata, _ := json.Marshal(map[string]any{
		"exception_id": record.ID, "exception_key": record.ExceptionKey, "acceptance_type": record.AcceptanceType,
		"status": record.Status, "actor": actor, "source_gate_item": record.SourceGateItem,
	})
	var id int64
	if err := tx.QueryRow(ctx, `
INSERT INTO audit_events (project_id, action, capability, resource_type, resource, decision, reason, metadata)
VALUES ($1, $2, 'release_exception', 'release_exception', $3, 'allowed', $4, $5::jsonb)
RETURNING id`, record.ProjectID, action, record.ExceptionKey, reason, string(metadata)).Scan(&id); err != nil {
		return 0, fmt.Errorf("insert release exception audit event: %w", err)
	}
	return id, nil
}

func releaseExceptionRequestHash(record Record, options RequestReleaseExceptionOptions, draft ReleaseExceptionRecordDraft) (string, error) {
	payload, err := json.Marshal(map[string]any{"project_id": record.ID, "exception_key": options.ExceptionKey, "actor": options.Actor,
		"reason": options.Reason, "owner": options.Owner, "review_at": options.ReviewAt, "expires_at": options.ExpiresAt,
		"source_gate_item": draft.SourceGateItem, "acceptance_type": draft.AcceptanceType})
	if err != nil {
		return "", err
	}
	return releaseExceptionSHA256Hex(payload), nil
}

func releaseExceptionDecisionHash(record Record, options DecideReleaseExceptionOptions, status string) string {
	payload, _ := json.Marshal(map[string]any{"project_id": record.ID, "exception_key": options.ExceptionKey, "actor": options.Actor, "reason": options.Reason, "status": status})
	return releaseExceptionSHA256Hex(payload)
}

func releaseExceptionSHA256Hex(payload []byte) string {
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:])
}

func releaseExceptionIdempotencyKey(commandType string, projectKey string, exceptionKey string, requestHash string) string {
	return fmt.Sprintf("%s:%s:%s:%s", commandType, projectKey, exceptionKey, requestHash[:12])
}

func releaseExceptionCommandResponse(record ReleaseExceptionRecord) map[string]any {
	return map[string]any{"release_exception_id": record.ID, "exception_key": record.ExceptionKey, "status": record.Status, "audit_event_id": record.AuditEventID}
}

func loadReleaseExceptionCommandResult(ctx context.Context, tx pgx.Tx, project Record, commandType string, idempotencyKey string) (ReleaseExceptionRecord, error) {
	var responseRaw []byte
	if err := tx.QueryRow(ctx, `SELECT response FROM command_requests WHERE project_id = $1 AND command_type = $2 AND idempotency_key = $3`, project.ID, commandType, idempotencyKey).Scan(&responseRaw); err != nil {
		return ReleaseExceptionRecord{}, fmt.Errorf("load release exception command response: %w", err)
	}
	var response map[string]any
	if err := json.Unmarshal(responseRaw, &response); err != nil {
		return ReleaseExceptionRecord{}, fmt.Errorf("decode release exception command response: %w", err)
	}
	id := metadataInt64(response, "release_exception_id")
	result, err := scanReleaseException(tx.QueryRow(ctx, `
SELECT id, project_id, exception_key, source_gate_item, source_decision, acceptance_type,
       status, owner, reason, required_evidence, rollback_plan, review_required,
       review_at, expires_at, requested_by, COALESCE(approved_by, ''), COALESCE(revoked_by, ''),
       COALESCE(decision_reason, ''), COALESCE(audit_event_id, 0), metadata,
       created_at, updated_at, approved_at, revoked_at
FROM release_exceptions WHERE id = $1`, id), project.Key)
	if err != nil {
		return ReleaseExceptionRecord{}, fmt.Errorf("load idempotent release exception: %w", err)
	}
	result.IdempotencyKey = idempotencyKey
	return result, nil
}
