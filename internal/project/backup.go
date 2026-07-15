package project

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"
)

type BackupManifestOptions struct {
	GeneratedAt time.Time
	ProjectID   int64
	ProjectKey  string
}

type BackupTableCount struct {
	Table string
	Rows  int64
}

type BackupArtifactSummary struct {
	ID                int64
	ProjectID         int64
	WorkflowVersionID int64
	WorkflowItemID    int64
	ArtifactType      string
	StorageBackend    string
	URI               string
	SourcePath        string
	SHA256            string
	SizeBytes         int64
	ContentType       string
	CreatedAt         time.Time
}

type BackupProjectManifest struct {
	Project       Record
	Inventory     ImportInventory
	Artifacts     []BackupArtifactSummary
	ArtifactCount int64
}

type BackupManifest struct {
	Status           string
	Mode             string
	Scope            string
	ProjectKey       string
	SchemaVersion    int
	GeneratedAt      time.Time
	TableCounts      []BackupTableCount
	Projects         []BackupProjectManifest
	Capabilities     []string
	ForbiddenActions []string
	ManifestHash     string
}

func (s Store) BackupManifest(ctx context.Context, options BackupManifestOptions) (BackupManifest, error) {
	options = normalizeBackupManifestOptions(options)
	tableCounts, err := s.backupTableCounts(ctx)
	if err != nil {
		return BackupManifest{}, err
	}
	projects, err := s.backupProjectManifests(ctx, options)
	if err != nil {
		return BackupManifest{}, err
	}
	scope := "platform"
	if options.ProjectID > 0 || options.ProjectKey != "" {
		scope = "project"
	}
	projectKey := options.ProjectKey
	if scope == "project" && projectKey == "" && len(projects) == 1 {
		projectKey = projects[0].Project.Key
	}
	manifest := BackupManifest{
		Status:        "ready",
		Mode:          "read_only_manifest",
		Scope:         scope,
		ProjectKey:    projectKey,
		SchemaVersion: 1,
		GeneratedAt:   options.GeneratedAt,
		TableCounts:   tableCounts,
		Projects:      projects,
		Capabilities: []string{
			"export_postgres_metadata",
			"export_artifact_metadata",
			"verify_manifest_hash",
		},
		ForbiddenActions: []string{
			"read_artifact_contents",
			"write_project_files",
			"restore_database",
			"delete_existing_state",
			"resolve_secrets",
		},
	}
	hash, err := backupManifestHash(manifest)
	if err != nil {
		return BackupManifest{}, err
	}
	manifest.ManifestHash = hash
	return manifest, nil
}

func normalizeBackupManifestOptions(options BackupManifestOptions) BackupManifestOptions {
	options.ProjectKey = strings.TrimSpace(options.ProjectKey)
	if options.GeneratedAt.IsZero() {
		options.GeneratedAt = time.Now().UTC()
	}
	return options
}

func (s Store) backupTableCounts(ctx context.Context) ([]BackupTableCount, error) {
	tables := []string{
		"actors",
		"projects",
		"project_connections",
		"project_permissions",
		"project_configs",
		"workflow_versions",
		"workflow_items",
		"workflow_item_links",
		"gate_results",
		"workflow_transition_previews",
		"approval_records",
		"residuals",
		"runs",
		"run_tasks",
		"run_attempts",
		"workers",
		"worker_heartbeats",
		"leases",
		"artifacts",
		"artifact_locations",
		"artifact_snapshots",
		"status_projections",
		"events",
		"audit_events",
		"users",
		"teams",
		"memberships",
		"adapters",
		"workflow_profiles",
		"secret_refs",
		"engine_profiles",
		"api_tokens",
		"webhooks",
		"command_requests",
		"project_scheduling_policies",
	}
	counts := make([]BackupTableCount, 0, len(tables))
	for _, table := range tables {
		var rows int64
		if err := s.pool.QueryRow(ctx, fmt.Sprintf("SELECT COUNT(*) FROM %s", table)).Scan(&rows); err != nil {
			return nil, fmt.Errorf("count backup table %s: %w", table, err)
		}
		counts = append(counts, BackupTableCount{Table: table, Rows: rows})
	}
	return counts, nil
}

func (s Store) backupProjectManifests(ctx context.Context, options BackupManifestOptions) ([]BackupProjectManifest, error) {
	options = normalizeBackupManifestOptions(options)
	records, err := s.backupProjectRecords(ctx)
	if err != nil {
		return nil, err
	}
	projects := make([]BackupProjectManifest, 0, len(records))
	for _, record := range records {
		if options.ProjectID > 0 && record.ID != options.ProjectID {
			continue
		}
		if options.ProjectKey != "" && record.Key != options.ProjectKey {
			continue
		}
		manifest, err := s.backupProjectManifest(ctx, record)
		if err != nil {
			return nil, err
		}
		projects = append(projects, manifest)
	}
	return projects, nil
}

func (s Store) backupProjectManifest(ctx context.Context, record Record) (BackupProjectManifest, error) {
	inventory, err := s.ImportInventory(ctx, record.ID)
	if err != nil {
		return BackupProjectManifest{}, err
	}
	artifacts, artifactCount, err := s.backupArtifactSummaries(ctx, record.ID)
	if err != nil {
		return BackupProjectManifest{}, err
	}
	return BackupProjectManifest{
		Project:       record,
		Inventory:     inventory,
		Artifacts:     artifacts,
		ArtifactCount: artifactCount,
	}, nil
}

func (s Store) backupProjectRecords(ctx context.Context) ([]Record, error) {
	rows, err := s.pool.Query(ctx, `
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
ORDER BY p.project_key`)
	if err != nil {
		return nil, fmt.Errorf("list backup projects: %w", err)
	}
	defer rows.Close()

	records := []Record{}
	for rows.Next() {
		var record Record
		if err := rows.Scan(
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
		); err != nil {
			return nil, fmt.Errorf("scan backup project: %w", err)
		}
		records = append(records, record)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate backup projects: %w", err)
	}
	return records, nil
}

func (s Store) backupArtifactSummaries(ctx context.Context, projectID int64) ([]BackupArtifactSummary, int64, error) {
	var total int64
	if err := s.pool.QueryRow(ctx, `SELECT COUNT(*) FROM artifacts WHERE project_id = $1`, projectID).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count project artifacts: %w", err)
	}
	rows, err := s.pool.Query(ctx, `
SELECT id, project_id, COALESCE(workflow_version_id, 0), COALESCE(workflow_item_id, 0),
       artifact_type, storage_backend, uri, COALESCE(source_path, ''), COALESCE(sha256, ''),
       COALESCE(size_bytes, 0), COALESCE(content_type, ''), created_at
FROM artifacts
WHERE project_id = $1
ORDER BY created_at DESC, id DESC`,
		projectID,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("list backup artifacts: %w", err)
	}
	defer rows.Close()

	artifacts := []BackupArtifactSummary{}
	for rows.Next() {
		var artifact BackupArtifactSummary
		if err := rows.Scan(
			&artifact.ID,
			&artifact.ProjectID,
			&artifact.WorkflowVersionID,
			&artifact.WorkflowItemID,
			&artifact.ArtifactType,
			&artifact.StorageBackend,
			&artifact.URI,
			&artifact.SourcePath,
			&artifact.SHA256,
			&artifact.SizeBytes,
			&artifact.ContentType,
			&artifact.CreatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("scan backup artifact: %w", err)
		}
		artifacts = append(artifacts, artifact)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate backup artifacts: %w", err)
	}
	return artifacts, total, nil
}

func backupManifestHash(manifest BackupManifest) (string, error) {
	shape := struct {
		SchemaVersion int
		TableCounts   []BackupTableCount
		Projects      []backupProjectHashShape
	}{
		SchemaVersion: manifest.SchemaVersion,
		TableCounts:   append([]BackupTableCount(nil), manifest.TableCounts...),
		Projects:      make([]backupProjectHashShape, 0, len(manifest.Projects)),
	}
	sort.Slice(shape.TableCounts, func(i, j int) bool {
		return shape.TableCounts[i].Table < shape.TableCounts[j].Table
	})
	for _, project := range manifest.Projects {
		artifacts := make([]BackupArtifactSummary, len(project.Artifacts))
		copy(artifacts, project.Artifacts)
		sort.Slice(artifacts, func(i, j int) bool {
			return artifacts[i].ID < artifacts[j].ID
		})
		shape.Projects = append(shape.Projects, backupProjectHashShape{
			Key:           project.Project.Key,
			Inventory:     project.Inventory,
			ArtifactCount: project.ArtifactCount,
			Artifacts:     artifacts,
		})
	}
	sort.Slice(shape.Projects, func(i, j int) bool {
		return shape.Projects[i].Key < shape.Projects[j].Key
	})
	raw, err := json.Marshal(shape)
	if err != nil {
		return "", fmt.Errorf("marshal backup manifest hash shape: %w", err)
	}
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:]), nil
}

type backupProjectHashShape struct {
	Key           string
	Inventory     ImportInventory
	ArtifactCount int64
	Artifacts     []BackupArtifactSummary
}
