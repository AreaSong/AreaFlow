package project

import (
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

var ErrInvalidResourceCursor = errors.New("invalid resource cursor")

type ResourcePageOptions struct {
	ProjectKey string
	Key        string
	Status     string
	Kind       string
	Type       string
	Cursor     string
	Limit      int
}

type WorkflowPageOptions struct {
	ResourcePageOptions
	ImportMode string
}

type RunPageOptions struct {
	ResourcePageOptions
	DryRun *bool
}

type ArtifactPageOptions struct {
	ResourcePageOptions
	StorageBackend    string
	SHA256            string
	RunID             int64
	WorkflowVersionID int64
}

type AuditEventPageOptions struct {
	ProjectID    int64
	ActorID      int64
	Action       string
	Decision     string
	ResourceType string
	Resource     string
	From         *time.Time
	To           *time.Time
	Cursor       string
	Limit        int
}

type WorkflowCollectionItem struct {
	Project  Record
	Workflow WorkflowVersion
}

type RunCollectionItem struct {
	Project  Record
	Workflow WorkflowVersion
	Run      RunRecord
}

type WorkerCollectionItem struct {
	Project Record
	Worker  WorkerRecord
}

type ArtifactCollectionItem struct {
	Project  Record
	Artifact ArtifactRecord
}

type WorkflowCollectionPage struct {
	Items      []WorkflowCollectionItem
	NextCursor string
}

type RunCollectionPage struct {
	Items      []RunCollectionItem
	NextCursor string
}

type WorkerCollectionPage struct {
	Items      []WorkerCollectionItem
	NextCursor string
}

type ArtifactCollectionPage struct {
	Items      []ArtifactCollectionItem
	NextCursor string
}

type AuditEventPage struct {
	Items      []AuditEventRecord
	NextCursor string
}

type resourceCursor struct {
	Time time.Time `json:"time"`
	ID   int64     `json:"id"`
}

func encodeResourceCursor(at time.Time, id int64) string {
	content, _ := json.Marshal(resourceCursor{Time: at.UTC(), ID: id})
	return base64.RawURLEncoding.EncodeToString(content)
}

func decodeResourceCursor(value string) (resourceCursor, error) {
	if strings.TrimSpace(value) == "" {
		return resourceCursor{}, nil
	}
	content, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil {
		return resourceCursor{}, ErrInvalidResourceCursor
	}
	var cursor resourceCursor
	if err := json.Unmarshal(content, &cursor); err != nil || cursor.Time.IsZero() || cursor.ID <= 0 {
		return resourceCursor{}, ErrInvalidResourceCursor
	}
	return cursor, nil
}

func normalizeResourcePageOptions(options ResourcePageOptions) ResourcePageOptions {
	options.ProjectKey = strings.TrimSpace(options.ProjectKey)
	options.Key = strings.TrimSpace(options.Key)
	options.Status = strings.TrimSpace(options.Status)
	options.Kind = strings.TrimSpace(options.Kind)
	options.Type = strings.TrimSpace(options.Type)
	options.Cursor = strings.TrimSpace(options.Cursor)
	if options.Limit <= 0 {
		options.Limit = 50
	}
	if options.Limit > 200 {
		options.Limit = 200
	}
	return options
}

type resourceQuery struct {
	conditions []string
	args       []any
}

func (q *resourceQuery) add(format string, value any) {
	q.args = append(q.args, value)
	q.conditions = append(q.conditions, fmt.Sprintf(format, len(q.args)))
}

func (q resourceQuery) where() string {
	if len(q.conditions) == 0 {
		return ""
	}
	return " WHERE " + strings.Join(q.conditions, " AND ")
}

func addResourceCursor(q *resourceQuery, column string, cursor resourceCursor) {
	if cursor.Time.IsZero() {
		return
	}
	q.args = append(q.args, cursor.Time, cursor.ID)
	q.conditions = append(q.conditions, fmt.Sprintf("(%s, id) < ($%d, $%d)", column, len(q.args)-1, len(q.args)))
}

const projectCollectionColumns = `
    p.id, p.project_key, p.name, p.kind, p.adapter, p.workflow_profile,
    COALESCE(p.default_branch, ''), COALESCE(local_path.root_path, ''),
    COALESCE(artifact_store.remote_url, ''), COALESCE(artifact_store.root_path, '')`

const projectCollectionJoins = `
LEFT JOIN LATERAL (
    SELECT root_path FROM project_connections
    WHERE project_id = p.id AND connection_type = 'local_path'
    ORDER BY updated_at DESC, id DESC LIMIT 1
) local_path ON true
LEFT JOIN LATERAL (
    SELECT root_path, remote_url FROM project_connections
    WHERE project_id = p.id AND connection_type = 'artifact_store'
    ORDER BY updated_at DESC, id DESC LIMIT 1
) artifact_store ON true`

func projectRecordDest(record *Record) []any {
	return []any{&record.ID, &record.Key, &record.Name, &record.Kind, &record.Adapter,
		&record.WorkflowProfile, &record.DefaultBranch, &record.RootPath,
		&record.ArtifactBackend, &record.ArtifactRoot}
}

func (s Store) ListAuditEventCollection(ctx context.Context, options AuditEventPageOptions) (AuditEventPage, error) {
	if options.Limit <= 0 {
		options.Limit = 50
	}
	if options.Limit > 200 {
		options.Limit = 200
	}
	cursor, err := decodeResourceCursor(strings.TrimSpace(options.Cursor))
	if err != nil {
		return AuditEventPage{}, err
	}
	q := resourceQuery{}
	if options.ProjectID > 0 {
		q.add("project_id = $%d", options.ProjectID)
	}
	if options.ActorID > 0 {
		q.add("actor_id = $%d", options.ActorID)
	}
	if value := strings.TrimSpace(options.Action); value != "" {
		q.add("action = $%d", value)
	}
	if value := strings.TrimSpace(options.Decision); value != "" {
		q.add("decision = $%d", value)
	}
	if value := strings.TrimSpace(options.ResourceType); value != "" {
		q.add("resource_type = $%d", value)
	}
	if value := strings.TrimSpace(options.Resource); value != "" {
		q.add("resource = $%d", value)
	}
	if options.From != nil {
		q.add("created_at >= $%d", options.From.UTC())
	}
	if options.To != nil {
		q.add("created_at <= $%d", options.To.UTC())
	}
	addResourceCursor(&q, "created_at", cursor)
	q.args = append(q.args, options.Limit+1)
	rows, err := s.pool.Query(ctx, `SELECT id, COALESCE(project_id, 0), COALESCE(actor_id, 0), action,
       COALESCE(capability, ''), COALESCE(resource_type, ''), COALESCE(resource, ''),
       decision, COALESCE(reason, ''), metadata, created_at
FROM audit_events`+q.where()+fmt.Sprintf(" ORDER BY created_at DESC, id DESC LIMIT $%d", len(q.args)), q.args...)
	if err != nil {
		return AuditEventPage{}, fmt.Errorf("list audit event collection: %w", err)
	}
	defer rows.Close()
	items := []AuditEventRecord{}
	for rows.Next() {
		var item AuditEventRecord
		var raw []byte
		if err := rows.Scan(&item.ID, &item.ProjectID, &item.ActorID, &item.Action, &item.Capability,
			&item.ResourceType, &item.Resource, &item.Decision, &item.Reason, &raw, &item.CreatedAt); err != nil {
			return AuditEventPage{}, fmt.Errorf("scan audit event collection: %w", err)
		}
		if err := json.Unmarshal(raw, &item.Metadata); err != nil {
			return AuditEventPage{}, fmt.Errorf("parse audit event metadata: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return AuditEventPage{}, fmt.Errorf("iterate audit event collection: %w", err)
	}
	page := AuditEventPage{Items: items}
	if len(page.Items) > options.Limit {
		page.Items = page.Items[:options.Limit]
		last := page.Items[len(page.Items)-1]
		page.NextCursor = encodeResourceCursor(last.CreatedAt, last.ID)
	}
	return page, nil
}

func (s Store) ListWorkflowCollection(ctx context.Context, options WorkflowPageOptions) (WorkflowCollectionPage, error) {
	options.ResourcePageOptions = normalizeResourcePageOptions(options.ResourcePageOptions)
	cursor, err := decodeResourceCursor(options.Cursor)
	if err != nil {
		return WorkflowCollectionPage{}, err
	}
	q := resourceQuery{conditions: []string{"p.archived_at IS NULL"}}
	if options.ProjectKey != "" {
		q.add("p.project_key = $%d", options.ProjectKey)
	}
	if options.Status != "" {
		q.add("wv.lifecycle_status = $%d", options.Status)
	}
	if options.Kind != "" {
		q.add("wv.version_kind = $%d", options.Kind)
	}
	if options.ImportMode != "" {
		q.add("wv.import_mode = $%d", strings.TrimSpace(options.ImportMode))
	}
	if !cursor.Time.IsZero() {
		q.args = append(q.args, cursor.Time, cursor.ID)
		q.conditions = append(q.conditions, fmt.Sprintf("(wv.updated_at, wv.id) < ($%d, $%d)", len(q.args)-1, len(q.args)))
	}
	q.args = append(q.args, options.Limit+1)
	rows, err := s.pool.Query(ctx, `SELECT `+projectCollectionColumns+`,
    wv.id, wv.project_id, wv.display_label, wv.version_kind, wv.lifecycle_status,
    COALESCE(wv.source_path, ''), COALESCE(wv.source_hash, ''), wv.import_mode,
    wv.immutable, wv.status_summary, wv.created_at, wv.updated_at, wv.imported_at
FROM workflow_versions wv JOIN projects p ON p.id = wv.project_id `+projectCollectionJoins+
		q.where()+fmt.Sprintf(" ORDER BY wv.updated_at DESC, wv.id DESC LIMIT $%d", len(q.args)), q.args...)
	if err != nil {
		return WorkflowCollectionPage{}, fmt.Errorf("list workflow collection: %w", err)
	}
	defer rows.Close()
	items := []WorkflowCollectionItem{}
	for rows.Next() {
		var item WorkflowCollectionItem
		var statusRaw []byte
		var imported sql.NullTime
		dest := projectRecordDest(&item.Project)
		dest = append(dest, &item.Workflow.ID, &item.Workflow.ProjectID, &item.Workflow.DisplayLabel,
			&item.Workflow.VersionKind, &item.Workflow.LifecycleStatus, &item.Workflow.SourcePath,
			&item.Workflow.SourceHash, &item.Workflow.ImportMode, &item.Workflow.Immutable, &statusRaw,
			&item.Workflow.CreatedAt, &item.Workflow.UpdatedAt, &imported)
		if err := rows.Scan(dest...); err != nil {
			return WorkflowCollectionPage{}, fmt.Errorf("scan workflow collection: %w", err)
		}
		if err := json.Unmarshal(statusRaw, &item.Workflow.StatusSummary); err != nil {
			return WorkflowCollectionPage{}, err
		}
		if imported.Valid {
			item.Workflow.ImportedAt = &imported.Time
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return WorkflowCollectionPage{}, fmt.Errorf("iterate workflow collection: %w", err)
	}
	page := WorkflowCollectionPage{Items: items}
	if len(page.Items) > options.Limit {
		page.Items = page.Items[:options.Limit]
		last := page.Items[len(page.Items)-1].Workflow
		page.NextCursor = encodeResourceCursor(last.UpdatedAt, last.ID)
	}
	return page, nil
}

func (s Store) ListRunCollection(ctx context.Context, options RunPageOptions) (RunCollectionPage, error) {
	options.ResourcePageOptions = normalizeResourcePageOptions(options.ResourcePageOptions)
	cursor, err := decodeResourceCursor(options.Cursor)
	if err != nil {
		return RunCollectionPage{}, err
	}
	q := resourceQuery{conditions: []string{"p.archived_at IS NULL"}}
	if options.ProjectKey != "" {
		q.add("p.project_key = $%d", options.ProjectKey)
	}
	if options.Status != "" {
		q.add("r.status = $%d", options.Status)
	}
	if options.Kind != "" {
		q.add("COALESCE(r.run_kind, '') = $%d", options.Kind)
	}
	if options.Type != "" {
		q.add("r.run_type = $%d", options.Type)
	}
	if options.DryRun != nil {
		q.add("r.dry_run = $%d", *options.DryRun)
	}
	if !cursor.Time.IsZero() {
		q.args = append(q.args, cursor.Time, cursor.ID)
		q.conditions = append(q.conditions, fmt.Sprintf("(r.started_at, r.id) < ($%d, $%d)", len(q.args)-1, len(q.args)))
	}
	q.args = append(q.args, options.Limit+1)
	rows, err := s.pool.Query(ctx, `SELECT `+projectCollectionColumns+`,
    COALESCE(wv.id, 0), COALESCE(wv.project_id, 0), COALESCE(wv.display_label, ''),
    COALESCE(wv.version_kind, ''), COALESCE(wv.lifecycle_status, ''), COALESCE(wv.source_path, ''),
    COALESCE(wv.source_hash, ''), COALESCE(wv.import_mode, ''), COALESCE(wv.immutable, false),
    COALESCE(wv.status_summary, '{}'::jsonb), wv.created_at, wv.updated_at, wv.imported_at,
    r.id, COALESCE(r.project_id, 0), COALESCE(r.workflow_version_id, 0), r.run_type,
    COALESCE(r.run_kind, ''), r.status, r.risk_level, r.risk_policy, r.dry_run,
    r.summary, r.metadata, r.started_at, r.finished_at
FROM runs r JOIN projects p ON p.id = r.project_id
LEFT JOIN workflow_versions wv ON wv.id = r.workflow_version_id `+projectCollectionJoins+
		q.where()+fmt.Sprintf(" ORDER BY r.started_at DESC, r.id DESC LIMIT $%d", len(q.args)), q.args...)
	if err != nil {
		return RunCollectionPage{}, fmt.Errorf("list run collection: %w", err)
	}
	defer rows.Close()
	items := []RunCollectionItem{}
	for rows.Next() {
		var item RunCollectionItem
		var workflowStatus, runSummary, runMetadata []byte
		var wCreated, wUpdated, wImported sql.NullTime
		var runFinished sql.NullTime
		dest := projectRecordDest(&item.Project)
		dest = append(dest, &item.Workflow.ID, &item.Workflow.ProjectID, &item.Workflow.DisplayLabel,
			&item.Workflow.VersionKind, &item.Workflow.LifecycleStatus, &item.Workflow.SourcePath,
			&item.Workflow.SourceHash, &item.Workflow.ImportMode, &item.Workflow.Immutable, &workflowStatus,
			&wCreated, &wUpdated, &wImported, &item.Run.ID, &item.Run.ProjectID,
			&item.Run.WorkflowVersionID, &item.Run.RunType, &item.Run.RunKind, &item.Run.Status,
			&item.Run.RiskLevel, &item.Run.RiskPolicy, &item.Run.DryRun, &runSummary, &runMetadata,
			&item.Run.StartedAt, &runFinished)
		if err := rows.Scan(dest...); err != nil {
			return RunCollectionPage{}, fmt.Errorf("scan run collection: %w", err)
		}
		_ = json.Unmarshal(workflowStatus, &item.Workflow.StatusSummary)
		if wCreated.Valid {
			item.Workflow.CreatedAt = wCreated.Time
		}
		if wUpdated.Valid {
			item.Workflow.UpdatedAt = wUpdated.Time
		}
		if wImported.Valid {
			item.Workflow.ImportedAt = &wImported.Time
		}
		if err := json.Unmarshal(runSummary, &item.Run.Summary); err != nil {
			return RunCollectionPage{}, err
		}
		if err := json.Unmarshal(runMetadata, &item.Run.Metadata); err != nil {
			return RunCollectionPage{}, err
		}
		if runFinished.Valid {
			item.Run.FinishedAt = &runFinished.Time
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return RunCollectionPage{}, fmt.Errorf("iterate run collection: %w", err)
	}
	page := RunCollectionPage{Items: items}
	if len(page.Items) > options.Limit {
		page.Items = page.Items[:options.Limit]
		last := page.Items[len(page.Items)-1].Run
		page.NextCursor = encodeResourceCursor(last.StartedAt, last.ID)
	}
	return page, nil
}

func (s Store) ListWorkerCollection(ctx context.Context, options ResourcePageOptions) (WorkerCollectionPage, error) {
	options = normalizeResourcePageOptions(options)
	cursor, err := decodeResourceCursor(options.Cursor)
	if err != nil {
		return WorkerCollectionPage{}, err
	}
	q := resourceQuery{conditions: []string{"p.archived_at IS NULL"}}
	if options.ProjectKey != "" {
		q.add("p.project_key = $%d", options.ProjectKey)
	}
	if options.Key != "" {
		q.add("w.worker_key = $%d", options.Key)
	}
	if options.Status != "" {
		q.add("w.status = $%d", options.Status)
	}
	if options.Type != "" {
		q.add("w.worker_type = $%d", options.Type)
	}
	if options.Kind != "" {
		q.add("w.capabilities ? $%d", options.Kind)
	}
	if !cursor.Time.IsZero() {
		q.args = append(q.args, cursor.Time, cursor.ID)
		q.conditions = append(q.conditions, fmt.Sprintf("(w.updated_at, w.id) < ($%d, $%d)", len(q.args)-1, len(q.args)))
	}
	q.args = append(q.args, options.Limit+1)
	rows, err := s.pool.Query(ctx, `SELECT `+projectCollectionColumns+`,
    w.id, w.project_id, COALESCE(w.actor_id, 0), w.worker_key, w.worker_type, w.status,
    COALESCE(w.hostname, ''), COALESCE(w.pid, 0), w.capabilities, w.metadata,
    w.registered_at, w.last_heartbeat_at, w.heartbeat_interval_seconds,
    w.lease_timeout_seconds, w.updated_at
FROM workers w JOIN projects p ON p.id = w.project_id `+projectCollectionJoins+
		q.where()+fmt.Sprintf(" ORDER BY w.updated_at DESC, w.id DESC LIMIT $%d", len(q.args)), q.args...)
	if err != nil {
		return WorkerCollectionPage{}, fmt.Errorf("list worker collection: %w", err)
	}
	defer rows.Close()
	items := []WorkerCollectionItem{}
	for rows.Next() {
		var item WorkerCollectionItem
		var capabilitiesRaw, metadataRaw []byte
		var lastHeartbeat sql.NullTime
		dest := projectRecordDest(&item.Project)
		dest = append(dest, &item.Worker.ID, &item.Worker.ProjectID, &item.Worker.ActorID,
			&item.Worker.WorkerKey, &item.Worker.WorkerType, &item.Worker.Status,
			&item.Worker.Hostname, &item.Worker.PID, &capabilitiesRaw, &metadataRaw,
			&item.Worker.RegisteredAt, &lastHeartbeat, &item.Worker.HeartbeatIntervalSeconds,
			&item.Worker.LeaseTimeoutSeconds, &item.Worker.UpdatedAt)
		if err := rows.Scan(dest...); err != nil {
			return WorkerCollectionPage{}, fmt.Errorf("scan worker collection: %w", err)
		}
		if err := json.Unmarshal(capabilitiesRaw, &item.Worker.Capabilities); err != nil {
			return WorkerCollectionPage{}, err
		}
		if err := json.Unmarshal(metadataRaw, &item.Worker.Metadata); err != nil {
			return WorkerCollectionPage{}, err
		}
		if lastHeartbeat.Valid {
			item.Worker.LastHeartbeatAt = &lastHeartbeat.Time
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return WorkerCollectionPage{}, fmt.Errorf("iterate worker collection: %w", err)
	}
	page := WorkerCollectionPage{Items: items}
	if len(page.Items) > options.Limit {
		page.Items = page.Items[:options.Limit]
		last := page.Items[len(page.Items)-1].Worker
		page.NextCursor = encodeResourceCursor(last.UpdatedAt, last.ID)
	}
	return page, nil
}

func (s Store) ListArtifactCollection(ctx context.Context, options ArtifactPageOptions) (ArtifactCollectionPage, error) {
	options.ResourcePageOptions = normalizeResourcePageOptions(options.ResourcePageOptions)
	cursor, err := decodeResourceCursor(options.Cursor)
	if err != nil {
		return ArtifactCollectionPage{}, err
	}
	q := resourceQuery{conditions: []string{"p.archived_at IS NULL"}}
	if options.ProjectKey != "" {
		q.add("p.project_key = $%d", options.ProjectKey)
	}
	if options.Type != "" {
		q.add("a.artifact_type = $%d", options.Type)
	}
	if options.StorageBackend != "" {
		q.add("a.storage_backend = $%d", strings.TrimSpace(options.StorageBackend))
	}
	if options.SHA256 != "" {
		q.add("a.sha256 = $%d", strings.TrimSpace(options.SHA256))
	}
	if options.RunID > 0 {
		q.add("a.run_id = $%d", options.RunID)
	}
	if options.WorkflowVersionID > 0 {
		q.add("a.workflow_version_id = $%d", options.WorkflowVersionID)
	}
	if !cursor.Time.IsZero() {
		q.args = append(q.args, cursor.Time, cursor.ID)
		q.conditions = append(q.conditions, fmt.Sprintf("(a.created_at, a.id) < ($%d, $%d)", len(q.args)-1, len(q.args)))
	}
	q.args = append(q.args, options.Limit+1)
	rows, err := s.pool.Query(ctx, `SELECT `+projectCollectionColumns+`,
    a.id, a.project_id, COALESCE(a.workflow_version_id, 0), COALESCE(a.run_id, 0),
    COALESCE(a.workflow_item_id, 0), a.artifact_type, a.storage_backend, a.uri,
    COALESCE(a.source_path, ''), COALESCE(a.sha256, ''), COALESCE(a.size_bytes, 0),
    COALESCE(a.content_type, ''), a.metadata, a.created_at
FROM artifacts a JOIN projects p ON p.id = a.project_id `+projectCollectionJoins+
		q.where()+fmt.Sprintf(" ORDER BY a.created_at DESC, a.id DESC LIMIT $%d", len(q.args)), q.args...)
	if err != nil {
		return ArtifactCollectionPage{}, fmt.Errorf("list artifact collection: %w", err)
	}
	defer rows.Close()
	items := []ArtifactCollectionItem{}
	for rows.Next() {
		var item ArtifactCollectionItem
		var metadataRaw []byte
		dest := projectRecordDest(&item.Project)
		dest = append(dest, &item.Artifact.ID, &item.Artifact.ProjectID,
			&item.Artifact.WorkflowVersionID, &item.Artifact.RunID, &item.Artifact.WorkflowItemID,
			&item.Artifact.ArtifactType, &item.Artifact.StorageBackend, &item.Artifact.URI,
			&item.Artifact.SourcePath, &item.Artifact.SHA256, &item.Artifact.SizeBytes,
			&item.Artifact.ContentType, &metadataRaw, &item.Artifact.CreatedAt)
		if err := rows.Scan(dest...); err != nil {
			return ArtifactCollectionPage{}, fmt.Errorf("scan artifact collection: %w", err)
		}
		if err := json.Unmarshal(metadataRaw, &item.Artifact.Metadata); err != nil {
			return ArtifactCollectionPage{}, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return ArtifactCollectionPage{}, fmt.Errorf("iterate artifact collection: %w", err)
	}
	page := ArtifactCollectionPage{Items: items}
	if len(page.Items) > options.Limit {
		page.Items = page.Items[:options.Limit]
		last := page.Items[len(page.Items)-1].Artifact
		page.NextCursor = encodeResourceCursor(last.CreatedAt, last.ID)
	}
	return page, nil
}
