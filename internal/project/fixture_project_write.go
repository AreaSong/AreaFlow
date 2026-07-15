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

var ErrFixtureProjectWriteBlocked = errors.New("fixture project write blocked")

const (
	fixtureProjectWriteQueueCommandType = "run.fixture_project_write_queue"
	fixtureProjectWriteApplyCommandType = "worker.fixture_project_write"
	maxFixtureProjectWriteContentBytes  = 256 * 1024
	maxFixtureProjectWriteFileBytes     = 2 * 1024 * 1024
)

type FixtureProjectWriteQueueOptions struct {
	TargetPath           string
	Content              string
	ExpectedBeforeSHA256 string
	ExpectedBeforeSize   int64
	IdempotencyKey       string
	Actor                string
	Reason               string
}

type FixtureProjectWriteQueueResult struct {
	Project                       Record
	Version                       WorkflowVersion
	Run                           RunRecord
	Task                          RunTaskRecord
	WriteSetArtifact              ArtifactRecord
	TargetPath                    string
	ExpectedBeforeSHA256          string
	ExpectedBeforeSize            int64
	AfterSHA256                   string
	AfterSize                     int64
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

type FixtureProjectWriteOptions struct {
	WorkerKey           string
	RunID               int64
	AllowedCapabilities []string
	LeaseTimeoutSeconds int
	Metadata            map[string]any
	IdempotencyKey      string
	Actor               string
	Reason              string
}

type FixtureProjectWriteResult struct {
	Project                       Record
	Version                       WorkflowVersion
	Run                           RunRecord
	Worker                        WorkerRecord
	Lease                         LeaseRecord
	Task                          RunTaskRecord
	CopyAttempt                   RunAttemptRecord
	VerifyAttempt                 RunAttemptRecord
	RollbackAttempt               RunAttemptRecord
	WriteSetArtifact              ArtifactRecord
	PreimageArtifact              ArtifactRecord
	Artifact                      ArtifactRecord
	Gate                          ExecutionApprovalGate
	TargetPath                    string
	ExpectedBeforeSHA256          string
	ExpectedBeforeSize            int64
	AfterSHA256                   string
	AfterSize                     int64
	RestoredSHA256                string
	RestoredSize                  int64
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
	ProjectWriteAllowed           bool
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
	WriteSetPassed                bool
	VerificationPassed            bool
	RollbackAttempted             bool
	RollbackVerified              bool
}

type fixtureProjectWriteSet struct {
	Operation              string   `json:"operation"`
	TargetPath             string   `json:"target_path"`
	TargetPathKind         string   `json:"target_path_kind"`
	ExpectedBeforeSHA256   string   `json:"expected_before_sha256"`
	ExpectedBeforeSize     int64    `json:"expected_before_size"`
	AfterSHA256            string   `json:"after_sha256"`
	AfterSize              int64    `json:"after_size"`
	Content                string   `json:"content"`
	PermissionCapabilities []string `json:"permission_capabilities"`
	ApprovalRequired       bool     `json:"approval_required"`
	RollbackMode           string   `json:"rollback_mode"`
}

type fixtureProjectFileImage struct {
	Content   []byte
	SHA256    string
	SizeBytes int64
	Mode      os.FileMode
}

func (s Store) QueueFixtureProjectWrite(ctx context.Context, record Record, label string, options FixtureProjectWriteQueueOptions) (FixtureProjectWriteQueueResult, error) {
	version, err := s.GetWorkflowVersion(ctx, record, label)
	if err != nil {
		return FixtureProjectWriteQueueResult{}, err
	}
	if version.ImportMode != "authored" {
		return FixtureProjectWriteQueueResult{}, fmt.Errorf("%w: %s", ErrWorkflowVersionNotAuthored, label)
	}
	options = normalizeFixtureProjectWriteQueueOptions(record, version, options)
	if options.TargetPath == "" {
		return FixtureProjectWriteQueueResult{}, fmt.Errorf("target path is required")
	}
	if options.ExpectedBeforeSHA256 == "" {
		return FixtureProjectWriteQueueResult{}, fmt.Errorf("expected before sha256 is required")
	}
	if len([]byte(options.Content)) > maxFixtureProjectWriteContentBytes {
		return FixtureProjectWriteQueueResult{}, fmt.Errorf("fixture project write content exceeds %d bytes", maxFixtureProjectWriteContentBytes)
	}
	requestHash, err := fixtureProjectWriteQueueRequestHash(record, version, options)
	if err != nil {
		return FixtureProjectWriteQueueResult{}, err
	}
	if options.IdempotencyKey == "" {
		options.IdempotencyKey = fixtureProjectWriteQueueIdempotencyKey(record, version, requestHash)
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return FixtureProjectWriteQueueResult{}, fmt.Errorf("begin fixture project write queue: %w", err)
	}
	defer tx.Rollback(ctx)

	created, err := reserveCommandRequest(ctx, tx, record.ID, fixtureProjectWriteQueueCommandType, options.IdempotencyKey, requestHash)
	if err != nil {
		return FixtureProjectWriteQueueResult{}, err
	}
	if !created {
		result, err := loadFixtureProjectWriteQueueByCommandResponse(ctx, tx, record, version, options.IdempotencyKey)
		if err != nil {
			return FixtureProjectWriteQueueResult{}, err
		}
		if err := tx.Commit(ctx); err != nil {
			return FixtureProjectWriteQueueResult{}, fmt.Errorf("commit fixture project write queue replay: %w", err)
		}
		result.Created = false
		return result, nil
	}

	run, err := insertFixtureProjectWriteRun(ctx, tx, record, version, options)
	if err != nil {
		return FixtureProjectWriteQueueResult{}, err
	}
	task, err := insertFixtureProjectWriteTask(ctx, tx, record, version, run, options)
	if err != nil {
		return FixtureProjectWriteQueueResult{}, err
	}
	writeSetArtifact, afterSHA, afterSize, err := writeAndInsertFixtureProjectWriteSetArtifact(ctx, tx, record, version, run, task, options)
	if err != nil {
		return FixtureProjectWriteQueueResult{}, err
	}
	if err := updateRunTaskMetadata(ctx, tx, task.ID, map[string]any{
		"write_set_artifact_id":     writeSetArtifact.ID,
		"target_path":               options.TargetPath,
		"expected_before_sha256":    options.ExpectedBeforeSHA256,
		"expected_before_size":      options.ExpectedBeforeSize,
		"after_sha256":              afterSHA,
		"after_size":                afterSize,
		"fixture_project_write":     true,
		"approved_write_set_source": "area_flow_artifact",
	}); err != nil {
		return FixtureProjectWriteQueueResult{}, err
	}
	task.Metadata["write_set_artifact_id"] = float64(writeSetArtifact.ID)
	result := FixtureProjectWriteQueueResult{
		Project:                       record,
		Version:                       version,
		Run:                           run,
		Task:                          task,
		WriteSetArtifact:              writeSetArtifact,
		TargetPath:                    options.TargetPath,
		ExpectedBeforeSHA256:          options.ExpectedBeforeSHA256,
		ExpectedBeforeSize:            options.ExpectedBeforeSize,
		AfterSHA256:                   afterSHA,
		AfterSize:                     afterSize,
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
	}
	eventID, err := insertFixtureProjectWriteQueueEvent(ctx, tx, result, options)
	if err != nil {
		return FixtureProjectWriteQueueResult{}, err
	}
	result.EventID = eventID
	auditEventID, err := insertFixtureProjectWriteQueueAuditEvent(ctx, tx, result, options)
	if err != nil {
		return FixtureProjectWriteQueueResult{}, err
	}
	result.AuditEventID = auditEventID
	if err := completeCommandRequestResponse(ctx, tx, record.ID, fixtureProjectWriteQueueCommandType, options.IdempotencyKey, fixtureProjectWriteQueueCommandResponse(result)); err != nil {
		return FixtureProjectWriteQueueResult{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return FixtureProjectWriteQueueResult{}, fmt.Errorf("commit fixture project write queue: %w", err)
	}
	return result, nil
}

func (s Store) WriteFixtureProject(ctx context.Context, record Record, options FixtureProjectWriteOptions) (FixtureProjectWriteResult, error) {
	if options.RunID <= 0 {
		return FixtureProjectWriteResult{}, fmt.Errorf("run id is required")
	}
	options = normalizeFixtureProjectWriteOptions(options)
	requestHash, err := fixtureProjectWriteRequestHash(record, options)
	if err != nil {
		return FixtureProjectWriteResult{}, err
	}
	if options.IdempotencyKey == "" {
		options.IdempotencyKey = fixtureProjectWriteIdempotencyKey(record, options, requestHash)
	}

	gate, err := s.ExecutionApprovalGate(ctx, options.RunID, ExecutionApprovalGateOptions{
		RequiredCapabilities: options.AllowedCapabilities,
		SkipEnginePreview:    true,
		Mode:                 "fixture_project_write_gate",
	})
	if err != nil {
		return FixtureProjectWriteResult{}, err
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return FixtureProjectWriteResult{}, fmt.Errorf("begin fixture project write: %w", err)
	}
	defer tx.Rollback(ctx)

	run, err := loadRunForUpdate(ctx, tx, options.RunID)
	if err != nil {
		return FixtureProjectWriteResult{}, err
	}
	if run.ProjectID != record.ID {
		return FixtureProjectWriteResult{}, fmt.Errorf("%w: run %d does not belong to project %s", ErrRunNotFound, options.RunID, record.Key)
	}
	version, err := workflowVersionByIDTx(ctx, tx, record.ID, run.WorkflowVersionID)
	if err != nil {
		return FixtureProjectWriteResult{}, err
	}
	worker, err := loadWorkerForUpdate(ctx, tx, record.ID, options.WorkerKey)
	if err != nil {
		return FixtureProjectWriteResult{}, err
	}

	created, err := reserveCommandRequest(ctx, tx, record.ID, fixtureProjectWriteApplyCommandType, options.IdempotencyKey, requestHash)
	if err != nil {
		return FixtureProjectWriteResult{}, err
	}
	if !created {
		result, err := loadFixtureProjectWriteByCommandResponse(ctx, tx, record, version, gate, options.IdempotencyKey)
		if err != nil {
			return FixtureProjectWriteResult{}, err
		}
		if err := tx.Commit(ctx); err != nil {
			return FixtureProjectWriteResult{}, fmt.Errorf("commit fixture project write replay: %w", err)
		}
		result.Created = false
		return result, nil
	}

	if gate.Status != "pass" {
		result := deniedFixtureProjectWriteResult(record, version, run, worker, gate, options, "fixture project write gate blocked", gate.Blockers)
		if err := finishDeniedFixtureProjectWrite(ctx, tx, result, options); err != nil {
			return FixtureProjectWriteResult{}, err
		}
		if err := tx.Commit(ctx); err != nil {
			return FixtureProjectWriteResult{}, fmt.Errorf("commit blocked fixture project write: %w", err)
		}
		return result, fmt.Errorf("%w: %s", ErrFixtureProjectWriteBlocked, strings.Join(result.Blockers, "; "))
	}
	if !isFixtureProjectRecord(record) {
		result := deniedFixtureProjectWriteResult(record, version, run, worker, gate, options, "project is not marked as fixture", []string{"fixture_project_write requires project key or kind to contain fixture"})
		if err := finishDeniedFixtureProjectWrite(ctx, tx, result, options); err != nil {
			return FixtureProjectWriteResult{}, err
		}
		if err := tx.Commit(ctx); err != nil {
			return FixtureProjectWriteResult{}, fmt.Errorf("commit non-fixture project write denial: %w", err)
		}
		return result, fmt.Errorf("%w: project is not fixture scoped", ErrFixtureProjectWriteBlocked)
	}
	if missing := missingWorkerCapabilities(worker.Capabilities, options.AllowedCapabilities); len(missing) > 0 {
		blockers := []string{"worker missing required capabilities: " + strings.Join(missing, ",")}
		result := deniedFixtureProjectWriteResult(record, version, run, worker, gate, options, "worker capability denied", blockers)
		if err := finishDeniedFixtureProjectWrite(ctx, tx, result, options); err != nil {
			return FixtureProjectWriteResult{}, err
		}
		if err := tx.Commit(ctx); err != nil {
			return FixtureProjectWriteResult{}, fmt.Errorf("commit denied fixture project write: %w", err)
		}
		return result, fmt.Errorf("%w: missing %s", ErrWorkerCapabilityDenied, strings.Join(missing, ","))
	}
	if allowed, reason, err := canProjectCapabilityInTx(ctx, tx, record.ID, "write_artifacts"); err != nil {
		return FixtureProjectWriteResult{}, err
	} else if !allowed {
		blockers := []string{"project write_artifacts capability denied: " + reason}
		result := deniedFixtureProjectWriteResult(record, version, run, worker, gate, options, "project artifact write denied", blockers)
		if err := finishDeniedFixtureProjectWrite(ctx, tx, result, options); err != nil {
			return FixtureProjectWriteResult{}, err
		}
		if err := tx.Commit(ctx); err != nil {
			return FixtureProjectWriteResult{}, fmt.Errorf("commit artifact permission denial: %w", err)
		}
		return result, fmt.Errorf("%w: %s", ErrFixtureProjectWriteBlocked, strings.Join(blockers, "; "))
	}

	task, ok, err := nextFixtureProjectWriteTaskForLease(ctx, tx, record.ID, run.ID)
	if err != nil {
		return FixtureProjectWriteResult{}, err
	}
	if !ok {
		result := deniedFixtureProjectWriteResult(record, version, run, worker, gate, options, "no queued fixture project write task", []string{"no queued or needs_recovery fixture project write task is available"})
		if err := finishDeniedFixtureProjectWrite(ctx, tx, result, options); err != nil {
			return FixtureProjectWriteResult{}, err
		}
		if err := tx.Commit(ctx); err != nil {
			return FixtureProjectWriteResult{}, fmt.Errorf("commit idle fixture project write: %w", err)
		}
		return result, fmt.Errorf("%w: no queued task", ErrNoLeaseAvailable)
	}
	targetPath := metadataString(task.Metadata, "target_path")
	if allowed, reason, err := canProjectPathInTx(ctx, tx, record.ID, "read_project", targetPath); err != nil {
		return FixtureProjectWriteResult{}, err
	} else if !allowed {
		blockers := []string{"target path is not readable: " + reason}
		result := deniedFixtureProjectWriteResult(record, version, run, worker, gate, options, "read path denied", blockers)
		result.TargetPath = targetPath
		if err := finishDeniedFixtureProjectWrite(ctx, tx, result, options); err != nil {
			return FixtureProjectWriteResult{}, err
		}
		if err := tx.Commit(ctx); err != nil {
			return FixtureProjectWriteResult{}, fmt.Errorf("commit read path denial: %w", err)
		}
		return result, fmt.Errorf("%w: %s", ErrFixtureProjectWriteBlocked, strings.Join(blockers, "; "))
	}
	if allowed, reason, err := canProjectPathInTx(ctx, tx, record.ID, "write_code", targetPath); err != nil {
		return FixtureProjectWriteResult{}, err
	} else if !allowed {
		blockers := []string{"target path is not writable: " + reason}
		result := deniedFixtureProjectWriteResult(record, version, run, worker, gate, options, "write path denied", blockers)
		result.TargetPath = targetPath
		if err := finishDeniedFixtureProjectWrite(ctx, tx, result, options); err != nil {
			return FixtureProjectWriteResult{}, err
		}
		if err := tx.Commit(ctx); err != nil {
			return FixtureProjectWriteResult{}, fmt.Errorf("commit write path denial: %w", err)
		}
		return result, fmt.Errorf("%w: %s", ErrFixtureProjectWriteBlocked, strings.Join(blockers, "; "))
	}
	fullPath, err := safeFixtureProjectWritePath(record.RootPath, targetPath)
	if err != nil {
		blockers := []string{"target path is unsafe: " + err.Error()}
		result := deniedFixtureProjectWriteResult(record, version, run, worker, gate, options, "write path unsafe", blockers)
		result.TargetPath = targetPath
		if err := finishDeniedFixtureProjectWrite(ctx, tx, result, options); err != nil {
			return FixtureProjectWriteResult{}, err
		}
		if err := tx.Commit(ctx); err != nil {
			return FixtureProjectWriteResult{}, fmt.Errorf("commit unsafe path denial: %w", err)
		}
		return result, fmt.Errorf("%w: %s", ErrFixtureProjectWriteBlocked, strings.Join(blockers, "; "))
	}
	writeSet, writeSetArtifact, err := loadFixtureProjectWriteSet(ctx, tx, record, metadataInt64(task.Metadata, "write_set_artifact_id"))
	if err != nil {
		return FixtureProjectWriteResult{}, err
	}
	if writeSet.TargetPath != targetPath {
		return FixtureProjectWriteResult{}, fmt.Errorf("write-set target path mismatch")
	}

	preimage, err := readFixtureProjectFileImage(fullPath)
	if err != nil {
		return FixtureProjectWriteResult{}, err
	}
	if preimage.SHA256 != writeSet.ExpectedBeforeSHA256 || preimage.SizeBytes != writeSet.ExpectedBeforeSize {
		blockers := []string{"expected-before hash or size does not match current fixture file"}
		result := deniedFixtureProjectWriteResult(record, version, run, worker, gate, options, "expected-before mismatch", blockers)
		result.TargetPath = targetPath
		result.WriteSetArtifact = writeSetArtifact
		result.ProjectReadAttempted = true
		result.ProjectReadAllowed = true
		result.ExpectedBeforeSHA256 = writeSet.ExpectedBeforeSHA256
		result.ExpectedBeforeSize = writeSet.ExpectedBeforeSize
		result.AfterSHA256 = writeSet.AfterSHA256
		result.AfterSize = writeSet.AfterSize
		if err := finishDeniedFixtureProjectWrite(ctx, tx, result, options); err != nil {
			return FixtureProjectWriteResult{}, err
		}
		if err := tx.Commit(ctx); err != nil {
			return FixtureProjectWriteResult{}, fmt.Errorf("commit expected-before denial: %w", err)
		}
		return result, fmt.Errorf("%w: %s", ErrFixtureProjectWriteBlocked, strings.Join(blockers, "; "))
	}

	lease, err := insertLeaseForTask(ctx, tx, record.ID, worker, task, "fixture_project_write", options.AllowedCapabilities, map[string]any{
		"run_id":                task.RunID,
		"run_task_id":           task.ID,
		"task_key":              task.TaskKey,
		"task_kind":             task.TaskKind,
		"target_path":           targetPath,
		"fixture_project_write": true,
		"approval_gated":        true,
	}, options.Metadata, options.LeaseTimeoutSeconds)
	if err != nil {
		return FixtureProjectWriteResult{}, err
	}
	if err := updateRunTaskStatus(ctx, tx, task.ID, "leased"); err != nil {
		return FixtureProjectWriteResult{}, err
	}

	preimageArtifact, err := writeAndInsertFixtureProjectPreimageArtifact(ctx, tx, record, version, run, task, targetPath, preimage, options)
	if err != nil {
		return FixtureProjectWriteResult{}, err
	}
	projectFileRestored := false
	projectFileChanged := false
	defer func() {
		if projectFileChanged && !projectFileRestored {
			_ = os.WriteFile(fullPath, preimage.Content, preimage.Mode.Perm())
		}
	}()

	if err := os.WriteFile(fullPath, []byte(writeSet.Content), preimage.Mode.Perm()); err != nil {
		return FixtureProjectWriteResult{}, fmt.Errorf("write fixture project target: %w", err)
	}
	projectFileChanged = true
	afterImage, err := readFixtureProjectFileImage(fullPath)
	if err != nil {
		return FixtureProjectWriteResult{}, err
	}
	copyAttempt, err := insertFixtureProjectWriteAttempt(ctx, tx, record, task, lease, "copy", "passed", map[string]any{
		"target_path":            targetPath,
		"expected_before_sha256": writeSet.ExpectedBeforeSHA256,
		"after_sha256":           afterImage.SHA256,
		"after_size":             afterImage.SizeBytes,
		"preimage_artifact_id":   preimageArtifact.ID,
		"write_set_artifact_id":  writeSetArtifact.ID,
	}, options)
	if err != nil {
		return FixtureProjectWriteResult{}, err
	}
	verifyStatus := "passed"
	if afterImage.SHA256 != writeSet.AfterSHA256 || afterImage.SizeBytes != writeSet.AfterSize {
		verifyStatus = "failed"
	}
	verifyAttempt, err := insertFixtureProjectWriteAttempt(ctx, tx, record, task, lease, "verify", verifyStatus, map[string]any{
		"target_path":     targetPath,
		"after_sha256":    afterImage.SHA256,
		"after_size":      afterImage.SizeBytes,
		"expected_sha256": writeSet.AfterSHA256,
		"expected_size":   writeSet.AfterSize,
	}, options)
	if err != nil {
		return FixtureProjectWriteResult{}, err
	}
	if verifyStatus != "passed" {
		return FixtureProjectWriteResult{}, fmt.Errorf("fixture project write verify failed")
	}
	if afterImage.SHA256 != writeSet.AfterSHA256 {
		return FixtureProjectWriteResult{}, fmt.Errorf("fixture project write after hash mismatch")
	}
	if err := os.WriteFile(fullPath, preimage.Content, preimage.Mode.Perm()); err != nil {
		return FixtureProjectWriteResult{}, fmt.Errorf("rollback fixture project target: %w", err)
	}
	restoredImage, err := readFixtureProjectFileImage(fullPath)
	if err != nil {
		return FixtureProjectWriteResult{}, err
	}
	if restoredImage.SHA256 != preimage.SHA256 || restoredImage.SizeBytes != preimage.SizeBytes {
		return FixtureProjectWriteResult{}, fmt.Errorf("fixture project rollback did not restore expected hash")
	}
	projectFileRestored = true
	rollbackAttempt, err := insertFixtureProjectWriteAttempt(ctx, tx, record, task, lease, "rollback", "passed", map[string]any{
		"target_path":       targetPath,
		"restored_sha256":   restoredImage.SHA256,
		"restored_size":     restoredImage.SizeBytes,
		"preimage_artifact": preimageArtifact.ID,
	}, options)
	if err != nil {
		return FixtureProjectWriteResult{}, err
	}
	reportArtifact, err := writeAndInsertFixtureProjectWriteReport(ctx, tx, record, version, run, worker, task, lease, gate, writeSetArtifact, preimageArtifact, copyAttempt, verifyAttempt, rollbackAttempt, writeSet, preimage, afterImage, restoredImage, options)
	if err != nil {
		return FixtureProjectWriteResult{}, err
	}
	released, err := releaseLeaseForWorker(ctx, tx, record.ID, worker, lease.ID, "completed", map[string]any{
		"fixture_project_write": true,
		"target_path":           targetPath,
		"attempt_ids":           []int64{copyAttempt.ID, verifyAttempt.ID, rollbackAttempt.ID},
		"artifact_id":           reportArtifact.ID,
	})
	if err != nil {
		return FixtureProjectWriteResult{}, err
	}
	if err := updateRunTaskStatus(ctx, tx, task.ID, "rollback_verified"); err != nil {
		return FixtureProjectWriteResult{}, err
	}
	task.Status = "rollback_verified"
	run, err = updateFixtureProjectWriteRunAfterTask(ctx, tx, run, options, reportArtifact.ID, rollbackAttempt.ID, targetPath, writeSet, restoredImage)
	if err != nil {
		return FixtureProjectWriteResult{}, err
	}
	result := FixtureProjectWriteResult{
		Project:                       record,
		Version:                       version,
		Run:                           run,
		Worker:                        worker,
		Lease:                         released,
		Task:                          task,
		CopyAttempt:                   copyAttempt,
		VerifyAttempt:                 verifyAttempt,
		RollbackAttempt:               rollbackAttempt,
		WriteSetArtifact:              writeSetArtifact,
		PreimageArtifact:              preimageArtifact,
		Artifact:                      reportArtifact,
		Gate:                          gate,
		TargetPath:                    targetPath,
		ExpectedBeforeSHA256:          writeSet.ExpectedBeforeSHA256,
		ExpectedBeforeSize:            writeSet.ExpectedBeforeSize,
		AfterSHA256:                   afterImage.SHA256,
		AfterSize:                     afterImage.SizeBytes,
		RestoredSHA256:                restoredImage.SHA256,
		RestoredSize:                  restoredImage.SizeBytes,
		Status:                        "rollback_verified",
		Decision:                      "allowed",
		Message:                       "fixture project write verified and rolled back",
		Created:                       true,
		IdempotencyKey:                options.IdempotencyKey,
		ProjectReadAttempted:          true,
		ProjectReadAllowed:            true,
		ProjectWriteAttempted:         true,
		ProjectWriteAllowed:           true,
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
		WriteSetPassed:                true,
		VerificationPassed:            true,
		RollbackAttempted:             true,
		RollbackVerified:              true,
	}
	eventID, err := insertFixtureProjectWriteEvent(ctx, tx, result, options)
	if err != nil {
		return FixtureProjectWriteResult{}, err
	}
	result.EventID = eventID
	auditEventID, err := insertFixtureProjectWriteAuditEvent(ctx, tx, result, options)
	if err != nil {
		return FixtureProjectWriteResult{}, err
	}
	result.AuditEventID = auditEventID
	if err := completeCommandRequestResponse(ctx, tx, record.ID, fixtureProjectWriteApplyCommandType, options.IdempotencyKey, fixtureProjectWriteCommandResponse(result)); err != nil {
		return FixtureProjectWriteResult{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return FixtureProjectWriteResult{}, fmt.Errorf("commit fixture project write: %w", err)
	}
	return result, nil
}

func normalizeFixtureProjectWriteQueueOptions(record Record, version WorkflowVersion, options FixtureProjectWriteQueueOptions) FixtureProjectWriteQueueOptions {
	options.TargetPath = normalizeProjectRelativePath(options.TargetPath)
	options.ExpectedBeforeSHA256 = strings.ToLower(strings.TrimSpace(options.ExpectedBeforeSHA256))
	options.IdempotencyKey = strings.TrimSpace(options.IdempotencyKey)
	options.Actor = strings.TrimSpace(options.Actor)
	options.Reason = strings.TrimSpace(options.Reason)
	if options.Actor == "" {
		options.Actor = "local-user"
	}
	if options.Reason == "" {
		options.Reason = "queue fixture project write run"
	}
	if options.IdempotencyKey == "" && options.TargetPath != "" && options.ExpectedBeforeSHA256 != "" {
		hash, err := fixtureProjectWriteQueueRequestHash(record, version, options)
		if err == nil {
			options.IdempotencyKey = fixtureProjectWriteQueueIdempotencyKey(record, version, hash)
		}
	}
	return options
}

func normalizeFixtureProjectWriteOptions(options FixtureProjectWriteOptions) FixtureProjectWriteOptions {
	options.WorkerKey = strings.TrimSpace(options.WorkerKey)
	options.IdempotencyKey = strings.TrimSpace(options.IdempotencyKey)
	options.Actor = strings.TrimSpace(options.Actor)
	options.Reason = strings.TrimSpace(options.Reason)
	if len(options.AllowedCapabilities) == 0 {
		options.AllowedCapabilities = []string{"read_project", "write_artifacts", "write_code"}
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
		options.Reason = "approval-gated fixture project write"
	}
	return options
}

func fixtureProjectWriteQueueRequestHash(record Record, version WorkflowVersion, options FixtureProjectWriteQueueOptions) (string, error) {
	afterSHA, afterSize := hashBytes([]byte(options.Content))
	payload := map[string]any{
		"command_type":            fixtureProjectWriteQueueCommandType,
		"project_id":              record.ID,
		"project_key":             record.Key,
		"version_id":              version.ID,
		"display_label":           version.DisplayLabel,
		"target_path":             options.TargetPath,
		"expected_before_sha256":  options.ExpectedBeforeSHA256,
		"expected_before_size":    options.ExpectedBeforeSize,
		"after_sha256":            afterSHA,
		"after_size":              afterSize,
		"content_sha256":          afterSHA,
		"content_size":            afterSize,
		"fixture_project_write":   true,
		"rollback_drill_required": true,
		"actor":                   options.Actor,
		"reason":                  options.Reason,
	}
	content, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("marshal fixture project write queue request hash payload: %w", err)
	}
	sum := sha256.Sum256(content)
	return hex.EncodeToString(sum[:]), nil
}

func fixtureProjectWriteRequestHash(record Record, options FixtureProjectWriteOptions) (string, error) {
	payload := map[string]any{
		"command_type":          fixtureProjectWriteApplyCommandType,
		"project_id":            record.ID,
		"project_key":           record.Key,
		"worker_key":            options.WorkerKey,
		"run_id":                options.RunID,
		"allowed_capabilities":  options.AllowedCapabilities,
		"lease_timeout_seconds": options.LeaseTimeoutSeconds,
		"metadata":              options.Metadata,
		"actor":                 options.Actor,
		"reason":                options.Reason,
		"fixture_project_write": true,
	}
	content, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("marshal fixture project write request hash payload: %w", err)
	}
	sum := sha256.Sum256(content)
	return hex.EncodeToString(sum[:]), nil
}

func fixtureProjectWriteQueueIdempotencyKey(record Record, version WorkflowVersion, requestHash string) string {
	return fmt.Sprintf("%s:%s:%s:%s", fixtureProjectWriteQueueCommandType, record.Key, version.DisplayLabel, commandHashPrefix(requestHash))
}

func fixtureProjectWriteIdempotencyKey(record Record, options FixtureProjectWriteOptions, requestHash string) string {
	return fmt.Sprintf("%s:%s:%s:%d:%s", fixtureProjectWriteApplyCommandType, record.Key, options.WorkerKey, options.RunID, commandHashPrefix(requestHash))
}

func insertFixtureProjectWriteRun(ctx context.Context, tx pgx.Tx, record Record, version WorkflowVersion, options FixtureProjectWriteQueueOptions) (RunRecord, error) {
	afterSHA, afterSize := hashBytes([]byte(options.Content))
	summary, err := json.Marshal(map[string]any{
		"fixture_project_write":      true,
		"approval_gated":             true,
		"target_path":                options.TargetPath,
		"expected_before_sha256":     options.ExpectedBeforeSHA256,
		"expected_before_size":       options.ExpectedBeforeSize,
		"after_sha256":               afterSHA,
		"after_size":                 afterSize,
		"project_read_attempted":     false,
		"project_write_attempted":    false,
		"project_write_allowed":      false,
		"rollback_drill_required":    true,
		"rollback_verified":          false,
		"area_flow_artifact_written": false,
		"area_flow_execution_state":  "queued",
		"execution_write_attempted":  false,
		"engine_call_attempted":      false,
		"commands_run":               false,
		"secrets_resolved":           false,
		"network_used":               false,
	})
	if err != nil {
		return RunRecord{}, fmt.Errorf("marshal fixture project write run summary: %w", err)
	}
	metadata, err := json.Marshal(map[string]any{
		"phase":                 "v0.6n",
		"fixture_project_write": true,
		"approval_gated":        true,
		"target_path":           options.TargetPath,
		"actor":                 options.Actor,
		"reason":                options.Reason,
	})
	if err != nil {
		return RunRecord{}, fmt.Errorf("marshal fixture project write run metadata: %w", err)
	}
	run, err := scanRun(tx.QueryRow(ctx, `
INSERT INTO runs (project_id, workflow_version_id, run_type, run_kind, status, risk_level, risk_policy, dry_run, summary, metadata)
VALUES ($1, $2, 'fixture_project_write', 'execution', 'queued', 'medium', 'pause', false, $3::jsonb, $4::jsonb)
RETURNING id, COALESCE(project_id, 0), COALESCE(workflow_version_id, 0), run_type,
          COALESCE(run_kind, ''), status, risk_level, risk_policy, dry_run,
          summary, metadata, started_at, finished_at`,
		record.ID,
		version.ID,
		string(summary),
		string(metadata),
	))
	if err != nil {
		return RunRecord{}, fmt.Errorf("insert fixture project write run: %w", err)
	}
	return run, nil
}

func insertFixtureProjectWriteTask(ctx context.Context, tx pgx.Tx, record Record, version WorkflowVersion, run RunRecord, options FixtureProjectWriteQueueOptions) (RunTaskRecord, error) {
	afterSHA, afterSize := hashBytes([]byte(options.Content))
	metadata, err := json.Marshal(map[string]any{
		"phase":                  "v0.6n",
		"fixture_project_write":  true,
		"approval_gated":         true,
		"operation":              "modify",
		"target_path":            options.TargetPath,
		"target_path_kind":       "fixture_file",
		"expected_before_sha256": options.ExpectedBeforeSHA256,
		"expected_before_size":   options.ExpectedBeforeSize,
		"after_sha256":           afterSHA,
		"after_size":             afterSize,
		"actor":                  options.Actor,
		"reason":                 options.Reason,
	})
	if err != nil {
		return RunTaskRecord{}, fmt.Errorf("marshal fixture project write task metadata: %w", err)
	}
	task, err := scanRunTask(tx.QueryRow(ctx, `
INSERT INTO run_tasks (
    project_id, workflow_version_id, run_id, task_key, task_kind, status, risk_level, sequence, metadata
)
VALUES ($1, $2, $3, $4, 'fixture_project_write_task', 'queued', 'medium', 1, $5::jsonb)
RETURNING id, project_id, COALESCE(workflow_version_id, 0), COALESCE(workflow_item_id, 0),
          run_id, task_key, task_kind, status, risk_level, sequence, metadata,
          created_at, updated_at`,
		record.ID,
		version.ID,
		run.ID,
		version.DisplayLabel+":fixture-project-write:"+options.TargetPath,
		string(metadata),
	))
	if err != nil {
		return RunTaskRecord{}, fmt.Errorf("insert fixture project write task: %w", err)
	}
	return task, nil
}

func updateRunTaskMetadata(ctx context.Context, tx pgx.Tx, taskID int64, additions map[string]any) error {
	current := map[string]any{}
	var raw []byte
	if err := tx.QueryRow(ctx, `SELECT metadata FROM run_tasks WHERE id = $1`, taskID).Scan(&raw); err != nil {
		return fmt.Errorf("load run task metadata: %w", err)
	}
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &current); err != nil {
			return fmt.Errorf("parse run task metadata: %w", err)
		}
	}
	for key, value := range additions {
		current[key] = value
	}
	updated, err := json.Marshal(current)
	if err != nil {
		return fmt.Errorf("marshal run task metadata: %w", err)
	}
	if _, err := tx.Exec(ctx, `UPDATE run_tasks SET metadata = $2::jsonb, updated_at = now() WHERE id = $1`, taskID, string(updated)); err != nil {
		return fmt.Errorf("update run task metadata: %w", err)
	}
	return nil
}

func writeAndInsertFixtureProjectWriteSetArtifact(ctx context.Context, tx pgx.Tx, record Record, version WorkflowVersion, run RunRecord, task RunTaskRecord, options FixtureProjectWriteQueueOptions) (ArtifactRecord, string, int64, error) {
	afterSHA, afterSize := hashBytes([]byte(options.Content))
	writeSet := fixtureProjectWriteSet{
		Operation:              "modify",
		TargetPath:             options.TargetPath,
		TargetPathKind:         "fixture_file",
		ExpectedBeforeSHA256:   options.ExpectedBeforeSHA256,
		ExpectedBeforeSize:     options.ExpectedBeforeSize,
		AfterSHA256:            afterSHA,
		AfterSize:              afterSize,
		Content:                options.Content,
		PermissionCapabilities: []string{"read_project", "write_artifacts", "write_code"},
		ApprovalRequired:       true,
		RollbackMode:           "restore_preimage",
	}
	content, err := json.MarshalIndent(writeSet, "", "  ")
	if err != nil {
		return ArtifactRecord{}, "", 0, fmt.Errorf("marshal fixture project write-set: %w", err)
	}
	relativePath := filepath.Join("versions", version.DisplayLabel, "fixture-project-write", fmt.Sprintf("run-%d-task-%d-write-set.json", run.ID, task.ID))
	stored, err := writeProjectArtifact(record, relativePath, content, "application/json")
	if err != nil {
		return ArtifactRecord{}, "", 0, err
	}
	metadata, err := json.Marshal(map[string]any{
		"phase":                  "v0.6n",
		"owned_by":               "areaflow",
		"fixture_project_write":  true,
		"artifact_role":          "approved_write_set",
		"target_path":            options.TargetPath,
		"expected_before_sha256": options.ExpectedBeforeSHA256,
		"expected_before_size":   options.ExpectedBeforeSize,
		"after_sha256":           afterSHA,
		"after_size":             afterSize,
		"actor":                  options.Actor,
		"reason":                 options.Reason,
	})
	if err != nil {
		return ArtifactRecord{}, "", 0, fmt.Errorf("marshal fixture project write-set artifact metadata: %w", err)
	}
	artifact, err := insertRunArtifactRecord(ctx, tx, record.ID, version.ID, run.ID, task.WorkflowItemID, "fixture_project_write_set", relativePath, stored, string(metadata))
	if err != nil {
		return ArtifactRecord{}, "", 0, err
	}
	return artifact, afterSHA, afterSize, nil
}

func nextFixtureProjectWriteTaskForLease(ctx context.Context, tx pgx.Tx, projectID int64, runID int64) (RunTaskRecord, bool, error) {
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
  AND rt.task_kind = 'fixture_project_write_task'
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
		return RunTaskRecord{}, false, fmt.Errorf("load next fixture project write task: %w", err)
	}
	return task, true, nil
}

func isFixtureProjectRecord(record Record) bool {
	key := strings.ToLower(record.Key)
	kind := strings.ToLower(record.Kind)
	return strings.Contains(key, "fixture") || strings.Contains(kind, "fixture")
}

func safeFixtureProjectWritePath(root string, targetPath string) (string, error) {
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
	info, err := os.Lstat(full)
	if err != nil {
		return "", fmt.Errorf("fixture target must already exist: %w", err)
	}
	if info.IsDir() {
		return "", fmt.Errorf("fixture target must be a file")
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return "", fmt.Errorf("symlink target is not supported")
	}
	fullReal, err := filepath.EvalSymlinks(full)
	if err != nil {
		return "", fmt.Errorf("resolve fixture target: %w", err)
	}
	rel, err := filepath.Rel(rootReal, fullReal)
	if err != nil {
		return "", fmt.Errorf("compare fixture target path: %w", err)
	}
	if rel == "." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || rel == ".." || filepath.IsAbs(rel) {
		return "", fmt.Errorf("target path escapes project root")
	}
	return fullReal, nil
}

func readFixtureProjectFileImage(path string) (fixtureProjectFileImage, error) {
	info, err := os.Stat(path)
	if err != nil {
		return fixtureProjectFileImage{}, fmt.Errorf("stat fixture target: %w", err)
	}
	if info.IsDir() {
		return fixtureProjectFileImage{}, fmt.Errorf("fixture target must be a file")
	}
	if info.Size() > maxFixtureProjectWriteFileBytes {
		return fixtureProjectFileImage{}, fmt.Errorf("fixture target exceeds %d bytes", maxFixtureProjectWriteFileBytes)
	}
	content, err := os.ReadFile(path)
	if err != nil {
		return fixtureProjectFileImage{}, fmt.Errorf("read fixture target: %w", err)
	}
	sha, size := hashBytes(content)
	return fixtureProjectFileImage{
		Content:   content,
		SHA256:    sha,
		SizeBytes: size,
		Mode:      info.Mode(),
	}, nil
}

func hashBytes(content []byte) (string, int64) {
	sum := sha256.Sum256(content)
	return hex.EncodeToString(sum[:]), int64(len(content))
}

func loadFixtureProjectWriteSet(ctx context.Context, tx pgx.Tx, record Record, artifactID int64) (fixtureProjectWriteSet, ArtifactRecord, error) {
	if artifactID <= 0 {
		return fixtureProjectWriteSet{}, ArtifactRecord{}, fmt.Errorf("write-set artifact id is required")
	}
	artifact, err := loadArtifactByIDTx(ctx, tx, record.ID, artifactID)
	if err != nil {
		return fixtureProjectWriteSet{}, ArtifactRecord{}, err
	}
	artifactContent, err := ReadArtifactContent(artifact)
	if err != nil {
		return fixtureProjectWriteSet{}, ArtifactRecord{}, fmt.Errorf("read fixture write-set artifact: %w", err)
	}
	content := artifactContent.Content
	var writeSet fixtureProjectWriteSet
	if err := json.Unmarshal(content, &writeSet); err != nil {
		return fixtureProjectWriteSet{}, ArtifactRecord{}, fmt.Errorf("parse fixture write-set artifact: %w", err)
	}
	if writeSet.Operation != "modify" {
		return fixtureProjectWriteSet{}, ArtifactRecord{}, fmt.Errorf("only modify operation is supported")
	}
	if writeSet.TargetPath == "" || writeSet.ExpectedBeforeSHA256 == "" || writeSet.AfterSHA256 == "" {
		return fixtureProjectWriteSet{}, ArtifactRecord{}, fmt.Errorf("write-set artifact is incomplete")
	}
	return writeSet, artifact, nil
}

func writeAndInsertFixtureProjectPreimageArtifact(ctx context.Context, tx pgx.Tx, record Record, version WorkflowVersion, run RunRecord, task RunTaskRecord, targetPath string, preimage fixtureProjectFileImage, options FixtureProjectWriteOptions) (ArtifactRecord, error) {
	relativePath := filepath.Join("versions", version.DisplayLabel, "fixture-project-write", fmt.Sprintf("run-%d-task-%d-preimage.bin", run.ID, task.ID))
	stored, err := writeProjectArtifact(record, relativePath, preimage.Content, "application/octet-stream")
	if err != nil {
		return ArtifactRecord{}, err
	}
	metadata, err := json.Marshal(map[string]any{
		"phase":                 "v0.6n",
		"owned_by":              "areaflow",
		"fixture_project_write": true,
		"artifact_role":         "preimage",
		"target_path":           targetPath,
		"target_sha256":         preimage.SHA256,
		"target_size":           preimage.SizeBytes,
		"actor":                 options.Actor,
		"reason":                options.Reason,
	})
	if err != nil {
		return ArtifactRecord{}, fmt.Errorf("marshal fixture project preimage metadata: %w", err)
	}
	return insertRunArtifactRecord(ctx, tx, record.ID, version.ID, run.ID, task.WorkflowItemID, "fixture_project_write_preimage", relativePath, stored, string(metadata))
}

func insertFixtureProjectWriteAttempt(ctx context.Context, tx pgx.Tx, record Record, task RunTaskRecord, lease LeaseRecord, attemptKind string, status string, metadata map[string]any, options FixtureProjectWriteOptions) (RunAttemptRecord, error) {
	base := map[string]any{
		"actor":                             options.Actor,
		"reason":                            options.Reason,
		"fixture_project_write":             true,
		"approval_gated":                    true,
		"project_read_attempted":            true,
		"project_write_attempted":           true,
		"project_write_allowed":             true,
		"execution_write_attempted":         false,
		"area_flow_artifact_written":        true,
		"area_flow_execution_state_written": true,
		"engine_call_attempted":             false,
		"commands_run":                      false,
		"secrets_resolved":                  false,
		"network_used":                      false,
		"lease_id":                          lease.ID,
		"worker_id":                         lease.WorkerID,
	}
	for key, value := range metadata {
		base[key] = value
	}
	metadataJSON, err := json.Marshal(base)
	if err != nil {
		return RunAttemptRecord{}, fmt.Errorf("marshal fixture project write attempt metadata: %w", err)
	}
	attempt, err := scanRunAttempt(tx.QueryRow(ctx, `
INSERT INTO run_attempts (
    project_id, workflow_version_id, workflow_item_id, run_id, run_task_id,
    attempt_kind, status, dry_run, finished_at, metadata
)
VALUES ($1, $2, $3, $4, $5, $6, $7, false, now(), $8::jsonb)
RETURNING id, project_id, COALESCE(workflow_version_id, 0), COALESCE(workflow_item_id, 0),
          run_id, COALESCE(run_task_id, 0), attempt_kind, status, dry_run,
          metadata, started_at, finished_at`,
		record.ID,
		nullableInt64(task.WorkflowVersionID),
		nullableInt64(task.WorkflowItemID),
		task.RunID,
		task.ID,
		attemptKind,
		status,
		string(metadataJSON),
	))
	if err != nil {
		return RunAttemptRecord{}, fmt.Errorf("insert fixture project write %s attempt: %w", attemptKind, err)
	}
	return attempt, nil
}

func writeAndInsertFixtureProjectWriteReport(ctx context.Context, tx pgx.Tx, record Record, version WorkflowVersion, run RunRecord, worker WorkerRecord, task RunTaskRecord, lease LeaseRecord, gate ExecutionApprovalGate, writeSetArtifact ArtifactRecord, preimageArtifact ArtifactRecord, copyAttempt RunAttemptRecord, verifyAttempt RunAttemptRecord, rollbackAttempt RunAttemptRecord, writeSet fixtureProjectWriteSet, preimage fixtureProjectFileImage, afterImage fixtureProjectFileImage, restoredImage fixtureProjectFileImage, options FixtureProjectWriteOptions) (ArtifactRecord, error) {
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
		"fixture_project_write":             true,
		"fixture_only":                      true,
		"approval_gated":                    true,
		"execution_gate_status":             gate.Status,
		"target_path":                       writeSet.TargetPath,
		"operation":                         writeSet.Operation,
		"write_set_artifact_id":             writeSetArtifact.ID,
		"preimage_artifact_id":              preimageArtifact.ID,
		"copy_attempt_id":                   copyAttempt.ID,
		"verify_attempt_id":                 verifyAttempt.ID,
		"rollback_attempt_id":               rollbackAttempt.ID,
		"expected_before_sha256":            writeSet.ExpectedBeforeSHA256,
		"expected_before_size":              writeSet.ExpectedBeforeSize,
		"preimage_sha256":                   preimage.SHA256,
		"preimage_size":                     preimage.SizeBytes,
		"after_sha256":                      afterImage.SHA256,
		"after_size":                        afterImage.SizeBytes,
		"rollback_restored_sha256":          restoredImage.SHA256,
		"rollback_restored_size":            restoredImage.SizeBytes,
		"project_read_attempted":            true,
		"project_read_allowed":              true,
		"project_write_attempted":           true,
		"project_write_allowed":             true,
		"execution_write_attempted":         false,
		"area_flow_artifact_written":        true,
		"area_flow_execution_state_written": true,
		"engine_call_attempted":             false,
		"commands_run":                      false,
		"secrets_resolved":                  false,
		"network_used":                      false,
		"write_set_passed":                  true,
		"verification_passed":               true,
		"rollback_attempted":                true,
		"rollback_verified":                 true,
		"unsupported_operations":            []string{"create", "delete", "move", "chmod", "binary_rewrite", "symlink_target", "project_root_escape", "glob_bulk_write"},
		"generated_at":                      time.Now().UTC().Format(time.RFC3339),
	}, "", "  ")
	if err != nil {
		return ArtifactRecord{}, fmt.Errorf("marshal fixture project write report: %w", err)
	}
	relativePath := filepath.Join("versions", version.DisplayLabel, "fixture-project-write", fmt.Sprintf("run-%d-task-%d-report.json", run.ID, task.ID))
	stored, err := writeProjectArtifact(record, relativePath, content, "application/json")
	if err != nil {
		return ArtifactRecord{}, err
	}
	metadata, err := json.Marshal(map[string]any{
		"phase":                 "v0.6n",
		"owned_by":              "areaflow",
		"fixture_project_write": true,
		"artifact_role":         "fixture_project_write_report",
		"target_path":           writeSet.TargetPath,
		"write_set_artifact_id": writeSetArtifact.ID,
		"preimage_artifact_id":  preimageArtifact.ID,
		"copy_attempt_id":       copyAttempt.ID,
		"verify_attempt_id":     verifyAttempt.ID,
		"rollback_attempt_id":   rollbackAttempt.ID,
		"rollback_verified":     true,
		"execution_gate_status": gate.Status,
		"actor":                 options.Actor,
		"reason":                options.Reason,
	})
	if err != nil {
		return ArtifactRecord{}, fmt.Errorf("marshal fixture project write report metadata: %w", err)
	}
	return insertRunArtifactRecord(ctx, tx, record.ID, version.ID, run.ID, task.WorkflowItemID, "fixture_project_write_report", relativePath, stored, string(metadata))
}

func updateFixtureProjectWriteRunAfterTask(ctx context.Context, tx pgx.Tx, run RunRecord, options FixtureProjectWriteOptions, artifactID int64, attemptID int64, targetPath string, writeSet fixtureProjectWriteSet, restoredImage fixtureProjectFileImage) (RunRecord, error) {
	summary := copyMap(run.Summary)
	summary["fixture_project_write"] = true
	summary["approval_gated"] = true
	summary["target_path"] = targetPath
	summary["last_artifact_id"] = artifactID
	summary["last_attempt_id"] = attemptID
	summary["project_read_attempted"] = true
	summary["project_read_allowed"] = true
	summary["project_write_attempted"] = true
	summary["project_write_allowed"] = true
	summary["execution_write_attempted"] = false
	summary["area_flow_artifact_written"] = true
	summary["area_flow_execution_state_written"] = true
	summary["engine_call_attempted"] = false
	summary["commands_run"] = false
	summary["secrets_resolved"] = false
	summary["network_used"] = false
	summary["expected_before_sha256"] = writeSet.ExpectedBeforeSHA256
	summary["after_sha256"] = writeSet.AfterSHA256
	summary["restored_sha256"] = restoredImage.SHA256
	summary["rollback_attempted"] = true
	summary["rollback_verified"] = true
	summary["verified_task_count"] = rollbackVerifiedRunTaskCount(ctx, tx, run.ID)
	remaining, err := remainingFixtureProjectWriteTaskCount(ctx, tx, run.ID)
	if err != nil {
		return RunRecord{}, err
	}
	status := "running"
	var finishedAtExpr string
	if remaining == 0 {
		status = "rollback_verified"
		finishedAtExpr = "now()"
	} else {
		finishedAtExpr = "finished_at"
	}
	summary["remaining_task_count"] = remaining
	summaryJSON, err := json.Marshal(summary)
	if err != nil {
		return RunRecord{}, fmt.Errorf("marshal fixture project write run summary: %w", err)
	}
	metadata := copyMap(run.Metadata)
	metadata["last_fixture_project_write_actor"] = options.Actor
	metadata["last_fixture_project_write_reason"] = options.Reason
	metadata["last_fixture_project_write_at"] = time.Now().UTC().Format(time.RFC3339)
	metadata["last_artifact_id"] = artifactID
	metadata["last_attempt_id"] = attemptID
	metadata["fixture_project_write"] = true
	metadata["approval_gated"] = true
	metadataJSON, err := json.Marshal(metadata)
	if err != nil {
		return RunRecord{}, fmt.Errorf("marshal fixture project write run metadata: %w", err)
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
		return RunRecord{}, fmt.Errorf("update fixture project write run: %w", err)
	}
	return updated, nil
}

func rollbackVerifiedRunTaskCount(ctx context.Context, tx pgx.Tx, runID int64) int64 {
	var count int64
	_ = tx.QueryRow(ctx, `SELECT count(*) FROM run_tasks WHERE run_id = $1 AND status = 'rollback_verified'`, runID).Scan(&count)
	return count
}

func remainingFixtureProjectWriteTaskCount(ctx context.Context, tx pgx.Tx, runID int64) (int64, error) {
	var count int64
	if err := tx.QueryRow(ctx, `
SELECT count(*)
FROM run_tasks
WHERE run_id = $1
  AND task_kind = 'fixture_project_write_task'
  AND status IN ('queued', 'pending', 'needs_recovery', 'leased')`,
		runID,
	).Scan(&count); err != nil {
		return 0, fmt.Errorf("count remaining fixture project write tasks: %w", err)
	}
	return count, nil
}

func insertFixtureProjectWriteQueueEvent(ctx context.Context, tx pgx.Tx, result FixtureProjectWriteQueueResult, options FixtureProjectWriteQueueOptions) (int64, error) {
	metadata, err := json.Marshal(fixtureProjectWriteQueueMetadata(result, options))
	if err != nil {
		return 0, fmt.Errorf("marshal fixture project write queue event metadata: %w", err)
	}
	var eventID int64
	if err := tx.QueryRow(ctx, `
INSERT INTO events (project_id, run_id, workflow_version_id, event_type, severity, message, metadata)
VALUES ($1, $2, $3, 'run.fixture_project_write_queue.created', 'info', 'Fixture project write run queued', $4::jsonb)
RETURNING id`,
		result.Project.ID,
		result.Run.ID,
		result.Version.ID,
		string(metadata),
	).Scan(&eventID); err != nil {
		return 0, fmt.Errorf("insert fixture project write queue event: %w", err)
	}
	return eventID, nil
}

func insertFixtureProjectWriteQueueAuditEvent(ctx context.Context, tx pgx.Tx, result FixtureProjectWriteQueueResult, options FixtureProjectWriteQueueOptions) (int64, error) {
	metadata, err := json.Marshal(fixtureProjectWriteQueueMetadata(result, options))
	if err != nil {
		return 0, fmt.Errorf("marshal fixture project write queue audit metadata: %w", err)
	}
	var auditEventID int64
	if err := tx.QueryRow(ctx, `
INSERT INTO audit_events (project_id, action, capability, resource_type, resource, decision, reason, metadata)
VALUES ($1, $2, 'manage_runs', 'run', $3, 'allowed', $4, $5::jsonb)
RETURNING id`,
		result.Project.ID,
		fixtureProjectWriteQueueCommandType,
		fmt.Sprintf("%d", result.Run.ID),
		options.Reason,
		string(metadata),
	).Scan(&auditEventID); err != nil {
		return 0, fmt.Errorf("insert fixture project write queue audit event: %w", err)
	}
	return auditEventID, nil
}

func fixtureProjectWriteQueueMetadata(result FixtureProjectWriteQueueResult, options FixtureProjectWriteQueueOptions) map[string]any {
	return map[string]any{
		"project_key":                       result.Project.Key,
		"workflow_version_id":               result.Version.ID,
		"display_label":                     result.Version.DisplayLabel,
		"run_id":                            result.Run.ID,
		"run_task_id":                       result.Task.ID,
		"write_set_artifact_id":             result.WriteSetArtifact.ID,
		"target_path":                       result.TargetPath,
		"expected_before_sha256":            result.ExpectedBeforeSHA256,
		"expected_before_size":              result.ExpectedBeforeSize,
		"after_sha256":                      result.AfterSHA256,
		"after_size":                        result.AfterSize,
		"actor":                             options.Actor,
		"idempotency_key":                   options.IdempotencyKey,
		"fixture_project_write":             true,
		"approval_gated":                    true,
		"project_read_attempted":            false,
		"project_write_attempted":           false,
		"execution_write_attempted":         false,
		"area_flow_artifact_written":        true,
		"area_flow_execution_state_written": true,
		"engine_call_attempted":             false,
		"commands_run":                      false,
		"secrets_resolved":                  false,
		"network_used":                      false,
	}
}

func insertFixtureProjectWriteEvent(ctx context.Context, tx pgx.Tx, result FixtureProjectWriteResult, options FixtureProjectWriteOptions) (int64, error) {
	severity := "info"
	if result.Decision == "denied" {
		severity = "warning"
	}
	metadata, err := json.Marshal(fixtureProjectWriteMetadata(result, options))
	if err != nil {
		return 0, fmt.Errorf("marshal fixture project write event metadata: %w", err)
	}
	var eventID int64
	if err := tx.QueryRow(ctx, `
INSERT INTO events (project_id, run_id, workflow_version_id, event_type, severity, message, metadata)
VALUES ($1, $2, $3, $4, $5, $6, $7::jsonb)
RETURNING id`,
		result.Project.ID,
		result.Run.ID,
		nullableInt64(result.Version.ID),
		"worker.fixture_project_write."+result.Decision,
		severity,
		result.Message,
		string(metadata),
	).Scan(&eventID); err != nil {
		return 0, fmt.Errorf("insert fixture project write event: %w", err)
	}
	return eventID, nil
}

func insertFixtureProjectWriteAuditEvent(ctx context.Context, tx pgx.Tx, result FixtureProjectWriteResult, options FixtureProjectWriteOptions) (int64, error) {
	metadata, err := json.Marshal(fixtureProjectWriteMetadata(result, options))
	if err != nil {
		return 0, fmt.Errorf("marshal fixture project write audit metadata: %w", err)
	}
	var auditEventID int64
	if err := tx.QueryRow(ctx, `
INSERT INTO audit_events (project_id, actor_id, action, capability, resource_type, resource, decision, reason, metadata)
VALUES ($1, $2, $3, 'write_code', 'path', $4, $5, $6, $7::jsonb)
RETURNING id`,
		result.Project.ID,
		nullableInt64(result.Worker.ActorID),
		fixtureProjectWriteApplyCommandType,
		result.TargetPath,
		result.Decision,
		options.Reason,
		string(metadata),
	).Scan(&auditEventID); err != nil {
		return 0, fmt.Errorf("insert fixture project write audit event: %w", err)
	}
	return auditEventID, nil
}

func finishDeniedFixtureProjectWrite(ctx context.Context, tx pgx.Tx, result FixtureProjectWriteResult, options FixtureProjectWriteOptions) error {
	eventID, err := insertFixtureProjectWriteEvent(ctx, tx, result, options)
	if err != nil {
		return err
	}
	result.EventID = eventID
	auditEventID, err := insertFixtureProjectWriteAuditEvent(ctx, tx, result, options)
	if err != nil {
		return err
	}
	result.AuditEventID = auditEventID
	return completeCommandRequestResponse(ctx, tx, result.Project.ID, fixtureProjectWriteApplyCommandType, options.IdempotencyKey, fixtureProjectWriteCommandResponse(result))
}

func fixtureProjectWriteMetadata(result FixtureProjectWriteResult, options FixtureProjectWriteOptions) map[string]any {
	return map[string]any{
		"project_key":                       result.Project.Key,
		"workflow_version_id":               result.Version.ID,
		"display_label":                     result.Version.DisplayLabel,
		"run_id":                            result.Run.ID,
		"run_task_id":                       result.Task.ID,
		"lease_id":                          result.Lease.ID,
		"copy_attempt_id":                   result.CopyAttempt.ID,
		"verify_attempt_id":                 result.VerifyAttempt.ID,
		"rollback_attempt_id":               result.RollbackAttempt.ID,
		"write_set_artifact_id":             result.WriteSetArtifact.ID,
		"preimage_artifact_id":              result.PreimageArtifact.ID,
		"artifact_id":                       result.Artifact.ID,
		"worker_id":                         result.Worker.ID,
		"worker_key":                        result.Worker.WorkerKey,
		"target_path":                       result.TargetPath,
		"expected_before_sha256":            result.ExpectedBeforeSHA256,
		"expected_before_size":              result.ExpectedBeforeSize,
		"after_sha256":                      result.AfterSHA256,
		"after_size":                        result.AfterSize,
		"restored_sha256":                   result.RestoredSHA256,
		"restored_size":                     result.RestoredSize,
		"status":                            result.Status,
		"decision":                          result.Decision,
		"blockers":                          result.Blockers,
		"actor":                             options.Actor,
		"idempotency_key":                   options.IdempotencyKey,
		"fixture_project_write":             true,
		"approval_gated":                    true,
		"execution_gate_status":             result.Gate.Status,
		"project_read_attempted":            result.ProjectReadAttempted,
		"project_read_allowed":              result.ProjectReadAllowed,
		"project_write_attempted":           result.ProjectWriteAttempted,
		"project_write_allowed":             result.ProjectWriteAllowed,
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
		"write_set_passed":                  result.WriteSetPassed,
		"verification_passed":               result.VerificationPassed,
		"rollback_attempted":                result.RollbackAttempted,
		"rollback_verified":                 result.RollbackVerified,
	}
}

func deniedFixtureProjectWriteResult(record Record, version WorkflowVersion, run RunRecord, worker WorkerRecord, gate ExecutionApprovalGate, options FixtureProjectWriteOptions, message string, blockers []string) FixtureProjectWriteResult {
	return FixtureProjectWriteResult{
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
		ProjectWriteAllowed:           false,
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
		WriteSetPassed:                false,
		VerificationPassed:            false,
		RollbackAttempted:             false,
		RollbackVerified:              false,
	}
}

func fixtureProjectWriteQueueCommandResponse(result FixtureProjectWriteQueueResult) map[string]any {
	return map[string]any{
		"project_id":                        result.Project.ID,
		"project_key":                       result.Project.Key,
		"workflow_version_id":               result.Version.ID,
		"display_label":                     result.Version.DisplayLabel,
		"run_id":                            result.Run.ID,
		"run_task_id":                       result.Task.ID,
		"write_set_artifact_id":             result.WriteSetArtifact.ID,
		"target_path":                       result.TargetPath,
		"expected_before_sha256":            result.ExpectedBeforeSHA256,
		"expected_before_size":              result.ExpectedBeforeSize,
		"after_sha256":                      result.AfterSHA256,
		"after_size":                        result.AfterSize,
		"event_id":                          result.EventID,
		"audit_event_id":                    result.AuditEventID,
		"fixture_project_write":             true,
		"approval_gated":                    true,
		"project_read_attempted":            false,
		"project_write_attempted":           false,
		"execution_write_attempted":         false,
		"area_flow_artifact_written":        result.AreaFlowArtifactWritten,
		"area_flow_execution_state_written": result.AreaFlowExecutionStateWritten,
		"engine_call_attempted":             false,
		"commands_run":                      false,
		"secrets_resolved":                  false,
		"network_used":                      false,
	}
}

func fixtureProjectWriteCommandResponse(result FixtureProjectWriteResult) map[string]any {
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
		"copy_attempt_id":                   result.CopyAttempt.ID,
		"verify_attempt_id":                 result.VerifyAttempt.ID,
		"rollback_attempt_id":               result.RollbackAttempt.ID,
		"write_set_artifact_id":             result.WriteSetArtifact.ID,
		"preimage_artifact_id":              result.PreimageArtifact.ID,
		"artifact_id":                       result.Artifact.ID,
		"artifact_type":                     result.Artifact.ArtifactType,
		"target_path":                       result.TargetPath,
		"expected_before_sha256":            result.ExpectedBeforeSHA256,
		"expected_before_size":              result.ExpectedBeforeSize,
		"after_sha256":                      result.AfterSHA256,
		"after_size":                        result.AfterSize,
		"restored_sha256":                   result.RestoredSHA256,
		"restored_size":                     result.RestoredSize,
		"event_id":                          result.EventID,
		"audit_event_id":                    result.AuditEventID,
		"status":                            result.Status,
		"decision":                          result.Decision,
		"message":                           result.Message,
		"blockers":                          result.Blockers,
		"fixture_project_write":             true,
		"approval_gated":                    true,
		"execution_gate_status":             result.Gate.Status,
		"project_read_attempted":            result.ProjectReadAttempted,
		"project_read_allowed":              result.ProjectReadAllowed,
		"project_write_attempted":           result.ProjectWriteAttempted,
		"project_write_allowed":             result.ProjectWriteAllowed,
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
		"write_set_passed":                  result.WriteSetPassed,
		"verification_passed":               result.VerificationPassed,
		"rollback_attempted":                result.RollbackAttempted,
		"rollback_verified":                 result.RollbackVerified,
	}
}

func loadFixtureProjectWriteQueueByCommandResponse(ctx context.Context, tx pgx.Tx, record Record, version WorkflowVersion, idempotencyKey string) (FixtureProjectWriteQueueResult, error) {
	response, err := loadCommandResponse(ctx, tx, record.ID, fixtureProjectWriteQueueCommandType, idempotencyKey)
	if err != nil {
		return FixtureProjectWriteQueueResult{}, err
	}
	runID := metadataInt64(response, "run_id")
	taskID := metadataInt64(response, "run_task_id")
	run, err := loadRunForUpdate(ctx, tx, runID)
	if err != nil {
		return FixtureProjectWriteQueueResult{}, err
	}
	task, err := loadRunTaskByID(ctx, tx, record.ID, taskID)
	if err != nil {
		return FixtureProjectWriteQueueResult{}, err
	}
	var writeSetArtifact ArtifactRecord
	if artifactID := metadataInt64(response, "write_set_artifact_id"); artifactID != 0 {
		writeSetArtifact, err = loadArtifactByIDTx(ctx, tx, record.ID, artifactID)
		if err != nil {
			return FixtureProjectWriteQueueResult{}, err
		}
	}
	return FixtureProjectWriteQueueResult{
		Project:                       record,
		Version:                       version,
		Run:                           run,
		Task:                          task,
		WriteSetArtifact:              writeSetArtifact,
		TargetPath:                    metadataString(response, "target_path"),
		ExpectedBeforeSHA256:          metadataString(response, "expected_before_sha256"),
		ExpectedBeforeSize:            metadataInt64(response, "expected_before_size"),
		AfterSHA256:                   metadataString(response, "after_sha256"),
		AfterSize:                     metadataInt64(response, "after_size"),
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

func loadFixtureProjectWriteByCommandResponse(ctx context.Context, tx pgx.Tx, record Record, version WorkflowVersion, gate ExecutionApprovalGate, idempotencyKey string) (FixtureProjectWriteResult, error) {
	response, err := loadCommandResponse(ctx, tx, record.ID, fixtureProjectWriteApplyCommandType, idempotencyKey)
	if err != nil {
		return FixtureProjectWriteResult{}, err
	}
	runID := metadataInt64(response, "run_id")
	workerID := metadataInt64(response, "worker_id")
	run, err := loadRunForUpdate(ctx, tx, runID)
	if err != nil {
		return FixtureProjectWriteResult{}, err
	}
	worker := WorkerRecord{}
	if workerID != 0 {
		worker, err = loadWorkerByID(ctx, tx, record.ID, workerID)
		if err != nil {
			return FixtureProjectWriteResult{}, err
		}
	}
	result := FixtureProjectWriteResult{
		Project:                       record,
		Version:                       version,
		Run:                           run,
		Worker:                        worker,
		Gate:                          gate,
		TargetPath:                    metadataString(response, "target_path"),
		ExpectedBeforeSHA256:          metadataString(response, "expected_before_sha256"),
		ExpectedBeforeSize:            metadataInt64(response, "expected_before_size"),
		AfterSHA256:                   metadataString(response, "after_sha256"),
		AfterSize:                     metadataInt64(response, "after_size"),
		RestoredSHA256:                metadataString(response, "restored_sha256"),
		RestoredSize:                  metadataInt64(response, "restored_size"),
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
		ProjectWriteAllowed:           metadataBool(response, "project_write_allowed"),
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
		WriteSetPassed:                metadataBool(response, "write_set_passed"),
		VerificationPassed:            metadataBool(response, "verification_passed"),
		RollbackAttempted:             metadataBool(response, "rollback_attempted"),
		RollbackVerified:              metadataBool(response, "rollback_verified"),
	}
	if taskID := metadataInt64(response, "run_task_id"); taskID != 0 {
		result.Task, err = loadRunTaskByID(ctx, tx, record.ID, taskID)
		if err != nil {
			return FixtureProjectWriteResult{}, err
		}
	}
	if leaseID := metadataInt64(response, "lease_id"); leaseID != 0 {
		result.Lease, err = loadLeaseByID(ctx, tx, record.ID, leaseID)
		if err != nil {
			return FixtureProjectWriteResult{}, err
		}
	}
	if attemptID := metadataInt64(response, "copy_attempt_id"); attemptID != 0 {
		result.CopyAttempt, err = loadRunAttemptByID(ctx, tx, record.ID, attemptID)
		if err != nil {
			return FixtureProjectWriteResult{}, err
		}
	}
	if attemptID := metadataInt64(response, "verify_attempt_id"); attemptID != 0 {
		result.VerifyAttempt, err = loadRunAttemptByID(ctx, tx, record.ID, attemptID)
		if err != nil {
			return FixtureProjectWriteResult{}, err
		}
	}
	if attemptID := metadataInt64(response, "rollback_attempt_id"); attemptID != 0 {
		result.RollbackAttempt, err = loadRunAttemptByID(ctx, tx, record.ID, attemptID)
		if err != nil {
			return FixtureProjectWriteResult{}, err
		}
	}
	if artifactID := metadataInt64(response, "write_set_artifact_id"); artifactID != 0 {
		result.WriteSetArtifact, err = loadArtifactByIDTx(ctx, tx, record.ID, artifactID)
		if err != nil {
			return FixtureProjectWriteResult{}, err
		}
	}
	if artifactID := metadataInt64(response, "preimage_artifact_id"); artifactID != 0 {
		result.PreimageArtifact, err = loadArtifactByIDTx(ctx, tx, record.ID, artifactID)
		if err != nil {
			return FixtureProjectWriteResult{}, err
		}
	}
	if artifactID := metadataInt64(response, "artifact_id"); artifactID != 0 {
		result.Artifact, err = loadArtifactByIDTx(ctx, tx, record.ID, artifactID)
		if err != nil {
			return FixtureProjectWriteResult{}, err
		}
	}
	return result, nil
}
