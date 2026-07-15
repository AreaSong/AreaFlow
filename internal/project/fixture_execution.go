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

var ErrFixtureExecutionBlocked = errors.New("fixture execution blocked")

const (
	fixtureExecutionQueueCommandType = "run.fixture_queue"
	fixtureExecutionApplyCommandType = "worker.fixture_execute"
)

type FixtureExecutionQueueOptions struct {
	IdempotencyKey string
	Actor          string
	Reason         string
}

type FixtureExecutionQueueResult struct {
	Project                 Record
	Version                 WorkflowVersion
	Run                     RunRecord
	Task                    RunTaskRecord
	Created                 bool
	IdempotencyKey          string
	EventID                 int64
	AuditEventID            int64
	ProjectWriteAttempted   bool
	ExecutionWriteAttempted bool
	EngineCallAttempted     bool
	CommandsRun             bool
	SecretsResolved         bool
	NetworkUsed             bool
}

type FixtureExecutionOptions struct {
	WorkerKey           string
	RunID               int64
	AllowedCapabilities []string
	LeaseTimeoutSeconds int
	Metadata            map[string]any
	IdempotencyKey      string
	Actor               string
	Reason              string
}

type FixtureExecutionResult struct {
	Project                       Record
	Version                       WorkflowVersion
	Run                           RunRecord
	Worker                        WorkerRecord
	Lease                         LeaseRecord
	Task                          RunTaskRecord
	Attempt                       RunAttemptRecord
	Artifact                      ArtifactRecord
	Gate                          ExecutionApprovalGate
	Status                        string
	Decision                      string
	Message                       string
	Blockers                      []string
	Created                       bool
	IdempotencyKey                string
	EventID                       int64
	AuditEventID                  int64
	ProjectWriteAttempted         bool
	ExecutionWriteAttempted       bool
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
}

func (s Store) QueueFixtureExecution(ctx context.Context, record Record, label string, options FixtureExecutionQueueOptions) (FixtureExecutionQueueResult, error) {
	version, err := s.GetWorkflowVersion(ctx, record, label)
	if err != nil {
		return FixtureExecutionQueueResult{}, err
	}
	if version.ImportMode != "authored" {
		return FixtureExecutionQueueResult{}, fmt.Errorf("%w: %s", ErrWorkflowVersionNotAuthored, label)
	}
	options = normalizeFixtureExecutionQueueOptions(record, version, options)
	requestHash, err := fixtureExecutionQueueRequestHash(record, version, options)
	if err != nil {
		return FixtureExecutionQueueResult{}, err
	}
	if options.IdempotencyKey == "" {
		options.IdempotencyKey = fixtureExecutionQueueIdempotencyKey(record, version, requestHash)
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return FixtureExecutionQueueResult{}, fmt.Errorf("begin fixture execution queue: %w", err)
	}
	defer tx.Rollback(ctx)

	created, err := reserveCommandRequest(ctx, tx, record.ID, fixtureExecutionQueueCommandType, options.IdempotencyKey, requestHash)
	if err != nil {
		return FixtureExecutionQueueResult{}, err
	}
	if !created {
		result, err := loadFixtureExecutionQueueByCommandResponse(ctx, tx, record, version, options.IdempotencyKey)
		if err != nil {
			return FixtureExecutionQueueResult{}, err
		}
		if err := tx.Commit(ctx); err != nil {
			return FixtureExecutionQueueResult{}, fmt.Errorf("commit fixture execution queue replay: %w", err)
		}
		result.Created = false
		return result, nil
	}

	run, err := insertFixtureExecutionRun(ctx, tx, record, version, options)
	if err != nil {
		return FixtureExecutionQueueResult{}, err
	}
	task, err := insertFixtureExecutionTask(ctx, tx, record, version, run, options)
	if err != nil {
		return FixtureExecutionQueueResult{}, err
	}
	result := FixtureExecutionQueueResult{
		Project:                 record,
		Version:                 version,
		Run:                     run,
		Task:                    task,
		Created:                 true,
		IdempotencyKey:          options.IdempotencyKey,
		ProjectWriteAttempted:   false,
		ExecutionWriteAttempted: false,
		EngineCallAttempted:     false,
		CommandsRun:             false,
		SecretsResolved:         false,
		NetworkUsed:             false,
	}
	eventID, err := insertFixtureExecutionQueueEvent(ctx, tx, result, options)
	if err != nil {
		return FixtureExecutionQueueResult{}, err
	}
	result.EventID = eventID
	auditEventID, err := insertFixtureExecutionQueueAuditEvent(ctx, tx, result, options)
	if err != nil {
		return FixtureExecutionQueueResult{}, err
	}
	result.AuditEventID = auditEventID
	if err := completeCommandRequestResponse(ctx, tx, record.ID, fixtureExecutionQueueCommandType, options.IdempotencyKey, fixtureExecutionQueueCommandResponse(result)); err != nil {
		return FixtureExecutionQueueResult{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return FixtureExecutionQueueResult{}, fmt.Errorf("commit fixture execution queue: %w", err)
	}
	return result, nil
}

func (s Store) ExecuteFixture(ctx context.Context, record Record, options FixtureExecutionOptions) (FixtureExecutionResult, error) {
	if options.RunID <= 0 {
		return FixtureExecutionResult{}, fmt.Errorf("run id is required")
	}
	options = normalizeFixtureExecutionOptions(options)
	requestHash, err := fixtureExecutionRequestHash(record, options)
	if err != nil {
		return FixtureExecutionResult{}, err
	}
	if options.IdempotencyKey == "" {
		options.IdempotencyKey = fixtureExecutionIdempotencyKey(record, options, requestHash)
	}

	gate, err := s.ExecutionApprovalGate(ctx, options.RunID, ExecutionApprovalGateOptions{
		RequiredCapabilities: options.AllowedCapabilities,
	})
	if err != nil {
		return FixtureExecutionResult{}, err
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return FixtureExecutionResult{}, fmt.Errorf("begin fixture execution: %w", err)
	}
	defer tx.Rollback(ctx)

	run, err := loadRunForUpdate(ctx, tx, options.RunID)
	if err != nil {
		return FixtureExecutionResult{}, err
	}
	if run.ProjectID != record.ID {
		return FixtureExecutionResult{}, fmt.Errorf("%w: run %d does not belong to project %s", ErrRunNotFound, options.RunID, record.Key)
	}
	version, err := workflowVersionByIDTx(ctx, tx, record.ID, run.WorkflowVersionID)
	if err != nil {
		return FixtureExecutionResult{}, err
	}
	worker, err := loadWorkerForUpdate(ctx, tx, record.ID, options.WorkerKey)
	if err != nil {
		return FixtureExecutionResult{}, err
	}

	created, err := reserveCommandRequest(ctx, tx, record.ID, fixtureExecutionApplyCommandType, options.IdempotencyKey, requestHash)
	if err != nil {
		return FixtureExecutionResult{}, err
	}
	if !created {
		result, err := loadFixtureExecutionByCommandResponse(ctx, tx, record, version, gate, options.IdempotencyKey)
		if err != nil {
			return FixtureExecutionResult{}, err
		}
		if err := tx.Commit(ctx); err != nil {
			return FixtureExecutionResult{}, fmt.Errorf("commit fixture execution replay: %w", err)
		}
		result.Created = false
		return result, nil
	}

	if gate.Status != "pass" {
		result := deniedFixtureExecutionResult(record, version, run, worker, gate, options, "execution approval gate blocked", gate.Blockers)
		if err := finishDeniedFixtureExecution(ctx, tx, result, options); err != nil {
			return FixtureExecutionResult{}, err
		}
		if err := tx.Commit(ctx); err != nil {
			return FixtureExecutionResult{}, fmt.Errorf("commit blocked fixture execution: %w", err)
		}
		return result, fmt.Errorf("%w: %s", ErrFixtureExecutionBlocked, strings.Join(result.Blockers, "; "))
	}
	if missing := missingWorkerCapabilities(worker.Capabilities, options.AllowedCapabilities); len(missing) > 0 {
		blockers := []string{"worker missing required capabilities: " + strings.Join(missing, ",")}
		result := deniedFixtureExecutionResult(record, version, run, worker, gate, options, "worker capability denied", blockers)
		if err := finishDeniedFixtureExecution(ctx, tx, result, options); err != nil {
			return FixtureExecutionResult{}, err
		}
		if err := tx.Commit(ctx); err != nil {
			return FixtureExecutionResult{}, fmt.Errorf("commit denied fixture execution: %w", err)
		}
		return result, fmt.Errorf("%w: missing %s", ErrWorkerCapabilityDenied, strings.Join(missing, ","))
	}

	task, ok, err := nextFixtureExecutionTaskForLease(ctx, tx, record.ID, run.ID)
	if err != nil {
		return FixtureExecutionResult{}, err
	}
	if !ok {
		result := deniedFixtureExecutionResult(record, version, run, worker, gate, options, "no queued fixture execution task", []string{"no queued or needs_recovery fixture execution task is available"})
		if err := finishDeniedFixtureExecution(ctx, tx, result, options); err != nil {
			return FixtureExecutionResult{}, err
		}
		if err := tx.Commit(ctx); err != nil {
			return FixtureExecutionResult{}, fmt.Errorf("commit idle fixture execution: %w", err)
		}
		return result, fmt.Errorf("%w: no queued task", ErrNoLeaseAvailable)
	}

	lease, err := insertLeaseForTask(ctx, tx, record.ID, worker, task, "fixture_execution", options.AllowedCapabilities, map[string]any{
		"run_id":         task.RunID,
		"run_task_id":    task.ID,
		"task_key":       task.TaskKey,
		"task_kind":      task.TaskKind,
		"fixture_only":   true,
		"approval_gated": true,
	}, options.Metadata, options.LeaseTimeoutSeconds)
	if err != nil {
		return FixtureExecutionResult{}, err
	}
	if err := updateRunTaskStatus(ctx, tx, task.ID, "leased"); err != nil {
		return FixtureExecutionResult{}, err
	}
	artifact, err := writeAndInsertFixtureExecutionArtifact(ctx, tx, record, version, run, worker, task, lease, gate, options)
	if err != nil {
		return FixtureExecutionResult{}, err
	}
	attempt, err := insertFixtureExecutionAttempt(ctx, tx, record, task, lease, artifact, options)
	if err != nil {
		return FixtureExecutionResult{}, err
	}
	released, err := releaseLeaseForWorker(ctx, tx, record.ID, worker, lease.ID, "completed", map[string]any{
		"fixture_only": true,
		"dry_run":      false,
		"attempt_id":   attempt.ID,
		"artifact_id":  artifact.ID,
	})
	if err != nil {
		return FixtureExecutionResult{}, err
	}
	if err := updateRunTaskStatus(ctx, tx, task.ID, "passed"); err != nil {
		return FixtureExecutionResult{}, err
	}
	task.Status = "passed"
	run, err = updateFixtureExecutionRunAfterTask(ctx, tx, run, options, artifact.ID, attempt.ID)
	if err != nil {
		return FixtureExecutionResult{}, err
	}
	result := FixtureExecutionResult{
		Project:                       record,
		Version:                       version,
		Run:                           run,
		Worker:                        worker,
		Lease:                         released,
		Task:                          task,
		Attempt:                       attempt,
		Artifact:                      artifact,
		Gate:                          gate,
		Status:                        "passed",
		Decision:                      "allowed",
		Message:                       "fixture execution applied in AreaFlow state only",
		Created:                       true,
		IdempotencyKey:                options.IdempotencyKey,
		ProjectWriteAttempted:         false,
		ExecutionWriteAttempted:       false,
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
	}
	eventID, err := insertFixtureExecutionEvent(ctx, tx, result, options)
	if err != nil {
		return FixtureExecutionResult{}, err
	}
	result.EventID = eventID
	auditEventID, err := insertFixtureExecutionAuditEvent(ctx, tx, result, options)
	if err != nil {
		return FixtureExecutionResult{}, err
	}
	result.AuditEventID = auditEventID
	if err := completeCommandRequestResponse(ctx, tx, record.ID, fixtureExecutionApplyCommandType, options.IdempotencyKey, fixtureExecutionCommandResponse(result)); err != nil {
		return FixtureExecutionResult{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return FixtureExecutionResult{}, fmt.Errorf("commit fixture execution: %w", err)
	}
	return result, nil
}

func normalizeFixtureExecutionQueueOptions(record Record, version WorkflowVersion, options FixtureExecutionQueueOptions) FixtureExecutionQueueOptions {
	options.IdempotencyKey = strings.TrimSpace(options.IdempotencyKey)
	options.Actor = strings.TrimSpace(options.Actor)
	options.Reason = strings.TrimSpace(options.Reason)
	if options.Actor == "" {
		options.Actor = "local-user"
	}
	if options.Reason == "" {
		options.Reason = "queue fixture execution run"
	}
	if options.IdempotencyKey == "" {
		hash, err := fixtureExecutionQueueRequestHash(record, version, options)
		if err == nil {
			options.IdempotencyKey = fixtureExecutionQueueIdempotencyKey(record, version, hash)
		}
	}
	return options
}

func normalizeFixtureExecutionOptions(options FixtureExecutionOptions) FixtureExecutionOptions {
	options.WorkerKey = strings.TrimSpace(options.WorkerKey)
	options.IdempotencyKey = strings.TrimSpace(options.IdempotencyKey)
	options.Actor = strings.TrimSpace(options.Actor)
	options.Reason = strings.TrimSpace(options.Reason)
	if len(options.AllowedCapabilities) == 0 {
		options.AllowedCapabilities = []string{"execute_agents", "read_project", "run_commands", "write_artifacts"}
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
		options.Reason = "approval-gated fixture execution"
	}
	return options
}

func fixtureExecutionQueueRequestHash(record Record, version WorkflowVersion, options FixtureExecutionQueueOptions) (string, error) {
	payload, err := json.Marshal(map[string]any{
		"command_type": fixtureExecutionQueueCommandType,
		"project_key":  record.Key,
		"version":      version.DisplayLabel,
		"actor":        options.Actor,
		"reason":       options.Reason,
		"fixture_only": true,
		"dry_run":      false,
	})
	if err != nil {
		return "", fmt.Errorf("marshal fixture execution queue request hash payload: %w", err)
	}
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:]), nil
}

func fixtureExecutionRequestHash(record Record, options FixtureExecutionOptions) (string, error) {
	payload, err := json.Marshal(map[string]any{
		"command_type":         fixtureExecutionApplyCommandType,
		"project_key":          record.Key,
		"worker_key":           options.WorkerKey,
		"run_id":               options.RunID,
		"allowed_capabilities": options.AllowedCapabilities,
		"lease_timeout":        options.LeaseTimeoutSeconds,
		"metadata":             options.Metadata,
		"actor":                options.Actor,
		"reason":               options.Reason,
		"fixture_only":         true,
	})
	if err != nil {
		return "", fmt.Errorf("marshal fixture execution request hash payload: %w", err)
	}
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:]), nil
}

func fixtureExecutionQueueIdempotencyKey(record Record, version WorkflowVersion, requestHash string) string {
	return fmt.Sprintf("%s:%s:%s:%s", fixtureExecutionQueueCommandType, record.Key, version.DisplayLabel, commandHashPrefix(requestHash))
}

func fixtureExecutionIdempotencyKey(record Record, options FixtureExecutionOptions, requestHash string) string {
	return fmt.Sprintf("%s:%s:%s:%d:%s", fixtureExecutionApplyCommandType, record.Key, options.WorkerKey, options.RunID, commandHashPrefix(requestHash))
}

func insertFixtureExecutionRun(ctx context.Context, tx pgx.Tx, record Record, version WorkflowVersion, options FixtureExecutionQueueOptions) (RunRecord, error) {
	summary, err := json.Marshal(map[string]any{
		"fixture_only":              true,
		"approval_gated":            true,
		"dry_run":                   false,
		"task_count":                1,
		"project_write_attempted":   false,
		"execution_write_attempted": false,
		"engine_call_attempted":     false,
		"commands_run":              false,
		"secrets_resolved":          false,
		"network_used":              false,
	})
	if err != nil {
		return RunRecord{}, fmt.Errorf("marshal fixture execution run summary: %w", err)
	}
	metadata, err := json.Marshal(map[string]any{
		"phase":           "v0.6i",
		"owned_by":        "areaflow",
		"actor":           options.Actor,
		"reason":          options.Reason,
		"idempotency_key": options.IdempotencyKey,
		"fixture_only":    true,
		"approval_gated":  true,
		"dry_run":         false,
	})
	if err != nil {
		return RunRecord{}, fmt.Errorf("marshal fixture execution run metadata: %w", err)
	}
	run, err := scanRun(tx.QueryRow(ctx, `
INSERT INTO runs (
    project_id, workflow_version_id, run_type, run_kind, status, risk_level,
    risk_policy, dry_run, summary, metadata
)
VALUES ($1, $2, 'fixture_execution', 'execution', 'queued', 'low', 'pause', false, $3::jsonb, $4::jsonb)
RETURNING id, COALESCE(project_id, 0), COALESCE(workflow_version_id, 0), run_type,
          COALESCE(run_kind, ''), status, risk_level, risk_policy, dry_run,
          summary, metadata, started_at, finished_at`,
		record.ID,
		version.ID,
		string(summary),
		string(metadata),
	))
	if err != nil {
		return RunRecord{}, fmt.Errorf("insert fixture execution run: %w", err)
	}
	return run, nil
}

func insertFixtureExecutionTask(ctx context.Context, tx pgx.Tx, record Record, version WorkflowVersion, run RunRecord, options FixtureExecutionQueueOptions) (RunTaskRecord, error) {
	metadata, err := json.Marshal(map[string]any{
		"phase":            "v0.6i",
		"owned_by":         "areaflow",
		"actor":            options.Actor,
		"reason":           options.Reason,
		"fixture_only":     true,
		"approval_gated":   true,
		"dry_run":          false,
		"copy_ready":       "fixture",
		"verify_ready":     "fixture",
		"commands_run":     false,
		"secrets_resolved": false,
		"network_used":     false,
	})
	if err != nil {
		return RunTaskRecord{}, fmt.Errorf("marshal fixture execution task metadata: %w", err)
	}
	task, err := scanRunTask(tx.QueryRow(ctx, `
INSERT INTO run_tasks (
    project_id, workflow_version_id, run_id, task_key, task_kind,
    status, risk_level, sequence, metadata
)
VALUES ($1, $2, $3, $4, 'fixture_execution_task', 'queued', 'low', 1, $5::jsonb)
RETURNING id, project_id, COALESCE(workflow_version_id, 0), COALESCE(workflow_item_id, 0),
          run_id, task_key, task_kind, status, risk_level, sequence, metadata,
          created_at, updated_at`,
		record.ID,
		version.ID,
		run.ID,
		version.DisplayLabel+":fixture-execution",
		string(metadata),
	))
	if err != nil {
		return RunTaskRecord{}, fmt.Errorf("insert fixture execution task: %w", err)
	}
	return task, nil
}

func nextFixtureExecutionTaskForLease(ctx context.Context, tx pgx.Tx, projectID int64, runID int64) (RunTaskRecord, bool, error) {
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
  AND rt.task_kind = 'fixture_execution_task'
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
		return RunTaskRecord{}, false, fmt.Errorf("load next fixture execution task: %w", err)
	}
	return task, true, nil
}

func writeAndInsertFixtureExecutionArtifact(ctx context.Context, tx pgx.Tx, record Record, version WorkflowVersion, run RunRecord, worker WorkerRecord, task RunTaskRecord, lease LeaseRecord, gate ExecutionApprovalGate, options FixtureExecutionOptions) (ArtifactRecord, error) {
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
		"fixture_only":                      true,
		"approval_gated":                    true,
		"execution_gate_status":             gate.Status,
		"dry_run":                           false,
		"project_write_attempted":           false,
		"execution_write_attempted":         false,
		"area_flow_execution_state_written": true,
		"engine_call_attempted":             false,
		"commands_run":                      false,
		"secrets_resolved":                  false,
		"network_used":                      false,
		"allowed_capabilities":              options.AllowedCapabilities,
	}, "", "  ")
	if err != nil {
		return ArtifactRecord{}, fmt.Errorf("marshal fixture execution report: %w", err)
	}
	relativePath := filepath.Join("versions", version.DisplayLabel, "fixture-execution", fmt.Sprintf("run-%d-task-%d-report.json", run.ID, task.ID))
	stored, err := writeProjectArtifact(record, relativePath, content, "application/json")
	if err != nil {
		return ArtifactRecord{}, err
	}
	metadata, err := json.Marshal(map[string]any{
		"phase":                 "v0.6i",
		"owned_by":              "areaflow",
		"fixture_only":          true,
		"approval_gated":        true,
		"dry_run":               false,
		"worker_id":             worker.ID,
		"worker_key":            worker.WorkerKey,
		"run_id":                run.ID,
		"run_task_id":           task.ID,
		"lease_id":              lease.ID,
		"actor":                 options.Actor,
		"reason":                options.Reason,
		"execution_gate_status": gate.Status,
	})
	if err != nil {
		return ArtifactRecord{}, fmt.Errorf("marshal fixture execution artifact metadata: %w", err)
	}
	return insertRunArtifactRecord(ctx, tx, record.ID, version.ID, run.ID, task.WorkflowItemID, "fixture_execution_report", relativePath, stored, string(metadata))
}

func insertFixtureExecutionAttempt(ctx context.Context, tx pgx.Tx, record Record, task RunTaskRecord, lease LeaseRecord, report ArtifactRecord, options FixtureExecutionOptions) (RunAttemptRecord, error) {
	metadata, err := json.Marshal(map[string]any{
		"actor":                             options.Actor,
		"reason":                            options.Reason,
		"fixture_only":                      true,
		"approval_gated":                    true,
		"dry_run":                           false,
		"would_execute":                     false,
		"project_write_attempted":           false,
		"execution_write_attempted":         false,
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
		return RunAttemptRecord{}, fmt.Errorf("marshal fixture execution attempt metadata: %w", err)
	}
	attempt, err := scanRunAttempt(tx.QueryRow(ctx, `
INSERT INTO run_attempts (
    project_id, workflow_version_id, workflow_item_id, run_id, run_task_id,
    attempt_kind, status, dry_run, finished_at, metadata
)
VALUES ($1, $2, $3, $4, $5, 'fixture_execution', 'passed', false, now(), $6::jsonb)
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
		return RunAttemptRecord{}, fmt.Errorf("insert fixture execution attempt: %w", err)
	}
	return attempt, nil
}

func updateFixtureExecutionRunAfterTask(ctx context.Context, tx pgx.Tx, run RunRecord, options FixtureExecutionOptions, artifactID int64, attemptID int64) (RunRecord, error) {
	summary := copyMap(run.Summary)
	summary["fixture_only"] = true
	summary["approval_gated"] = true
	summary["last_artifact_id"] = artifactID
	summary["last_attempt_id"] = attemptID
	summary["project_write_attempted"] = false
	summary["execution_write_attempted"] = false
	summary["area_flow_execution_state_written"] = true
	summary["engine_call_attempted"] = false
	summary["commands_run"] = false
	summary["secrets_resolved"] = false
	summary["network_used"] = false
	summary["passed_task_count"] = passedRunTaskCount(ctx, tx, run.ID)
	remaining, err := remainingFixtureExecutionTaskCount(ctx, tx, run.ID)
	if err != nil {
		return RunRecord{}, err
	}
	status := "running"
	var finishedAtExpr string
	if remaining == 0 {
		status = "passed"
		finishedAtExpr = "now()"
	} else {
		finishedAtExpr = "finished_at"
	}
	summary["remaining_task_count"] = remaining
	summaryJSON, err := json.Marshal(summary)
	if err != nil {
		return RunRecord{}, fmt.Errorf("marshal fixture execution run summary: %w", err)
	}
	metadata := copyMap(run.Metadata)
	metadata["last_fixture_execution_actor"] = options.Actor
	metadata["last_fixture_execution_reason"] = options.Reason
	metadata["last_fixture_execution_at"] = time.Now().UTC().Format(time.RFC3339)
	metadata["last_artifact_id"] = artifactID
	metadata["last_attempt_id"] = attemptID
	metadata["fixture_only"] = true
	metadata["approval_gated"] = true
	metadataJSON, err := json.Marshal(metadata)
	if err != nil {
		return RunRecord{}, fmt.Errorf("marshal fixture execution run metadata: %w", err)
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
		return RunRecord{}, fmt.Errorf("update fixture execution run: %w", err)
	}
	return updated, nil
}

func passedRunTaskCount(ctx context.Context, tx pgx.Tx, runID int64) int64 {
	var count int64
	_ = tx.QueryRow(ctx, `SELECT count(*) FROM run_tasks WHERE run_id = $1 AND status = 'passed'`, runID).Scan(&count)
	return count
}

func remainingFixtureExecutionTaskCount(ctx context.Context, tx pgx.Tx, runID int64) (int64, error) {
	var count int64
	if err := tx.QueryRow(ctx, `
SELECT count(*)
FROM run_tasks
WHERE run_id = $1
  AND task_kind = 'fixture_execution_task'
  AND status IN ('queued', 'pending', 'needs_recovery', 'leased')`,
		runID,
	).Scan(&count); err != nil {
		return 0, fmt.Errorf("count remaining fixture execution tasks: %w", err)
	}
	return count, nil
}

func insertFixtureExecutionQueueEvent(ctx context.Context, tx pgx.Tx, result FixtureExecutionQueueResult, options FixtureExecutionQueueOptions) (int64, error) {
	metadata, err := json.Marshal(fixtureExecutionQueueMetadata(result, options))
	if err != nil {
		return 0, fmt.Errorf("marshal fixture execution queue event metadata: %w", err)
	}
	var eventID int64
	if err := tx.QueryRow(ctx, `
INSERT INTO events (project_id, run_id, workflow_version_id, event_type, severity, message, metadata)
VALUES ($1, $2, $3, 'run.fixture_queue.created', 'info', 'Fixture execution run queued', $4::jsonb)
RETURNING id`,
		result.Project.ID,
		result.Run.ID,
		result.Version.ID,
		string(metadata),
	).Scan(&eventID); err != nil {
		return 0, fmt.Errorf("insert fixture execution queue event: %w", err)
	}
	return eventID, nil
}

func insertFixtureExecutionQueueAuditEvent(ctx context.Context, tx pgx.Tx, result FixtureExecutionQueueResult, options FixtureExecutionQueueOptions) (int64, error) {
	metadata, err := json.Marshal(fixtureExecutionQueueMetadata(result, options))
	if err != nil {
		return 0, fmt.Errorf("marshal fixture execution queue audit metadata: %w", err)
	}
	var auditEventID int64
	if err := tx.QueryRow(ctx, `
INSERT INTO audit_events (project_id, action, capability, resource_type, resource, decision, reason, metadata)
VALUES ($1, $2, 'manage_runs', 'run', $3, 'allowed', $4, $5::jsonb)
RETURNING id`,
		result.Project.ID,
		fixtureExecutionQueueCommandType,
		fmt.Sprintf("%d", result.Run.ID),
		options.Reason,
		string(metadata),
	).Scan(&auditEventID); err != nil {
		return 0, fmt.Errorf("insert fixture execution queue audit event: %w", err)
	}
	return auditEventID, nil
}

func fixtureExecutionQueueMetadata(result FixtureExecutionQueueResult, options FixtureExecutionQueueOptions) map[string]any {
	return map[string]any{
		"project_key":               result.Project.Key,
		"workflow_version_id":       result.Version.ID,
		"display_label":             result.Version.DisplayLabel,
		"run_id":                    result.Run.ID,
		"run_task_id":               result.Task.ID,
		"actor":                     options.Actor,
		"idempotency_key":           options.IdempotencyKey,
		"fixture_only":              true,
		"approval_gated":            true,
		"dry_run":                   false,
		"project_write_attempted":   false,
		"execution_write_attempted": false,
		"engine_call_attempted":     false,
		"commands_run":              false,
		"secrets_resolved":          false,
		"network_used":              false,
	}
}

func insertFixtureExecutionEvent(ctx context.Context, tx pgx.Tx, result FixtureExecutionResult, options FixtureExecutionOptions) (int64, error) {
	severity := "info"
	if result.Decision == "denied" {
		severity = "warning"
	}
	metadata, err := json.Marshal(fixtureExecutionMetadata(result, options))
	if err != nil {
		return 0, fmt.Errorf("marshal fixture execution event metadata: %w", err)
	}
	var eventID int64
	if err := tx.QueryRow(ctx, `
INSERT INTO events (project_id, run_id, workflow_version_id, event_type, severity, message, metadata)
VALUES ($1, $2, $3, $4, $5, $6, $7::jsonb)
RETURNING id`,
		result.Project.ID,
		result.Run.ID,
		nullableInt64(result.Version.ID),
		"worker.fixture_execute."+result.Decision,
		severity,
		result.Message,
		string(metadata),
	).Scan(&eventID); err != nil {
		return 0, fmt.Errorf("insert fixture execution event: %w", err)
	}
	return eventID, nil
}

func insertFixtureExecutionAuditEvent(ctx context.Context, tx pgx.Tx, result FixtureExecutionResult, options FixtureExecutionOptions) (int64, error) {
	metadata, err := json.Marshal(fixtureExecutionMetadata(result, options))
	if err != nil {
		return 0, fmt.Errorf("marshal fixture execution audit metadata: %w", err)
	}
	var auditEventID int64
	if err := tx.QueryRow(ctx, `
INSERT INTO audit_events (project_id, actor_id, action, capability, resource_type, resource, decision, reason, metadata)
VALUES ($1, $2, $3, 'execute_agents', 'run', $4, $5, $6, $7::jsonb)
RETURNING id`,
		result.Project.ID,
		nullableInt64(result.Worker.ActorID),
		fixtureExecutionApplyCommandType,
		fmt.Sprintf("%d", result.Run.ID),
		result.Decision,
		options.Reason,
		string(metadata),
	).Scan(&auditEventID); err != nil {
		return 0, fmt.Errorf("insert fixture execution audit event: %w", err)
	}
	return auditEventID, nil
}

func finishDeniedFixtureExecution(ctx context.Context, tx pgx.Tx, result FixtureExecutionResult, options FixtureExecutionOptions) error {
	eventID, err := insertFixtureExecutionEvent(ctx, tx, result, options)
	if err != nil {
		return err
	}
	result.EventID = eventID
	auditEventID, err := insertFixtureExecutionAuditEvent(ctx, tx, result, options)
	if err != nil {
		return err
	}
	result.AuditEventID = auditEventID
	return completeCommandRequestResponse(ctx, tx, result.Project.ID, fixtureExecutionApplyCommandType, options.IdempotencyKey, fixtureExecutionCommandResponse(result))
}

func fixtureExecutionMetadata(result FixtureExecutionResult, options FixtureExecutionOptions) map[string]any {
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
		"status":                            result.Status,
		"decision":                          result.Decision,
		"blockers":                          result.Blockers,
		"actor":                             options.Actor,
		"idempotency_key":                   options.IdempotencyKey,
		"fixture_only":                      true,
		"approval_gated":                    true,
		"dry_run":                           false,
		"execution_gate_status":             result.Gate.Status,
		"task_claimed":                      result.TaskClaimed,
		"worker_started":                    result.WorkerStarted,
		"lease_created":                     result.LeaseCreated,
		"attempt_created":                   result.AttemptCreated,
		"artifact_created":                  result.ArtifactCreated,
		"area_flow_execution_state_written": result.AreaFlowExecutionStateWritten,
		"project_write_attempted":           result.ProjectWriteAttempted,
		"execution_write_attempted":         result.ExecutionWriteAttempted,
		"engine_call_attempted":             result.EngineCallAttempted,
		"commands_run":                      result.CommandsRun,
		"secrets_resolved":                  result.SecretsResolved,
		"network_used":                      result.NetworkUsed,
	}
}

func deniedFixtureExecutionResult(record Record, version WorkflowVersion, run RunRecord, worker WorkerRecord, gate ExecutionApprovalGate, options FixtureExecutionOptions, message string, blockers []string) FixtureExecutionResult {
	return FixtureExecutionResult{
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
		ProjectWriteAttempted:         false,
		ExecutionWriteAttempted:       false,
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
	}
}

func fixtureExecutionQueueCommandResponse(result FixtureExecutionQueueResult) map[string]any {
	return map[string]any{
		"project_id":                result.Project.ID,
		"project_key":               result.Project.Key,
		"workflow_version_id":       result.Version.ID,
		"display_label":             result.Version.DisplayLabel,
		"run_id":                    result.Run.ID,
		"run_task_id":               result.Task.ID,
		"event_id":                  result.EventID,
		"audit_event_id":            result.AuditEventID,
		"fixture_only":              true,
		"approval_gated":            true,
		"dry_run":                   false,
		"project_write_attempted":   false,
		"execution_write_attempted": false,
		"engine_call_attempted":     false,
		"commands_run":              false,
		"secrets_resolved":          false,
		"network_used":              false,
	}
}

func fixtureExecutionCommandResponse(result FixtureExecutionResult) map[string]any {
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
		"event_id":                          result.EventID,
		"audit_event_id":                    result.AuditEventID,
		"status":                            result.Status,
		"decision":                          result.Decision,
		"message":                           result.Message,
		"blockers":                          result.Blockers,
		"fixture_only":                      true,
		"approval_gated":                    true,
		"dry_run":                           false,
		"execution_gate_status":             result.Gate.Status,
		"task_claimed":                      result.TaskClaimed,
		"worker_started":                    result.WorkerStarted,
		"lease_created":                     result.LeaseCreated,
		"attempt_created":                   result.AttemptCreated,
		"artifact_created":                  result.ArtifactCreated,
		"area_flow_execution_state_written": result.AreaFlowExecutionStateWritten,
		"project_write_attempted":           result.ProjectWriteAttempted,
		"execution_write_attempted":         result.ExecutionWriteAttempted,
		"engine_call_attempted":             result.EngineCallAttempted,
		"commands_run":                      result.CommandsRun,
		"secrets_resolved":                  result.SecretsResolved,
		"network_used":                      result.NetworkUsed,
	}
}

func loadFixtureExecutionQueueByCommandResponse(ctx context.Context, tx pgx.Tx, record Record, version WorkflowVersion, idempotencyKey string) (FixtureExecutionQueueResult, error) {
	response, err := loadCommandResponse(ctx, tx, record.ID, fixtureExecutionQueueCommandType, idempotencyKey)
	if err != nil {
		return FixtureExecutionQueueResult{}, err
	}
	runID := metadataInt64(response, "run_id")
	taskID := metadataInt64(response, "run_task_id")
	run, err := loadRunForUpdate(ctx, tx, runID)
	if err != nil {
		return FixtureExecutionQueueResult{}, err
	}
	task, err := loadRunTaskByID(ctx, tx, record.ID, taskID)
	if err != nil {
		return FixtureExecutionQueueResult{}, err
	}
	return FixtureExecutionQueueResult{
		Project:                 record,
		Version:                 version,
		Run:                     run,
		Task:                    task,
		IdempotencyKey:          idempotencyKey,
		EventID:                 metadataInt64(response, "event_id"),
		AuditEventID:            metadataInt64(response, "audit_event_id"),
		ProjectWriteAttempted:   metadataBool(response, "project_write_attempted"),
		ExecutionWriteAttempted: metadataBool(response, "execution_write_attempted"),
		EngineCallAttempted:     metadataBool(response, "engine_call_attempted"),
		CommandsRun:             metadataBool(response, "commands_run"),
		SecretsResolved:         metadataBool(response, "secrets_resolved"),
		NetworkUsed:             metadataBool(response, "network_used"),
	}, nil
}

func loadFixtureExecutionByCommandResponse(ctx context.Context, tx pgx.Tx, record Record, version WorkflowVersion, gate ExecutionApprovalGate, idempotencyKey string) (FixtureExecutionResult, error) {
	response, err := loadCommandResponse(ctx, tx, record.ID, fixtureExecutionApplyCommandType, idempotencyKey)
	if err != nil {
		return FixtureExecutionResult{}, err
	}
	runID := metadataInt64(response, "run_id")
	workerID := metadataInt64(response, "worker_id")
	run, err := loadRunForUpdate(ctx, tx, runID)
	if err != nil {
		return FixtureExecutionResult{}, err
	}
	worker := WorkerRecord{}
	if workerID != 0 {
		worker, err = loadWorkerByID(ctx, tx, record.ID, workerID)
		if err != nil {
			return FixtureExecutionResult{}, err
		}
	}
	result := FixtureExecutionResult{
		Project:                       record,
		Version:                       version,
		Run:                           run,
		Worker:                        worker,
		Gate:                          gate,
		Status:                        metadataString(response, "status"),
		Decision:                      metadataString(response, "decision"),
		Message:                       metadataString(response, "message"),
		Blockers:                      metadataStringSlice(response, "blockers"),
		IdempotencyKey:                idempotencyKey,
		EventID:                       metadataInt64(response, "event_id"),
		AuditEventID:                  metadataInt64(response, "audit_event_id"),
		ProjectWriteAttempted:         metadataBool(response, "project_write_attempted"),
		ExecutionWriteAttempted:       metadataBool(response, "execution_write_attempted"),
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
	}
	if taskID := metadataInt64(response, "run_task_id"); taskID != 0 {
		result.Task, err = loadRunTaskByID(ctx, tx, record.ID, taskID)
		if err != nil {
			return FixtureExecutionResult{}, err
		}
	}
	if leaseID := metadataInt64(response, "lease_id"); leaseID != 0 {
		result.Lease, err = loadLeaseByID(ctx, tx, record.ID, leaseID)
		if err != nil {
			return FixtureExecutionResult{}, err
		}
	}
	if attemptID := metadataInt64(response, "attempt_id"); attemptID != 0 {
		result.Attempt, err = loadRunAttemptByID(ctx, tx, record.ID, attemptID)
		if err != nil {
			return FixtureExecutionResult{}, err
		}
	}
	if artifactID := metadataInt64(response, "artifact_id"); artifactID != 0 {
		result.Artifact, err = loadArtifactByIDTx(ctx, tx, record.ID, artifactID)
		if err != nil {
			return FixtureExecutionResult{}, err
		}
	}
	return result, nil
}

func loadRunTaskByID(ctx context.Context, tx pgx.Tx, projectID int64, taskID int64) (RunTaskRecord, error) {
	task, err := scanRunTask(tx.QueryRow(ctx, `
SELECT id, project_id, COALESCE(workflow_version_id, 0), COALESCE(workflow_item_id, 0),
       run_id, task_key, task_kind, status, risk_level, sequence, metadata,
       created_at, updated_at
FROM run_tasks
WHERE project_id = $1 AND id = $2`,
		projectID,
		taskID,
	))
	if err != nil {
		return RunTaskRecord{}, fmt.Errorf("load run task by id: %w", err)
	}
	return task, nil
}

func loadRunAttemptByID(ctx context.Context, tx pgx.Tx, projectID int64, attemptID int64) (RunAttemptRecord, error) {
	attempt, err := scanRunAttempt(tx.QueryRow(ctx, `
SELECT id, project_id, COALESCE(workflow_version_id, 0), COALESCE(workflow_item_id, 0),
       run_id, COALESCE(run_task_id, 0), attempt_kind, status, dry_run,
       metadata, started_at, finished_at
FROM run_attempts
WHERE project_id = $1 AND id = $2`,
		projectID,
		attemptID,
	))
	if err != nil {
		return RunAttemptRecord{}, fmt.Errorf("load run attempt by id: %w", err)
	}
	return attempt, nil
}

func loadArtifactByIDTx(ctx context.Context, tx pgx.Tx, projectID int64, artifactID int64) (ArtifactRecord, error) {
	artifact, err := scanArtifactRecord(tx.QueryRow(ctx, `
SELECT id, project_id, COALESCE(workflow_version_id, 0), COALESCE(workflow_item_id, 0),
       artifact_type, storage_backend, uri, COALESCE(source_path, ''), COALESCE(sha256, ''),
       COALESCE(size_bytes, 0), COALESCE(content_type, ''), metadata, created_at
FROM artifacts
WHERE project_id = $1 AND id = $2`,
		projectID,
		artifactID,
	))
	if err != nil {
		return ArtifactRecord{}, fmt.Errorf("load artifact by id: %w", err)
	}
	return artifact, nil
}

func workflowVersionByIDTx(ctx context.Context, tx pgx.Tx, projectID int64, versionID int64) (WorkflowVersion, error) {
	version, err := scanWorkflowVersion(tx.QueryRow(ctx, `
SELECT id, project_id, display_label, version_kind, lifecycle_status,
       COALESCE(source_path, ''), COALESCE(source_hash, ''), import_mode,
       immutable, status_summary, created_at, updated_at, imported_at
FROM workflow_versions
WHERE project_id = $1 AND id = $2`,
		projectID,
		versionID,
	))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return WorkflowVersion{}, fmt.Errorf("%w: id %d", ErrWorkflowVersionNotFound, versionID)
		}
		return WorkflowVersion{}, fmt.Errorf("load workflow version by id: %w", err)
	}
	return version, nil
}
