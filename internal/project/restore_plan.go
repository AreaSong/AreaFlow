package project

import (
	"context"
	"strings"
	"time"
)

type RestorePlanOptions struct {
	GeneratedAt time.Time
	ProjectID   int64
	ProjectKey  string
}

type RestorePlanItem struct {
	Key      string
	Category string
	Status   string
	Message  string
	Metadata map[string]any
}

type RestorePlan struct {
	Status           string
	Mode             string
	Scope            string
	ProjectKey       string
	SchemaVersion    int
	ManifestHash     string
	Projects         []Record
	Items            []RestorePlanItem
	Capabilities     []string
	ForbiddenActions []string
	GeneratedAt      time.Time
}

func (s Store) RestorePlan(ctx context.Context, options RestorePlanOptions) (RestorePlan, error) {
	options = normalizeRestorePlanOptions(options)
	manifest, err := s.BackupManifest(ctx, BackupManifestOptions{GeneratedAt: options.GeneratedAt, ProjectID: options.ProjectID, ProjectKey: options.ProjectKey})
	if err != nil {
		return RestorePlan{}, err
	}
	plan := BuildRestorePlan(manifest, options)
	for _, projectRecord := range plan.Projects {
		integrity, err := s.ArtifactIntegrity(ctx, projectRecord, ArtifactIntegrityOptions{GeneratedAt: options.GeneratedAt})
		if err != nil {
			return RestorePlan{}, err
		}
		plan.Items = append(plan.Items, restorePlanItemFromArtifactIntegrity(integrity))
		if worseRestorePlanStatus(plan.Items[len(plan.Items)-1].Status, plan.Status) {
			plan.Status = plan.Items[len(plan.Items)-1].Status
		}
	}
	return plan, nil
}

func normalizeRestorePlanOptions(options RestorePlanOptions) RestorePlanOptions {
	options.ProjectKey = strings.TrimSpace(options.ProjectKey)
	if options.GeneratedAt.IsZero() {
		options.GeneratedAt = time.Now().UTC()
	}
	return options
}

func BuildRestorePlan(manifest BackupManifest, options RestorePlanOptions) RestorePlan {
	options = normalizeRestorePlanOptions(options)
	projects := filterRestorePlanProjects(manifest.Projects, options)
	scope := "platform"
	if options.ProjectID > 0 || options.ProjectKey != "" {
		scope = "project"
	}
	projectKey := options.ProjectKey
	if scope == "project" && projectKey == "" && len(projects) == 1 {
		projectKey = projects[0].Project.Key
	}
	plan := RestorePlan{
		Status:        "ready",
		Mode:          "read_only_restore_plan",
		Scope:         scope,
		ProjectKey:    projectKey,
		SchemaVersion: manifest.SchemaVersion,
		ManifestHash:  manifest.ManifestHash,
		Projects:      make([]Record, 0, len(projects)),
		Items:         []RestorePlanItem{},
		Capabilities: []string{
			"read_backup_manifest",
			"verify_manifest_hash",
			"check_artifact_integrity",
			"generate_restore_plan",
		},
		ForbiddenActions: []string{
			"restore_database",
			"write_project_files",
			"write_artifact_store",
			"delete_existing_state",
			"resolve_secrets",
			"apply_restore",
		},
		GeneratedAt: options.GeneratedAt,
	}
	add := func(item RestorePlanItem) {
		plan.Items = append(plan.Items, item)
		if worseRestorePlanStatus(item.Status, plan.Status) {
			plan.Status = item.Status
		}
	}
	scopedManifest := manifest
	scopedManifest.Projects = projects
	for _, projectManifest := range scopedManifest.Projects {
		plan.Projects = append(plan.Projects, projectManifest.Project)
	}
	add(checkRestoreManifest(scopedManifest))
	add(checkRestoreProjectInventory(scopedManifest))
	add(checkRestoreArtifactInventory(scopedManifest))
	add(checkRestoreForbiddenActions(scopedManifest))
	return plan
}

func filterRestorePlanProjects(projects []BackupProjectManifest, options RestorePlanOptions) []BackupProjectManifest {
	options = normalizeRestorePlanOptions(options)
	if options.ProjectID == 0 && options.ProjectKey == "" {
		return append([]BackupProjectManifest(nil), projects...)
	}
	filtered := []BackupProjectManifest{}
	for _, projectManifest := range projects {
		record := projectManifest.Project
		if options.ProjectID > 0 && record.ID != options.ProjectID {
			continue
		}
		if options.ProjectKey != "" && record.Key != options.ProjectKey {
			continue
		}
		filtered = append(filtered, projectManifest)
	}
	return filtered
}

func checkRestoreManifest(manifest BackupManifest) RestorePlanItem {
	missing := []string{}
	if manifest.SchemaVersion <= 0 {
		missing = append(missing, "schema_version")
	}
	if manifest.ManifestHash == "" {
		missing = append(missing, "manifest_hash")
	}
	if len(missing) > 0 {
		return RestorePlanItem{
			Key:      "manifest_shape",
			Category: "manifest",
			Status:   "blocked",
			Message:  "backup manifest is missing required restore metadata",
			Metadata: map[string]any{"missing": missing},
		}
	}
	return RestorePlanItem{
		Key:      "manifest_shape",
		Category: "manifest",
		Status:   "ready",
		Message:  "backup manifest has schema version and stable hash",
		Metadata: map[string]any{
			"schema_version": manifest.SchemaVersion,
			"manifest_hash":  manifest.ManifestHash,
		},
	}
}

func checkRestoreProjectInventory(manifest BackupManifest) RestorePlanItem {
	if len(manifest.Projects) == 0 {
		return RestorePlanItem{
			Key:      "project_inventory",
			Category: "project",
			Status:   "blocked",
			Message:  "backup manifest contains no projects",
		}
	}
	keys := make([]string, 0, len(manifest.Projects))
	for _, projectManifest := range manifest.Projects {
		keys = append(keys, projectManifest.Project.Key)
	}
	return RestorePlanItem{
		Key:      "project_inventory",
		Category: "project",
		Status:   "ready",
		Message:  "backup manifest contains project inventory",
		Metadata: map[string]any{"project_keys": keys, "projects": len(keys)},
	}
}

func checkRestoreArtifactInventory(manifest BackupManifest) RestorePlanItem {
	total := int64(0)
	local := int64(0)
	referenced := int64(0)
	for _, projectManifest := range manifest.Projects {
		for _, artifact := range projectManifest.Artifacts {
			total++
			switch artifact.StorageBackend {
			case "local":
				local++
			case "external_project", "project_reference":
				referenced++
			}
		}
	}
	status := "ready"
	message := "backup manifest contains artifact metadata"
	if total == 0 {
		status = "blocked"
		message = "backup manifest contains no artifact metadata"
	} else if referenced > 0 {
		status = "needs_attention"
		message = "some artifact originals remain as project references"
	}
	return RestorePlanItem{
		Key:      "artifact_inventory",
		Category: "artifact",
		Status:   status,
		Message:  message,
		Metadata: map[string]any{
			"total_artifacts":      total,
			"local_artifacts":      local,
			"referenced_artifacts": referenced,
		},
	}
}

func checkRestoreForbiddenActions(manifest BackupManifest) RestorePlanItem {
	required := []string{
		"restore_database",
		"write_project_files",
		"delete_existing_state",
		"resolve_secrets",
	}
	missing := []string{}
	for _, action := range required {
		if !containsRestoreString(manifest.ForbiddenActions, action) {
			missing = append(missing, action)
		}
	}
	if len(missing) > 0 {
		return RestorePlanItem{
			Key:      "dry_run_guardrails",
			Category: "safety",
			Status:   "blocked",
			Message:  "restore dry-run guardrails are incomplete",
			Metadata: map[string]any{"missing_forbidden_actions": missing},
		}
	}
	return RestorePlanItem{
		Key:      "dry_run_guardrails",
		Category: "safety",
		Status:   "ready",
		Message:  "restore plan is read-only and forbids apply actions",
		Metadata: map[string]any{"forbidden_actions": manifest.ForbiddenActions},
	}
}

func restorePlanItemFromArtifactIntegrity(report ArtifactIntegrityReport) RestorePlanItem {
	status := "ready"
	message := "artifact integrity is ready for restore planning"
	switch report.Status {
	case "fail":
		status = "blocked"
		message = "artifact integrity has failures that block restore planning"
	case "warn":
		status = "needs_attention"
		message = "artifact integrity has warnings or skipped references"
	}
	return RestorePlanItem{
		Key:      "artifact_integrity:" + report.Project.Key,
		Category: "artifact",
		Status:   status,
		Message:  message,
		Metadata: map[string]any{
			"project_key":       report.Project.Key,
			"checked_artifacts": report.CheckedArtifacts,
			"passed_artifacts":  report.PassedArtifacts,
			"warn_artifacts":    report.WarnArtifacts,
			"failed_artifacts":  report.FailedArtifacts,
			"skipped_artifacts": report.SkippedArtifacts,
			"integrity_status":  report.Status,
		},
	}
}

func worseRestorePlanStatus(candidate string, current string) bool {
	return restorePlanStatusRank(candidate) > restorePlanStatusRank(current)
}

func restorePlanStatusRank(status string) int {
	switch status {
	case "blocked":
		return 3
	case "needs_attention":
		return 2
	case "ready":
		return 1
	default:
		return 0
	}
}

func containsRestoreString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
