package project

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

var ErrRunControlBlocked = errors.New("run control blocked")

type RunControlOptions struct {
	Action         string
	IdempotencyKey string
	Actor          string
	Reason         string
}

type RunControlResult struct {
	Project                  Record
	Run                      RunRecord
	PreviousStatus           string
	Status                   string
	Decision                 string
	Message                  string
	Blockers                 []string
	EventID                  int64
	AuditEventID             int64
	IdempotencyKey           string
	Created                  bool
	ProjectWriteAttempted    bool
	ExecutionWriteAttempted  bool
	AreaMatrixWriteAttempted bool
	EngineCallAttempted      bool
}

func (s Store) ControlRun(ctx context.Context, runID int64, options RunControlOptions) (RunControlResult, error) {
	if runID <= 0 {
		return RunControlResult{}, fmt.Errorf("run id is required")
	}
	options = normalizeRunControlOptions(runID, options)
	commandType := runControlCommandType(options.Action)
	requestHash, err := runControlRequestHash(runID, options)
	if err != nil {
		return RunControlResult{}, err
	}
	if options.IdempotencyKey == "" {
		options.IdempotencyKey = runControlIdempotencyKey(runID, options, requestHash)
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return RunControlResult{}, fmt.Errorf("begin run control: %w", err)
	}
	defer tx.Rollback(ctx)

	run, err := loadRunForUpdate(ctx, tx, runID)
	if err != nil {
		return RunControlResult{}, err
	}
	record, err := loadProjectRecordByID(ctx, tx, run.ProjectID)
	if err != nil {
		return RunControlResult{}, err
	}

	created, err := reserveCommandRequest(ctx, tx, record.ID, commandType, options.IdempotencyKey, requestHash)
	if err != nil {
		return RunControlResult{}, err
	}
	if !created {
		result, err := loadRunControlByCommandResponse(ctx, tx, record, commandType, options.IdempotencyKey)
		if err != nil {
			return RunControlResult{}, err
		}
		if err := tx.Commit(ctx); err != nil {
			return RunControlResult{}, fmt.Errorf("commit idempotent run control: %w", err)
		}
		result.Created = false
		return result, nil
	}

	result := evaluateRunControl(record, run, options)
	if result.Decision == "allowed" {
		updated, err := updateRunControlStatus(ctx, tx, run, result.Status, options, result)
		if err != nil {
			return RunControlResult{}, err
		}
		result.Run = updated
	}
	eventID, err := insertRunControlEvent(ctx, tx, result, options)
	if err != nil {
		return RunControlResult{}, err
	}
	result.EventID = eventID
	auditEventID, err := insertRunControlAuditEvent(ctx, tx, result, options)
	if err != nil {
		return RunControlResult{}, err
	}
	result.AuditEventID = auditEventID
	result.IdempotencyKey = options.IdempotencyKey
	result.Created = true
	if err := completeCommandRequestResponse(ctx, tx, record.ID, commandType, options.IdempotencyKey, runControlCommandResponse(result)); err != nil {
		return RunControlResult{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return RunControlResult{}, fmt.Errorf("commit run control: %w", err)
	}
	if result.Decision == "denied" {
		return result, fmt.Errorf("%w: %s", ErrRunControlBlocked, strings.Join(result.Blockers, "; "))
	}
	return result, nil
}

func normalizeRunControlOptions(runID int64, options RunControlOptions) RunControlOptions {
	options.Action = strings.TrimSpace(options.Action)
	options.IdempotencyKey = strings.TrimSpace(options.IdempotencyKey)
	options.Actor = strings.TrimSpace(options.Actor)
	options.Reason = strings.TrimSpace(options.Reason)
	if options.Actor == "" {
		options.Actor = "local-user"
	}
	if options.Reason == "" {
		options.Reason = fmt.Sprintf("run %s protected control", options.Action)
	}
	if options.IdempotencyKey == "" && runID <= 0 {
		options.IdempotencyKey = "run.control:unknown"
	}
	return options
}

func runControlCommandType(action string) string {
	return "run." + strings.TrimSpace(action)
}

func runControlRequestHash(runID int64, options RunControlOptions) (string, error) {
	payload, err := json.Marshal(map[string]any{
		"command_type": runControlCommandType(options.Action),
		"run_id":       runID,
		"actor":        options.Actor,
		"reason":       options.Reason,
		"protected":    true,
		"dry_run_only": true,
	})
	if err != nil {
		return "", fmt.Errorf("marshal run control command request hash payload: %w", err)
	}
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:]), nil
}

func runControlIdempotencyKey(runID int64, options RunControlOptions, requestHash string) string {
	prefix := requestHash
	if len(prefix) > 16 {
		prefix = prefix[:16]
	}
	return fmt.Sprintf("run.%s:%d:%s", options.Action, runID, prefix)
}

func evaluateRunControl(record Record, run RunRecord, options RunControlOptions) RunControlResult {
	result := RunControlResult{
		Project:                  record,
		Run:                      run,
		PreviousStatus:           run.Status,
		Status:                   run.Status,
		Decision:                 "allowed",
		Message:                  "run control applied in AreaFlow state",
		ProjectWriteAttempted:    false,
		ExecutionWriteAttempted:  false,
		AreaMatrixWriteAttempted: false,
		EngineCallAttempted:      false,
	}
	blockers := []string{}
	switch options.Action {
	case "start":
		if run.Status == "running" {
			result.Message = "run was already running"
			return result
		}
		if run.Status != "queued" {
			blockers = append(blockers, fmt.Sprintf("run status %q cannot be started; expected queued", run.Status))
		} else {
			result.Status = "running"
			result.Message = "run marked running in protected mode"
		}
	case "drain":
		if run.Status == "draining" || run.Status == "drained" {
			result.Message = "run was already draining or drained"
			return result
		}
		if run.Status != "running" {
			blockers = append(blockers, fmt.Sprintf("run status %q cannot be drained; expected running", run.Status))
		} else {
			result.Status = "draining"
			result.Message = "run drain requested in protected mode"
		}
	case "cancel":
		if run.Status == "cancelled" || run.Status == "cancelling" {
			result.Message = "run was already cancelled or cancelling"
			return result
		}
		switch run.Status {
		case "queued":
			result.Status = "cancelled"
			result.Message = "queued run cancelled in protected mode"
		case "running", "draining":
			result.Status = "cancelling"
			result.Message = "run cancellation requested in protected mode"
		default:
			blockers = append(blockers, fmt.Sprintf("run status %q cannot be cancelled", run.Status))
		}
	default:
		blockers = append(blockers, fmt.Sprintf("run control action %q is not supported", options.Action))
	}
	if !run.DryRun {
		blockers = append(blockers, "protected run control is only enabled for dry-run runs")
	}
	if len(blockers) > 0 {
		result.Decision = "denied"
		result.Message = "run control blocked by protected boundary"
		result.Blockers = blockers
		result.Status = run.Status
	}
	return result
}

func updateRunControlStatus(ctx context.Context, tx pgx.Tx, run RunRecord, status string, options RunControlOptions, result RunControlResult) (RunRecord, error) {
	summary := copyMap(run.Summary)
	summary["control_status"] = status
	summary["last_control_action"] = options.Action
	summary["last_control_actor"] = options.Actor
	summary["last_control_reason"] = options.Reason
	summary["last_control_at"] = time.Now().UTC().Format(time.RFC3339)
	summary["project_write_attempted"] = false
	summary["execution_write_attempted"] = false
	summary["area_matrix_write_attempted"] = false
	summary["engine_call_attempted"] = false
	summaryJSON, err := json.Marshal(summary)
	if err != nil {
		return RunRecord{}, fmt.Errorf("marshal run control summary: %w", err)
	}
	metadata := copyMap(run.Metadata)
	metadata["protected_control"] = true
	metadata["last_control_action"] = options.Action
	metadata["last_control_actor"] = options.Actor
	metadata["last_control_reason"] = options.Reason
	metadata["last_control_idempotency_key"] = options.IdempotencyKey
	metadata["project_write_attempted"] = false
	metadata["execution_write_attempted"] = false
	metadata["area_matrix_write_attempted"] = false
	metadata["engine_call_attempted"] = false
	metadataJSON, err := json.Marshal(metadata)
	if err != nil {
		return RunRecord{}, fmt.Errorf("marshal run control metadata: %w", err)
	}

	var finishedAtExpr string
	switch status {
	case "cancelled", "drained":
		finishedAtExpr = "now()"
	default:
		if run.FinishedAt != nil && result.PreviousStatus != status {
			finishedAtExpr = "NULL"
		} else {
			finishedAtExpr = "finished_at"
		}
	}
	query := fmt.Sprintf(`
UPDATE runs
SET status = $2,
    summary = $3::jsonb,
    metadata = $4::jsonb,
    finished_at = %s
WHERE id = $1
RETURNING id, COALESCE(project_id, 0), COALESCE(workflow_version_id, 0), run_type,
          COALESCE(run_kind, ''), status, risk_level, risk_policy, dry_run,
          summary, metadata, started_at, finished_at`, finishedAtExpr)
	updated, err := scanRun(tx.QueryRow(ctx, query, run.ID, status, string(summaryJSON), string(metadataJSON)))
	if err != nil {
		return RunRecord{}, fmt.Errorf("update run control status: %w", err)
	}
	if options.Action == "cancel" && status == "cancelled" {
		if _, err := tx.Exec(ctx, `
UPDATE run_tasks
SET status = 'cancelled',
    updated_at = now()
WHERE run_id = $1
  AND status IN ('pending', 'queued', 'retry_waiting', 'needs_recovery')`,
			run.ID,
		); err != nil {
			return RunRecord{}, fmt.Errorf("cancel queued run tasks: %w", err)
		}
	}
	if options.Action == "cancel" && status == "cancelling" {
		if _, err := tx.Exec(ctx, `
UPDATE run_tasks
SET status = 'cancel_requested',
    updated_at = now()
WHERE run_id = $1
  AND status IN ('pending', 'queued', 'retry_waiting', 'needs_recovery')`,
			run.ID,
		); err != nil {
			return RunRecord{}, fmt.Errorf("request run task cancellation: %w", err)
		}
	}
	return updated, nil
}

func insertRunControlEvent(ctx context.Context, tx pgx.Tx, result RunControlResult, options RunControlOptions) (int64, error) {
	severity := "info"
	if result.Decision == "denied" {
		severity = "warning"
	}
	metadata, err := json.Marshal(runControlMetadata(result, options))
	if err != nil {
		return 0, fmt.Errorf("marshal run control event metadata: %w", err)
	}
	var eventID int64
	if err := tx.QueryRow(ctx, `
INSERT INTO events (project_id, run_id, workflow_version_id, event_type, severity, message, metadata)
VALUES ($1, $2, $3, $4, $5, $6, $7::jsonb)
RETURNING id`,
		result.Project.ID,
		result.Run.ID,
		nullableInt64(result.Run.WorkflowVersionID),
		fmt.Sprintf("run.%s.%s", options.Action, result.Decision),
		severity,
		result.Message,
		string(metadata),
	).Scan(&eventID); err != nil {
		return 0, fmt.Errorf("insert run control event: %w", err)
	}
	return eventID, nil
}

func insertRunControlAuditEvent(ctx context.Context, tx pgx.Tx, result RunControlResult, options RunControlOptions) (int64, error) {
	metadata, err := json.Marshal(runControlMetadata(result, options))
	if err != nil {
		return 0, fmt.Errorf("marshal run control audit metadata: %w", err)
	}
	var auditEventID int64
	if err := tx.QueryRow(ctx, `
INSERT INTO audit_events (project_id, action, capability, resource_type, resource, decision, reason, metadata)
VALUES ($1, $2, 'manage_runs', 'run', $3, $4, $5, $6::jsonb)
RETURNING id`,
		result.Project.ID,
		runControlCommandType(options.Action),
		fmt.Sprintf("%d", result.Run.ID),
		result.Decision,
		options.Reason,
		string(metadata),
	).Scan(&auditEventID); err != nil {
		return 0, fmt.Errorf("insert run control audit event: %w", err)
	}
	return auditEventID, nil
}

func runControlMetadata(result RunControlResult, options RunControlOptions) map[string]any {
	return map[string]any{
		"run_id":                      result.Run.ID,
		"workflow_version_id":         result.Run.WorkflowVersionID,
		"action":                      options.Action,
		"previous_status":             result.PreviousStatus,
		"status":                      result.Status,
		"decision":                    result.Decision,
		"actor":                       options.Actor,
		"idempotency_key":             options.IdempotencyKey,
		"protected_control":           true,
		"dry_run":                     result.Run.DryRun,
		"project_write_attempted":     false,
		"execution_write_attempted":   false,
		"area_matrix_write_attempted": false,
		"engine_call_attempted":       false,
		"blockers":                    result.Blockers,
	}
}

func runControlCommandResponse(result RunControlResult) map[string]any {
	return map[string]any{
		"project_id":                  result.Project.ID,
		"project_key":                 result.Project.Key,
		"run_id":                      result.Run.ID,
		"workflow_version_id":         result.Run.WorkflowVersionID,
		"previous_status":             result.PreviousStatus,
		"status":                      result.Status,
		"decision":                    result.Decision,
		"message":                     result.Message,
		"blockers":                    result.Blockers,
		"event_id":                    result.EventID,
		"audit_event_id":              result.AuditEventID,
		"dry_run":                     result.Run.DryRun,
		"task_claimed":                false,
		"worker_started":              false,
		"project_write_attempted":     false,
		"execution_write_attempted":   false,
		"area_matrix_write_attempted": false,
		"engine_call_attempted":       false,
		"commands_run":                false,
		"secrets_resolved":            false,
		"network_used":                false,
	}
}

func loadRunControlByCommandResponse(ctx context.Context, tx pgx.Tx, record Record, commandType string, idempotencyKey string) (RunControlResult, error) {
	response, err := loadCommandResponse(ctx, tx, record.ID, commandType, idempotencyKey)
	if err != nil {
		return RunControlResult{}, err
	}
	runID := metadataInt64(response, "run_id")
	run, err := loadRunForUpdate(ctx, tx, runID)
	if err != nil {
		return RunControlResult{}, err
	}
	return RunControlResult{
		Project:                  record,
		Run:                      run,
		PreviousStatus:           metadataString(response, "previous_status"),
		Status:                   metadataString(response, "status"),
		Decision:                 metadataString(response, "decision"),
		Message:                  metadataString(response, "message"),
		Blockers:                 metadataStringSlice(response, "blockers"),
		EventID:                  metadataInt64(response, "event_id"),
		AuditEventID:             metadataInt64(response, "audit_event_id"),
		IdempotencyKey:           idempotencyKey,
		ProjectWriteAttempted:    metadataBool(response, "project_write_attempted"),
		ExecutionWriteAttempted:  metadataBool(response, "execution_write_attempted"),
		AreaMatrixWriteAttempted: metadataBool(response, "area_matrix_write_attempted"),
		EngineCallAttempted:      metadataBool(response, "engine_call_attempted"),
	}, nil
}

func loadRunForUpdate(ctx context.Context, tx pgx.Tx, runID int64) (RunRecord, error) {
	run, err := scanRun(tx.QueryRow(ctx, `
SELECT id, COALESCE(project_id, 0), COALESCE(workflow_version_id, 0), run_type,
       COALESCE(run_kind, ''), status, risk_level, risk_policy, dry_run,
       summary, metadata, started_at, finished_at
FROM runs
WHERE id = $1
FOR UPDATE`,
		runID,
	))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return RunRecord{}, ErrRunNotFound
		}
		return RunRecord{}, fmt.Errorf("load run for update: %w", err)
	}
	return run, nil
}

func loadProjectRecordByID(ctx context.Context, tx pgx.Tx, projectID int64) (Record, error) {
	if projectID <= 0 {
		return Record{}, fmt.Errorf("run project id is required")
	}
	var record Record
	err := tx.QueryRow(ctx, `
SELECT p.id, p.project_key, p.name, p.kind, p.adapter, p.workflow_profile, COALESCE(p.default_branch, ''),
       COALESCE(c.root_path, ''), COALESCE(a.remote_url, ''), COALESCE(a.root_path, '')
FROM projects p
LEFT JOIN LATERAL (
    SELECT root_path
    FROM project_connections
    WHERE project_id = p.id AND connection_type = 'local_path'
    ORDER BY updated_at DESC, id DESC
    LIMIT 1
) c ON true
LEFT JOIN LATERAL (
    SELECT root_path, remote_url
    FROM project_connections
    WHERE project_id = p.id AND connection_type = 'artifact_store'
    ORDER BY updated_at DESC, id DESC
    LIMIT 1
) a ON true
WHERE p.id = $1`,
		projectID,
	).Scan(
		&record.ID,
		&record.Key,
		&record.Name,
		&record.Kind,
		&record.Adapter,
		&record.WorkflowProfile,
		&record.DefaultBranch,
		&record.RootPath,
		&record.ArtifactBackend,
		&record.ArtifactRoot,
	)
	if err != nil {
		return Record{}, fmt.Errorf("load run project: %w", err)
	}
	return record, nil
}

func copyMap(in map[string]any) map[string]any {
	out := map[string]any{}
	for key, value := range in {
		out[key] = value
	}
	return out
}

func metadataBool(metadata map[string]any, key string) bool {
	value, ok := metadata[key]
	if !ok {
		return false
	}
	switch typed := value.(type) {
	case bool:
		return typed
	case string:
		return typed == "true"
	default:
		return false
	}
}

func metadataStringSlice(metadata map[string]any, key string) []string {
	value, ok := metadata[key]
	if !ok || value == nil {
		return nil
	}
	switch typed := value.(type) {
	case []string:
		return append([]string{}, typed...)
	case []any:
		out := make([]string, 0, len(typed))
		for _, item := range typed {
			if text, ok := item.(string); ok {
				out = append(out, text)
			}
		}
		return out
	default:
		return nil
	}
}
