package project

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

type ArtifactArchivePreviewOptions struct {
	RetentionClass string
	Limit          int
	IdempotencyKey string
	Actor          string
	Reason         string
}

type ArtifactArchivePreviewItem struct {
	ArtifactID     int64
	ArtifactType   string
	StorageBackend string
	URI            string
	SourcePath     string
	RetentionClass string
	ArchiveState   string
	Action         string
	Decision       string
	Reason         string
	Metadata       map[string]any
}

type ArtifactArchivePreviewSummary struct {
	TotalArtifacts    int
	ArchiveCandidates int
	RetainedArtifacts int
	ExternalRefs      int
	NeedsPolicy       int
}

type ArtifactArchivePreviewResult struct {
	Project                 Record
	Status                  string
	Mode                    string
	Summary                 ArtifactArchivePreviewSummary
	Items                   []ArtifactArchivePreviewItem
	EventID                 int64
	AuditEventID            int64
	IdempotencyKey          string
	Created                 bool
	GeneratedAt             time.Time
	ProjectWriteAttempted   bool
	StorageWriteAttempted   bool
	ArtifactDeleteAttempted bool
}

const artifactArchivePreviewCommandType = "artifact.archive.preview"

type artifactArchivePreviewQuerier interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}

func (s Store) ArtifactArchivePreview(ctx context.Context, record Record, options ArtifactArchivePreviewOptions) (ArtifactArchivePreviewResult, error) {
	options = normalizeArtifactArchivePreviewOptions(options)
	requestHash, err := artifactArchivePreviewRequestHash(record, options)
	if err != nil {
		return ArtifactArchivePreviewResult{}, err
	}
	if options.IdempotencyKey == "" {
		options.IdempotencyKey = artifactArchivePreviewIdempotencyKey(record, options, requestHash)
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return ArtifactArchivePreviewResult{}, fmt.Errorf("begin artifact archive preview: %w", err)
	}
	defer tx.Rollback(ctx)

	created, err := reserveCommandRequest(ctx, tx, record.ID, artifactArchivePreviewCommandType, options.IdempotencyKey, requestHash)
	if err != nil {
		return ArtifactArchivePreviewResult{}, err
	}
	if !created {
		result, err := loadArtifactArchivePreviewByCommandResponse(ctx, tx, record, options.IdempotencyKey)
		if err != nil {
			return ArtifactArchivePreviewResult{}, err
		}
		if err := tx.Commit(ctx); err != nil {
			return ArtifactArchivePreviewResult{}, fmt.Errorf("commit idempotent artifact archive preview: %w", err)
		}
		result.Created = false
		return result, nil
	}

	artifacts, err := listProjectArtifactsForArchivePreview(ctx, tx, record.ID, options)
	if err != nil {
		return ArtifactArchivePreviewResult{}, err
	}
	result := buildArtifactArchivePreview(record, artifacts, options)
	eventID, err := insertArtifactArchivePreviewEvent(ctx, tx, result, options)
	if err != nil {
		return ArtifactArchivePreviewResult{}, err
	}
	result.EventID = eventID
	auditEventID, err := insertArtifactArchivePreviewAuditEvent(ctx, tx, result, options)
	if err != nil {
		return ArtifactArchivePreviewResult{}, err
	}
	result.AuditEventID = auditEventID
	result.IdempotencyKey = options.IdempotencyKey
	result.Created = true
	if err := completeCommandRequestResponse(ctx, tx, record.ID, artifactArchivePreviewCommandType, options.IdempotencyKey, artifactArchivePreviewCommandResponse(result)); err != nil {
		return ArtifactArchivePreviewResult{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return ArtifactArchivePreviewResult{}, fmt.Errorf("commit artifact archive preview: %w", err)
	}
	return result, nil
}

func (s Store) ArtifactArchivePreviewReadOnly(ctx context.Context, record Record, options ArtifactArchivePreviewOptions) (ArtifactArchivePreviewResult, error) {
	options = normalizeArtifactArchivePreviewOptions(options)
	artifacts, err := s.listAllProjectArtifacts(ctx, record.ID)
	if err != nil {
		return ArtifactArchivePreviewResult{}, err
	}
	artifacts = filterArtifactsByArchivePreviewRetentionClass(artifacts, options.RetentionClass)
	return buildArtifactArchivePreview(record, artifacts, options), nil
}

func normalizeArtifactArchivePreviewOptions(options ArtifactArchivePreviewOptions) ArtifactArchivePreviewOptions {
	options.RetentionClass = strings.TrimSpace(options.RetentionClass)
	options.IdempotencyKey = strings.TrimSpace(options.IdempotencyKey)
	options.Actor = strings.TrimSpace(options.Actor)
	options.Reason = strings.TrimSpace(options.Reason)
	if options.Limit <= 0 {
		options.Limit = 100
	}
	if options.Limit > 500 {
		options.Limit = 500
	}
	if options.Actor == "" {
		options.Actor = "local-user"
	}
	if options.Reason == "" {
		options.Reason = "artifact archive preview"
	}
	return options
}

func artifactArchivePreviewRequestHash(record Record, options ArtifactArchivePreviewOptions) (string, error) {
	payload, err := json.Marshal(map[string]any{
		"command_type":    artifactArchivePreviewCommandType,
		"project_key":     record.Key,
		"project_id":      record.ID,
		"retention_class": options.RetentionClass,
		"limit":           options.Limit,
		"actor":           options.Actor,
		"reason":          options.Reason,
		"preview_only":    true,
	})
	if err != nil {
		return "", fmt.Errorf("marshal artifact archive preview request hash payload: %w", err)
	}
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:]), nil
}

func artifactArchivePreviewIdempotencyKey(record Record, options ArtifactArchivePreviewOptions, requestHash string) string {
	prefix := requestHash
	if len(prefix) > 16 {
		prefix = prefix[:16]
	}
	retentionClass := options.RetentionClass
	if retentionClass == "" {
		retentionClass = "all"
	}
	return fmt.Sprintf("artifact.archive.preview:%s:%s:%d:%s", record.Key, retentionClass, options.Limit, prefix)
}

func listProjectArtifactsForArchivePreview(ctx context.Context, querier artifactArchivePreviewQuerier, projectID int64, options ArtifactArchivePreviewOptions) ([]ArtifactRecord, error) {
	rows, err := querier.Query(ctx, `
SELECT id, project_id, COALESCE(workflow_version_id, 0), COALESCE(workflow_item_id, 0),
       artifact_type, storage_backend, uri, COALESCE(source_path, ''), COALESCE(sha256, ''),
       COALESCE(size_bytes, 0), COALESCE(content_type, ''), metadata, created_at
FROM artifacts
WHERE project_id = $1
ORDER BY created_at DESC, id DESC
LIMIT $2`,
		projectID,
		options.Limit,
	)
	if err != nil {
		return nil, fmt.Errorf("list artifacts for archive preview: %w", err)
	}
	defer rows.Close()
	artifacts := []ArtifactRecord{}
	for rows.Next() {
		artifact, err := scanArtifactRecord(rows)
		if err != nil {
			return nil, err
		}
		artifacts = append(artifacts, artifact)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate artifacts for archive preview: %w", err)
	}
	return filterArtifactsByArchivePreviewRetentionClass(artifacts, options.RetentionClass), nil
}

func filterArtifactsByArchivePreviewRetentionClass(artifacts []ArtifactRecord, retentionClass string) []ArtifactRecord {
	if retentionClass == "" {
		return artifacts
	}
	filtered := make([]ArtifactRecord, 0, len(artifacts))
	for _, artifact := range artifacts {
		if retentionClassForArtifact(artifact) == retentionClass {
			filtered = append(filtered, artifact)
		}
	}
	return filtered
}

func buildArtifactArchivePreview(record Record, artifacts []ArtifactRecord, options ArtifactArchivePreviewOptions) ArtifactArchivePreviewResult {
	result := ArtifactArchivePreviewResult{
		Project:                 record,
		Status:                  "ready",
		Mode:                    "metadata_only_archive_preview",
		Items:                   make([]ArtifactArchivePreviewItem, 0, len(artifacts)),
		GeneratedAt:             time.Now().UTC(),
		ProjectWriteAttempted:   false,
		StorageWriteAttempted:   false,
		ArtifactDeleteAttempted: false,
	}
	_ = options
	for _, artifact := range artifacts {
		item := artifactArchivePreviewItem(artifact)
		result.Items = append(result.Items, item)
		result.Summary.TotalArtifacts++
		switch item.ArchiveState {
		case "archive_candidate":
			result.Summary.ArchiveCandidates++
		case "retained":
			result.Summary.RetainedArtifacts++
		case "metadata_only_reference":
			result.Summary.ExternalRefs++
		case "needs_policy":
			result.Summary.NeedsPolicy++
		}
	}
	if result.Summary.ExternalRefs > 0 || result.Summary.NeedsPolicy > 0 {
		result.Status = "needs_attention"
	}
	return result
}

func artifactArchivePreviewItem(artifact ArtifactRecord) ArtifactArchivePreviewItem {
	retentionClass := retentionClassForArtifact(artifact)
	item := ArtifactArchivePreviewItem{
		ArtifactID:     artifact.ID,
		ArtifactType:   artifact.ArtifactType,
		StorageBackend: artifact.StorageBackend,
		URI:            artifact.URI,
		SourcePath:     artifact.SourcePath,
		RetentionClass: retentionClass,
		Metadata: map[string]any{
			"sha256":                    artifact.SHA256,
			"size_bytes":                artifact.SizeBytes,
			"content_type":              artifact.ContentType,
			"workflow_version_id":       artifact.WorkflowVersionID,
			"workflow_item_id":          artifact.WorkflowItemID,
			"project_write_attempted":   false,
			"storage_write_attempted":   false,
			"artifact_delete_attempted": false,
		},
	}
	switch retentionClass {
	case "ephemeral":
		item.ArchiveState = "archive_candidate"
		item.Action = "eligible_for_future_gc_preview"
		item.Decision = "preview_only"
		item.Reason = "ephemeral artifacts may be cleaned only by a future explicit GC command"
	case "external_ref":
		item.ArchiveState = "metadata_only_reference"
		item.Action = "keep_metadata_only"
		item.Decision = "requires_archive_ownership_decision"
		item.Reason = "project reference originals remain in the managed project and are not copied or deleted"
	case "run_evidence", "audit", "release":
		item.ArchiveState = "retained"
		item.Action = "keep"
		item.Decision = "protected_retention"
		item.Reason = "retention class is protected from ordinary archive or GC"
	default:
		item.ArchiveState = "needs_policy"
		item.Action = "manual_review"
		item.Decision = "needs_policy"
		item.Reason = "artifact retention class is unknown"
	}
	return item
}

func retentionClassForArtifact(artifact ArtifactRecord) string {
	if value := metadataString(artifact.Metadata, "retention_class"); value != "" {
		return value
	}
	switch artifact.StorageBackend {
	case "external_project", "project_reference":
		return "external_ref"
	}
	if metadataBool(artifact.Metadata, "dry_run") || strings.Contains(artifact.ArtifactType, "preview") {
		return "ephemeral"
	}
	if strings.Contains(artifact.ArtifactType, "release") || strings.Contains(artifact.ArtifactType, "distribution") {
		return "release"
	}
	if strings.Contains(artifact.ArtifactType, "audit") || strings.Contains(artifact.ArtifactType, "approval") {
		return "audit"
	}
	if artifact.StorageBackend == "local" {
		return "run_evidence"
	}
	return "unknown"
}

func insertArtifactArchivePreviewEvent(ctx context.Context, tx pgx.Tx, result ArtifactArchivePreviewResult, options ArtifactArchivePreviewOptions) (int64, error) {
	metadata, err := json.Marshal(artifactArchivePreviewMetadata(result, options))
	if err != nil {
		return 0, fmt.Errorf("marshal artifact archive preview event metadata: %w", err)
	}
	var eventID int64
	if err := tx.QueryRow(ctx, `
INSERT INTO events (project_id, event_type, severity, message, metadata)
VALUES ($1, 'artifact.archive.preview.completed', 'info', 'Artifact archive preview completed', $2::jsonb)
RETURNING id`,
		result.Project.ID,
		string(metadata),
	).Scan(&eventID); err != nil {
		return 0, fmt.Errorf("insert artifact archive preview event: %w", err)
	}
	return eventID, nil
}

func insertArtifactArchivePreviewAuditEvent(ctx context.Context, tx pgx.Tx, result ArtifactArchivePreviewResult, options ArtifactArchivePreviewOptions) (int64, error) {
	metadata, err := json.Marshal(artifactArchivePreviewMetadata(result, options))
	if err != nil {
		return 0, fmt.Errorf("marshal artifact archive preview audit metadata: %w", err)
	}
	var auditEventID int64
	if err := tx.QueryRow(ctx, `
INSERT INTO audit_events (project_id, action, capability, resource_type, resource, decision, reason, metadata)
VALUES ($1, 'artifact.archive.preview', 'manage_artifacts', 'project', $2, 'allowed', $3, $4::jsonb)
RETURNING id`,
		result.Project.ID,
		result.Project.Key,
		options.Reason,
		string(metadata),
	).Scan(&auditEventID); err != nil {
		return 0, fmt.Errorf("insert artifact archive preview audit event: %w", err)
	}
	return auditEventID, nil
}

func artifactArchivePreviewMetadata(result ArtifactArchivePreviewResult, options ArtifactArchivePreviewOptions) map[string]any {
	return map[string]any{
		"project_key":               result.Project.Key,
		"status":                    result.Status,
		"mode":                      result.Mode,
		"retention_class":           options.RetentionClass,
		"limit":                     options.Limit,
		"actor":                     options.Actor,
		"idempotency_key":           options.IdempotencyKey,
		"summary":                   artifactArchivePreviewSummaryMap(result.Summary),
		"project_write_attempted":   false,
		"storage_write_attempted":   false,
		"artifact_delete_attempted": false,
	}
}

func artifactArchivePreviewSummaryMap(summary ArtifactArchivePreviewSummary) map[string]any {
	return map[string]any{
		"total_artifacts":    summary.TotalArtifacts,
		"archive_candidates": summary.ArchiveCandidates,
		"retained_artifacts": summary.RetainedArtifacts,
		"external_refs":      summary.ExternalRefs,
		"needs_policy":       summary.NeedsPolicy,
	}
}

func artifactArchivePreviewCommandResponse(result ArtifactArchivePreviewResult) map[string]any {
	items := make([]map[string]any, 0, len(result.Items))
	for _, item := range result.Items {
		items = append(items, artifactArchivePreviewItemMap(item))
	}
	return map[string]any{
		"project_id":                result.Project.ID,
		"project_key":               result.Project.Key,
		"status":                    result.Status,
		"mode":                      result.Mode,
		"summary":                   artifactArchivePreviewSummaryMap(result.Summary),
		"items":                     items,
		"event_id":                  result.EventID,
		"audit_event_id":            result.AuditEventID,
		"generated_at":              result.GeneratedAt.Format(time.RFC3339),
		"project_write_attempted":   false,
		"storage_write_attempted":   false,
		"artifact_delete_attempted": false,
	}
}

func artifactArchivePreviewItemMap(item ArtifactArchivePreviewItem) map[string]any {
	return map[string]any{
		"artifact_id":     item.ArtifactID,
		"artifact_type":   item.ArtifactType,
		"storage_backend": item.StorageBackend,
		"uri":             item.URI,
		"source_path":     item.SourcePath,
		"retention_class": item.RetentionClass,
		"archive_state":   item.ArchiveState,
		"action":          item.Action,
		"decision":        item.Decision,
		"reason":          item.Reason,
		"metadata":        item.Metadata,
	}
}

func loadArtifactArchivePreviewByCommandResponse(ctx context.Context, tx pgx.Tx, record Record, idempotencyKey string) (ArtifactArchivePreviewResult, error) {
	response, err := loadCommandResponse(ctx, tx, record.ID, artifactArchivePreviewCommandType, idempotencyKey)
	if err != nil {
		return ArtifactArchivePreviewResult{}, err
	}
	return ArtifactArchivePreviewResult{
		Project:                 record,
		Status:                  metadataString(response, "status"),
		Mode:                    metadataString(response, "mode"),
		Summary:                 artifactArchivePreviewSummaryFromAny(response["summary"]),
		Items:                   artifactArchivePreviewItemsFromAny(response["items"]),
		EventID:                 metadataInt64(response, "event_id"),
		AuditEventID:            metadataInt64(response, "audit_event_id"),
		IdempotencyKey:          idempotencyKey,
		GeneratedAt:             metadataTime(response, "generated_at"),
		ProjectWriteAttempted:   metadataBool(response, "project_write_attempted"),
		StorageWriteAttempted:   metadataBool(response, "storage_write_attempted"),
		ArtifactDeleteAttempted: metadataBool(response, "artifact_delete_attempted"),
	}, nil
}

func artifactArchivePreviewSummaryFromAny(value any) ArtifactArchivePreviewSummary {
	metadata, ok := value.(map[string]any)
	if !ok {
		return ArtifactArchivePreviewSummary{}
	}
	return ArtifactArchivePreviewSummary{
		TotalArtifacts:    int(metadataInt64(metadata, "total_artifacts")),
		ArchiveCandidates: int(metadataInt64(metadata, "archive_candidates")),
		RetainedArtifacts: int(metadataInt64(metadata, "retained_artifacts")),
		ExternalRefs:      int(metadataInt64(metadata, "external_refs")),
		NeedsPolicy:       int(metadataInt64(metadata, "needs_policy")),
	}
}

func artifactArchivePreviewItemsFromAny(value any) []ArtifactArchivePreviewItem {
	items, ok := value.([]any)
	if !ok {
		return nil
	}
	out := make([]ArtifactArchivePreviewItem, 0, len(items))
	for _, itemValue := range items {
		metadata, ok := itemValue.(map[string]any)
		if !ok {
			continue
		}
		item := ArtifactArchivePreviewItem{
			ArtifactID:     metadataInt64(metadata, "artifact_id"),
			ArtifactType:   metadataString(metadata, "artifact_type"),
			StorageBackend: metadataString(metadata, "storage_backend"),
			URI:            metadataString(metadata, "uri"),
			SourcePath:     metadataString(metadata, "source_path"),
			RetentionClass: metadataString(metadata, "retention_class"),
			ArchiveState:   metadataString(metadata, "archive_state"),
			Action:         metadataString(metadata, "action"),
			Decision:       metadataString(metadata, "decision"),
			Reason:         metadataString(metadata, "reason"),
			Metadata:       mapFromAny(metadata["metadata"]),
		}
		out = append(out, item)
	}
	return out
}

func metadataTime(metadata map[string]any, key string) time.Time {
	value := metadataString(metadata, key)
	if value == "" {
		return time.Time{}
	}
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return time.Time{}
	}
	return parsed
}

func mapFromAny(value any) map[string]any {
	if value == nil {
		return map[string]any{}
	}
	metadata, ok := value.(map[string]any)
	if !ok {
		return map[string]any{}
	}
	return metadata
}
