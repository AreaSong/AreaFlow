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

	"github.com/jackc/pgx/v5"
)

var ErrApprovedArtifactWriteBlocked = errors.New("approved artifact write blocked")

const (
	approvedArtifactWriteQueueCommandType = "run.approved_artifact_write_queue"
	approvedArtifactWriteApplyCommandType = "worker.approved_artifact_write"
)

type ApprovedArtifactWriteQueueOptions struct {
	ArtifactLabel  string
	IdempotencyKey string
	Actor          string
	Reason         string
}

type ApprovedArtifactWriteQueueResult struct {
	Project                       Record
	Version                       WorkflowVersion
	Run                           RunRecord
	Task                          RunTaskRecord
	ArtifactLabel                 string
	Created                       bool
	IdempotencyKey                string
	EventID                       int64
	AuditEventID                  int64
	ProjectReadAttempted          bool
	ProjectWriteAttempted         bool
	ExecutionWriteAttempted       bool
	AreaFlowArtifactWritten       bool
	AreaFlowExecutionStateWritten bool
	EngineCallAttempted           bool
	CommandsRun                   bool
	SecretsResolved               bool
	NetworkUsed                   bool
}

type ApprovedArtifactWriteOptions struct {
	WorkerKey           string
	RunID               int64
	AllowedCapabilities []string
	LeaseTimeoutSeconds int
	Metadata            map[string]any
	IdempotencyKey      string
	Actor               string
	Reason              string
}

type ApprovedArtifactWriteResult struct {
	Project                       Record
	Version                       WorkflowVersion
	Run                           RunRecord
	Worker                        WorkerRecord
	Lease                         LeaseRecord
	Task                          RunTaskRecord
	Attempt                       RunAttemptRecord
	Artifact                      ArtifactRecord
	Gate                          ExecutionApprovalGate
	ArtifactLabel                 string
	Status                        string
	Decision                      string
	Message                       string
	Blockers                      []string
	Created                       bool
	IdempotencyKey                string
	EventID                       int64
	AuditEventID                  int64
	ProjectReadAttempted          bool
	ProjectWriteAttempted         bool
	ExecutionWriteAttempted       bool
	AreaFlowArtifactWritten       bool
	AreaFlowExecutionStateWritten bool
	EngineCallAttempted           bool
	CommandsRun                   bool
	SecretsResolved               bool
	NetworkUsed                   bool
	TaskClaimed                   bool
	WorkerStarted                 bool
	LeaseCreated                  bool
	AttemptCreated                bool
	ArtifactCreated               bool
	ArtifactWritePassed           bool
}

func (s Store) QueueApprovedArtifactWrite(ctx context.Context, record Record, label string, options ApprovedArtifactWriteQueueOptions) (ApprovedArtifactWriteQueueResult, error) {
	version, err := s.GetWorkflowVersion(ctx, record, label)
	if err != nil {
		return ApprovedArtifactWriteQueueResult{}, err
	}
	if version.ImportMode != "authored" {
		return ApprovedArtifactWriteQueueResult{}, fmt.Errorf("%w: %s", ErrWorkflowVersionNotAuthored, label)
	}
	options = normalizeApprovedArtifactWriteQueueOptions(record, version, options)
	requestHash, err := approvedArtifactWriteQueueRequestHash(record, version, options)
	if err != nil {
		return ApprovedArtifactWriteQueueResult{}, err
	}
	if options.IdempotencyKey == "" {
		options.IdempotencyKey = approvedArtifactWriteQueueIdempotencyKey(record, version, requestHash)
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return ApprovedArtifactWriteQueueResult{}, fmt.Errorf("begin approved artifact write queue: %w", err)
	}
	defer tx.Rollback(ctx)

	created, err := reserveCommandRequest(ctx, tx, record.ID, approvedArtifactWriteQueueCommandType, options.IdempotencyKey, requestHash)
	if err != nil {
		return ApprovedArtifactWriteQueueResult{}, err
	}
	if !created {
		result, err := loadApprovedArtifactWriteQueueByCommandResponse(ctx, tx, record, version, options.IdempotencyKey)
		if err != nil {
			return ApprovedArtifactWriteQueueResult{}, err
		}
		if err := tx.Commit(ctx); err != nil {
			return ApprovedArtifactWriteQueueResult{}, fmt.Errorf("commit approved artifact write queue replay: %w", err)
		}
		result.Created = false
		return result, nil
	}

	run, err := insertApprovedArtifactWriteRun(ctx, tx, record, version, options)
	if err != nil {
		return ApprovedArtifactWriteQueueResult{}, err
	}
	task, err := insertApprovedArtifactWriteTask(ctx, tx, record, version, run, options)
	if err != nil {
		return ApprovedArtifactWriteQueueResult{}, err
	}
	result := ApprovedArtifactWriteQueueResult{
		Project:                       record,
		Version:                       version,
		Run:                           run,
		Task:                          task,
		ArtifactLabel:                 options.ArtifactLabel,
		Created:                       true,
		IdempotencyKey:                options.IdempotencyKey,
		ProjectReadAttempted:          false,
		ProjectWriteAttempted:         false,
		ExecutionWriteAttempted:       false,
		AreaFlowArtifactWritten:       false,
		AreaFlowExecutionStateWritten: true,
		EngineCallAttempted:           false,
		CommandsRun:                   false,
		SecretsResolved:               false,
		NetworkUsed:                   false,
	}
	eventID, err := insertApprovedArtifactWriteQueueEvent(ctx, tx, result, options)
	if err != nil {
		return ApprovedArtifactWriteQueueResult{}, err
	}
	result.EventID = eventID
	auditEventID, err := insertApprovedArtifactWriteQueueAuditEvent(ctx, tx, result, options)
	if err != nil {
		return ApprovedArtifactWriteQueueResult{}, err
	}
	result.AuditEventID = auditEventID
	if err := completeCommandRequestResponse(ctx, tx, record.ID, approvedArtifactWriteQueueCommandType, options.IdempotencyKey, approvedArtifactWriteQueueCommandResponse(result)); err != nil {
		return ApprovedArtifactWriteQueueResult{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return ApprovedArtifactWriteQueueResult{}, fmt.Errorf("commit approved artifact write queue: %w", err)
	}
	return result, nil
}

func (s Store) WriteApprovedArtifact(ctx context.Context, record Record, options ApprovedArtifactWriteOptions) (ApprovedArtifactWriteResult, error) {
	if options.RunID <= 0 {
		return ApprovedArtifactWriteResult{}, fmt.Errorf("run id is required")
	}
	options = normalizeApprovedArtifactWriteOptions(options)
	requestHash, err := approvedArtifactWriteRequestHash(record, options)
	if err != nil {
		return ApprovedArtifactWriteResult{}, err
	}
	if options.IdempotencyKey == "" {
		options.IdempotencyKey = approvedArtifactWriteIdempotencyKey(record, options, requestHash)
	}

	gate, err := s.ExecutionApprovalGate(ctx, options.RunID, ExecutionApprovalGateOptions{
		RequiredCapabilities: options.AllowedCapabilities,
		SkipEnginePreview:    true,
		Mode:                 "approved_artifact_write_gate",
	})
	if err != nil {
		return ApprovedArtifactWriteResult{}, err
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return ApprovedArtifactWriteResult{}, fmt.Errorf("begin approved artifact write: %w", err)
	}
	defer tx.Rollback(ctx)

	run, err := loadRunForUpdate(ctx, tx, options.RunID)
	if err != nil {
		return ApprovedArtifactWriteResult{}, err
	}
	if run.ProjectID != record.ID {
		return ApprovedArtifactWriteResult{}, fmt.Errorf("%w: run %d does not belong to project %s", ErrRunNotFound, options.RunID, record.Key)
	}
	version, err := workflowVersionByIDTx(ctx, tx, record.ID, run.WorkflowVersionID)
	if err != nil {
		return ApprovedArtifactWriteResult{}, err
	}
	worker, err := loadWorkerForUpdate(ctx, tx, record.ID, options.WorkerKey)
	if err != nil {
		return ApprovedArtifactWriteResult{}, err
	}

	created, err := reserveCommandRequest(ctx, tx, record.ID, approvedArtifactWriteApplyCommandType, options.IdempotencyKey, requestHash)
	if err != nil {
		return ApprovedArtifactWriteResult{}, err
	}
	if !created {
		result, err := loadApprovedArtifactWriteByCommandResponse(ctx, tx, record, version, gate, options.IdempotencyKey)
		if err != nil {
			return ApprovedArtifactWriteResult{}, err
		}
		if err := tx.Commit(ctx); err != nil {
			return ApprovedArtifactWriteResult{}, fmt.Errorf("commit approved artifact write replay: %w", err)
		}
		result.Created = false
		return result, nil
	}

	if gate.Status != "pass" {
		result := deniedApprovedArtifactWriteResult(record, version, run, worker, gate, options, "approved artifact write gate blocked", gate.Blockers)
		if err := finishDeniedApprovedArtifactWrite(ctx, tx, result, options); err != nil {
			return ApprovedArtifactWriteResult{}, err
		}
		if err := tx.Commit(ctx); err != nil {
			return ApprovedArtifactWriteResult{}, fmt.Errorf("commit blocked approved artifact write: %w", err)
		}
		return result, fmt.Errorf("%w: %s", ErrApprovedArtifactWriteBlocked, strings.Join(result.Blockers, "; "))
	}
	if missing := missingWorkerCapabilities(worker.Capabilities, options.AllowedCapabilities); len(missing) > 0 {
		blockers := []string{"worker missing required capabilities: " + strings.Join(missing, ",")}
		result := deniedApprovedArtifactWriteResult(record, version, run, worker, gate, options, "worker capability denied", blockers)
		if err := finishDeniedApprovedArtifactWrite(ctx, tx, result, options); err != nil {
			return ApprovedArtifactWriteResult{}, err
		}
		if err := tx.Commit(ctx); err != nil {
			return ApprovedArtifactWriteResult{}, fmt.Errorf("commit denied approved artifact write: %w", err)
		}
		return result, fmt.Errorf("%w: missing %s", ErrWorkerCapabilityDenied, strings.Join(missing, ","))
	}
	allowed, reason, err := canProjectCapabilityInTx(ctx, tx, record.ID, "write_artifacts")
	if err != nil {
		return ApprovedArtifactWriteResult{}, err
	}
	if !allowed {
		blockers := []string{"project write_artifacts capability denied: " + reason}
		result := deniedApprovedArtifactWriteResult(record, version, run, worker, gate, options, "project artifact write denied", blockers)
		if err := finishDeniedApprovedArtifactWrite(ctx, tx, result, options); err != nil {
			return ApprovedArtifactWriteResult{}, err
		}
		if err := tx.Commit(ctx); err != nil {
			return ApprovedArtifactWriteResult{}, fmt.Errorf("commit approved artifact write permission denied: %w", err)
		}
		return result, fmt.Errorf("%w: %s", ErrApprovedArtifactWriteBlocked, strings.Join(blockers, "; "))
	}

	task, ok, err := nextApprovedArtifactWriteTaskForLease(ctx, tx, record.ID, run.ID)
	if err != nil {
		return ApprovedArtifactWriteResult{}, err
	}
	if !ok {
		result := deniedApprovedArtifactWriteResult(record, version, run, worker, gate, options, "no queued approved artifact write task", []string{"no queued or needs_recovery approved artifact write task is available"})
		if err := finishDeniedApprovedArtifactWrite(ctx, tx, result, options); err != nil {
			return ApprovedArtifactWriteResult{}, err
		}
		if err := tx.Commit(ctx); err != nil {
			return ApprovedArtifactWriteResult{}, fmt.Errorf("commit idle approved artifact write: %w", err)
		}
		return result, fmt.Errorf("%w: no queued task", ErrNoLeaseAvailable)
	}
	artifactLabel := metadataString(task.Metadata, "artifact_label")
	if artifactLabel == "" {
		artifactLabel = "approved-artifact"
	}

	lease, err := insertLeaseForTask(ctx, tx, record.ID, worker, task, "approved_artifact_write", options.AllowedCapabilities, map[string]any{
		"run_id":                  task.RunID,
		"run_task_id":             task.ID,
		"task_key":                task.TaskKey,
		"task_kind":               task.TaskKind,
		"artifact_label":          artifactLabel,
		"approved_artifact_write": true,
		"approval_gated":          true,
	}, options.Metadata, options.LeaseTimeoutSeconds)
	if err != nil {
		return ApprovedArtifactWriteResult{}, err
	}
	if err := updateRunTaskStatus(ctx, tx, task.ID, "leased"); err != nil {
		return ApprovedArtifactWriteResult{}, err
	}
	artifact, err := writeAndInsertApprovedArtifact(ctx, tx, record, version, run, worker, task, lease, gate, artifactLabel, options)
	if err != nil {
		return ApprovedArtifactWriteResult{}, err
	}
	attempt, err := insertApprovedArtifactWriteAttempt(ctx, tx, record, task, lease, artifact, artifactLabel, options)
	if err != nil {
		return ApprovedArtifactWriteResult{}, err
	}
	released, err := releaseLeaseForWorker(ctx, tx, record.ID, worker, lease.ID, "completed", map[string]any{
		"approved_artifact_write": true,
		"artifact_label":          artifactLabel,
		"attempt_id":              attempt.ID,
		"artifact_id":             artifact.ID,
	})
	if err != nil {
		return ApprovedArtifactWriteResult{}, err
	}
	if err := updateRunTaskStatus(ctx, tx, task.ID, "artifact_written"); err != nil {
		return ApprovedArtifactWriteResult{}, err
	}
	task.Status = "artifact_written"
	run, err = updateApprovedArtifactWriteRunAfterTask(ctx, tx, run, options, artifact.ID, attempt.ID)
	if err != nil {
		return ApprovedArtifactWriteResult{}, err
	}
	result := ApprovedArtifactWriteResult{
		Project:                       record,
		Version:                       version,
		Run:                           run,
		Worker:                        worker,
		Lease:                         released,
		Task:                          task,
		Attempt:                       attempt,
		Artifact:                      artifact,
		Gate:                          gate,
		ArtifactLabel:                 artifactLabel,
		Status:                        "artifact_written",
		Decision:                      "allowed",
		Message:                       "approved artifact write completed in AreaFlow artifact store only",
		Created:                       true,
		IdempotencyKey:                options.IdempotencyKey,
		ProjectReadAttempted:          false,
		ProjectWriteAttempted:         false,
		ExecutionWriteAttempted:       false,
		AreaFlowArtifactWritten:       true,
		AreaFlowExecutionStateWritten: true,
		EngineCallAttempted:           false,
		CommandsRun:                   false,
		SecretsResolved:               false,
		NetworkUsed:                   false,
		TaskClaimed:                   true,
		WorkerStarted:                 false,
		LeaseCreated:                  true,
		AttemptCreated:                true,
		ArtifactCreated:               true,
		ArtifactWritePassed:           true,
	}
	eventID, err := insertApprovedArtifactWriteEvent(ctx, tx, result, options)
	if err != nil {
		return ApprovedArtifactWriteResult{}, err
	}
	result.EventID = eventID
	auditEventID, err := insertApprovedArtifactWriteAuditEvent(ctx, tx, result, options)
	if err != nil {
		return ApprovedArtifactWriteResult{}, err
	}
	result.AuditEventID = auditEventID
	if err := completeCommandRequestResponse(ctx, tx, record.ID, approvedArtifactWriteApplyCommandType, options.IdempotencyKey, approvedArtifactWriteCommandResponse(result)); err != nil {
		return ApprovedArtifactWriteResult{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return ApprovedArtifactWriteResult{}, fmt.Errorf("commit approved artifact write: %w", err)
	}
	return result, nil
}

func normalizeApprovedArtifactWriteQueueOptions(record Record, version WorkflowVersion, options ApprovedArtifactWriteQueueOptions) ApprovedArtifactWriteQueueOptions {
	options.ArtifactLabel = normalizeApprovedArtifactLabel(options.ArtifactLabel)
	options.IdempotencyKey = strings.TrimSpace(options.IdempotencyKey)
	options.Actor = strings.TrimSpace(options.Actor)
	options.Reason = strings.TrimSpace(options.Reason)
	if options.Actor == "" {
		options.Actor = "local-user"
	}
	if options.Reason == "" {
		options.Reason = "queue approved artifact write run"
	}
	if options.IdempotencyKey == "" {
		hash, err := approvedArtifactWriteQueueRequestHash(record, version, options)
		if err == nil {
			options.IdempotencyKey = approvedArtifactWriteQueueIdempotencyKey(record, version, hash)
		}
	}
	return options
}

func normalizeApprovedArtifactWriteOptions(options ApprovedArtifactWriteOptions) ApprovedArtifactWriteOptions {
	options.WorkerKey = strings.TrimSpace(options.WorkerKey)
	options.IdempotencyKey = strings.TrimSpace(options.IdempotencyKey)
	options.Actor = strings.TrimSpace(options.Actor)
	options.Reason = strings.TrimSpace(options.Reason)
	if len(options.AllowedCapabilities) == 0 {
		options.AllowedCapabilities = []string{"write_artifacts"}
	}
	options.AllowedCapabilities = normalizeCapabilityList(options.AllowedCapabilities)
	if options.LeaseTimeoutSeconds <= 0 {
		options.LeaseTimeoutSeconds = 300
	}
	if options.Metadata == nil {
		options.Metadata = map[string]any{}
	}
	if options.Actor == "" {
		options.Actor = "local-user"
	}
	if options.Reason == "" {
		options.Reason = "approval-gated approved artifact write"
	}
	return options
}

func normalizeApprovedArtifactLabel(label string) string {
	label = strings.TrimSpace(label)
	if label == "" {
		return "approved-artifact"
	}
	var out strings.Builder
	lastDash := false
	for i := 0; i < len(label); i++ {
		ch := label[i]
		if (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9') || ch == '_' || ch == '.' {
			out.WriteByte(ch)
			lastDash = false
			continue
		}
		if ch == '-' {
			if !lastDash {
				out.WriteByte(ch)
				lastDash = true
			}
			continue
		}
		if !lastDash {
			out.WriteByte('-')
			lastDash = true
		}
	}
	normalized := strings.Trim(out.String(), "-.")
	if normalized == "" {
		return "approved-artifact"
	}
	if len(normalized) > 80 {
		normalized = strings.Trim(normalized[:80], "-.")
	}
	if normalized == "" {
		return "approved-artifact"
	}
	return normalized
}

func approvedArtifactWriteQueueRequestHash(record Record, version WorkflowVersion, options ApprovedArtifactWriteQueueOptions) (string, error) {
	payload := map[string]any{
		"command_type":   approvedArtifactWriteQueueCommandType,
		"project_id":     record.ID,
		"project_key":    record.Key,
		"version_id":     version.ID,
		"display_label":  version.DisplayLabel,
		"artifact_label": options.ArtifactLabel,
		"actor":          options.Actor,
		"reason":         options.Reason,
		"artifact_write": true,
	}
	content, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("marshal approved artifact write queue request hash payload: %w", err)
	}
	sum := sha256.Sum256(content)
	return hex.EncodeToString(sum[:]), nil
}

func approvedArtifactWriteRequestHash(record Record, options ApprovedArtifactWriteOptions) (string, error) {
	payload := map[string]any{
		"command_type":          approvedArtifactWriteApplyCommandType,
		"project_id":            record.ID,
		"project_key":           record.Key,
		"worker_key":            options.WorkerKey,
		"run_id":                options.RunID,
		"allowed_capabilities":  options.AllowedCapabilities,
		"lease_timeout_seconds": options.LeaseTimeoutSeconds,
		"metadata":              options.Metadata,
		"actor":                 options.Actor,
		"reason":                options.Reason,
		"artifact_write":        true,
	}
	content, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("marshal approved artifact write request hash payload: %w", err)
	}
	sum := sha256.Sum256(content)
	return hex.EncodeToString(sum[:]), nil
}

func approvedArtifactWriteQueueIdempotencyKey(record Record, version WorkflowVersion, requestHash string) string {
	return fmt.Sprintf("%s:%s:%s:%s", approvedArtifactWriteQueueCommandType, record.Key, version.DisplayLabel, commandHashPrefix(requestHash))
}

func approvedArtifactWriteIdempotencyKey(record Record, options ApprovedArtifactWriteOptions, requestHash string) string {
	return fmt.Sprintf("%s:%s:%s:%d:%s", approvedArtifactWriteApplyCommandType, record.Key, options.WorkerKey, options.RunID, commandHashPrefix(requestHash))
}

func insertApprovedArtifactWriteRun(ctx context.Context, tx pgx.Tx, record Record, version WorkflowVersion, options ApprovedArtifactWriteQueueOptions) (RunRecord, error) {
	summary, err := json.Marshal(map[string]any{
		"approved_artifact_write":       true,
		"approval_gated":                true,
		"artifact_label":                options.ArtifactLabel,
		"project_read_attempted":        false,
		"project_write_attempted":       false,
		"execution_write_attempted":     false,
		"area_flow_artifact_written":    false,
		"engine_call_attempted":         false,
		"commands_run":                  false,
		"secrets_resolved":              false,
		"network_used":                  false,
		"area_flow_execution_state":     "queued",
		"area_flow_execution_state_set": true,
	})
	if err != nil {
		return RunRecord{}, fmt.Errorf("marshal approved artifact write run summary: %w", err)
	}
	metadata, err := json.Marshal(map[string]any{
		"phase":                   "v0.6k",
		"approved_artifact_write": true,
		"approval_gated":          true,
		"artifact_label":          options.ArtifactLabel,
		"actor":                   options.Actor,
		"reason":                  options.Reason,
	})
	if err != nil {
		return RunRecord{}, fmt.Errorf("marshal approved artifact write run metadata: %w", err)
	}
	run, err := scanRun(tx.QueryRow(ctx, `
INSERT INTO runs (project_id, workflow_version_id, run_type, run_kind, status, risk_level, risk_policy, dry_run, summary, metadata)
VALUES ($1, $2, 'approved_artifact_write', 'execution', 'queued', 'low', 'pause', false, $3::jsonb, $4::jsonb)
RETURNING id, COALESCE(project_id, 0), COALESCE(workflow_version_id, 0), run_type,
          COALESCE(run_kind, ''), status, risk_level, risk_policy, dry_run,
          summary, metadata, started_at, finished_at`,
		record.ID,
		version.ID,
		string(summary),
		string(metadata),
	))
	if err != nil {
		return RunRecord{}, fmt.Errorf("insert approved artifact write run: %w", err)
	}
	return run, nil
}

func insertApprovedArtifactWriteTask(ctx context.Context, tx pgx.Tx, record Record, version WorkflowVersion, run RunRecord, options ApprovedArtifactWriteQueueOptions) (RunTaskRecord, error) {
	metadata, err := json.Marshal(map[string]any{
		"approved_artifact_write": true,
		"approval_gated":          true,
		"artifact_label":          options.ArtifactLabel,
		"actor":                   options.Actor,
		"reason":                  options.Reason,
	})
	if err != nil {
		return RunTaskRecord{}, fmt.Errorf("marshal approved artifact write task metadata: %w", err)
	}
	task, err := scanRunTask(tx.QueryRow(ctx, `
INSERT INTO run_tasks (
    project_id, workflow_version_id, run_id, task_key, task_kind, status, risk_level, sequence, metadata
)
VALUES ($1, $2, $3, $4, 'approved_artifact_write_task', 'queued', 'low', 1, $5::jsonb)
RETURNING id, project_id, COALESCE(workflow_version_id, 0), COALESCE(workflow_item_id, 0),
          run_id, task_key, task_kind, status, risk_level, sequence, metadata,
          created_at, updated_at`,
		record.ID,
		version.ID,
		run.ID,
		version.DisplayLabel+":approved-artifact-write:"+options.ArtifactLabel,
		string(metadata),
	))
	if err != nil {
		return RunTaskRecord{}, fmt.Errorf("insert approved artifact write task: %w", err)
	}
	return task, nil
}

func nextApprovedArtifactWriteTaskForLease(ctx context.Context, tx pgx.Tx, projectID int64, runID int64) (RunTaskRecord, bool, error) {
	task, err := scanRunTask(tx.QueryRow(ctx, `
SELECT rt.id, rt.project_id, COALESCE(rt.workflow_version_id, 0), COALESCE(rt.workflow_item_id, 0),
       rt.run_id, rt.task_key, rt.task_kind, rt.status, rt.risk_level, rt.sequence, rt.metadata,
       rt.created_at, rt.updated_at
FROM run_tasks rt
JOIN runs r ON r.id = rt.run_id
WHERE rt.project_id = $1
  AND rt.run_id = $2
  AND r.dry_run = false
  AND r.run_kind = 'execution'
  AND r.status = 'queued'
  AND rt.task_kind = 'approved_artifact_write_task'
  AND rt.status IN ('queued', 'needs_recovery')
  AND NOT EXISTS (
      SELECT 1
      FROM leases l
      WHERE l.run_task_id = rt.id
        AND l.status = 'active'
  )
ORDER BY rt.sequence ASC, rt.id ASC
LIMIT 1
FOR UPDATE SKIP LOCKED`,
		projectID,
		runID,
	))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return RunTaskRecord{}, false, nil
		}
		return RunTaskRecord{}, false, fmt.Errorf("load next approved artifact write task: %w", err)
	}
	return task, true, nil
}

func writeAndInsertApprovedArtifact(ctx context.Context, tx pgx.Tx, record Record, version WorkflowVersion, run RunRecord, worker WorkerRecord, task RunTaskRecord, lease LeaseRecord, gate ExecutionApprovalGate, artifactLabel string, options ApprovedArtifactWriteOptions) (ArtifactRecord, error) {
	content, err := json.MarshalIndent(map[string]any{
		"project":                           record.Key,
		"workflow_version":                  version.DisplayLabel,
		"run_id":                            run.ID,
		"run_task_id":                       task.ID,
		"task_key":                          task.TaskKey,
		"task_kind":                         task.TaskKind,
		"worker_id":                         worker.ID,
		"worker_key":                        worker.WorkerKey,
		"lease_id":                          lease.ID,
		"artifact_label":                    artifactLabel,
		"approved_artifact_write":           true,
		"approval_gated":                    true,
		"execution_gate_status":             gate.Status,
		"project_read_attempted":            false,
		"project_write_attempted":           false,
		"execution_write_attempted":         false,
		"area_flow_artifact_written":        true,
		"area_flow_execution_state_written": true,
		"engine_call_attempted":             false,
		"commands_run":                      false,
		"secrets_resolved":                  false,
		"network_used":                      false,
		"allowed_capabilities":              options.AllowedCapabilities,
		"generated_at":                      time.Now().UTC().Format(time.RFC3339),
	}, "", "  ")
	if err != nil {
		return ArtifactRecord{}, fmt.Errorf("marshal approved artifact write report: %w", err)
	}
	relativePath := filepath.Join("versions", version.DisplayLabel, "approved-artifact-write", fmt.Sprintf("run-%d-task-%d-%s.json", run.ID, task.ID, artifactLabel))
	stored, err := writeProjectArtifact(record, relativePath, content, "application/json")
	if err != nil {
		return ArtifactRecord{}, err
	}
	metadata, err := json.Marshal(map[string]any{
		"phase":                   "v0.6k",
		"owned_by":                "areaflow",
		"approved_artifact_write": true,
		"approval_gated":          true,
		"artifact_label":          artifactLabel,
		"worker_id":               worker.ID,
		"worker_key":              worker.WorkerKey,
		"run_id":                  run.ID,
		"run_task_id":             task.ID,
		"lease_id":                lease.ID,
		"actor":                   options.Actor,
		"reason":                  options.Reason,
		"execution_gate_status":   gate.Status,
	})
	if err != nil {
		return ArtifactRecord{}, fmt.Errorf("marshal approved artifact write artifact metadata: %w", err)
	}
	return insertRunArtifactRecord(ctx, tx, record.ID, version.ID, run.ID, task.WorkflowItemID, "approved_artifact_write_report", relativePath, stored, string(metadata))
}

func insertApprovedArtifactWriteAttempt(ctx context.Context, tx pgx.Tx, record Record, task RunTaskRecord, lease LeaseRecord, report ArtifactRecord, artifactLabel string, options ApprovedArtifactWriteOptions) (RunAttemptRecord, error) {
	metadata, err := json.Marshal(map[string]any{
		"actor":                             options.Actor,
		"reason":                            options.Reason,
		"approved_artifact_write":           true,
		"approval_gated":                    true,
		"artifact_label":                    artifactLabel,
		"project_read_attempted":            false,
		"project_write_attempted":           false,
		"execution_write_attempted":         false,
		"area_flow_artifact_written":        true,
		"area_flow_execution_state_written": true,
		"engine_call_attempted":             false,
		"commands_run":                      false,
		"secrets_resolved":                  false,
		"network_used":                      false,
		"lease_id":                          lease.ID,
		"worker_id":                         lease.WorkerID,
		"evidence_artifact_id":              report.ID,
	})
	if err != nil {
		return RunAttemptRecord{}, fmt.Errorf("marshal approved artifact write attempt metadata: %w", err)
	}
	attempt, err := scanRunAttempt(tx.QueryRow(ctx, `
INSERT INTO run_attempts (
    project_id, workflow_version_id, workflow_item_id, run_id, run_task_id,
    attempt_kind, status, dry_run, finished_at, metadata
)
VALUES ($1, $2, $3, $4, $5, 'approved_artifact_write', 'passed', false, now(), $6::jsonb)
RETURNING id, project_id, COALESCE(workflow_version_id, 0), COALESCE(workflow_item_id, 0),
          run_id, COALESCE(run_task_id, 0), attempt_kind, status, dry_run,
          metadata, started_at, finished_at`,
		record.ID,
		nullableInt64(task.WorkflowVersionID),
		nullableInt64(task.WorkflowItemID),
		task.RunID,
		task.ID,
		string(metadata),
	))
	if err != nil {
		return RunAttemptRecord{}, fmt.Errorf("insert approved artifact write attempt: %w", err)
	}
	return attempt, nil
}

func updateApprovedArtifactWriteRunAfterTask(ctx context.Context, tx pgx.Tx, run RunRecord, options ApprovedArtifactWriteOptions, artifactID int64, attemptID int64) (RunRecord, error) {
	summary := copyMap(run.Summary)
	summary["approved_artifact_write"] = true
	summary["approval_gated"] = true
	summary["last_artifact_id"] = artifactID
	summary["last_attempt_id"] = attemptID
	summary["project_read_attempted"] = false
	summary["project_write_attempted"] = false
	summary["execution_write_attempted"] = false
	summary["area_flow_artifact_written"] = true
	summary["area_flow_execution_state_written"] = true
	summary["engine_call_attempted"] = false
	summary["commands_run"] = false
	summary["secrets_resolved"] = false
	summary["network_used"] = false
	summary["artifact_written_task_count"] = artifactWrittenRunTaskCount(ctx, tx, run.ID)
	remaining, err := remainingApprovedArtifactWriteTaskCount(ctx, tx, run.ID)
	if err != nil {
		return RunRecord{}, err
	}
	status := "running"
	var finishedAtExpr string
	if remaining == 0 {
		status = "artifact_written"
		finishedAtExpr = "now()"
	} else {
		finishedAtExpr = "finished_at"
	}
	summary["remaining_task_count"] = remaining
	summaryJSON, err := json.Marshal(summary)
	if err != nil {
		return RunRecord{}, fmt.Errorf("marshal approved artifact write run summary: %w", err)
	}
	metadata := copyMap(run.Metadata)
	metadata["last_approved_artifact_write_actor"] = options.Actor
	metadata["last_approved_artifact_write_reason"] = options.Reason
	metadata["last_approved_artifact_write_at"] = time.Now().UTC().Format(time.RFC3339)
	metadata["last_artifact_id"] = artifactID
	metadata["last_attempt_id"] = attemptID
	metadata["approved_artifact_write"] = true
	metadata["approval_gated"] = true
	metadataJSON, err := json.Marshal(metadata)
	if err != nil {
		return RunRecord{}, fmt.Errorf("marshal approved artifact write run metadata: %w", err)
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
		return RunRecord{}, fmt.Errorf("update approved artifact write run: %w", err)
	}
	return updated, nil
}

func artifactWrittenRunTaskCount(ctx context.Context, tx pgx.Tx, runID int64) int64 {
	var count int64
	_ = tx.QueryRow(ctx, `SELECT count(*) FROM run_tasks WHERE run_id = $1 AND status = 'artifact_written'`, runID).Scan(&count)
	return count
}

func remainingApprovedArtifactWriteTaskCount(ctx context.Context, tx pgx.Tx, runID int64) (int64, error) {
	var count int64
	if err := tx.QueryRow(ctx, `
SELECT count(*)
FROM run_tasks
WHERE run_id = $1
  AND task_kind = 'approved_artifact_write_task'
  AND status IN ('queued', 'pending', 'needs_recovery', 'leased')`,
		runID,
	).Scan(&count); err != nil {
		return 0, fmt.Errorf("count remaining approved artifact write tasks: %w", err)
	}
	return count, nil
}

func canProjectCapabilityInTx(ctx context.Context, tx pgx.Tx, projectID int64, capability string) (bool, string, error) {
	rows, err := tx.Query(ctx, `
SELECT effect, capability, pattern
FROM project_permissions
WHERE project_id = $1 AND resource_type = 'capability'
ORDER BY id`,
		projectID,
	)
	if err != nil {
		return false, "", fmt.Errorf("load project capability permissions: %w", err)
	}
	defer rows.Close()

	allowed := false
	for rows.Next() {
		var effect, permissionCapability, pattern string
		if err := rows.Scan(&effect, &permissionCapability, &pattern); err != nil {
			return false, "", fmt.Errorf("scan project capability permission: %w", err)
		}
		if permissionCapability != capability && pattern != capability {
			continue
		}
		if effect == "deny" {
			return false, "capability denied", nil
		}
		if effect == "allow" {
			allowed = true
		}
	}
	if err := rows.Err(); err != nil {
		return false, "", fmt.Errorf("iterate project capability permissions: %w", err)
	}
	if !allowed {
		return false, "capability not allowed", nil
	}
	return true, "allowed", nil
}

func insertApprovedArtifactWriteQueueEvent(ctx context.Context, tx pgx.Tx, result ApprovedArtifactWriteQueueResult, options ApprovedArtifactWriteQueueOptions) (int64, error) {
	metadata, err := json.Marshal(approvedArtifactWriteQueueMetadata(result, options))
	if err != nil {
		return 0, fmt.Errorf("marshal approved artifact write queue event metadata: %w", err)
	}
	var eventID int64
	if err := tx.QueryRow(ctx, `
INSERT INTO events (project_id, run_id, workflow_version_id, event_type, severity, message, metadata)
VALUES ($1, $2, $3, 'run.approved_artifact_write_queue.created', 'info', 'Approved artifact write run queued', $4::jsonb)
RETURNING id`,
		result.Project.ID,
		result.Run.ID,
		result.Version.ID,
		string(metadata),
	).Scan(&eventID); err != nil {
		return 0, fmt.Errorf("insert approved artifact write queue event: %w", err)
	}
	return eventID, nil
}

func insertApprovedArtifactWriteQueueAuditEvent(ctx context.Context, tx pgx.Tx, result ApprovedArtifactWriteQueueResult, options ApprovedArtifactWriteQueueOptions) (int64, error) {
	metadata, err := json.Marshal(approvedArtifactWriteQueueMetadata(result, options))
	if err != nil {
		return 0, fmt.Errorf("marshal approved artifact write queue audit metadata: %w", err)
	}
	var auditEventID int64
	if err := tx.QueryRow(ctx, `
INSERT INTO audit_events (project_id, action, capability, resource_type, resource, decision, reason, metadata)
VALUES ($1, $2, 'manage_runs', 'run', $3, 'allowed', $4, $5::jsonb)
RETURNING id`,
		result.Project.ID,
		approvedArtifactWriteQueueCommandType,
		fmt.Sprintf("%d", result.Run.ID),
		options.Reason,
		string(metadata),
	).Scan(&auditEventID); err != nil {
		return 0, fmt.Errorf("insert approved artifact write queue audit event: %w", err)
	}
	return auditEventID, nil
}

func approvedArtifactWriteQueueMetadata(result ApprovedArtifactWriteQueueResult, options ApprovedArtifactWriteQueueOptions) map[string]any {
	return map[string]any{
		"project_key":                       result.Project.Key,
		"workflow_version_id":               result.Version.ID,
		"display_label":                     result.Version.DisplayLabel,
		"run_id":                            result.Run.ID,
		"run_task_id":                       result.Task.ID,
		"artifact_label":                    result.ArtifactLabel,
		"actor":                             options.Actor,
		"idempotency_key":                   options.IdempotencyKey,
		"approved_artifact_write":           true,
		"approval_gated":                    true,
		"project_read_attempted":            false,
		"project_write_attempted":           false,
		"execution_write_attempted":         false,
		"area_flow_artifact_written":        false,
		"area_flow_execution_state_written": true,
		"engine_call_attempted":             false,
		"commands_run":                      false,
		"secrets_resolved":                  false,
		"network_used":                      false,
	}
}

func insertApprovedArtifactWriteEvent(ctx context.Context, tx pgx.Tx, result ApprovedArtifactWriteResult, options ApprovedArtifactWriteOptions) (int64, error) {
	severity := "info"
	if result.Decision == "denied" {
		severity = "warning"
	}
	metadata, err := json.Marshal(approvedArtifactWriteMetadata(result, options))
	if err != nil {
		return 0, fmt.Errorf("marshal approved artifact write event metadata: %w", err)
	}
	var eventID int64
	if err := tx.QueryRow(ctx, `
INSERT INTO events (project_id, run_id, workflow_version_id, event_type, severity, message, metadata)
VALUES ($1, $2, $3, $4, $5, $6, $7::jsonb)
RETURNING id`,
		result.Project.ID,
		result.Run.ID,
		nullableInt64(result.Version.ID),
		"worker.approved_artifact_write."+result.Decision,
		severity,
		result.Message,
		string(metadata),
	).Scan(&eventID); err != nil {
		return 0, fmt.Errorf("insert approved artifact write event: %w", err)
	}
	return eventID, nil
}

func insertApprovedArtifactWriteAuditEvent(ctx context.Context, tx pgx.Tx, result ApprovedArtifactWriteResult, options ApprovedArtifactWriteOptions) (int64, error) {
	metadata, err := json.Marshal(approvedArtifactWriteMetadata(result, options))
	if err != nil {
		return 0, fmt.Errorf("marshal approved artifact write audit metadata: %w", err)
	}
	var auditEventID int64
	if err := tx.QueryRow(ctx, `
INSERT INTO audit_events (project_id, actor_id, action, capability, resource_type, resource, decision, reason, metadata)
VALUES ($1, $2, $3, 'write_artifacts', 'artifact', $4, $5, $6, $7::jsonb)
RETURNING id`,
		result.Project.ID,
		nullableInt64(result.Worker.ActorID),
		approvedArtifactWriteApplyCommandType,
		result.ArtifactLabel,
		result.Decision,
		options.Reason,
		string(metadata),
	).Scan(&auditEventID); err != nil {
		return 0, fmt.Errorf("insert approved artifact write audit event: %w", err)
	}
	return auditEventID, nil
}

func finishDeniedApprovedArtifactWrite(ctx context.Context, tx pgx.Tx, result ApprovedArtifactWriteResult, options ApprovedArtifactWriteOptions) error {
	eventID, err := insertApprovedArtifactWriteEvent(ctx, tx, result, options)
	if err != nil {
		return err
	}
	result.EventID = eventID
	auditEventID, err := insertApprovedArtifactWriteAuditEvent(ctx, tx, result, options)
	if err != nil {
		return err
	}
	result.AuditEventID = auditEventID
	return completeCommandRequestResponse(ctx, tx, result.Project.ID, approvedArtifactWriteApplyCommandType, options.IdempotencyKey, approvedArtifactWriteCommandResponse(result))
}

func approvedArtifactWriteMetadata(result ApprovedArtifactWriteResult, options ApprovedArtifactWriteOptions) map[string]any {
	return map[string]any{
		"project_key":                       result.Project.Key,
		"workflow_version_id":               result.Version.ID,
		"display_label":                     result.Version.DisplayLabel,
		"run_id":                            result.Run.ID,
		"run_task_id":                       result.Task.ID,
		"lease_id":                          result.Lease.ID,
		"attempt_id":                        result.Attempt.ID,
		"artifact_id":                       result.Artifact.ID,
		"worker_id":                         result.Worker.ID,
		"worker_key":                        result.Worker.WorkerKey,
		"artifact_label":                    result.ArtifactLabel,
		"status":                            result.Status,
		"decision":                          result.Decision,
		"blockers":                          result.Blockers,
		"actor":                             options.Actor,
		"idempotency_key":                   options.IdempotencyKey,
		"approved_artifact_write":           true,
		"approval_gated":                    true,
		"execution_gate_status":             result.Gate.Status,
		"task_claimed":                      result.TaskClaimed,
		"worker_started":                    result.WorkerStarted,
		"lease_created":                     result.LeaseCreated,
		"attempt_created":                   result.AttemptCreated,
		"artifact_created":                  result.ArtifactCreated,
		"artifact_write_passed":             result.ArtifactWritePassed,
		"area_flow_artifact_written":        result.AreaFlowArtifactWritten,
		"area_flow_execution_state_written": result.AreaFlowExecutionStateWritten,
		"project_read_attempted":            result.ProjectReadAttempted,
		"project_write_attempted":           result.ProjectWriteAttempted,
		"execution_write_attempted":         result.ExecutionWriteAttempted,
		"engine_call_attempted":             result.EngineCallAttempted,
		"commands_run":                      result.CommandsRun,
		"secrets_resolved":                  result.SecretsResolved,
		"network_used":                      result.NetworkUsed,
	}
}

func deniedApprovedArtifactWriteResult(record Record, version WorkflowVersion, run RunRecord, worker WorkerRecord, gate ExecutionApprovalGate, options ApprovedArtifactWriteOptions, message string, blockers []string) ApprovedArtifactWriteResult {
	return ApprovedArtifactWriteResult{
		Project:                       record,
		Version:                       version,
		Run:                           run,
		Worker:                        worker,
		Gate:                          gate,
		Status:                        "blocked",
		Decision:                      "denied",
		Message:                       message,
		Blockers:                      blockers,
		Created:                       true,
		IdempotencyKey:                options.IdempotencyKey,
		ProjectReadAttempted:          false,
		ProjectWriteAttempted:         false,
		ExecutionWriteAttempted:       false,
		AreaFlowArtifactWritten:       false,
		AreaFlowExecutionStateWritten: false,
		EngineCallAttempted:           false,
		CommandsRun:                   false,
		SecretsResolved:               false,
		NetworkUsed:                   false,
		TaskClaimed:                   false,
		WorkerStarted:                 false,
		LeaseCreated:                  false,
		AttemptCreated:                false,
		ArtifactCreated:               false,
		ArtifactWritePassed:           false,
	}
}

func approvedArtifactWriteQueueCommandResponse(result ApprovedArtifactWriteQueueResult) map[string]any {
	return map[string]any{
		"project_id":                        result.Project.ID,
		"project_key":                       result.Project.Key,
		"workflow_version_id":               result.Version.ID,
		"display_label":                     result.Version.DisplayLabel,
		"run_id":                            result.Run.ID,
		"run_task_id":                       result.Task.ID,
		"artifact_label":                    result.ArtifactLabel,
		"event_id":                          result.EventID,
		"audit_event_id":                    result.AuditEventID,
		"approved_artifact_write":           true,
		"approval_gated":                    true,
		"project_read_attempted":            false,
		"project_write_attempted":           false,
		"execution_write_attempted":         false,
		"area_flow_artifact_written":        false,
		"area_flow_execution_state_written": result.AreaFlowExecutionStateWritten,
		"engine_call_attempted":             false,
		"commands_run":                      false,
		"secrets_resolved":                  false,
		"network_used":                      false,
	}
}

func approvedArtifactWriteCommandResponse(result ApprovedArtifactWriteResult) map[string]any {
	return map[string]any{
		"project_id":                        result.Project.ID,
		"project_key":                       result.Project.Key,
		"workflow_version_id":               result.Version.ID,
		"display_label":                     result.Version.DisplayLabel,
		"run_id":                            result.Run.ID,
		"run_status":                        result.Run.Status,
		"worker_id":                         result.Worker.ID,
		"worker_key":                        result.Worker.WorkerKey,
		"run_task_id":                       result.Task.ID,
		"task_status":                       result.Task.Status,
		"lease_id":                          result.Lease.ID,
		"attempt_id":                        result.Attempt.ID,
		"artifact_id":                       result.Artifact.ID,
		"artifact_type":                     result.Artifact.ArtifactType,
		"artifact_label":                    result.ArtifactLabel,
		"event_id":                          result.EventID,
		"audit_event_id":                    result.AuditEventID,
		"status":                            result.Status,
		"decision":                          result.Decision,
		"message":                           result.Message,
		"blockers":                          result.Blockers,
		"approved_artifact_write":           true,
		"approval_gated":                    true,
		"execution_gate_status":             result.Gate.Status,
		"project_read_attempted":            result.ProjectReadAttempted,
		"project_write_attempted":           result.ProjectWriteAttempted,
		"execution_write_attempted":         result.ExecutionWriteAttempted,
		"area_flow_artifact_written":        result.AreaFlowArtifactWritten,
		"area_flow_execution_state_written": result.AreaFlowExecutionStateWritten,
		"engine_call_attempted":             result.EngineCallAttempted,
		"commands_run":                      result.CommandsRun,
		"secrets_resolved":                  result.SecretsResolved,
		"network_used":                      result.NetworkUsed,
		"task_claimed":                      result.TaskClaimed,
		"worker_started":                    result.WorkerStarted,
		"lease_created":                     result.LeaseCreated,
		"attempt_created":                   result.AttemptCreated,
		"artifact_created":                  result.ArtifactCreated,
		"artifact_write_passed":             result.ArtifactWritePassed,
	}
}

func loadApprovedArtifactWriteQueueByCommandResponse(ctx context.Context, tx pgx.Tx, record Record, version WorkflowVersion, idempotencyKey string) (ApprovedArtifactWriteQueueResult, error) {
	response, err := loadCommandResponse(ctx, tx, record.ID, approvedArtifactWriteQueueCommandType, idempotencyKey)
	if err != nil {
		return ApprovedArtifactWriteQueueResult{}, err
	}
	runID := metadataInt64(response, "run_id")
	taskID := metadataInt64(response, "run_task_id")
	run, err := loadRunForUpdate(ctx, tx, runID)
	if err != nil {
		return ApprovedArtifactWriteQueueResult{}, err
	}
	task, err := loadRunTaskByID(ctx, tx, record.ID, taskID)
	if err != nil {
		return ApprovedArtifactWriteQueueResult{}, err
	}
	return ApprovedArtifactWriteQueueResult{
		Project:                       record,
		Version:                       version,
		Run:                           run,
		Task:                          task,
		ArtifactLabel:                 metadataString(response, "artifact_label"),
		IdempotencyKey:                idempotencyKey,
		EventID:                       metadataInt64(response, "event_id"),
		AuditEventID:                  metadataInt64(response, "audit_event_id"),
		ProjectReadAttempted:          metadataBool(response, "project_read_attempted"),
		ProjectWriteAttempted:         metadataBool(response, "project_write_attempted"),
		ExecutionWriteAttempted:       metadataBool(response, "execution_write_attempted"),
		AreaFlowArtifactWritten:       metadataBool(response, "area_flow_artifact_written"),
		AreaFlowExecutionStateWritten: metadataBool(response, "area_flow_execution_state_written"),
		EngineCallAttempted:           metadataBool(response, "engine_call_attempted"),
		CommandsRun:                   metadataBool(response, "commands_run"),
		SecretsResolved:               metadataBool(response, "secrets_resolved"),
		NetworkUsed:                   metadataBool(response, "network_used"),
	}, nil
}

func loadApprovedArtifactWriteByCommandResponse(ctx context.Context, tx pgx.Tx, record Record, version WorkflowVersion, gate ExecutionApprovalGate, idempotencyKey string) (ApprovedArtifactWriteResult, error) {
	response, err := loadCommandResponse(ctx, tx, record.ID, approvedArtifactWriteApplyCommandType, idempotencyKey)
	if err != nil {
		return ApprovedArtifactWriteResult{}, err
	}
	runID := metadataInt64(response, "run_id")
	workerID := metadataInt64(response, "worker_id")
	run, err := loadRunForUpdate(ctx, tx, runID)
	if err != nil {
		return ApprovedArtifactWriteResult{}, err
	}
	worker := WorkerRecord{}
	if workerID != 0 {
		worker, err = loadWorkerByID(ctx, tx, record.ID, workerID)
		if err != nil {
			return ApprovedArtifactWriteResult{}, err
		}
	}
	result := ApprovedArtifactWriteResult{
		Project:                       record,
		Version:                       version,
		Run:                           run,
		Worker:                        worker,
		Gate:                          gate,
		ArtifactLabel:                 metadataString(response, "artifact_label"),
		Status:                        metadataString(response, "status"),
		Decision:                      metadataString(response, "decision"),
		Message:                       metadataString(response, "message"),
		Blockers:                      metadataStringSlice(response, "blockers"),
		IdempotencyKey:                idempotencyKey,
		EventID:                       metadataInt64(response, "event_id"),
		AuditEventID:                  metadataInt64(response, "audit_event_id"),
		ProjectReadAttempted:          metadataBool(response, "project_read_attempted"),
		ProjectWriteAttempted:         metadataBool(response, "project_write_attempted"),
		ExecutionWriteAttempted:       metadataBool(response, "execution_write_attempted"),
		AreaFlowArtifactWritten:       metadataBool(response, "area_flow_artifact_written"),
		AreaFlowExecutionStateWritten: metadataBool(response, "area_flow_execution_state_written"),
		EngineCallAttempted:           metadataBool(response, "engine_call_attempted"),
		CommandsRun:                   metadataBool(response, "commands_run"),
		SecretsResolved:               metadataBool(response, "secrets_resolved"),
		NetworkUsed:                   metadataBool(response, "network_used"),
		TaskClaimed:                   metadataBool(response, "task_claimed"),
		WorkerStarted:                 metadataBool(response, "worker_started"),
		LeaseCreated:                  metadataBool(response, "lease_created"),
		AttemptCreated:                metadataBool(response, "attempt_created"),
		ArtifactCreated:               metadataBool(response, "artifact_created"),
		ArtifactWritePassed:           metadataBool(response, "artifact_write_passed"),
	}
	if taskID := metadataInt64(response, "run_task_id"); taskID != 0 {
		result.Task, err = loadRunTaskByID(ctx, tx, record.ID, taskID)
		if err != nil {
			return ApprovedArtifactWriteResult{}, err
		}
	}
	if leaseID := metadataInt64(response, "lease_id"); leaseID != 0 {
		result.Lease, err = loadLeaseByID(ctx, tx, record.ID, leaseID)
		if err != nil {
			return ApprovedArtifactWriteResult{}, err
		}
	}
	if attemptID := metadataInt64(response, "attempt_id"); attemptID != 0 {
		result.Attempt, err = loadRunAttemptByID(ctx, tx, record.ID, attemptID)
		if err != nil {
			return ApprovedArtifactWriteResult{}, err
		}
	}
	if artifactID := metadataInt64(response, "artifact_id"); artifactID != 0 {
		result.Artifact, err = loadArtifactByIDTx(ctx, tx, record.ID, artifactID)
		if err != nil {
			return ApprovedArtifactWriteResult{}, err
		}
	}
	return result, nil
}
