package project

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

var ErrReadOnlyVerifyBlocked = errors.New("read-only verify blocked")

const (
	readOnlyVerifyQueueCommandType = "run.read_only_verify_queue"
	readOnlyVerifyApplyCommandType = "worker.read_only_verify"
	maxReadOnlyVerifyFileBytes     = 2 * 1024 * 1024
)

type ReadOnlyVerifyQueueOptions struct {
	TargetPath     string
	IdempotencyKey string
	Actor          string
	Reason         string
}

type ReadOnlyVerifyQueueResult struct {
	Project                 Record
	Version                 WorkflowVersion
	Run                     RunRecord
	Task                    RunTaskRecord
	TargetPath              string
	Created                 bool
	IdempotencyKey          string
	EventID                 int64
	AuditEventID            int64
	ProjectReadAttempted    bool
	ProjectReadAllowed      bool
	ProjectWriteAttempted   bool
	ExecutionWriteAttempted bool
	EngineCallAttempted     bool
	CommandsRun             bool
	SecretsResolved         bool
	NetworkUsed             bool
}

type ReadOnlyVerifyOptions struct {
	WorkerKey           string
	RunID               int64
	AllowedCapabilities []string
	LeaseTimeoutSeconds int
	Metadata            map[string]any
	IdempotencyKey      string
	Actor               string
	Reason              string
}

type ReadOnlyVerifyResult struct {
	Project                       Record
	Version                       WorkflowVersion
	Run                           RunRecord
	Worker                        WorkerRecord
	Lease                         LeaseRecord
	Task                          RunTaskRecord
	Attempt                       RunAttemptRecord
	Artifact                      ArtifactRecord
	Gate                          ExecutionApprovalGate
	TargetPath                    string
	TargetSHA256                  string
	TargetSizeBytes               int64
	Status                        string
	Decision                      string
	Message                       string
	Blockers                      []string
	Created                       bool
	IdempotencyKey                string
	EventID                       int64
	AuditEventID                  int64
	ProjectReadAttempted          bool
	ProjectReadAllowed            bool
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
	VerificationPassed            bool
}

func (s Store) QueueReadOnlyVerify(ctx context.Context, record Record, label string, options ReadOnlyVerifyQueueOptions) (ReadOnlyVerifyQueueResult, error) {
	version, err := s.GetWorkflowVersion(ctx, record, label)
	if err != nil {
		return ReadOnlyVerifyQueueResult{}, err
	}
	if version.ImportMode != "authored" {
		return ReadOnlyVerifyQueueResult{}, fmt.Errorf("%w: %s", ErrWorkflowVersionNotAuthored, label)
	}
	options = normalizeReadOnlyVerifyQueueOptions(record, version, options)
	if options.TargetPath == "" {
		return ReadOnlyVerifyQueueResult{}, fmt.Errorf("target path is required")
	}
	requestHash, err := readOnlyVerifyQueueRequestHash(record, version, options)
	if err != nil {
		return ReadOnlyVerifyQueueResult{}, err
	}
	if options.IdempotencyKey == "" {
		options.IdempotencyKey = readOnlyVerifyQueueIdempotencyKey(record, version, requestHash)
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return ReadOnlyVerifyQueueResult{}, fmt.Errorf("begin read-only verify queue: %w", err)
	}
	defer tx.Rollback(ctx)

	created, err := reserveCommandRequest(ctx, tx, record.ID, readOnlyVerifyQueueCommandType, options.IdempotencyKey, requestHash)
	if err != nil {
		return ReadOnlyVerifyQueueResult{}, err
	}
	if !created {
		result, err := loadReadOnlyVerifyQueueByCommandResponse(ctx, tx, record, version, options.IdempotencyKey)
		if err != nil {
			return ReadOnlyVerifyQueueResult{}, err
		}
		if err := tx.Commit(ctx); err != nil {
			return ReadOnlyVerifyQueueResult{}, fmt.Errorf("commit read-only verify queue replay: %w", err)
		}
		result.Created = false
		return result, nil
	}

	run, err := insertReadOnlyVerifyRun(ctx, tx, record, version, options)
	if err != nil {
		return ReadOnlyVerifyQueueResult{}, err
	}
	task, err := insertReadOnlyVerifyTask(ctx, tx, record, version, run, options)
	if err != nil {
		return ReadOnlyVerifyQueueResult{}, err
	}
	result := ReadOnlyVerifyQueueResult{
		Project:                 record,
		Version:                 version,
		Run:                     run,
		Task:                    task,
		TargetPath:              options.TargetPath,
		Created:                 true,
		IdempotencyKey:          options.IdempotencyKey,
		ProjectReadAttempted:    false,
		ProjectReadAllowed:      false,
		ProjectWriteAttempted:   false,
		ExecutionWriteAttempted: false,
		EngineCallAttempted:     false,
		CommandsRun:             false,
		SecretsResolved:         false,
		NetworkUsed:             false,
	}
	eventID, err := insertReadOnlyVerifyQueueEvent(ctx, tx, result, options)
	if err != nil {
		return ReadOnlyVerifyQueueResult{}, err
	}
	result.EventID = eventID
	auditEventID, err := insertReadOnlyVerifyQueueAuditEvent(ctx, tx, result, options)
	if err != nil {
		return ReadOnlyVerifyQueueResult{}, err
	}
	result.AuditEventID = auditEventID
	if err := completeCommandRequestResponse(ctx, tx, record.ID, readOnlyVerifyQueueCommandType, options.IdempotencyKey, readOnlyVerifyQueueCommandResponse(result)); err != nil {
		return ReadOnlyVerifyQueueResult{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return ReadOnlyVerifyQueueResult{}, fmt.Errorf("commit read-only verify queue: %w", err)
	}
	return result, nil
}

func (s Store) VerifyReadOnly(ctx context.Context, record Record, options ReadOnlyVerifyOptions) (ReadOnlyVerifyResult, error) {
	if options.RunID <= 0 {
		return ReadOnlyVerifyResult{}, fmt.Errorf("run id is required")
	}
	options = normalizeReadOnlyVerifyOptions(options)
	requestHash, err := readOnlyVerifyRequestHash(record, options)
	if err != nil {
		return ReadOnlyVerifyResult{}, err
	}
	if options.IdempotencyKey == "" {
		options.IdempotencyKey = readOnlyVerifyIdempotencyKey(record, options, requestHash)
	}

	gate, err := s.ExecutionApprovalGate(ctx, options.RunID, ExecutionApprovalGateOptions{
		RequiredCapabilities: options.AllowedCapabilities,
		SkipEnginePreview:    true,
		Mode:                 "read_only_verify_gate",
	})
	if err != nil {
		return ReadOnlyVerifyResult{}, err
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return ReadOnlyVerifyResult{}, fmt.Errorf("begin read-only verify: %w", err)
	}
	defer tx.Rollback(ctx)

	run, err := loadRunForUpdate(ctx, tx, options.RunID)
	if err != nil {
		return ReadOnlyVerifyResult{}, err
	}
	if run.ProjectID != record.ID {
		return ReadOnlyVerifyResult{}, fmt.Errorf("%w: run %d does not belong to project %s", ErrRunNotFound, options.RunID, record.Key)
	}
	version, err := workflowVersionByIDTx(ctx, tx, record.ID, run.WorkflowVersionID)
	if err != nil {
		return ReadOnlyVerifyResult{}, err
	}
	worker, err := loadWorkerForUpdate(ctx, tx, record.ID, options.WorkerKey)
	if err != nil {
		return ReadOnlyVerifyResult{}, err
	}

	created, err := reserveCommandRequest(ctx, tx, record.ID, readOnlyVerifyApplyCommandType, options.IdempotencyKey, requestHash)
	if err != nil {
		return ReadOnlyVerifyResult{}, err
	}
	if !created {
		result, err := loadReadOnlyVerifyByCommandResponse(ctx, tx, record, version, gate, options.IdempotencyKey)
		if err != nil {
			return ReadOnlyVerifyResult{}, err
		}
		if err := tx.Commit(ctx); err != nil {
			return ReadOnlyVerifyResult{}, fmt.Errorf("commit read-only verify replay: %w", err)
		}
		result.Created = false
		return result, nil
	}

	if gate.Status != "pass" {
		result := deniedReadOnlyVerifyResult(record, version, run, worker, gate, options, "read-only verify gate blocked", gate.Blockers)
		if err := finishDeniedReadOnlyVerify(ctx, tx, result, options); err != nil {
			return ReadOnlyVerifyResult{}, err
		}
		if err := tx.Commit(ctx); err != nil {
			return ReadOnlyVerifyResult{}, fmt.Errorf("commit blocked read-only verify: %w", err)
		}
		return result, fmt.Errorf("%w: %s", ErrReadOnlyVerifyBlocked, strings.Join(result.Blockers, "; "))
	}
	if missing := missingWorkerCapabilities(worker.Capabilities, options.AllowedCapabilities); len(missing) > 0 {
		blockers := []string{"worker missing required capabilities: " + strings.Join(missing, ",")}
		result := deniedReadOnlyVerifyResult(record, version, run, worker, gate, options, "worker capability denied", blockers)
		if err := finishDeniedReadOnlyVerify(ctx, tx, result, options); err != nil {
			return ReadOnlyVerifyResult{}, err
		}
		if err := tx.Commit(ctx); err != nil {
			return ReadOnlyVerifyResult{}, fmt.Errorf("commit denied read-only verify: %w", err)
		}
		return result, fmt.Errorf("%w: missing %s", ErrWorkerCapabilityDenied, strings.Join(missing, ","))
	}

	task, ok, err := nextReadOnlyVerifyTaskForLease(ctx, tx, record.ID, run.ID)
	if err != nil {
		return ReadOnlyVerifyResult{}, err
	}
	if !ok {
		result := deniedReadOnlyVerifyResult(record, version, run, worker, gate, options, "no queued read-only verify task", []string{"no queued or needs_recovery read-only verify task is available"})
		if err := finishDeniedReadOnlyVerify(ctx, tx, result, options); err != nil {
			return ReadOnlyVerifyResult{}, err
		}
		if err := tx.Commit(ctx); err != nil {
			return ReadOnlyVerifyResult{}, fmt.Errorf("commit idle read-only verify: %w", err)
		}
		return result, fmt.Errorf("%w: no queued task", ErrNoLeaseAvailable)
	}
	targetPath := metadataString(task.Metadata, "target_path")
	allowed, reason, err := canProjectPathInTx(ctx, tx, record.ID, "read_project", targetPath)
	if err != nil {
		return ReadOnlyVerifyResult{}, err
	}
	if !allowed {
		blockers := []string{"target path is not readable: " + reason}
		result := deniedReadOnlyVerifyResult(record, version, run, worker, gate, options, "read path denied", blockers)
		result.TargetPath = targetPath
		result.ProjectReadAttempted = false
		result.ProjectReadAllowed = false
		if err := finishDeniedReadOnlyVerify(ctx, tx, result, options); err != nil {
			return ReadOnlyVerifyResult{}, err
		}
		if err := tx.Commit(ctx); err != nil {
			return ReadOnlyVerifyResult{}, fmt.Errorf("commit read path denied: %w", err)
		}
		return result, fmt.Errorf("%w: %s", ErrReadOnlyVerifyBlocked, strings.Join(blockers, "; "))
	}
	fullPath, err := safeProjectReadPath(record.RootPath, targetPath)
	if err != nil {
		blockers := []string{"target path escaped project root: " + err.Error()}
		result := deniedReadOnlyVerifyResult(record, version, run, worker, gate, options, "read path unsafe", blockers)
		result.TargetPath = targetPath
		if err := finishDeniedReadOnlyVerify(ctx, tx, result, options); err != nil {
			return ReadOnlyVerifyResult{}, err
		}
		if err := tx.Commit(ctx); err != nil {
			return ReadOnlyVerifyResult{}, fmt.Errorf("commit read path unsafe: %w", err)
		}
		return result, fmt.Errorf("%w: %s", ErrReadOnlyVerifyBlocked, strings.Join(blockers, "; "))
	}

	lease, err := insertLeaseForTask(ctx, tx, record.ID, worker, task, "read_only_verify", options.AllowedCapabilities, map[string]any{
		"run_id":           task.RunID,
		"run_task_id":      task.ID,
		"task_key":         task.TaskKey,
		"task_kind":        task.TaskKind,
		"target_path":      targetPath,
		"read_only_verify": true,
		"approval_gated":   true,
	}, options.Metadata, options.LeaseTimeoutSeconds)
	if err != nil {
		return ReadOnlyVerifyResult{}, err
	}
	if err := updateRunTaskStatus(ctx, tx, task.ID, "leased"); err != nil {
		return ReadOnlyVerifyResult{}, err
	}
	readResult, err := readProjectFileForVerify(fullPath)
	if err != nil {
		return ReadOnlyVerifyResult{}, err
	}
	artifact, err := writeAndInsertReadOnlyVerifyArtifact(ctx, tx, record, version, run, worker, task, lease, gate, targetPath, readResult, options)
	if err != nil {
		return ReadOnlyVerifyResult{}, err
	}
	attempt, err := insertReadOnlyVerifyAttempt(ctx, tx, record, task, lease, artifact, targetPath, readResult, options)
	if err != nil {
		return ReadOnlyVerifyResult{}, err
	}
	released, err := releaseLeaseForWorker(ctx, tx, record.ID, worker, lease.ID, "completed", map[string]any{
		"read_only_verify": true,
		"target_path":      targetPath,
		"target_sha256":    readResult.SHA256,
		"target_size":      readResult.SizeBytes,
		"attempt_id":       attempt.ID,
		"artifact_id":      artifact.ID,
	})
	if err != nil {
		return ReadOnlyVerifyResult{}, err
	}
	if err := updateRunTaskStatus(ctx, tx, task.ID, "verified"); err != nil {
		return ReadOnlyVerifyResult{}, err
	}
	task.Status = "verified"
	run, err = updateReadOnlyVerifyRunAfterTask(ctx, tx, run, options, artifact.ID, attempt.ID)
	if err != nil {
		return ReadOnlyVerifyResult{}, err
	}
	result := ReadOnlyVerifyResult{
		Project:                       record,
		Version:                       version,
		Run:                           run,
		Worker:                        worker,
		Lease:                         released,
		Task:                          task,
		Attempt:                       attempt,
		Artifact:                      artifact,
		Gate:                          gate,
		TargetPath:                    targetPath,
		TargetSHA256:                  readResult.SHA256,
		TargetSizeBytes:               readResult.SizeBytes,
		Status:                        "verified",
		Decision:                      "allowed",
		Message:                       "read-only verify completed without managed project writes",
		Created:                       true,
		IdempotencyKey:                options.IdempotencyKey,
		ProjectReadAttempted:          true,
		ProjectReadAllowed:            true,
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
		VerificationPassed:            true,
	}
	eventID, err := insertReadOnlyVerifyEvent(ctx, tx, result, options)
	if err != nil {
		return ReadOnlyVerifyResult{}, err
	}
	result.EventID = eventID
	auditEventID, err := insertReadOnlyVerifyAuditEvent(ctx, tx, result, options)
	if err != nil {
		return ReadOnlyVerifyResult{}, err
	}
	result.AuditEventID = auditEventID
	if err := completeCommandRequestResponse(ctx, tx, record.ID, readOnlyVerifyApplyCommandType, options.IdempotencyKey, readOnlyVerifyCommandResponse(result)); err != nil {
		return ReadOnlyVerifyResult{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return ReadOnlyVerifyResult{}, fmt.Errorf("commit read-only verify: %w", err)
	}
	return result, nil
}

type readOnlyVerifyFileResult struct {
	SHA256    string
	SizeBytes int64
}

func normalizeReadOnlyVerifyQueueOptions(record Record, version WorkflowVersion, options ReadOnlyVerifyQueueOptions) ReadOnlyVerifyQueueOptions {
	options.TargetPath = normalizeProjectRelativePath(options.TargetPath)
	options.IdempotencyKey = strings.TrimSpace(options.IdempotencyKey)
	options.Actor = strings.TrimSpace(options.Actor)
	options.Reason = strings.TrimSpace(options.Reason)
	if options.Actor == "" {
		options.Actor = "local-user"
	}
	if options.Reason == "" {
		options.Reason = "queue read-only verify run"
	}
	if options.IdempotencyKey == "" && options.TargetPath != "" {
		hash, err := readOnlyVerifyQueueRequestHash(record, version, options)
		if err == nil {
			options.IdempotencyKey = readOnlyVerifyQueueIdempotencyKey(record, version, hash)
		}
	}
	return options
}

func normalizeReadOnlyVerifyOptions(options ReadOnlyVerifyOptions) ReadOnlyVerifyOptions {
	options.WorkerKey = strings.TrimSpace(options.WorkerKey)
	options.IdempotencyKey = strings.TrimSpace(options.IdempotencyKey)
	options.Actor = strings.TrimSpace(options.Actor)
	options.Reason = strings.TrimSpace(options.Reason)
	if len(options.AllowedCapabilities) == 0 {
		options.AllowedCapabilities = []string{"read_project", "write_artifacts"}
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
		options.Reason = "approval-gated read-only verify"
	}
	return options
}

func normalizeProjectRelativePath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}
	clean := filepath.Clean(path)
	if clean == "." || filepath.IsAbs(clean) || strings.HasPrefix(clean, ".."+string(filepath.Separator)) || clean == ".." {
		return ""
	}
	return filepath.ToSlash(clean)
}

func safeProjectReadPath(root string, targetPath string) (string, error) {
	if strings.TrimSpace(root) == "" {
		return "", fmt.Errorf("project root is empty")
	}
	relative := normalizeProjectRelativePath(targetPath)
	if relative == "" {
		return "", fmt.Errorf("target path must stay under project root")
	}
	rootReal, err := filepath.EvalSymlinks(root)
	if err != nil {
		return "", fmt.Errorf("resolve project root: %w", err)
	}
	full := filepath.Join(rootReal, filepath.FromSlash(relative))
	fullReal, err := filepath.EvalSymlinks(full)
	if err != nil {
		return "", fmt.Errorf("resolve target path: %w", err)
	}
	rel, err := filepath.Rel(rootReal, fullReal)
	if err != nil {
		return "", fmt.Errorf("compare target path: %w", err)
	}
	if rel == "." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || rel == ".." || filepath.IsAbs(rel) {
		return "", fmt.Errorf("target path escapes project root")
	}
	return fullReal, nil
}

func readProjectFileForVerify(path string) (readOnlyVerifyFileResult, error) {
	info, err := os.Stat(path)
	if err != nil {
		return readOnlyVerifyFileResult{}, fmt.Errorf("stat verify target: %w", err)
	}
	if info.IsDir() {
		return readOnlyVerifyFileResult{}, fmt.Errorf("verify target must be a file")
	}
	if info.Size() > maxReadOnlyVerifyFileBytes {
		return readOnlyVerifyFileResult{}, fmt.Errorf("verify target exceeds %d bytes", maxReadOnlyVerifyFileBytes)
	}
	content, err := os.ReadFile(path)
	if err != nil {
		return readOnlyVerifyFileResult{}, fmt.Errorf("read verify target: %w", err)
	}
	sum := sha256.Sum256(content)
	return readOnlyVerifyFileResult{
		SHA256:    hex.EncodeToString(sum[:]),
		SizeBytes: int64(len(content)),
	}, nil
}

func readOnlyVerifyQueueRequestHash(record Record, version WorkflowVersion, options ReadOnlyVerifyQueueOptions) (string, error) {
	payload := map[string]any{
		"command_type":  readOnlyVerifyQueueCommandType,
		"project_id":    record.ID,
		"project_key":   record.Key,
		"version_id":    version.ID,
		"display_label": version.DisplayLabel,
		"target_path":   options.TargetPath,
		"actor":         options.Actor,
		"reason":        options.Reason,
		"read_only":     true,
	}
	content, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("marshal read-only verify queue request hash payload: %w", err)
	}
	sum := sha256.Sum256(content)
	return hex.EncodeToString(sum[:]), nil
}

func readOnlyVerifyRequestHash(record Record, options ReadOnlyVerifyOptions) (string, error) {
	payload := map[string]any{
		"command_type":          readOnlyVerifyApplyCommandType,
		"project_id":            record.ID,
		"project_key":           record.Key,
		"worker_key":            options.WorkerKey,
		"run_id":                options.RunID,
		"allowed_capabilities":  options.AllowedCapabilities,
		"lease_timeout_seconds": options.LeaseTimeoutSeconds,
		"actor":                 options.Actor,
		"reason":                options.Reason,
		"read_only":             true,
	}
	content, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("marshal read-only verify request hash payload: %w", err)
	}
	sum := sha256.Sum256(content)
	return hex.EncodeToString(sum[:]), nil
}

func readOnlyVerifyQueueIdempotencyKey(record Record, version WorkflowVersion, requestHash string) string {
	return fmt.Sprintf("%s:%s:%s:%s", readOnlyVerifyQueueCommandType, record.Key, version.DisplayLabel, commandHashPrefix(requestHash))
}

func readOnlyVerifyIdempotencyKey(record Record, options ReadOnlyVerifyOptions, requestHash string) string {
	return fmt.Sprintf("%s:%s:%s:%d:%s", readOnlyVerifyApplyCommandType, record.Key, options.WorkerKey, options.RunID, commandHashPrefix(requestHash))
}

func insertReadOnlyVerifyRun(ctx context.Context, tx pgx.Tx, record Record, version WorkflowVersion, options ReadOnlyVerifyQueueOptions) (RunRecord, error) {
	summary, err := json.Marshal(map[string]any{
		"read_only_verify":        true,
		"approval_gated":          true,
		"target_path":             options.TargetPath,
		"project_read_attempted":  false,
		"project_write_attempted": false,
		"engine_call_attempted":   false,
		"commands_run":            false,
		"secrets_resolved":        false,
		"network_used":            false,
	})
	if err != nil {
		return RunRecord{}, fmt.Errorf("marshal read-only verify run summary: %w", err)
	}
	metadata, err := json.Marshal(map[string]any{
		"phase":            "v0.6j",
		"read_only_verify": true,
		"approval_gated":   true,
		"target_path":      options.TargetPath,
		"actor":            options.Actor,
		"reason":           options.Reason,
	})
	if err != nil {
		return RunRecord{}, fmt.Errorf("marshal read-only verify run metadata: %w", err)
	}
	run, err := scanRun(tx.QueryRow(ctx, `
INSERT INTO runs (project_id, workflow_version_id, run_type, run_kind, status, risk_level, risk_policy, dry_run, summary, metadata)
VALUES ($1, $2, 'read_only_verify', 'execution', 'queued', 'low', 'pause', false, $3::jsonb, $4::jsonb)
RETURNING id, COALESCE(project_id, 0), COALESCE(workflow_version_id, 0), run_type,
          COALESCE(run_kind, ''), status, risk_level, risk_policy, dry_run,
          summary, metadata, started_at, finished_at`,
		record.ID,
		version.ID,
		string(summary),
		string(metadata),
	))
	if err != nil {
		return RunRecord{}, fmt.Errorf("insert read-only verify run: %w", err)
	}
	return run, nil
}

func insertReadOnlyVerifyTask(ctx context.Context, tx pgx.Tx, record Record, version WorkflowVersion, run RunRecord, options ReadOnlyVerifyQueueOptions) (RunTaskRecord, error) {
	metadata, err := json.Marshal(map[string]any{
		"read_only_verify": true,
		"approval_gated":   true,
		"target_path":      options.TargetPath,
		"actor":            options.Actor,
		"reason":           options.Reason,
	})
	if err != nil {
		return RunTaskRecord{}, fmt.Errorf("marshal read-only verify task metadata: %w", err)
	}
	task, err := scanRunTask(tx.QueryRow(ctx, `
INSERT INTO run_tasks (
    project_id, workflow_version_id, run_id, task_key, task_kind, status, risk_level, sequence, metadata
)
VALUES ($1, $2, $3, $4, 'read_only_verify_task', 'queued', 'low', 1, $5::jsonb)
RETURNING id, project_id, COALESCE(workflow_version_id, 0), COALESCE(workflow_item_id, 0),
          run_id, task_key, task_kind, status, risk_level, sequence, metadata,
          created_at, updated_at`,
		record.ID,
		version.ID,
		run.ID,
		version.DisplayLabel+":read-only-verify:"+options.TargetPath,
		string(metadata),
	))
	if err != nil {
		return RunTaskRecord{}, fmt.Errorf("insert read-only verify task: %w", err)
	}
	return task, nil
}

func nextReadOnlyVerifyTaskForLease(ctx context.Context, tx pgx.Tx, projectID int64, runID int64) (RunTaskRecord, bool, error) {
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
  AND rt.task_kind = 'read_only_verify_task'
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
		return RunTaskRecord{}, false, fmt.Errorf("load next read-only verify task: %w", err)
	}
	return task, true, nil
}

func writeAndInsertReadOnlyVerifyArtifact(ctx context.Context, tx pgx.Tx, record Record, version WorkflowVersion, run RunRecord, worker WorkerRecord, task RunTaskRecord, lease LeaseRecord, gate ExecutionApprovalGate, targetPath string, readResult readOnlyVerifyFileResult, options ReadOnlyVerifyOptions) (ArtifactRecord, error) {
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
		"read_only_verify":                  true,
		"approval_gated":                    true,
		"execution_gate_status":             gate.Status,
		"target_path":                       targetPath,
		"target_sha256":                     readResult.SHA256,
		"target_size_bytes":                 readResult.SizeBytes,
		"project_read_attempted":            true,
		"project_read_allowed":              true,
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
		return ArtifactRecord{}, fmt.Errorf("marshal read-only verify report: %w", err)
	}
	relativePath := filepath.Join("versions", version.DisplayLabel, "read-only-verify", fmt.Sprintf("run-%d-task-%d-report.json", run.ID, task.ID))
	stored, err := writeProjectArtifact(record, relativePath, content, "application/json")
	if err != nil {
		return ArtifactRecord{}, err
	}
	metadata, err := json.Marshal(map[string]any{
		"phase":                 "v0.6j",
		"owned_by":              "areaflow",
		"read_only_verify":      true,
		"approval_gated":        true,
		"worker_id":             worker.ID,
		"worker_key":            worker.WorkerKey,
		"run_id":                run.ID,
		"run_task_id":           task.ID,
		"lease_id":              lease.ID,
		"actor":                 options.Actor,
		"reason":                options.Reason,
		"target_path":           targetPath,
		"target_sha256":         readResult.SHA256,
		"target_size_bytes":     readResult.SizeBytes,
		"execution_gate_status": gate.Status,
	})
	if err != nil {
		return ArtifactRecord{}, fmt.Errorf("marshal read-only verify artifact metadata: %w", err)
	}
	return insertRunArtifactRecord(ctx, tx, record.ID, version.ID, run.ID, task.WorkflowItemID, "read_only_verify_report", relativePath, stored, string(metadata))
}

func insertReadOnlyVerifyAttempt(ctx context.Context, tx pgx.Tx, record Record, task RunTaskRecord, lease LeaseRecord, report ArtifactRecord, targetPath string, readResult readOnlyVerifyFileResult, options ReadOnlyVerifyOptions) (RunAttemptRecord, error) {
	metadata, err := json.Marshal(map[string]any{
		"actor":                             options.Actor,
		"reason":                            options.Reason,
		"read_only_verify":                  true,
		"approval_gated":                    true,
		"target_path":                       targetPath,
		"target_sha256":                     readResult.SHA256,
		"target_size_bytes":                 readResult.SizeBytes,
		"project_read_attempted":            true,
		"project_read_allowed":              true,
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
		return RunAttemptRecord{}, fmt.Errorf("marshal read-only verify attempt metadata: %w", err)
	}
	attempt, err := scanRunAttempt(tx.QueryRow(ctx, `
INSERT INTO run_attempts (
    project_id, workflow_version_id, workflow_item_id, run_id, run_task_id,
    attempt_kind, status, dry_run, finished_at, metadata
)
VALUES ($1, $2, $3, $4, $5, 'read_only_verify', 'passed', false, now(), $6::jsonb)
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
		return RunAttemptRecord{}, fmt.Errorf("insert read-only verify attempt: %w", err)
	}
	return attempt, nil
}

func updateReadOnlyVerifyRunAfterTask(ctx context.Context, tx pgx.Tx, run RunRecord, options ReadOnlyVerifyOptions, artifactID int64, attemptID int64) (RunRecord, error) {
	summary := copyMap(run.Summary)
	summary["read_only_verify"] = true
	summary["approval_gated"] = true
	summary["last_artifact_id"] = artifactID
	summary["last_attempt_id"] = attemptID
	summary["project_read_attempted"] = true
	summary["project_read_allowed"] = true
	summary["project_write_attempted"] = false
	summary["execution_write_attempted"] = false
	summary["area_flow_execution_state_written"] = true
	summary["engine_call_attempted"] = false
	summary["commands_run"] = false
	summary["secrets_resolved"] = false
	summary["network_used"] = false
	summary["verified_task_count"] = verifiedRunTaskCount(ctx, tx, run.ID)
	remaining, err := remainingReadOnlyVerifyTaskCount(ctx, tx, run.ID)
	if err != nil {
		return RunRecord{}, err
	}
	status := "running"
	var finishedAtExpr string
	if remaining == 0 {
		status = "verified"
		finishedAtExpr = "now()"
	} else {
		finishedAtExpr = "finished_at"
	}
	summary["remaining_task_count"] = remaining
	summaryJSON, err := json.Marshal(summary)
	if err != nil {
		return RunRecord{}, fmt.Errorf("marshal read-only verify run summary: %w", err)
	}
	metadata := copyMap(run.Metadata)
	metadata["last_read_only_verify_actor"] = options.Actor
	metadata["last_read_only_verify_reason"] = options.Reason
	metadata["last_read_only_verify_at"] = time.Now().UTC().Format(time.RFC3339)
	metadata["last_artifact_id"] = artifactID
	metadata["last_attempt_id"] = attemptID
	metadata["read_only_verify"] = true
	metadata["approval_gated"] = true
	metadataJSON, err := json.Marshal(metadata)
	if err != nil {
		return RunRecord{}, fmt.Errorf("marshal read-only verify run metadata: %w", err)
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
		return RunRecord{}, fmt.Errorf("update read-only verify run: %w", err)
	}
	return updated, nil
}

func verifiedRunTaskCount(ctx context.Context, tx pgx.Tx, runID int64) int64 {
	var count int64
	_ = tx.QueryRow(ctx, `SELECT count(*) FROM run_tasks WHERE run_id = $1 AND status = 'verified'`, runID).Scan(&count)
	return count
}

func remainingReadOnlyVerifyTaskCount(ctx context.Context, tx pgx.Tx, runID int64) (int64, error) {
	var count int64
	if err := tx.QueryRow(ctx, `
SELECT count(*)
FROM run_tasks
WHERE run_id = $1
  AND task_kind = 'read_only_verify_task'
  AND status IN ('queued', 'pending', 'needs_recovery', 'leased')`,
		runID,
	).Scan(&count); err != nil {
		return 0, fmt.Errorf("count remaining read-only verify tasks: %w", err)
	}
	return count, nil
}

func canProjectPathInTx(ctx context.Context, tx pgx.Tx, projectID int64, capability string, path string) (bool, string, error) {
	rows, err := tx.Query(ctx, `
SELECT effect, capability, resource_type, pattern
FROM project_permissions
WHERE project_id = $1 AND resource_type IN ('capability', 'path')
ORDER BY id`,
		projectID,
	)
	if err != nil {
		return false, "", fmt.Errorf("load project permissions: %w", err)
	}
	defer rows.Close()

	capabilityAllowed := false
	pathAllowed := false
	for rows.Next() {
		var effect, permissionCapability, resourceType, pattern string
		if err := rows.Scan(&effect, &permissionCapability, &resourceType, &pattern); err != nil {
			return false, "", fmt.Errorf("scan project permission: %w", err)
		}
		if effect == "deny" && resourceType == "path" && globMatch(pattern, path) {
			return false, "path denied by forbidden path", nil
		}
		if resourceType == "capability" && permissionCapability == capability && effect == "allow" {
			capabilityAllowed = true
		}
		if resourceType == "path" && permissionCapability == capability && effect == "allow" && globMatch(pattern, path) {
			pathAllowed = true
		}
	}
	if err := rows.Err(); err != nil {
		return false, "", fmt.Errorf("iterate project permissions: %w", err)
	}
	if !capabilityAllowed {
		return false, "capability not allowed", nil
	}
	if !pathAllowed {
		return false, "path not allowed", nil
	}
	return true, "allowed", nil
}

func insertReadOnlyVerifyQueueEvent(ctx context.Context, tx pgx.Tx, result ReadOnlyVerifyQueueResult, options ReadOnlyVerifyQueueOptions) (int64, error) {
	metadata, err := json.Marshal(readOnlyVerifyQueueMetadata(result, options))
	if err != nil {
		return 0, fmt.Errorf("marshal read-only verify queue event metadata: %w", err)
	}
	var eventID int64
	if err := tx.QueryRow(ctx, `
INSERT INTO events (project_id, run_id, workflow_version_id, event_type, severity, message, metadata)
VALUES ($1, $2, $3, 'run.read_only_verify_queue.created', 'info', 'Read-only verify run queued', $4::jsonb)
RETURNING id`,
		result.Project.ID,
		result.Run.ID,
		result.Version.ID,
		string(metadata),
	).Scan(&eventID); err != nil {
		return 0, fmt.Errorf("insert read-only verify queue event: %w", err)
	}
	return eventID, nil
}

func insertReadOnlyVerifyQueueAuditEvent(ctx context.Context, tx pgx.Tx, result ReadOnlyVerifyQueueResult, options ReadOnlyVerifyQueueOptions) (int64, error) {
	metadata, err := json.Marshal(readOnlyVerifyQueueMetadata(result, options))
	if err != nil {
		return 0, fmt.Errorf("marshal read-only verify queue audit metadata: %w", err)
	}
	var auditEventID int64
	if err := tx.QueryRow(ctx, `
INSERT INTO audit_events (project_id, action, capability, resource_type, resource, decision, reason, metadata)
VALUES ($1, $2, 'manage_runs', 'run', $3, 'allowed', $4, $5::jsonb)
RETURNING id`,
		result.Project.ID,
		readOnlyVerifyQueueCommandType,
		fmt.Sprintf("%d", result.Run.ID),
		options.Reason,
		string(metadata),
	).Scan(&auditEventID); err != nil {
		return 0, fmt.Errorf("insert read-only verify queue audit event: %w", err)
	}
	return auditEventID, nil
}

func readOnlyVerifyQueueMetadata(result ReadOnlyVerifyQueueResult, options ReadOnlyVerifyQueueOptions) map[string]any {
	return map[string]any{
		"project_key":               result.Project.Key,
		"workflow_version_id":       result.Version.ID,
		"display_label":             result.Version.DisplayLabel,
		"run_id":                    result.Run.ID,
		"run_task_id":               result.Task.ID,
		"target_path":               result.TargetPath,
		"actor":                     options.Actor,
		"idempotency_key":           options.IdempotencyKey,
		"read_only_verify":          true,
		"approval_gated":            true,
		"project_read_attempted":    false,
		"project_read_allowed":      false,
		"project_write_attempted":   false,
		"execution_write_attempted": false,
		"engine_call_attempted":     false,
		"commands_run":              false,
		"secrets_resolved":          false,
		"network_used":              false,
	}
}

func insertReadOnlyVerifyEvent(ctx context.Context, tx pgx.Tx, result ReadOnlyVerifyResult, options ReadOnlyVerifyOptions) (int64, error) {
	severity := "info"
	if result.Decision == "denied" {
		severity = "warning"
	}
	metadata, err := json.Marshal(readOnlyVerifyMetadata(result, options))
	if err != nil {
		return 0, fmt.Errorf("marshal read-only verify event metadata: %w", err)
	}
	var eventID int64
	if err := tx.QueryRow(ctx, `
INSERT INTO events (project_id, run_id, workflow_version_id, event_type, severity, message, metadata)
VALUES ($1, $2, $3, $4, $5, $6, $7::jsonb)
RETURNING id`,
		result.Project.ID,
		result.Run.ID,
		nullableInt64(result.Version.ID),
		"worker.read_only_verify."+result.Decision,
		severity,
		result.Message,
		string(metadata),
	).Scan(&eventID); err != nil {
		return 0, fmt.Errorf("insert read-only verify event: %w", err)
	}
	return eventID, nil
}

func insertReadOnlyVerifyAuditEvent(ctx context.Context, tx pgx.Tx, result ReadOnlyVerifyResult, options ReadOnlyVerifyOptions) (int64, error) {
	metadata, err := json.Marshal(readOnlyVerifyMetadata(result, options))
	if err != nil {
		return 0, fmt.Errorf("marshal read-only verify audit metadata: %w", err)
	}
	var auditEventID int64
	if err := tx.QueryRow(ctx, `
INSERT INTO audit_events (project_id, actor_id, action, capability, resource_type, resource, decision, reason, metadata)
VALUES ($1, $2, $3, 'read_project', 'path', $4, $5, $6, $7::jsonb)
RETURNING id`,
		result.Project.ID,
		nullableInt64(result.Worker.ActorID),
		readOnlyVerifyApplyCommandType,
		result.TargetPath,
		result.Decision,
		options.Reason,
		string(metadata),
	).Scan(&auditEventID); err != nil {
		return 0, fmt.Errorf("insert read-only verify audit event: %w", err)
	}
	return auditEventID, nil
}

func finishDeniedReadOnlyVerify(ctx context.Context, tx pgx.Tx, result ReadOnlyVerifyResult, options ReadOnlyVerifyOptions) error {
	eventID, err := insertReadOnlyVerifyEvent(ctx, tx, result, options)
	if err != nil {
		return err
	}
	result.EventID = eventID
	auditEventID, err := insertReadOnlyVerifyAuditEvent(ctx, tx, result, options)
	if err != nil {
		return err
	}
	result.AuditEventID = auditEventID
	return completeCommandRequestResponse(ctx, tx, result.Project.ID, readOnlyVerifyApplyCommandType, options.IdempotencyKey, readOnlyVerifyCommandResponse(result))
}

func readOnlyVerifyMetadata(result ReadOnlyVerifyResult, options ReadOnlyVerifyOptions) map[string]any {
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
		"target_path":                       result.TargetPath,
		"target_sha256":                     result.TargetSHA256,
		"target_size_bytes":                 result.TargetSizeBytes,
		"status":                            result.Status,
		"decision":                          result.Decision,
		"blockers":                          result.Blockers,
		"actor":                             options.Actor,
		"idempotency_key":                   options.IdempotencyKey,
		"read_only_verify":                  true,
		"approval_gated":                    true,
		"execution_gate_status":             result.Gate.Status,
		"project_read_attempted":            result.ProjectReadAttempted,
		"project_read_allowed":              result.ProjectReadAllowed,
		"task_claimed":                      result.TaskClaimed,
		"worker_started":                    result.WorkerStarted,
		"lease_created":                     result.LeaseCreated,
		"attempt_created":                   result.AttemptCreated,
		"artifact_created":                  result.ArtifactCreated,
		"verification_passed":               result.VerificationPassed,
		"area_flow_execution_state_written": result.AreaFlowExecutionStateWritten,
		"project_write_attempted":           result.ProjectWriteAttempted,
		"execution_write_attempted":         result.ExecutionWriteAttempted,
		"engine_call_attempted":             result.EngineCallAttempted,
		"commands_run":                      result.CommandsRun,
		"secrets_resolved":                  result.SecretsResolved,
		"network_used":                      result.NetworkUsed,
	}
}

func deniedReadOnlyVerifyResult(record Record, version WorkflowVersion, run RunRecord, worker WorkerRecord, gate ExecutionApprovalGate, options ReadOnlyVerifyOptions, message string, blockers []string) ReadOnlyVerifyResult {
	return ReadOnlyVerifyResult{
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
		ProjectReadAllowed:            false,
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
		VerificationPassed:            false,
	}
}

func readOnlyVerifyQueueCommandResponse(result ReadOnlyVerifyQueueResult) map[string]any {
	return map[string]any{
		"project_id":                result.Project.ID,
		"project_key":               result.Project.Key,
		"workflow_version_id":       result.Version.ID,
		"display_label":             result.Version.DisplayLabel,
		"run_id":                    result.Run.ID,
		"run_task_id":               result.Task.ID,
		"target_path":               result.TargetPath,
		"event_id":                  result.EventID,
		"audit_event_id":            result.AuditEventID,
		"read_only_verify":          true,
		"approval_gated":            true,
		"project_read_attempted":    false,
		"project_read_allowed":      false,
		"project_write_attempted":   false,
		"execution_write_attempted": false,
		"engine_call_attempted":     false,
		"commands_run":              false,
		"secrets_resolved":          false,
		"network_used":              false,
	}
}

func readOnlyVerifyCommandResponse(result ReadOnlyVerifyResult) map[string]any {
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
		"target_path":                       result.TargetPath,
		"target_sha256":                     result.TargetSHA256,
		"target_size_bytes":                 result.TargetSizeBytes,
		"event_id":                          result.EventID,
		"audit_event_id":                    result.AuditEventID,
		"status":                            result.Status,
		"decision":                          result.Decision,
		"message":                           result.Message,
		"blockers":                          result.Blockers,
		"read_only_verify":                  true,
		"approval_gated":                    true,
		"execution_gate_status":             result.Gate.Status,
		"project_read_attempted":            result.ProjectReadAttempted,
		"project_read_allowed":              result.ProjectReadAllowed,
		"task_claimed":                      result.TaskClaimed,
		"worker_started":                    result.WorkerStarted,
		"lease_created":                     result.LeaseCreated,
		"attempt_created":                   result.AttemptCreated,
		"artifact_created":                  result.ArtifactCreated,
		"verification_passed":               result.VerificationPassed,
		"area_flow_execution_state_written": result.AreaFlowExecutionStateWritten,
		"project_write_attempted":           result.ProjectWriteAttempted,
		"execution_write_attempted":         result.ExecutionWriteAttempted,
		"engine_call_attempted":             result.EngineCallAttempted,
		"commands_run":                      result.CommandsRun,
		"secrets_resolved":                  result.SecretsResolved,
		"network_used":                      result.NetworkUsed,
	}
}

func loadReadOnlyVerifyQueueByCommandResponse(ctx context.Context, tx pgx.Tx, record Record, version WorkflowVersion, idempotencyKey string) (ReadOnlyVerifyQueueResult, error) {
	response, err := loadCommandResponse(ctx, tx, record.ID, readOnlyVerifyQueueCommandType, idempotencyKey)
	if err != nil {
		return ReadOnlyVerifyQueueResult{}, err
	}
	runID := metadataInt64(response, "run_id")
	taskID := metadataInt64(response, "run_task_id")
	run, err := loadRunForUpdate(ctx, tx, runID)
	if err != nil {
		return ReadOnlyVerifyQueueResult{}, err
	}
	task, err := loadRunTaskByID(ctx, tx, record.ID, taskID)
	if err != nil {
		return ReadOnlyVerifyQueueResult{}, err
	}
	return ReadOnlyVerifyQueueResult{
		Project:                 record,
		Version:                 version,
		Run:                     run,
		Task:                    task,
		TargetPath:              metadataString(response, "target_path"),
		IdempotencyKey:          idempotencyKey,
		EventID:                 metadataInt64(response, "event_id"),
		AuditEventID:            metadataInt64(response, "audit_event_id"),
		ProjectReadAttempted:    metadataBool(response, "project_read_attempted"),
		ProjectReadAllowed:      metadataBool(response, "project_read_allowed"),
		ProjectWriteAttempted:   metadataBool(response, "project_write_attempted"),
		ExecutionWriteAttempted: metadataBool(response, "execution_write_attempted"),
		EngineCallAttempted:     metadataBool(response, "engine_call_attempted"),
		CommandsRun:             metadataBool(response, "commands_run"),
		SecretsResolved:         metadataBool(response, "secrets_resolved"),
		NetworkUsed:             metadataBool(response, "network_used"),
	}, nil
}

func loadReadOnlyVerifyByCommandResponse(ctx context.Context, tx pgx.Tx, record Record, version WorkflowVersion, gate ExecutionApprovalGate, idempotencyKey string) (ReadOnlyVerifyResult, error) {
	response, err := loadCommandResponse(ctx, tx, record.ID, readOnlyVerifyApplyCommandType, idempotencyKey)
	if err != nil {
		return ReadOnlyVerifyResult{}, err
	}
	runID := metadataInt64(response, "run_id")
	workerID := metadataInt64(response, "worker_id")
	run, err := loadRunForUpdate(ctx, tx, runID)
	if err != nil {
		return ReadOnlyVerifyResult{}, err
	}
	worker := WorkerRecord{}
	if workerID != 0 {
		worker, err = loadWorkerByID(ctx, tx, record.ID, workerID)
		if err != nil {
			return ReadOnlyVerifyResult{}, err
		}
	}
	result := ReadOnlyVerifyResult{
		Project:                       record,
		Version:                       version,
		Run:                           run,
		Worker:                        worker,
		Gate:                          gate,
		TargetPath:                    metadataString(response, "target_path"),
		TargetSHA256:                  metadataString(response, "target_sha256"),
		TargetSizeBytes:               metadataInt64(response, "target_size_bytes"),
		Status:                        metadataString(response, "status"),
		Decision:                      metadataString(response, "decision"),
		Message:                       metadataString(response, "message"),
		Blockers:                      metadataStringSlice(response, "blockers"),
		IdempotencyKey:                idempotencyKey,
		EventID:                       metadataInt64(response, "event_id"),
		AuditEventID:                  metadataInt64(response, "audit_event_id"),
		ProjectReadAttempted:          metadataBool(response, "project_read_attempted"),
		ProjectReadAllowed:            metadataBool(response, "project_read_allowed"),
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
		VerificationPassed:            metadataBool(response, "verification_passed"),
	}
	if taskID := metadataInt64(response, "run_task_id"); taskID != 0 {
		result.Task, err = loadRunTaskByID(ctx, tx, record.ID, taskID)
		if err != nil {
			return ReadOnlyVerifyResult{}, err
		}
	}
	if leaseID := metadataInt64(response, "lease_id"); leaseID != 0 {
		result.Lease, err = loadLeaseByID(ctx, tx, record.ID, leaseID)
		if err != nil {
			return ReadOnlyVerifyResult{}, err
		}
	}
	if attemptID := metadataInt64(response, "attempt_id"); attemptID != 0 {
		result.Attempt, err = loadRunAttemptByID(ctx, tx, record.ID, attemptID)
		if err != nil {
			return ReadOnlyVerifyResult{}, err
		}
	}
	if artifactID := metadataInt64(response, "artifact_id"); artifactID != 0 {
		result.Artifact, err = loadArtifactByIDTx(ctx, tx, record.ID, artifactID)
		if err != nil {
			return ReadOnlyVerifyResult{}, err
		}
	}
	return result, nil
}
