package project

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/areasong/areaflow/internal/artifact"
	"github.com/jackc/pgx/v5"
)

var (
	ErrRunnerPreviewBlocked       = errors.New("runner preview blocked")
	ErrRunNotFound                = errors.New("run not found")
	ErrArtifactNotFound           = errors.New("artifact not found")
	ErrArtifactContentUnavailable = errors.New("artifact content unavailable")
	ErrArtifactContentMismatch    = errors.New("artifact content mismatch")
)

const runnerPreviewCommandType = "runner.preview"

type RunnerPreviewOptions struct {
	Actor           string
	Reason          string
	RiskLevel       string
	RiskPolicy      string
	IdempotencyKey  string
	RequireApproval bool
}

type RunnerPreviewResult struct {
	Project        Record
	Version        WorkflowVersion
	Run            RunRecord
	Tasks          []RunTaskRecord
	Attempts       []RunAttemptRecord
	Artifacts      []ArtifactRecord
	Preflight      RunnerPreflight
	Created        bool
	IdempotencyKey string
	EventID        int64
	AuditEventID   int64
}

type RunDetail struct {
	Run       RunRecord
	Tasks     []RunTaskRecord
	Attempts  []RunAttemptRecord
	Artifacts []ArtifactRecord
}

type ArtifactContent struct {
	Artifact    ArtifactRecord
	Content     []byte
	ContentType string
}

type RunnerPreflight struct {
	Status   string
	Checks   []ReadinessItem
	Blockers []string
}

type RunRecord struct {
	ID                int64
	ProjectID         int64
	WorkflowVersionID int64
	RunType           string
	RunKind           string
	Status            string
	RiskLevel         string
	RiskPolicy        string
	DryRun            bool
	Summary           map[string]any
	Metadata          map[string]any
	StartedAt         time.Time
	FinishedAt        *time.Time
}

type RunTaskRecord struct {
	ID                int64
	ProjectID         int64
	WorkflowVersionID int64
	WorkflowItemID    int64
	RunID             int64
	TaskKey           string
	TaskKind          string
	Status            string
	RiskLevel         string
	Sequence          int
	Metadata          map[string]any
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

type RunAttemptRecord struct {
	ID                int64
	ProjectID         int64
	WorkflowVersionID int64
	WorkflowItemID    int64
	RunID             int64
	RunTaskID         int64
	AttemptKind       string
	Status            string
	DryRun            bool
	Metadata          map[string]any
	StartedAt         time.Time
	FinishedAt        *time.Time
}

func (s Store) PreviewRunner(ctx context.Context, record Record, label string, options RunnerPreviewOptions) (RunnerPreviewResult, error) {
	version, err := s.GetWorkflowVersion(ctx, record, label)
	if err != nil {
		return RunnerPreviewResult{}, err
	}
	if version.ImportMode != "authored" {
		return RunnerPreviewResult{}, fmt.Errorf("%w: %s", ErrWorkflowVersionNotAuthored, label)
	}
	options = normalizeRunnerPreviewOptions(record, version, options)
	preflight := EvaluateRunnerPreflight(record, version, options)
	if preflight.Status == "blocked" {
		return RunnerPreviewResult{}, fmt.Errorf("%w: %s", ErrRunnerPreviewBlocked, strings.Join(preflight.Blockers, "; "))
	}
	requestHash, err := runnerPreviewRequestHash(record, version, options)
	if err != nil {
		return RunnerPreviewResult{}, err
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return RunnerPreviewResult{}, fmt.Errorf("begin runner preview: %w", err)
	}
	defer tx.Rollback(ctx)

	created, err := reserveCommandRequest(ctx, tx, record.ID, runnerPreviewCommandType, options.IdempotencyKey, requestHash)
	if err != nil {
		return RunnerPreviewResult{}, err
	}
	if !created {
		result, err := s.loadRunnerPreviewByIdempotency(ctx, record, version, options.IdempotencyKey)
		if err != nil {
			return RunnerPreviewResult{}, err
		}
		if response, err := loadCommandResponse(ctx, tx, record.ID, runnerPreviewCommandType, options.IdempotencyKey); err == nil {
			result.EventID = metadataInt64(response, "event_id")
			result.AuditEventID = metadataInt64(response, "audit_event_id")
		}
		return result, nil
	}

	run, err := insertRunnerPreviewRun(ctx, tx, record, version, options, preflight)
	if err != nil {
		return RunnerPreviewResult{}, err
	}
	task, err := insertRunnerPreviewTask(ctx, tx, record, version, run, options, preflight)
	if err != nil {
		return RunnerPreviewResult{}, err
	}
	report, err := writeAndInsertRunnerPreviewArtifact(ctx, tx, record, version, run, task, options, preflight)
	if err != nil {
		return RunnerPreviewResult{}, err
	}
	attempts := make([]RunAttemptRecord, 0, 2)
	for _, attemptKind := range []string{"copy", "verify"} {
		attempt, err := insertRunnerPreviewAttempt(ctx, tx, record, version, run, task, report.ID, attemptKind, options)
		if err != nil {
			return RunnerPreviewResult{}, err
		}
		attempts = append(attempts, attempt)
	}
	eventID, err := insertRunnerPreviewEvent(ctx, tx, record, version, run, task, report, options, preflight)
	if err != nil {
		return RunnerPreviewResult{}, err
	}
	auditEventID, err := insertRunnerPreviewAuditEvent(ctx, tx, record, version, run, task, report, options, preflight)
	if err != nil {
		return RunnerPreviewResult{}, err
	}
	result := RunnerPreviewResult{
		Project:        record,
		Version:        version,
		Run:            run,
		Tasks:          []RunTaskRecord{task},
		Attempts:       attempts,
		Artifacts:      []ArtifactRecord{report},
		Preflight:      preflight,
		Created:        true,
		IdempotencyKey: options.IdempotencyKey,
		EventID:        eventID,
		AuditEventID:   auditEventID,
	}
	if err := completeCommandRequestResponse(ctx, tx, record.ID, runnerPreviewCommandType, options.IdempotencyKey, runnerPreviewCommandResponse(result)); err != nil {
		return RunnerPreviewResult{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return RunnerPreviewResult{}, fmt.Errorf("commit runner preview: %w", err)
	}
	return result, nil
}

func EvaluateRunnerPreflight(record Record, version WorkflowVersion, options RunnerPreviewOptions) RunnerPreflight {
	preflight := RunnerPreflight{
		Status: "pass",
	}
	addPreflight := func(key string, status string, message string, metadata map[string]any) {
		preflight.Checks = append(preflight.Checks, ReadinessItem{
			Key:      key,
			Status:   status,
			Message:  message,
			Metadata: metadata,
		})
		if status == "blocked" || status == "fail" {
			preflight.Status = "blocked"
			preflight.Blockers = append(preflight.Blockers, fmt.Sprintf("%s: %s", key, message))
		}
	}
	addPreflight("dry_run", "pass", "runner preview is dry-run only", map[string]any{
		"dry_run": true,
	})
	addPreflight("workflow_version_authored", statusBool(version.ImportMode == "authored"), runnerVersionMessage(version), map[string]any{
		"display_label": version.DisplayLabel,
		"import_mode":   version.ImportMode,
	})
	addPreflight("write_permissions", "pass", "write operations are not attempted in runner preview", map[string]any{
		"write_attempted": false,
		"capability":      "write_code",
	})
	addPreflight("command_permissions", "pass", "commands are not executed in runner preview", map[string]any{
		"command_attempted": false,
		"capability":        "run_commands",
	})
	addPreflight("secret_permissions", "pass", "secrets are not resolved in runner preview", map[string]any{
		"secret_attempted": false,
		"capability":       "use_secrets",
	})
	addPreflight("network_permissions", "pass", "network access is not attempted in runner preview", map[string]any{
		"network_attempted": false,
		"capability":        "network",
	})
	if (options.RiskLevel == "high" || options.RiskLevel == "mission_critical") && options.RiskPolicy != "allow" {
		addPreflight("risk_gate", "blocked", "high risk runner preview requires risk policy allow", map[string]any{
			"risk_level":  options.RiskLevel,
			"risk_policy": options.RiskPolicy,
		})
	} else {
		addPreflight("risk_gate", "pass", "risk policy allows dry-run preview", map[string]any{
			"risk_level":  options.RiskLevel,
			"risk_policy": options.RiskPolicy,
		})
	}
	_ = record
	return preflight
}

func normalizeRunnerPreviewOptions(record Record, version WorkflowVersion, options RunnerPreviewOptions) RunnerPreviewOptions {
	options.Actor = strings.TrimSpace(options.Actor)
	options.Reason = strings.TrimSpace(options.Reason)
	options.RiskLevel = strings.TrimSpace(options.RiskLevel)
	options.RiskPolicy = strings.TrimSpace(options.RiskPolicy)
	options.IdempotencyKey = strings.TrimSpace(options.IdempotencyKey)
	if options.Actor == "" {
		options.Actor = "local-user"
	}
	if options.Reason == "" {
		options.Reason = "runner preview dry-run"
	}
	if options.RiskLevel == "" {
		options.RiskLevel = "low"
	}
	if options.RiskPolicy == "" {
		options.RiskPolicy = "pause"
	}
	if options.IdempotencyKey == "" {
		options.IdempotencyKey = fmt.Sprintf("runner.preview:%s:%s", record.Key, version.DisplayLabel)
	}
	return options
}

func runnerPreviewRequestHash(record Record, version WorkflowVersion, options RunnerPreviewOptions) (string, error) {
	payload, err := json.Marshal(map[string]any{
		"command_type":        runnerPreviewCommandType,
		"project_id":          record.ID,
		"project_key":         record.Key,
		"workflow_version_id": version.ID,
		"display_label":       version.DisplayLabel,
		"actor":               options.Actor,
		"reason":              options.Reason,
		"risk_level":          options.RiskLevel,
		"risk_policy":         options.RiskPolicy,
		"require_approval":    options.RequireApproval,
		"dry_run":             true,
	})
	if err != nil {
		return "", fmt.Errorf("marshal runner preview request hash payload: %w", err)
	}
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:]), nil
}

func insertRunnerPreviewRun(ctx context.Context, tx pgx.Tx, record Record, version WorkflowVersion, options RunnerPreviewOptions, preflight RunnerPreflight) (RunRecord, error) {
	summary, err := json.Marshal(map[string]any{
		"dry_run":          true,
		"preflight_status": preflight.Status,
		"task_count":       1,
		"attempt_count":    2,
		"artifact_count":   1,
	})
	if err != nil {
		return RunRecord{}, fmt.Errorf("marshal runner preview run summary: %w", err)
	}
	metadata, err := json.Marshal(map[string]any{
		"actor":           options.Actor,
		"reason":          options.Reason,
		"idempotency_key": options.IdempotencyKey,
		"phase":           "v0.5",
		"dry_run":         true,
	})
	if err != nil {
		return RunRecord{}, fmt.Errorf("marshal runner preview run metadata: %w", err)
	}
	run, err := scanRun(tx.QueryRow(ctx, `
INSERT INTO runs (
    project_id, workflow_version_id, run_type, run_kind, status, risk_level,
    risk_policy, dry_run, summary, metadata, finished_at
)
VALUES ($1, $2, 'runner_preview', 'execution', 'passed', $3, $4, true, $5::jsonb, $6::jsonb, now())
RETURNING id, COALESCE(project_id, 0), COALESCE(workflow_version_id, 0), run_type,
          COALESCE(run_kind, ''), status, risk_level, risk_policy, dry_run,
          summary, metadata, started_at, finished_at`,
		record.ID,
		version.ID,
		options.RiskLevel,
		options.RiskPolicy,
		string(summary),
		string(metadata),
	))
	if err != nil {
		return RunRecord{}, fmt.Errorf("insert runner preview run: %w", err)
	}
	return run, nil
}

func insertRunnerPreviewTask(ctx context.Context, tx pgx.Tx, record Record, version WorkflowVersion, run RunRecord, options RunnerPreviewOptions, preflight RunnerPreflight) (RunTaskRecord, error) {
	metadata, err := json.Marshal(map[string]any{
		"actor":               options.Actor,
		"reason":              options.Reason,
		"preflight_status":    preflight.Status,
		"copy_ready_source":   "not_materialized_in_v0.5_preview",
		"verify_ready_source": "not_materialized_in_v0.5_preview",
	})
	if err != nil {
		return RunTaskRecord{}, fmt.Errorf("marshal runner preview task metadata: %w", err)
	}
	task, err := scanRunTask(tx.QueryRow(ctx, `
INSERT INTO run_tasks (
    project_id, workflow_version_id, run_id, task_key, task_kind, status,
    risk_level, sequence, metadata
)
VALUES ($1, $2, $3, $4, 'workflow_item_preview', 'queued', $5, 1, $6::jsonb)
RETURNING id, project_id, COALESCE(workflow_version_id, 0), COALESCE(workflow_item_id, 0),
          run_id, task_key, task_kind, status, risk_level, sequence, metadata,
          created_at, updated_at`,
		record.ID,
		version.ID,
		run.ID,
		version.DisplayLabel+":runner-preview",
		options.RiskLevel,
		string(metadata),
	))
	if err != nil {
		return RunTaskRecord{}, fmt.Errorf("insert runner preview task: %w", err)
	}
	return task, nil
}

func writeAndInsertRunnerPreviewArtifact(ctx context.Context, tx pgx.Tx, record Record, version WorkflowVersion, run RunRecord, task RunTaskRecord, options RunnerPreviewOptions, preflight RunnerPreflight) (ArtifactRecord, error) {
	content, err := json.MarshalIndent(map[string]any{
		"project":          record.Key,
		"display_label":    version.DisplayLabel,
		"run_id":           run.ID,
		"run_task_id":      task.ID,
		"dry_run":          true,
		"risk_level":       options.RiskLevel,
		"risk_policy":      options.RiskPolicy,
		"preflight_status": preflight.Status,
		"preflight_checks": preflight.Checks,
		"attempts":         []string{"copy", "verify"},
		"writes_attempted": false,
		"commands_run":     false,
		"secrets_resolved": false,
		"network_used":     false,
	}, "", "  ")
	if err != nil {
		return ArtifactRecord{}, fmt.Errorf("marshal runner preview report: %w", err)
	}
	relativePath := filepath.Join(version.DisplayLabel, "runner-preview", fmt.Sprintf("run-%d-report.json", run.ID))
	stored, err := writeProjectArtifact(record, relativePath, content, "application/json")
	if err != nil {
		return ArtifactRecord{}, err
	}
	metadata, err := json.Marshal(map[string]any{
		"phase":           "v0.5",
		"owned_by":        "areaflow",
		"dry_run":         true,
		"run_id":          run.ID,
		"run_task_id":     task.ID,
		"actor":           options.Actor,
		"reason":          options.Reason,
		"idempotency_key": options.IdempotencyKey,
	})
	if err != nil {
		return ArtifactRecord{}, fmt.Errorf("marshal runner preview artifact metadata: %w", err)
	}
	return insertRunArtifactRecord(ctx, tx, record.ID, version.ID, run.ID, task.WorkflowItemID, "runner_preview_report", relativePath, stored, string(metadata))
}

func insertRunArtifactRecord(ctx context.Context, tx pgx.Tx, projectID int64, versionID int64, runID int64, itemID int64, artifactType string, sourcePath string, stored artifact.Stored, metadata string) (ArtifactRecord, error) {
	var nullableItem any
	if itemID != 0 {
		nullableItem = itemID
	}
	record, err := scanArtifactRecord(tx.QueryRow(ctx, `
INSERT INTO artifacts (
    project_id, workflow_version_id, run_id, workflow_item_id, artifact_type,
    storage_backend, uri, source_path, sha256, size_bytes, content_type, metadata
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12::jsonb)
RETURNING id, project_id, workflow_version_id, workflow_item_id, artifact_type,
          storage_backend, uri, COALESCE(source_path, ''), COALESCE(sha256, ''),
          COALESCE(size_bytes, 0), COALESCE(content_type, ''), metadata, created_at`,
		projectID,
		versionID,
		runID,
		nullableItem,
		artifactType,
		stored.Backend,
		stored.URI,
		sourcePath,
		stored.SHA256,
		stored.SizeBytes,
		stored.ContentType,
		metadata,
	))
	if err != nil {
		return ArtifactRecord{}, fmt.Errorf("insert runner preview artifact metadata: %w", err)
	}
	return record, nil
}

func insertRunnerPreviewAttempt(ctx context.Context, tx pgx.Tx, record Record, version WorkflowVersion, run RunRecord, task RunTaskRecord, reportArtifactID int64, attemptKind string, options RunnerPreviewOptions) (RunAttemptRecord, error) {
	metadata, err := json.Marshal(map[string]any{
		"actor":                    options.Actor,
		"reason":                   options.Reason,
		"dry_run":                  true,
		"would_execute":            false,
		"evidence_artifact_id":     reportArtifactID,
		"writes_attempted":         false,
		"commands_run":             false,
		"secrets_resolved":         false,
		"network_used":             false,
		"verify_can_mark_done":     false,
		"checkpoint_would_execute": false,
	})
	if err != nil {
		return RunAttemptRecord{}, fmt.Errorf("marshal runner preview attempt metadata: %w", err)
	}
	attempt, err := scanRunAttempt(tx.QueryRow(ctx, `
INSERT INTO run_attempts (
    project_id, workflow_version_id, run_id, run_task_id, attempt_kind,
    status, dry_run, finished_at, metadata
)
VALUES ($1, $2, $3, $4, $5, 'passed', true, now(), $6::jsonb)
RETURNING id, project_id, COALESCE(workflow_version_id, 0), COALESCE(workflow_item_id, 0),
          run_id, COALESCE(run_task_id, 0), attempt_kind, status, dry_run,
          metadata, started_at, finished_at`,
		record.ID,
		version.ID,
		run.ID,
		task.ID,
		attemptKind,
		string(metadata),
	))
	if err != nil {
		return RunAttemptRecord{}, fmt.Errorf("insert runner preview attempt: %w", err)
	}
	return attempt, nil
}

func insertRunnerPreviewEvent(ctx context.Context, tx pgx.Tx, record Record, version WorkflowVersion, run RunRecord, task RunTaskRecord, report ArtifactRecord, options RunnerPreviewOptions, preflight RunnerPreflight) (int64, error) {
	metadata, err := json.Marshal(map[string]any{
		"display_label":       version.DisplayLabel,
		"workflow_version_id": version.ID,
		"run_id":              run.ID,
		"run_task_id":         task.ID,
		"artifact_id":         report.ID,
		"actor":               options.Actor,
		"reason":              options.Reason,
		"risk_level":          options.RiskLevel,
		"risk_policy":         options.RiskPolicy,
		"dry_run":             true,
		"preflight_status":    preflight.Status,
	})
	if err != nil {
		return 0, fmt.Errorf("marshal runner preview event metadata: %w", err)
	}
	var eventID int64
	if err := tx.QueryRow(ctx, `
INSERT INTO events (project_id, run_id, workflow_version_id, event_type, severity, message, metadata)
VALUES ($1, $2, $3, 'runner.preview.completed', 'info', 'Runner preview dry-run completed', $4::jsonb)
RETURNING id`,
		record.ID,
		run.ID,
		version.ID,
		string(metadata),
	).Scan(&eventID); err != nil {
		return 0, fmt.Errorf("insert runner preview event: %w", err)
	}
	return eventID, nil
}

func insertRunnerPreviewAuditEvent(ctx context.Context, tx pgx.Tx, record Record, version WorkflowVersion, run RunRecord, task RunTaskRecord, report ArtifactRecord, options RunnerPreviewOptions, preflight RunnerPreflight) (int64, error) {
	metadata, err := json.Marshal(map[string]any{
		"display_label":       version.DisplayLabel,
		"workflow_version_id": version.ID,
		"run_id":              run.ID,
		"run_task_id":         task.ID,
		"artifact_id":         report.ID,
		"actor":               options.Actor,
		"risk_level":          options.RiskLevel,
		"risk_policy":         options.RiskPolicy,
		"dry_run":             true,
		"preflight_status":    preflight.Status,
		"checks":              preflight.Checks,
	})
	if err != nil {
		return 0, fmt.Errorf("marshal runner preview audit metadata: %w", err)
	}
	var auditEventID int64
	if err := tx.QueryRow(ctx, `
INSERT INTO audit_events (project_id, action, capability, resource_type, resource, decision, reason, metadata)
VALUES ($1, 'runner.preview', 'execute_agents', 'workflow_version', $2, 'allowed', $3, $4::jsonb)
RETURNING id`,
		record.ID,
		version.DisplayLabel,
		options.Reason,
		string(metadata),
	).Scan(&auditEventID); err != nil {
		return 0, fmt.Errorf("insert runner preview audit event: %w", err)
	}
	return auditEventID, nil
}

func runnerPreviewCommandResponse(result RunnerPreviewResult) map[string]any {
	taskIDs := make([]int64, 0, len(result.Tasks))
	for _, task := range result.Tasks {
		taskIDs = append(taskIDs, task.ID)
	}
	attemptIDs := make([]int64, 0, len(result.Attempts))
	for _, attempt := range result.Attempts {
		attemptIDs = append(attemptIDs, attempt.ID)
	}
	artifactIDs := make([]int64, 0, len(result.Artifacts))
	artifactType := ""
	artifactSHA256 := ""
	artifactSize := int64(0)
	for _, artifact := range result.Artifacts {
		artifactIDs = append(artifactIDs, artifact.ID)
		if artifactType == "" {
			artifactType = artifact.ArtifactType
			artifactSHA256 = artifact.SHA256
			artifactSize = artifact.SizeBytes
		}
	}
	return map[string]any{
		"project_id":                  result.Project.ID,
		"project_key":                 result.Project.Key,
		"workflow_version_id":         result.Version.ID,
		"display_label":               result.Version.DisplayLabel,
		"run_id":                      result.Run.ID,
		"run_type":                    result.Run.RunType,
		"run_status":                  result.Run.Status,
		"dry_run":                     result.Run.DryRun,
		"run_task_ids":                taskIDs,
		"attempt_ids":                 attemptIDs,
		"artifact_ids":                artifactIDs,
		"artifact_type":               artifactType,
		"artifact_sha256":             artifactSHA256,
		"artifact_size_bytes":         artifactSize,
		"preflight_status":            result.Preflight.Status,
		"event_id":                    result.EventID,
		"audit_event_id":              result.AuditEventID,
		"idempotency_key":             result.IdempotencyKey,
		"project_write_attempted":     false,
		"execution_write_attempted":   false,
		"engine_call_attempted":       false,
		"commands_run":                false,
		"secrets_resolved":            false,
		"network_used":                false,
		"area_matrix_write_attempted": false,
	}
}

func (s Store) loadRunnerPreviewByIdempotency(ctx context.Context, record Record, version WorkflowVersion, idempotencyKey string) (RunnerPreviewResult, error) {
	run, err := scanRun(s.pool.QueryRow(ctx, `
SELECT id, COALESCE(project_id, 0), COALESCE(workflow_version_id, 0), run_type,
       COALESCE(run_kind, ''), status, risk_level, risk_policy, dry_run,
       summary, metadata, started_at, finished_at
FROM runs
WHERE project_id = $1
  AND workflow_version_id = $2
  AND run_type = 'runner_preview'
  AND metadata->>'idempotency_key' = $3
ORDER BY started_at DESC, id DESC
LIMIT 1`,
		record.ID,
		version.ID,
		idempotencyKey,
	))
	if err != nil {
		return RunnerPreviewResult{}, fmt.Errorf("load existing runner preview run: %w", err)
	}
	tasks, err := s.listRunTasks(ctx, run.ID)
	if err != nil {
		return RunnerPreviewResult{}, err
	}
	attempts, err := s.listRunAttempts(ctx, run.ID)
	if err != nil {
		return RunnerPreviewResult{}, err
	}
	artifacts, err := s.listRunArtifacts(ctx, run.ID)
	if err != nil {
		return RunnerPreviewResult{}, err
	}
	options := RunnerPreviewOptions{
		IdempotencyKey: idempotencyKey,
		RiskLevel:      run.RiskLevel,
		RiskPolicy:     run.RiskPolicy,
	}
	preflight := EvaluateRunnerPreflight(record, version, normalizeRunnerPreviewOptions(record, version, options))
	return RunnerPreviewResult{
		Project:        record,
		Version:        version,
		Run:            run,
		Tasks:          tasks,
		Attempts:       attempts,
		Artifacts:      artifacts,
		Preflight:      preflight,
		Created:        false,
		IdempotencyKey: idempotencyKey,
	}, nil
}

func (s Store) GetRun(ctx context.Context, runID int64) (RunDetail, error) {
	if runID <= 0 {
		return RunDetail{}, fmt.Errorf("run id is required")
	}
	run, err := scanRun(s.pool.QueryRow(ctx, `
SELECT id, COALESCE(project_id, 0), COALESCE(workflow_version_id, 0), run_type,
       COALESCE(run_kind, ''), status, risk_level, risk_policy, dry_run,
       summary, metadata, started_at, finished_at
FROM runs
WHERE id = $1`,
		runID,
	))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return RunDetail{}, ErrRunNotFound
		}
		return RunDetail{}, fmt.Errorf("load run: %w", err)
	}
	tasks, err := s.listRunTasks(ctx, run.ID)
	if err != nil {
		return RunDetail{}, err
	}
	attempts, err := s.listRunAttempts(ctx, run.ID)
	if err != nil {
		return RunDetail{}, err
	}
	artifacts, err := s.listRunArtifacts(ctx, run.ID)
	if err != nil {
		return RunDetail{}, err
	}
	return RunDetail{
		Run:       run,
		Tasks:     tasks,
		Attempts:  attempts,
		Artifacts: artifacts,
	}, nil
}

func (s Store) ListWorkflowVersionRuns(ctx context.Context, record Record, version WorkflowVersion, limit int) ([]RunRecord, error) {
	if record.ID <= 0 {
		return nil, fmt.Errorf("project id is required")
	}
	if version.ID <= 0 {
		return nil, fmt.Errorf("workflow version id is required")
	}
	if limit <= 0 {
		limit = 20
	}
	rows, err := s.pool.Query(ctx, `
SELECT id, COALESCE(project_id, 0), COALESCE(workflow_version_id, 0), run_type,
       COALESCE(run_kind, ''), status, risk_level, risk_policy, dry_run,
       summary, metadata, started_at, finished_at
FROM runs
WHERE project_id = $1
  AND workflow_version_id = $2
ORDER BY started_at DESC, id DESC
LIMIT $3`,
		record.ID,
		version.ID,
		limit,
	)
	if err != nil {
		return nil, fmt.Errorf("list workflow version runs: %w", err)
	}
	defer rows.Close()
	runs := []RunRecord{}
	for rows.Next() {
		run, err := scanRun(rows)
		if err != nil {
			return nil, err
		}
		runs = append(runs, run)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate workflow version runs: %w", err)
	}
	return runs, nil
}

func (s Store) ListRunEvents(ctx context.Context, runID int64, limit int) ([]EventRecord, error) {
	if runID <= 0 {
		return nil, fmt.Errorf("run id is required")
	}
	if limit <= 0 {
		limit = 20
	}
	rows, err := s.pool.Query(ctx, `
SELECT id, COALESCE(project_id, 0), COALESCE(run_id, 0), COALESCE(workflow_version_id, 0),
       event_type, severity, message, metadata, created_at
FROM events
WHERE run_id = $1
ORDER BY created_at DESC, id DESC
LIMIT $2`,
		runID,
		limit,
	)
	if err != nil {
		return nil, fmt.Errorf("list run events: %w", err)
	}
	defer rows.Close()
	return scanEventRows(rows)
}

func (s Store) GetArtifact(ctx context.Context, artifactID int64) (ArtifactRecord, error) {
	if artifactID <= 0 {
		return ArtifactRecord{}, fmt.Errorf("artifact id is required")
	}
	record, err := scanArtifactRecordWithRun(s.pool.QueryRow(ctx, `
SELECT id, project_id, COALESCE(workflow_version_id, 0), COALESCE(run_id, 0), COALESCE(workflow_item_id, 0),
       artifact_type, storage_backend, uri, COALESCE(source_path, ''), COALESCE(sha256, ''),
       COALESCE(size_bytes, 0), COALESCE(content_type, ''), metadata, created_at
FROM artifacts
WHERE id = $1`,
		artifactID,
	))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ArtifactRecord{}, ErrArtifactNotFound
		}
		return ArtifactRecord{}, fmt.Errorf("load artifact: %w", err)
	}
	return record, nil
}

func (s Store) GetArtifactContent(ctx context.Context, artifactID int64) (ArtifactContent, error) {
	record, err := s.GetArtifact(ctx, artifactID)
	if err != nil {
		return ArtifactContent{}, err
	}
	content, primaryErr := readArtifactRecordContent(ctx, record)
	if primaryErr == nil {
		return content, nil
	}
	locations, err := s.artifactReadLocations(ctx, artifactID)
	if err != nil {
		return ArtifactContent{}, errors.Join(primaryErr, err)
	}
	for _, location := range locations {
		if location.StorageBackend == record.StorageBackend && location.URI == record.URI {
			continue
		}
		if location.Role == "migration_source" && artifactMetadataString(record.Metadata, "artifact_migration_status") != "observing" {
			continue
		}
		fallback := record
		fallback.StorageBackend = location.StorageBackend
		fallback.URI = location.URI
		fallback.SHA256 = location.SHA256
		fallback.SizeBytes = location.SizeBytes
		fallback.ContentType = location.ContentType
		content, err := readArtifactRecordContent(ctx, fallback)
		if err == nil {
			content.Artifact = record
			return content, nil
		}
		primaryErr = errors.Join(primaryErr, err)
	}
	return ArtifactContent{}, primaryErr
}

func ReadArtifactContent(record ArtifactRecord) (ArtifactContent, error) {
	return readArtifactRecordContent(context.Background(), record)
}

func readArtifactRecordContent(ctx context.Context, record ArtifactRecord) (ArtifactContent, error) {
	if record.StorageBackend != "local" && record.StorageBackend != "s3" && record.StorageBackend != "object" {
		return ArtifactContent{}, fmt.Errorf("%w: storage backend %q is metadata-only for content API", ErrArtifactContentUnavailable, record.StorageBackend)
	}
	if strings.TrimSpace(record.URI) == "" {
		return ArtifactContent{}, fmt.Errorf("%w: artifact URI is missing", ErrArtifactContentUnavailable)
	}
	content, err := artifact.ReadConfigured(ctx, record.StorageBackend, record.URI)
	if err != nil {
		return ArtifactContent{}, fmt.Errorf("%w: read artifact: %v", ErrArtifactContentUnavailable, err)
	}
	sum := sha256.Sum256(content)
	actualSHA := hex.EncodeToString(sum[:])
	if record.SHA256 != "" && record.SHA256 != actualSHA {
		return ArtifactContent{}, fmt.Errorf("%w: sha256 mismatch", ErrArtifactContentMismatch)
	}
	if record.SizeBytes > 0 && record.SizeBytes != int64(len(content)) {
		return ArtifactContent{}, fmt.Errorf("%w: size mismatch", ErrArtifactContentMismatch)
	}
	contentType := strings.TrimSpace(record.ContentType)
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	return ArtifactContent{Artifact: record, Content: content, ContentType: contentType}, nil
}

func (s Store) ListProjectArtifacts(ctx context.Context, record Record, limit int) ([]ArtifactRecord, error) {
	if record.ID <= 0 {
		return nil, fmt.Errorf("project id is required")
	}
	if limit <= 0 {
		limit = 20
	}
	rows, err := s.pool.Query(ctx, `
SELECT id, project_id, COALESCE(workflow_version_id, 0), COALESCE(run_id, 0), COALESCE(workflow_item_id, 0),
       artifact_type, storage_backend, uri, COALESCE(source_path, ''), COALESCE(sha256, ''),
       COALESCE(size_bytes, 0), COALESCE(content_type, ''), metadata, created_at
FROM artifacts
WHERE project_id = $1
ORDER BY created_at DESC, id DESC
LIMIT $2`,
		record.ID,
		limit,
	)
	if err != nil {
		return nil, fmt.Errorf("list project artifacts: %w", err)
	}
	defer rows.Close()
	artifacts := []ArtifactRecord{}
	for rows.Next() {
		record, err := scanArtifactRecordWithRun(rows)
		if err != nil {
			return nil, err
		}
		artifacts = append(artifacts, record)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate project artifacts: %w", err)
	}
	return artifacts, nil
}

func (s Store) ListWorkflowVersionArtifacts(ctx context.Context, record Record, version WorkflowVersion, limit int) ([]ArtifactRecord, error) {
	if record.ID <= 0 {
		return nil, fmt.Errorf("project id is required")
	}
	if version.ID <= 0 {
		return nil, fmt.Errorf("workflow version id is required")
	}
	if limit <= 0 {
		limit = 20
	}
	rows, err := s.pool.Query(ctx, `
SELECT id, project_id, COALESCE(workflow_version_id, 0), COALESCE(run_id, 0), COALESCE(workflow_item_id, 0),
       artifact_type, storage_backend, uri, COALESCE(source_path, ''), COALESCE(sha256, ''),
       COALESCE(size_bytes, 0), COALESCE(content_type, ''), metadata, created_at
FROM artifacts
WHERE project_id = $1
  AND workflow_version_id = $2
ORDER BY created_at DESC, id DESC
LIMIT $3`,
		record.ID,
		version.ID,
		limit,
	)
	if err != nil {
		return nil, fmt.Errorf("list workflow version artifacts: %w", err)
	}
	defer rows.Close()
	artifacts := []ArtifactRecord{}
	for rows.Next() {
		record, err := scanArtifactRecordWithRun(rows)
		if err != nil {
			return nil, err
		}
		artifacts = append(artifacts, record)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate workflow version artifacts: %w", err)
	}
	return artifacts, nil
}

func (s Store) listRunArtifacts(ctx context.Context, runID int64) ([]ArtifactRecord, error) {
	rows, err := s.pool.Query(ctx, `
SELECT id, project_id, COALESCE(workflow_version_id, 0), COALESCE(run_id, 0), COALESCE(workflow_item_id, 0),
       artifact_type, storage_backend, uri, COALESCE(source_path, ''), COALESCE(sha256, ''),
       COALESCE(size_bytes, 0), COALESCE(content_type, ''), metadata, created_at
FROM artifacts
WHERE run_id = $1
ORDER BY created_at ASC, id ASC`,
		runID,
	)
	if err != nil {
		return nil, fmt.Errorf("list run artifacts: %w", err)
	}
	defer rows.Close()
	artifacts := []ArtifactRecord{}
	for rows.Next() {
		record, err := scanArtifactRecordWithRun(rows)
		if err != nil {
			return nil, err
		}
		artifacts = append(artifacts, record)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate run artifacts: %w", err)
	}
	return artifacts, nil
}

func (s Store) listRunTasks(ctx context.Context, runID int64) ([]RunTaskRecord, error) {
	rows, err := s.pool.Query(ctx, `
SELECT id, project_id, COALESCE(workflow_version_id, 0), COALESCE(workflow_item_id, 0),
       run_id, task_key, task_kind, status, risk_level, sequence, metadata,
       created_at, updated_at
FROM run_tasks
WHERE run_id = $1
ORDER BY sequence ASC, id ASC`,
		runID,
	)
	if err != nil {
		return nil, fmt.Errorf("list run tasks: %w", err)
	}
	defer rows.Close()
	tasks := []RunTaskRecord{}
	for rows.Next() {
		task, err := scanRunTask(rows)
		if err != nil {
			return nil, err
		}
		tasks = append(tasks, task)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate run tasks: %w", err)
	}
	return tasks, nil
}

func (s Store) listRunAttempts(ctx context.Context, runID int64) ([]RunAttemptRecord, error) {
	rows, err := s.pool.Query(ctx, `
SELECT id, project_id, COALESCE(workflow_version_id, 0), COALESCE(workflow_item_id, 0),
       run_id, COALESCE(run_task_id, 0), attempt_kind, status, dry_run,
       metadata, started_at, finished_at
FROM run_attempts
WHERE run_id = $1
ORDER BY started_at ASC, id ASC`,
		runID,
	)
	if err != nil {
		return nil, fmt.Errorf("list run attempts: %w", err)
	}
	defer rows.Close()
	attempts := []RunAttemptRecord{}
	for rows.Next() {
		attempt, err := scanRunAttempt(rows)
		if err != nil {
			return nil, err
		}
		attempts = append(attempts, attempt)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate run attempts: %w", err)
	}
	return attempts, nil
}

func scanRun(row scanner) (RunRecord, error) {
	var run RunRecord
	var summaryRaw []byte
	var metadataRaw []byte
	if err := row.Scan(
		&run.ID,
		&run.ProjectID,
		&run.WorkflowVersionID,
		&run.RunType,
		&run.RunKind,
		&run.Status,
		&run.RiskLevel,
		&run.RiskPolicy,
		&run.DryRun,
		&summaryRaw,
		&metadataRaw,
		&run.StartedAt,
		&run.FinishedAt,
	); err != nil {
		return RunRecord{}, err
	}
	if err := json.Unmarshal(summaryRaw, &run.Summary); err != nil {
		return RunRecord{}, fmt.Errorf("parse run summary: %w", err)
	}
	if err := json.Unmarshal(metadataRaw, &run.Metadata); err != nil {
		return RunRecord{}, fmt.Errorf("parse run metadata: %w", err)
	}
	return run, nil
}

func scanRunTask(row scanner) (RunTaskRecord, error) {
	var task RunTaskRecord
	var metadataRaw []byte
	if err := row.Scan(
		&task.ID,
		&task.ProjectID,
		&task.WorkflowVersionID,
		&task.WorkflowItemID,
		&task.RunID,
		&task.TaskKey,
		&task.TaskKind,
		&task.Status,
		&task.RiskLevel,
		&task.Sequence,
		&metadataRaw,
		&task.CreatedAt,
		&task.UpdatedAt,
	); err != nil {
		return RunTaskRecord{}, err
	}
	if err := json.Unmarshal(metadataRaw, &task.Metadata); err != nil {
		return RunTaskRecord{}, fmt.Errorf("parse run task metadata: %w", err)
	}
	return task, nil
}

func scanRunAttempt(row scanner) (RunAttemptRecord, error) {
	var attempt RunAttemptRecord
	var metadataRaw []byte
	if err := row.Scan(
		&attempt.ID,
		&attempt.ProjectID,
		&attempt.WorkflowVersionID,
		&attempt.WorkflowItemID,
		&attempt.RunID,
		&attempt.RunTaskID,
		&attempt.AttemptKind,
		&attempt.Status,
		&attempt.DryRun,
		&metadataRaw,
		&attempt.StartedAt,
		&attempt.FinishedAt,
	); err != nil {
		return RunAttemptRecord{}, err
	}
	if err := json.Unmarshal(metadataRaw, &attempt.Metadata); err != nil {
		return RunAttemptRecord{}, fmt.Errorf("parse run attempt metadata: %w", err)
	}
	return attempt, nil
}

func runnerVersionMessage(version WorkflowVersion) string {
	if version.ImportMode == "authored" {
		return "workflow version is authored by AreaFlow"
	}
	return "runner preview requires an AreaFlow-authored workflow version"
}
