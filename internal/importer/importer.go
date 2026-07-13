package importer

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync/atomic"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/areasong/areaflow/internal/adapter/areamatrix"
	"github.com/areasong/areaflow/internal/project"
)

type Result struct {
	ProjectKey     string
	Versions       int
	Residuals      int
	Artifacts      int
	ActiveTasks    int
	V1Done         int
	V1Total        int
	StatusSnapshot string
	RunID          int64
	IdempotencyKey string
	Created        bool
}

type Options struct {
	IdempotencyKey string
	Actor          string
	Reason         string
}

var importDefaultKeySequence atomic.Int64

func ImportProject(ctx context.Context, pool *pgxpool.Pool, record project.Record, options Options) (Result, error) {
	options = normalizeOptions(options)
	if record.Adapter != "areamatrix" {
		return Result{}, fmt.Errorf("unsupported adapter %q", record.Adapter)
	}
	if record.RootPath == "" {
		return Result{}, fmt.Errorf("project %s has no local root path", record.Key)
	}

	snapshot, err := areamatrix.Load(record.RootPath)
	if err != nil {
		return Result{}, err
	}
	statusJSON, err := json.Marshal(snapshot.StatusSummary)
	if err != nil {
		return Result{}, fmt.Errorf("marshal status summary: %w", err)
	}
	requestHash, err := importRequestHash(record, snapshot.StatusSourceHash, statusJSON, options)
	if err != nil {
		return Result{}, err
	}
	if options.IdempotencyKey == "" {
		options.IdempotencyKey = importIdempotencyKey(record, snapshot.StatusSourceHash)
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		return Result{}, fmt.Errorf("begin import: %w", err)
	}
	defer tx.Rollback(ctx)

	createdCommand, err := reserveCommandRequest(ctx, tx, record.ID, "project.import", options.IdempotencyKey, requestHash)
	if err != nil {
		return Result{}, err
	}
	if !createdCommand {
		result, err := loadImportResultByCommandResponse(ctx, tx, record, options.IdempotencyKey)
		if err != nil {
			return Result{}, err
		}
		if err := tx.Commit(ctx); err != nil {
			return Result{}, fmt.Errorf("commit idempotent import: %w", err)
		}
		result.IdempotencyKey = options.IdempotencyKey
		result.Created = false
		return result, nil
	}

	runID, err := startRun(ctx, tx, record.ID)
	if err != nil {
		return Result{}, err
	}
	if err := clearCurrentImportIndex(ctx, tx, record.ID); err != nil {
		return Result{}, err
	}

	versionIDs := map[string]int64{}
	for _, version := range snapshot.Versions {
		versionID, err := upsertVersion(ctx, tx, record.ID, version)
		if err != nil {
			return Result{}, err
		}
		versionIDs[version.Label] = versionID
	}

	for _, residual := range snapshot.Residuals {
		if err := upsertResidual(ctx, tx, record.ID, versionIDs[residual.VersionLabel], residual); err != nil {
			return Result{}, err
		}
	}
	for _, artifact := range snapshot.Artifacts {
		if err := insertArtifact(ctx, tx, record.ID, versionIDs[artifact.VersionLabel], runID, artifact); err != nil {
			return Result{}, err
		}
	}

	if _, err := tx.Exec(ctx, `
INSERT INTO project_status_snapshots (project_id, snapshot_kind, summary, source_hash, created_by_actor_id)
VALUES ($1, 'import', $2::jsonb, $3, NULL)`,
		record.ID,
		string(statusJSON),
		snapshot.StatusSourceHash,
	); err != nil {
		return Result{}, fmt.Errorf("insert status snapshot: %w", err)
	}

	if _, err := tx.Exec(ctx, `
INSERT INTO events (project_id, run_id, event_type, severity, message, metadata)
VALUES ($1, $2, 'project.import.completed', 'info', 'AreaMatrix metadata import completed', $3::jsonb)`,
		record.ID,
		runID,
		string(statusJSON),
	); err != nil {
		return Result{}, fmt.Errorf("insert import event: %w", err)
	}

	result := Result{
		ProjectKey:     record.Key,
		Versions:       len(snapshot.Versions),
		Residuals:      len(snapshot.Residuals),
		Artifacts:      len(snapshot.Artifacts),
		ActiveTasks:    snapshot.TaskSummary.ActiveCount,
		V1Done:         snapshot.TaskSummary.V1ExecutionDone,
		V1Total:        snapshot.TaskSummary.V1ExecutionTotal,
		StatusSnapshot: snapshot.StatusSourceHash,
		RunID:          runID,
		IdempotencyKey: options.IdempotencyKey,
		Created:        true,
	}

	runSummary, _ := json.Marshal(map[string]any{
		"versions":  len(snapshot.Versions),
		"residuals": len(snapshot.Residuals),
		"artifacts": len(snapshot.Artifacts),
	})
	if _, err := tx.Exec(ctx, `
UPDATE runs
SET status = 'completed', finished_at = now(), summary = $2::jsonb
WHERE id = $1`,
		runID,
		string(runSummary),
	); err != nil {
		return Result{}, fmt.Errorf("complete import run: %w", err)
	}

	if _, err := tx.Exec(ctx, `
INSERT INTO audit_events (project_id, action, resource_type, resource, decision, reason, metadata)
VALUES ($1, 'project.import', 'project', $2, 'allowed', 'read-only AreaMatrix metadata import', $3::jsonb)`,
		record.ID,
		record.Key,
		string(runSummary),
	); err != nil {
		return Result{}, fmt.Errorf("insert import audit event: %w", err)
	}

	if err := completeCommandRequest(ctx, tx, record.ID, "project.import", options.IdempotencyKey, result); err != nil {
		return Result{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Result{}, fmt.Errorf("commit import: %w", err)
	}

	return result, nil
}

func normalizeOptions(options Options) Options {
	options.IdempotencyKey = strings.TrimSpace(options.IdempotencyKey)
	options.Actor = strings.TrimSpace(options.Actor)
	options.Reason = strings.TrimSpace(options.Reason)
	if options.Actor == "" {
		options.Actor = "local-user"
	}
	if options.Reason == "" {
		options.Reason = "read-only AreaMatrix metadata import"
	}
	return options
}

func importRequestHash(record project.Record, sourceHash string, statusJSON []byte, options Options) (string, error) {
	payload, err := json.Marshal(map[string]any{
		"command_type": "project.import",
		"project_key":  record.Key,
		"project_id":   record.ID,
		"adapter":      record.Adapter,
		"root_path":    record.RootPath,
		"source_hash":  sourceHash,
		"summary":      json.RawMessage(statusJSON),
		"actor":        options.Actor,
		"reason":       options.Reason,
	})
	if err != nil {
		return "", fmt.Errorf("marshal import command request hash payload: %w", err)
	}
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:]), nil
}

func importIdempotencyKey(record project.Record, sourceHash string) string {
	sequence := importDefaultKeySequence.Add(1)
	return fmt.Sprintf("project.import:%s:%s:%d:%d", record.Key, sourceHash, time.Now().UTC().UnixNano(), sequence)
}

func reserveCommandRequest(ctx context.Context, tx pgx.Tx, projectID int64, commandType string, idempotencyKey string, requestHash string) (bool, error) {
	var id int64
	err := tx.QueryRow(ctx, `
INSERT INTO command_requests (project_id, command_type, idempotency_key, request_hash)
VALUES ($1, $2, $3, $4)
ON CONFLICT DO NOTHING
RETURNING id`,
		projectID,
		commandType,
		idempotencyKey,
		requestHash,
	).Scan(&id)
	if err == nil {
		return true, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return false, fmt.Errorf("reserve command request: %w", err)
	}

	var existingHash string
	err = tx.QueryRow(ctx, `
SELECT request_hash
FROM command_requests
WHERE project_id = $1 AND command_type = $2 AND idempotency_key = $3`,
		projectID,
		commandType,
		idempotencyKey,
	).Scan(&existingHash)
	if err != nil {
		return false, fmt.Errorf("load existing command request: %w", err)
	}
	if existingHash != requestHash {
		return false, fmt.Errorf("%w: %s", project.ErrIdempotencyConflict, idempotencyKey)
	}
	return false, nil
}

func completeCommandRequest(ctx context.Context, tx pgx.Tx, projectID int64, commandType string, idempotencyKey string, result Result) error {
	response, err := json.Marshal(map[string]any{
		"project_key":     result.ProjectKey,
		"run_id":          result.RunID,
		"versions":        result.Versions,
		"residuals":       result.Residuals,
		"artifacts":       result.Artifacts,
		"active_tasks":    result.ActiveTasks,
		"v1_done":         result.V1Done,
		"v1_total":        result.V1Total,
		"status_snapshot": result.StatusSnapshot,
	})
	if err != nil {
		return fmt.Errorf("marshal import command response: %w", err)
	}
	if _, err := tx.Exec(ctx, `
UPDATE command_requests
SET response = $4::jsonb, completed_at = now()
WHERE project_id = $1 AND command_type = $2 AND idempotency_key = $3`,
		projectID,
		commandType,
		idempotencyKey,
		string(response),
	); err != nil {
		return fmt.Errorf("complete command request: %w", err)
	}
	return nil
}

func loadImportResultByCommandResponse(ctx context.Context, tx pgx.Tx, record project.Record, idempotencyKey string) (Result, error) {
	var responseRaw []byte
	var completedAt sql.NullTime
	if err := tx.QueryRow(ctx, `
SELECT response, completed_at
FROM command_requests
WHERE project_id = $1 AND command_type = 'project.import' AND idempotency_key = $2`,
		record.ID,
		idempotencyKey,
	).Scan(&responseRaw, &completedAt); err != nil {
		return Result{}, fmt.Errorf("load import command response: %w", err)
	}
	if !completedAt.Valid {
		return Result{}, fmt.Errorf("import command request is not complete: %s", idempotencyKey)
	}
	response := map[string]any{}
	if err := json.Unmarshal(responseRaw, &response); err != nil {
		return Result{}, fmt.Errorf("parse import command response: %w", err)
	}
	projectKey := stringValue(response["project_key"])
	if projectKey == "" {
		projectKey = record.Key
	}
	return Result{
		ProjectKey:     projectKey,
		Versions:       int(number(response["versions"])),
		Residuals:      int(number(response["residuals"])),
		Artifacts:      int(number(response["artifacts"])),
		ActiveTasks:    int(number(response["active_tasks"])),
		V1Done:         int(number(response["v1_done"])),
		V1Total:        int(number(response["v1_total"])),
		StatusSnapshot: stringValue(response["status_snapshot"]),
		RunID:          number(response["run_id"]),
	}, nil
}

func number(value any) int64 {
	switch typed := value.(type) {
	case int:
		return int64(typed)
	case int64:
		return typed
	case float64:
		return int64(typed)
	default:
		return 0
	}
}

func stringValue(value any) string {
	if typed, ok := value.(string); ok {
		return typed
	}
	return ""
}

func startRun(ctx context.Context, tx pgx.Tx, projectID int64) (int64, error) {
	var runID int64
	err := tx.QueryRow(ctx, `
INSERT INTO runs (project_id, run_type, status)
VALUES ($1, 'import', 'running')
RETURNING id`,
		projectID,
	).Scan(&runID)
	if err != nil {
		return 0, fmt.Errorf("start import run: %w", err)
	}
	return runID, nil
}

func clearCurrentImportIndex(ctx context.Context, tx pgx.Tx, projectID int64) error {
	statements := []string{
		`DELETE FROM artifacts WHERE project_id = $1 AND storage_backend = 'external_project'`,
		`DELETE FROM residuals WHERE project_id = $1 AND imported_at IS NOT NULL`,
		`DELETE FROM workflow_versions WHERE project_id = $1 AND import_mode = 'metadata_only'`,
	}
	for _, statement := range statements {
		if _, err := tx.Exec(ctx, statement, projectID); err != nil {
			return fmt.Errorf("clear current import index: %w", err)
		}
	}
	return nil
}

func upsertVersion(ctx context.Context, tx pgx.Tx, projectID int64, version areamatrix.Version) (int64, error) {
	summary, err := json.Marshal(map[string]any{
		"status":          version.StatusSummary,
		"artifact_counts": version.ArtifactCounts,
	})
	if err != nil {
		return 0, fmt.Errorf("marshal version summary: %w", err)
	}

	var id int64
	err = tx.QueryRow(ctx, `
INSERT INTO workflow_versions (
    project_id, display_label, version_kind, lifecycle_status, source_path,
    source_hash, import_mode, immutable, status_summary, imported_at
)
VALUES ($1, $2, 'workflow_version', $3, $4, $5, 'metadata_only', $6, $7::jsonb, now())
RETURNING id`,
		projectID,
		version.Label,
		version.Lifecycle,
		version.SourcePath,
		version.SourceHash,
		version.Immutable,
		string(summary),
	).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("insert workflow version %s: %w", version.Label, err)
	}
	return id, nil
}

func upsertResidual(ctx context.Context, tx pgx.Tx, projectID int64, versionID int64, residual areamatrix.Residual) error {
	metadata, err := json.Marshal(residual.Metadata)
	if err != nil {
		return fmt.Errorf("marshal residual metadata %s: %w", residual.Key, err)
	}
	var nullableVersion any
	if versionID != 0 {
		nullableVersion = versionID
	}
	if _, err := tx.Exec(ctx, `
INSERT INTO residuals (
    project_id, workflow_version_id, residual_key, status, type, title, source_path,
    current_impact, executable_task, promotion_required, close_condition, metadata,
    immutable, imported_at
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12::jsonb, $13, now())`,
		projectID,
		nullableVersion,
		residual.Key,
		residual.Status,
		residual.Type,
		residual.Title,
		residual.SourcePath,
		residual.CurrentImpact,
		residual.ExecutableTask,
		residual.PromotionRequired,
		residual.CloseCondition,
		string(metadata),
		residual.Immutable,
	); err != nil {
		return fmt.Errorf("insert residual %s: %w", residual.Key, err)
	}
	return nil
}

func insertArtifact(ctx context.Context, tx pgx.Tx, projectID int64, versionID int64, runID int64, artifact areamatrix.Artifact) error {
	metadata, err := json.Marshal(artifact.Metadata)
	if err != nil {
		return fmt.Errorf("marshal artifact metadata %s: %w", artifact.SourcePath, err)
	}
	var nullableVersion any
	if versionID != 0 {
		nullableVersion = versionID
	}
	if _, err := tx.Exec(ctx, `
INSERT INTO artifacts (
    project_id, workflow_version_id, run_id, artifact_type, storage_backend, uri,
    source_path, sha256, size_bytes, content_type, metadata
)
VALUES ($1, $2, $3, $4, 'external_project', $5, $5, $6, $7, $8, $9::jsonb)`,
		projectID,
		nullableVersion,
		runID,
		artifact.Type,
		artifact.SourcePath,
		artifact.SHA256,
		artifact.SizeBytes,
		artifact.ContentType,
		string(metadata),
	); err != nil {
		return fmt.Errorf("insert artifact %s: %w", artifact.SourcePath, err)
	}
	return nil
}
