package project

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/areasong/areaflow/internal/artifact"
)

type ArtifactLocation struct {
	ID             int64
	ProjectID      int64
	ArtifactID     int64
	Role           string
	StorageBackend string
	URI            string
	SHA256         string
	SizeBytes      int64
	ContentType    string
	VerifiedAt     *time.Time
	Metadata       map[string]any
	CreatedAt      time.Time
}

type ArtifactMigrationItem struct {
	Artifact ArtifactRecord
	Source   ArtifactLocation
	Target   ArtifactLocation
	Status   string
}

type ArtifactMigrationInventory struct {
	Project       Record
	SourceBackend string
	TargetBackend string
	Items         []ArtifactMigrationItem
	Pending       int
	Verified      int
	Activated     int
	Observing     int
	Stable        int
}

type CopyArtifactOptions struct {
	TargetBackend string
	TargetRoot    string
	Actor         string
	Reason        string
}

type ActivateArtifactOptions struct {
	TargetLocationID int64
	ObservationUntil time.Time
	Actor            string
	Reason           string
}

type CompleteArtifactObservationOptions struct {
	Actor  string
	Reason string
}

func (s Store) ArtifactMigrationInventory(ctx context.Context, record Record, sourceBackend, targetBackend string) (ArtifactMigrationInventory, error) {
	if record.ID <= 0 {
		return ArtifactMigrationInventory{}, fmt.Errorf("project id is required")
	}
	sourceBackend = normalizedArtifactBackend(sourceBackend)
	targetBackend = normalizedArtifactBackend(targetBackend)
	rows, err := s.pool.Query(ctx, `
SELECT a.id, a.project_id, COALESCE(a.workflow_version_id, 0), COALESCE(a.run_id, 0), COALESCE(a.workflow_item_id, 0),
       a.artifact_type, a.storage_backend, a.uri, COALESCE(a.source_path, ''), COALESCE(a.sha256, ''),
       COALESCE(a.size_bytes, 0), COALESCE(a.content_type, ''), a.metadata, a.created_at,
       COALESCE(source_location.id, 0), COALESCE(source_location.location_role, ''), COALESCE(source_location.storage_backend, ''), COALESCE(source_location.uri, ''),
       COALESCE(source_location.sha256, ''), COALESCE(source_location.size_bytes, 0), COALESCE(source_location.content_type, ''), source_location.verified_at,
       COALESCE(source_location.metadata, '{}'::jsonb), source_location.created_at,
       COALESCE(target_location.id, 0), COALESCE(target_location.location_role, ''), COALESCE(target_location.storage_backend, ''), COALESCE(target_location.uri, ''),
       COALESCE(target_location.sha256, ''), COALESCE(target_location.size_bytes, 0), COALESCE(target_location.content_type, ''), target_location.verified_at,
       COALESCE(target_location.metadata, '{}'::jsonb), target_location.created_at
FROM artifacts a
LEFT JOIN LATERAL (
    SELECT * FROM artifact_locations source
    WHERE source.artifact_id = a.id AND source.storage_backend = $2
    ORDER BY (source.location_role = 'migration_source') DESC, source.verified_at DESC NULLS LAST, source.id DESC
    LIMIT 1
) source_location ON true
LEFT JOIN LATERAL (
    SELECT * FROM artifact_locations target
    WHERE target.artifact_id = a.id AND target.storage_backend = $3
    ORDER BY (target.location_role = 'primary') DESC, target.verified_at DESC NULLS LAST, target.id DESC
    LIMIT 1
) target_location ON true
WHERE a.project_id = $1 AND (a.storage_backend = $2 OR source_location.id IS NOT NULL)
ORDER BY a.id`, record.ID, sourceBackend, targetBackend)
	if err != nil {
		return ArtifactMigrationInventory{}, fmt.Errorf("list artifact migration inventory: %w", err)
	}
	defer rows.Close()

	inventory := ArtifactMigrationInventory{Project: record, SourceBackend: sourceBackend, TargetBackend: targetBackend}
	for rows.Next() {
		var item ArtifactMigrationItem
		var source ArtifactLocation
		var target ArtifactLocation
		var artifactMetadata []byte
		var sourceMetadata []byte
		var targetMetadata []byte
		var sourceCreatedAt *time.Time
		var targetCreatedAt *time.Time
		if err := rows.Scan(
			&item.Artifact.ID, &item.Artifact.ProjectID, &item.Artifact.WorkflowVersionID, &item.Artifact.RunID, &item.Artifact.WorkflowItemID,
			&item.Artifact.ArtifactType, &item.Artifact.StorageBackend, &item.Artifact.URI, &item.Artifact.SourcePath, &item.Artifact.SHA256,
			&item.Artifact.SizeBytes, &item.Artifact.ContentType, &artifactMetadata, &item.Artifact.CreatedAt,
			&source.ID, &source.Role, &source.StorageBackend, &source.URI, &source.SHA256, &source.SizeBytes, &source.ContentType,
			&source.VerifiedAt, &sourceMetadata, &sourceCreatedAt,
			&target.ID, &target.Role, &target.StorageBackend, &target.URI, &target.SHA256, &target.SizeBytes, &target.ContentType,
			&target.VerifiedAt, &targetMetadata, &targetCreatedAt,
		); err != nil {
			return ArtifactMigrationInventory{}, fmt.Errorf("scan artifact migration inventory: %w", err)
		}
		if err := json.Unmarshal(artifactMetadata, &item.Artifact.Metadata); err != nil {
			return ArtifactMigrationInventory{}, fmt.Errorf("parse artifact migration metadata: %w", err)
		}
		if source.ID == 0 {
			item.Source = locationFromArtifact(item.Artifact, "primary")
		} else {
			source.ProjectID = record.ID
			source.ArtifactID = item.Artifact.ID
			if sourceCreatedAt != nil {
				source.CreatedAt = *sourceCreatedAt
			}
			if err := json.Unmarshal(sourceMetadata, &source.Metadata); err != nil {
				return ArtifactMigrationInventory{}, fmt.Errorf("parse source location metadata: %w", err)
			}
			item.Source = source
		}
		if target.ID != 0 {
			target.ProjectID = record.ID
			target.ArtifactID = item.Artifact.ID
			if targetCreatedAt != nil {
				target.CreatedAt = *targetCreatedAt
			}
			if err := json.Unmarshal(targetMetadata, &target.Metadata); err != nil {
				return ArtifactMigrationInventory{}, fmt.Errorf("parse target location metadata: %w", err)
			}
			item.Target = target
		}
		item.Status = artifactMigrationStatus(item)
		switch item.Status {
		case "activated", "observing", "stable":
			inventory.Activated++
			if item.Status == "observing" {
				inventory.Observing++
			}
			if item.Status == "stable" {
				inventory.Stable++
			}
		case "verified":
			inventory.Verified++
		default:
			inventory.Pending++
		}
		inventory.Items = append(inventory.Items, item)
	}
	if err := rows.Err(); err != nil {
		return ArtifactMigrationInventory{}, fmt.Errorf("iterate artifact migration inventory: %w", err)
	}
	return inventory, nil
}

func (s Store) CompleteArtifactObservation(ctx context.Context, record Record, artifactID int64, options CompleteArtifactObservationOptions) (ArtifactRecord, error) {
	if record.ID <= 0 || artifactID <= 0 {
		return ArtifactRecord{}, fmt.Errorf("project id and artifact id are required")
	}
	if strings.TrimSpace(options.Actor) == "" || strings.TrimSpace(options.Reason) == "" {
		return ArtifactRecord{}, fmt.Errorf("actor and reason are required")
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return ArtifactRecord{}, fmt.Errorf("begin artifact observation completion: %w", err)
	}
	defer tx.Rollback(ctx)
	recordState, err := loadArtifactForMigration(ctx, tx, record.ID, artifactID)
	if err != nil {
		return ArtifactRecord{}, err
	}
	if artifactMetadataString(recordState.Metadata, "artifact_migration_status") == "stable" {
		if err := tx.Commit(ctx); err != nil {
			return ArtifactRecord{}, fmt.Errorf("commit stable artifact observation replay: %w", err)
		}
		return recordState, nil
	}
	deadline, err := time.Parse(time.RFC3339, artifactMetadataString(recordState.Metadata, "artifact_migration_observation_until"))
	if err != nil {
		return ArtifactRecord{}, fmt.Errorf("artifact observation deadline is missing or invalid")
	}
	if time.Now().UTC().Before(deadline) {
		return ArtifactRecord{}, fmt.Errorf("artifact observation period has not completed")
	}
	result, err := tx.Exec(ctx, `
UPDATE artifact_locations
SET metadata = metadata || jsonb_build_object('status', 'stable', 'observed_at', now()::text, 'actor', $3::text, 'reason', $4::text)
WHERE project_id = $1 AND artifact_id = $2 AND location_role = 'primary' AND verified_at IS NOT NULL`, record.ID, artifactID, strings.TrimSpace(options.Actor), strings.TrimSpace(options.Reason))
	if err != nil {
		return ArtifactRecord{}, fmt.Errorf("complete artifact location observation: %w", err)
	}
	if result.RowsAffected() != 1 {
		return ArtifactRecord{}, fmt.Errorf("exactly one verified primary artifact location is required")
	}
	updated, err := scanArtifactRecordWithRun(tx.QueryRow(ctx, `
UPDATE artifacts
SET metadata = metadata || jsonb_build_object('artifact_migration_status', 'stable', 'artifact_migration_observed_at', now()::text)
WHERE project_id = $1 AND id = $2
RETURNING id, project_id, COALESCE(workflow_version_id, 0), COALESCE(run_id, 0), COALESCE(workflow_item_id, 0),
          artifact_type, storage_backend, uri, COALESCE(source_path, ''), COALESCE(sha256, ''),
          COALESCE(size_bytes, 0), COALESCE(content_type, ''), metadata, created_at`, record.ID, artifactID))
	if err != nil {
		return ArtifactRecord{}, fmt.Errorf("mark artifact observation stable: %w", err)
	}
	if err := insertArtifactMigrationAudit(ctx, tx, record.ID, "artifact.migration.observe.complete", artifactID, options.Actor, options.Reason, map[string]any{
		"storage_backend": updated.StorageBackend, "uri": updated.URI, "observation_until": deadline.UTC().Format(time.RFC3339),
	}); err != nil {
		return ArtifactRecord{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return ArtifactRecord{}, fmt.Errorf("commit artifact observation completion: %w", err)
	}
	return updated, nil
}

func (s Store) CopyArtifactToBackend(ctx context.Context, record Record, artifactID int64, options CopyArtifactOptions) (ArtifactLocation, error) {
	if record.ID <= 0 || artifactID <= 0 {
		return ArtifactLocation{}, fmt.Errorf("project id and artifact id are required")
	}
	options.TargetBackend = normalizedArtifactBackend(options.TargetBackend)
	if options.TargetBackend == "" || strings.TrimSpace(options.Actor) == "" || strings.TrimSpace(options.Reason) == "" {
		return ArtifactLocation{}, fmt.Errorf("target backend, actor and reason are required")
	}
	source, err := s.GetArtifact(ctx, artifactID)
	if err != nil {
		return ArtifactLocation{}, err
	}
	if source.ProjectID != record.ID {
		return ArtifactLocation{}, ErrArtifactNotFound
	}
	content, err := readArtifactRecordContent(ctx, source)
	if err != nil {
		return ArtifactLocation{}, err
	}
	relativePath, err := artifactMigrationRelativePath(record, source)
	if err != nil {
		return ArtifactLocation{}, err
	}
	stored, err := artifact.WriteConfigured(ctx, options.TargetBackend, options.TargetRoot, relativePath, content.Content, content.ContentType)
	if err != nil {
		return ArtifactLocation{}, fmt.Errorf("copy artifact to %s: %w", options.TargetBackend, err)
	}
	if err := verifyMigratedArtifact(source, stored); err != nil {
		return ArtifactLocation{}, err
	}
	verifiedContent, err := artifact.ReadConfigured(ctx, stored.Backend, stored.URI)
	if err != nil {
		return ArtifactLocation{}, fmt.Errorf("read back migrated artifact: %w", err)
	}
	if err := verifyMigratedArtifactContent(source, verifiedContent); err != nil {
		return ArtifactLocation{}, err
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return ArtifactLocation{}, fmt.Errorf("begin artifact migration copy: %w", err)
	}
	defer tx.Rollback(ctx)
	if _, err := upsertArtifactLocation(ctx, tx, source, locationFromArtifact(source, "migration_source"), nil, map[string]any{
		"status": "retained", "delete_allowed": false,
	}); err != nil {
		return ArtifactLocation{}, err
	}
	verifiedAt := time.Now().UTC()
	target := ArtifactLocation{ProjectID: source.ProjectID, ArtifactID: source.ID, Role: "migration_candidate", StorageBackend: stored.Backend, URI: stored.URI, SHA256: stored.SHA256, SizeBytes: stored.SizeBytes, ContentType: stored.ContentType}
	target, err = upsertArtifactLocation(ctx, tx, source, target, &verifiedAt, map[string]any{
		"status": "verified", "actor": strings.TrimSpace(options.Actor), "reason": strings.TrimSpace(options.Reason),
	})
	if err != nil {
		return ArtifactLocation{}, err
	}
	if err := insertArtifactMigrationAudit(ctx, tx, source.ProjectID, "artifact.migration.copy", source.ID, options.Actor, options.Reason, map[string]any{
		"source_backend": source.StorageBackend, "source_uri": source.URI, "target_backend": target.StorageBackend,
		"target_uri": target.URI, "target_location_id": target.ID, "sha256": target.SHA256, "size_bytes": target.SizeBytes,
	}); err != nil {
		return ArtifactLocation{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return ArtifactLocation{}, fmt.Errorf("commit artifact migration copy: %w", err)
	}
	return target, nil
}

func (s Store) ActivateArtifactLocation(ctx context.Context, record Record, artifactID int64, options ActivateArtifactOptions) (ArtifactRecord, error) {
	if record.ID <= 0 || artifactID <= 0 || options.TargetLocationID <= 0 {
		return ArtifactRecord{}, fmt.Errorf("project, artifact and target location ids are required")
	}
	if strings.TrimSpace(options.Actor) == "" || strings.TrimSpace(options.Reason) == "" || options.ObservationUntil.IsZero() {
		return ArtifactRecord{}, fmt.Errorf("actor, reason and observation deadline are required")
	}
	if !options.ObservationUntil.After(time.Now().UTC()) {
		return ArtifactRecord{}, fmt.Errorf("observation deadline must be in the future")
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return ArtifactRecord{}, fmt.Errorf("begin artifact migration activation: %w", err)
	}
	defer tx.Rollback(ctx)

	source, err := loadArtifactForMigration(ctx, tx, record.ID, artifactID)
	if err != nil {
		return ArtifactRecord{}, err
	}
	target, err := loadArtifactLocationForUpdate(ctx, tx, record.ID, artifactID, options.TargetLocationID)
	if err != nil {
		return ArtifactRecord{}, err
	}
	if target.VerifiedAt == nil || target.SHA256 != source.SHA256 || target.SizeBytes != source.SizeBytes {
		return ArtifactRecord{}, fmt.Errorf("target artifact location is not verified against source metadata")
	}
	if _, err := tx.Exec(ctx, `
UPDATE artifact_locations
SET location_role = CASE WHEN id = $3 THEN 'primary' WHEN location_role = 'primary' THEN 'migration_source' ELSE location_role END,
    metadata = metadata || CASE WHEN id = $3
        THEN jsonb_build_object('status', 'observing', 'observation_until', $4::text, 'actor', $5::text, 'reason', $6::text)
        ELSE jsonb_build_object('status', 'retained', 'delete_allowed', false) END
WHERE project_id = $1 AND artifact_id = $2`, record.ID, artifactID, options.TargetLocationID, options.ObservationUntil.UTC().Format(time.RFC3339), strings.TrimSpace(options.Actor), strings.TrimSpace(options.Reason)); err != nil {
		return ArtifactRecord{}, fmt.Errorf("promote artifact location: %w", err)
	}
	if _, err := tx.Exec(ctx, `
INSERT INTO artifact_locations (project_id, artifact_id, location_role, storage_backend, uri, sha256, size_bytes, content_type, verified_at, metadata)
VALUES ($1, $2, 'migration_source', $3, $4, $5, $6, $7, now(), jsonb_build_object('status', 'retained', 'delete_allowed', false))
ON CONFLICT (artifact_id, location_role, uri) DO UPDATE
SET metadata = artifact_locations.metadata || EXCLUDED.metadata`, source.ProjectID, source.ID, source.StorageBackend, source.URI, source.SHA256, source.SizeBytes, source.ContentType); err != nil {
		return ArtifactRecord{}, fmt.Errorf("retain artifact migration source: %w", err)
	}
	updated, err := scanArtifactRecordWithRun(tx.QueryRow(ctx, `
UPDATE artifacts
SET storage_backend = $3, uri = $4, sha256 = $5, size_bytes = $6, content_type = $7,
    metadata = metadata || jsonb_build_object('artifact_migration_status', 'observing', 'artifact_migration_observation_until', $8::text)
WHERE project_id = $1 AND id = $2
RETURNING id, project_id, COALESCE(workflow_version_id, 0), COALESCE(run_id, 0), COALESCE(workflow_item_id, 0),
          artifact_type, storage_backend, uri, COALESCE(source_path, ''), COALESCE(sha256, ''),
          COALESCE(size_bytes, 0), COALESCE(content_type, ''), metadata, created_at`, record.ID, artifactID, target.StorageBackend, target.URI, target.SHA256, target.SizeBytes, target.ContentType, options.ObservationUntil.UTC().Format(time.RFC3339)))
	if err != nil {
		return ArtifactRecord{}, fmt.Errorf("activate artifact location: %w", err)
	}
	if err := insertArtifactMigrationAudit(ctx, tx, source.ProjectID, "artifact.migration.activate", source.ID, options.Actor, options.Reason, map[string]any{
		"source_backend": source.StorageBackend, "source_uri": source.URI, "target_backend": target.StorageBackend,
		"target_uri": target.URI, "target_location_id": target.ID, "observation_until": options.ObservationUntil.UTC().Format(time.RFC3339),
	}); err != nil {
		return ArtifactRecord{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return ArtifactRecord{}, fmt.Errorf("commit artifact migration activation: %w", err)
	}
	return updated, nil
}

func (s Store) artifactReadLocations(ctx context.Context, artifactID int64) ([]ArtifactLocation, error) {
	rows, err := s.pool.Query(ctx, `
SELECT id, project_id, artifact_id, location_role, storage_backend, uri, COALESCE(sha256, ''),
       COALESCE(size_bytes, 0), COALESCE(content_type, ''), verified_at, metadata, created_at
FROM artifact_locations
WHERE artifact_id = $1 AND location_role IN ('primary', 'migration_candidate', 'migration_source')
ORDER BY CASE location_role WHEN 'primary' THEN 0 WHEN 'migration_candidate' THEN 1 ELSE 2 END, id DESC`, artifactID)
	if err != nil {
		return nil, fmt.Errorf("list artifact read locations: %w", err)
	}
	defer rows.Close()
	var locations []ArtifactLocation
	for rows.Next() {
		location, err := scanArtifactLocation(rows)
		if err != nil {
			return nil, err
		}
		locations = append(locations, location)
	}
	return locations, rows.Err()
}

func normalizedArtifactBackend(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "object" {
		return "s3"
	}
	return value
}

func artifactMigrationRelativePath(record Record, source ArtifactRecord) (string, error) {
	path := strings.TrimSpace(source.SourcePath)
	if path == "" {
		path = fmt.Sprintf("artifact-%d", source.ID)
	}
	path = filepath.Clean(path)
	if filepath.IsAbs(path) || path == "." || strings.HasPrefix(path, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("artifact source path is outside project namespace")
	}
	return filepath.Join(record.Key, "migration", fmt.Sprintf("artifact-%d", source.ID), path), nil
}

func verifyMigratedArtifact(source ArtifactRecord, stored artifact.Stored) error {
	if source.SHA256 != "" && source.SHA256 != stored.SHA256 {
		return fmt.Errorf("migrated artifact sha256 mismatch")
	}
	if source.SizeBytes > 0 && source.SizeBytes != stored.SizeBytes {
		return fmt.Errorf("migrated artifact size mismatch")
	}
	return nil
}

func verifyMigratedArtifactContent(source ArtifactRecord, content []byte) error {
	sha, size := hashBytes(content)
	if source.SHA256 != "" && source.SHA256 != sha {
		return fmt.Errorf("migrated artifact readback sha256 mismatch")
	}
	if source.SizeBytes > 0 && source.SizeBytes != size {
		return fmt.Errorf("migrated artifact readback size mismatch")
	}
	return nil
}

func artifactMigrationStatus(item ArtifactMigrationItem) string {
	if item.Target.ID == 0 {
		return "pending"
	}
	if item.Target.Role == "primary" {
		if status := artifactMetadataString(item.Target.Metadata, "status"); status == "observing" || status == "stable" {
			return status
		}
		return "activated"
	}
	if item.Target.VerifiedAt != nil && item.Target.SHA256 == item.Artifact.SHA256 && item.Target.SizeBytes == item.Artifact.SizeBytes {
		return "verified"
	}
	return "pending"
}

func artifactMetadataString(metadata map[string]any, key string) string {
	value, _ := metadata[key].(string)
	return strings.TrimSpace(value)
}

func locationFromArtifact(record ArtifactRecord, role string) ArtifactLocation {
	verifiedAt := record.CreatedAt
	return ArtifactLocation{ProjectID: record.ProjectID, ArtifactID: record.ID, Role: role, StorageBackend: record.StorageBackend, URI: record.URI, SHA256: record.SHA256, SizeBytes: record.SizeBytes, ContentType: record.ContentType, VerifiedAt: &verifiedAt}
}

func upsertArtifactLocation(ctx context.Context, tx pgx.Tx, source ArtifactRecord, location ArtifactLocation, verifiedAt *time.Time, metadata map[string]any) (ArtifactLocation, error) {
	metadataJSON, err := json.Marshal(metadata)
	if err != nil {
		return ArtifactLocation{}, fmt.Errorf("marshal artifact location metadata: %w", err)
	}
	return scanArtifactLocation(tx.QueryRow(ctx, `
INSERT INTO artifact_locations (project_id, artifact_id, location_role, storage_backend, uri, sha256, size_bytes, content_type, verified_at, metadata)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10::jsonb)
ON CONFLICT (artifact_id, location_role, uri) DO UPDATE
SET sha256 = EXCLUDED.sha256, size_bytes = EXCLUDED.size_bytes, content_type = EXCLUDED.content_type,
    verified_at = COALESCE(EXCLUDED.verified_at, artifact_locations.verified_at), metadata = artifact_locations.metadata || EXCLUDED.metadata
RETURNING id, project_id, artifact_id, location_role, storage_backend, uri, COALESCE(sha256, ''),
          COALESCE(size_bytes, 0), COALESCE(content_type, ''), verified_at, metadata, created_at`, source.ProjectID, source.ID, location.Role, location.StorageBackend, location.URI, location.SHA256, location.SizeBytes, location.ContentType, verifiedAt, string(metadataJSON)))
}

func loadArtifactForMigration(ctx context.Context, tx pgx.Tx, projectID, artifactID int64) (ArtifactRecord, error) {
	record, err := scanArtifactRecordWithRun(tx.QueryRow(ctx, `
SELECT id, project_id, COALESCE(workflow_version_id, 0), COALESCE(run_id, 0), COALESCE(workflow_item_id, 0),
       artifact_type, storage_backend, uri, COALESCE(source_path, ''), COALESCE(sha256, ''),
       COALESCE(size_bytes, 0), COALESCE(content_type, ''), metadata, created_at
FROM artifacts WHERE project_id = $1 AND id = $2 FOR UPDATE`, projectID, artifactID))
	if errors.Is(err, pgx.ErrNoRows) {
		return ArtifactRecord{}, ErrArtifactNotFound
	}
	return record, err
}

func loadArtifactLocationForUpdate(ctx context.Context, tx pgx.Tx, projectID, artifactID, locationID int64) (ArtifactLocation, error) {
	location, err := scanArtifactLocation(tx.QueryRow(ctx, `
SELECT id, project_id, artifact_id, location_role, storage_backend, uri, COALESCE(sha256, ''),
       COALESCE(size_bytes, 0), COALESCE(content_type, ''), verified_at, metadata, created_at
FROM artifact_locations WHERE project_id = $1 AND artifact_id = $2 AND id = $3 FOR UPDATE`, projectID, artifactID, locationID))
	if errors.Is(err, pgx.ErrNoRows) {
		return ArtifactLocation{}, fmt.Errorf("artifact migration target location not found")
	}
	return location, err
}

func scanArtifactLocation(row scanner) (ArtifactLocation, error) {
	var location ArtifactLocation
	var metadata []byte
	if err := row.Scan(&location.ID, &location.ProjectID, &location.ArtifactID, &location.Role, &location.StorageBackend, &location.URI,
		&location.SHA256, &location.SizeBytes, &location.ContentType, &location.VerifiedAt, &metadata, &location.CreatedAt); err != nil {
		return ArtifactLocation{}, err
	}
	if err := json.Unmarshal(metadata, &location.Metadata); err != nil {
		return ArtifactLocation{}, fmt.Errorf("parse artifact location metadata: %w", err)
	}
	return location, nil
}

func insertArtifactMigrationAudit(ctx context.Context, tx pgx.Tx, projectID int64, action string, artifactID int64, actor, reason string, metadata map[string]any) error {
	metadata["actor"] = strings.TrimSpace(actor)
	encoded, err := json.Marshal(metadata)
	if err != nil {
		return fmt.Errorf("marshal artifact migration audit metadata: %w", err)
	}
	if _, err := tx.Exec(ctx, `
INSERT INTO audit_events (project_id, action, capability, resource_type, resource, decision, reason, metadata)
VALUES ($1, $2, 'manage_artifacts', 'artifact', $3, 'allowed', $4, $5::jsonb)`, projectID, action, fmt.Sprintf("%d", artifactID), strings.TrimSpace(reason), string(encoded)); err != nil {
		return fmt.Errorf("insert artifact migration audit event: %w", err)
	}
	return nil
}
