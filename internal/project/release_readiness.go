package project

import (
	"context"
	"fmt"
	"time"
)

type ReleaseReadinessOptions struct {
	GeneratedAt time.Time
	ProjectID   int64
	ProjectKey  string
}

type ReleaseReadinessItem struct {
	Key      string
	Category string
	Status   string
	Message  string
	Metadata map[string]any
}

type ReleaseReadinessProject struct {
	Project             Record
	Permission          PermissionPolicyDoctor
	ArtifactIntegrity   ArtifactIntegrityReport
	Conformance         ConformanceReport
	Status              string
	NeedsAttentionItems int
	BlockedItems        int
}

type ReleaseReadiness struct {
	Real100Guardrail
	Status           string
	Mode             string
	Scope            string
	ProjectKey       string
	Backup           BackupManifest
	RestorePlan      RestorePlan
	AuditCoverage    AuditCoverage
	Projects         []ReleaseReadinessProject
	Items            []ReleaseReadinessItem
	Capabilities     []string
	ForbiddenActions []string
	GeneratedAt      time.Time
}

func (s Store) ReleaseReadiness(ctx context.Context, options ReleaseReadinessOptions) (ReleaseReadiness, error) {
	options = normalizeReleaseReadinessOptions(options)
	scope, err := s.resolveReleaseProjectScope(ctx, options.ProjectID, options.ProjectKey)
	if err != nil {
		return ReleaseReadiness{}, err
	}
	options.ProjectID = scope.ProjectID
	options.ProjectKey = scope.ProjectKey
	backup, err := s.BackupManifest(ctx, BackupManifestOptions{GeneratedAt: options.GeneratedAt, ProjectID: options.ProjectID, ProjectKey: options.ProjectKey})
	if err != nil {
		return ReleaseReadiness{}, err
	}
	restorePlan, err := s.RestorePlan(ctx, RestorePlanOptions{GeneratedAt: options.GeneratedAt, ProjectID: options.ProjectID, ProjectKey: options.ProjectKey})
	if err != nil {
		return ReleaseReadiness{}, err
	}
	auditCoverage, err := s.AuditCoverage(ctx, AuditCoverageOptions{GeneratedAt: options.GeneratedAt, ProjectID: options.ProjectID, ProjectKey: options.ProjectKey})
	if err != nil {
		return ReleaseReadiness{}, err
	}
	readiness := BuildReleaseReadiness(backup, restorePlan, auditCoverage, nil, options)
	for _, manifestProject := range backup.Projects {
		record := manifestProject.Project
		permission, err := s.PermissionPolicyDoctor(ctx, record, PermissionPolicyDoctorOptions{GeneratedAt: options.GeneratedAt})
		if err != nil {
			return ReleaseReadiness{}, err
		}
		integrity, err := s.ArtifactIntegrity(ctx, record, ArtifactIntegrityOptions{GeneratedAt: options.GeneratedAt})
		if err != nil {
			return ReleaseReadiness{}, err
		}
		conformance, err := s.ConformanceCheck(ctx, record, ConformanceOptions{GeneratedAt: options.GeneratedAt})
		if err != nil {
			return ReleaseReadiness{}, err
		}
		readiness.addProject(record, permission, integrity, conformance)
	}
	return readiness, nil
}

func normalizeReleaseReadinessOptions(options ReleaseReadinessOptions) ReleaseReadinessOptions {
	options.GeneratedAt, options.ProjectKey = normalizeReleaseScopeFields(options.GeneratedAt, options.ProjectKey)
	return options
}

func BuildReleaseReadiness(backup BackupManifest, restorePlan RestorePlan, auditCoverage AuditCoverage, projects []ReleaseReadinessProject, options ReleaseReadinessOptions) ReleaseReadiness {
	options = normalizeReleaseReadinessOptions(options)
	scope, projectKey := releaseScopeAndProjectKey(options.ProjectID, options.ProjectKey, backup.ProjectKey, restorePlan.ProjectKey, auditCoverage.ProjectKey)
	readiness := ReleaseReadiness{
		Real100Guardrail: ReleasePreviewReal100Guardrail(),
		Status:           "ready",
		Mode:             "read_only_release_readiness",
		Scope:            scope,
		ProjectKey:       projectKey,
		Backup:           backup,
		RestorePlan:      restorePlan,
		AuditCoverage:    auditCoverage,
		Projects:         []ReleaseReadinessProject{},
		Items:            []ReleaseReadinessItem{},
		Capabilities: []string{
			"read_backup_manifest",
			"read_restore_plan",
			"read_audit_coverage",
			"read_permission_policy",
			"read_artifact_integrity",
			"read_adapter_profile_conformance",
			"generate_release_readiness",
		},
		ForbiddenActions: []string{
			"restore_database",
			"write_project_files",
			"write_artifact_store",
			"delete_existing_state",
			"resolve_secrets",
			"execute_commands",
			"apply_cutover",
			"start_worker",
			"create_release_package",
		},
		GeneratedAt: options.GeneratedAt,
	}
	readiness.addItem(releaseItemFromBackup(backup))
	readiness.addItem(releaseItemFromRestorePlan(restorePlan))
	readiness.addItem(releaseItemFromAuditCoverage(auditCoverage))
	for _, project := range projects {
		readiness.addProject(project.Project, project.Permission, project.ArtifactIntegrity, project.Conformance)
	}
	return readiness
}

func (r *ReleaseReadiness) addProject(record Record, permission PermissionPolicyDoctor, integrity ArtifactIntegrityReport, conformance ConformanceReport) {
	project := ReleaseReadinessProject{
		Project:           record,
		Permission:        permission,
		ArtifactIntegrity: integrity,
		Conformance:       conformance,
		Status:            "ready",
	}
	projectItems := []ReleaseReadinessItem{
		releaseItemFromPermission(permission),
		releaseItemFromArtifactIntegrity(integrity),
		releaseItemFromConformance(conformance),
	}
	for _, item := range projectItems {
		if item.Status == "blocked" {
			project.BlockedItems++
		}
		if item.Status == "needs_attention" {
			project.NeedsAttentionItems++
		}
		if worseReleaseReadinessStatus(item.Status, project.Status) {
			project.Status = item.Status
		}
		r.addItem(item)
	}
	r.Projects = append(r.Projects, project)
}

func (r *ReleaseReadiness) addItem(item ReleaseReadinessItem) {
	if item.Metadata == nil {
		item.Metadata = map[string]any{}
	}
	r.Items = append(r.Items, item)
	if worseReleaseReadinessStatus(item.Status, r.Status) {
		r.Status = item.Status
	}
}

func releaseItemFromBackup(manifest BackupManifest) ReleaseReadinessItem {
	status := releaseStatusFromPassFail(manifest.Status)
	message := "backup manifest is ready for release metadata review"
	if status == "blocked" {
		message = "backup manifest blocks release readiness"
	}
	return ReleaseReadinessItem{
		Key:      "backup_manifest",
		Category: "backup",
		Status:   status,
		Message:  message,
		Metadata: map[string]any{
			"backup_status":  manifest.Status,
			"mode":           manifest.Mode,
			"schema_version": manifest.SchemaVersion,
			"manifest_hash":  manifest.ManifestHash,
			"projects":       len(manifest.Projects),
			"table_counts":   len(manifest.TableCounts),
		},
	}
}

func releaseItemFromRestorePlan(plan RestorePlan) ReleaseReadinessItem {
	status := releaseStatusFromPlan(plan.Status)
	message := "restore dry-run plan is ready"
	if status == "needs_attention" {
		message = "restore dry-run plan needs attention before release"
	}
	if status == "blocked" {
		message = "restore dry-run plan blocks release readiness"
	}
	return ReleaseReadinessItem{
		Key:      "restore_plan",
		Category: "restore",
		Status:   status,
		Message:  message,
		Metadata: map[string]any{
			"restore_status":    plan.Status,
			"mode":              plan.Mode,
			"schema_version":    plan.SchemaVersion,
			"manifest_hash":     plan.ManifestHash,
			"items":             len(plan.Items),
			"forbidden_actions": plan.ForbiddenActions,
		},
	}
}

func releaseItemFromAuditCoverage(coverage AuditCoverage) ReleaseReadinessItem {
	status := releaseStatusFromPassWarnFail(coverage.Status)
	message := "audit coverage is ready"
	if status == "needs_attention" {
		message = "audit coverage has gaps that must be accepted or closed before release"
	}
	if status == "blocked" {
		message = "audit coverage blocks release readiness"
	}
	return ReleaseReadinessItem{
		Key:      "audit_coverage",
		Category: "audit",
		Status:   status,
		Message:  message,
		Metadata: map[string]any{
			"audit_status":         coverage.Status,
			"scope":                coverage.Scope,
			"total_audit_events":   coverage.TotalAuditEvents,
			"covered_requirements": coverage.CoveredRequirements,
			"gap_requirements":     coverage.GapRequirements,
		},
	}
}

func releaseItemFromPermission(doctor PermissionPolicyDoctor) ReleaseReadinessItem {
	status := releaseStatusFromPassWarnFail(doctor.Status)
	message := "permission policy doctor is ready"
	if status == "needs_attention" {
		message = "permission policy doctor has warnings that need review"
	}
	if status == "blocked" {
		message = "permission policy doctor blocks release readiness"
	}
	key := "permission_policy"
	if doctor.Project.Key != "" {
		key = fmt.Sprintf("permission_policy:%s", doctor.Project.Key)
	}
	return ReleaseReadinessItem{
		Key:      key,
		Category: "permission",
		Status:   status,
		Message:  message,
		Metadata: map[string]any{
			"project_key":       doctor.Project.Key,
			"permission_status": doctor.Status,
			"mode":              doctor.Mode,
			"checks":            len(doctor.Checks),
		},
	}
}

func releaseItemFromArtifactIntegrity(report ArtifactIntegrityReport) ReleaseReadinessItem {
	status := releaseStatusFromPassWarnFail(report.Status)
	message := "artifact integrity is ready"
	if status == "needs_attention" {
		message = "artifact integrity has warnings or skipped references that need review"
	}
	if status == "blocked" {
		message = "artifact integrity blocks release readiness"
	}
	key := "artifact_integrity"
	if report.Project.Key != "" {
		key = fmt.Sprintf("artifact_integrity:%s", report.Project.Key)
	}
	return ReleaseReadinessItem{
		Key:      key,
		Category: "artifact",
		Status:   status,
		Message:  message,
		Metadata: map[string]any{
			"project_key":        report.Project.Key,
			"integrity_status":   report.Status,
			"checked_artifacts":  report.CheckedArtifacts,
			"passed_artifacts":   report.PassedArtifacts,
			"warn_artifacts":     report.WarnArtifacts,
			"failed_artifacts":   report.FailedArtifacts,
			"skipped_artifacts":  report.SkippedArtifacts,
			"metadata_only_refs": report.SkippedArtifacts,
		},
	}
}

func releaseItemFromConformance(report ConformanceReport) ReleaseReadinessItem {
	status := releaseStatusFromPassWarnFail(report.Status)
	message := "adapter/profile conformance is ready"
	if status == "needs_attention" {
		message = "adapter/profile conformance has warnings that need review"
	}
	if status == "blocked" {
		message = "adapter/profile conformance blocks release readiness"
	}
	key := "adapter_profile_conformance"
	if report.Project.Key != "" {
		key = fmt.Sprintf("adapter_profile_conformance:%s", report.Project.Key)
	}
	return ReleaseReadinessItem{
		Key:      key,
		Category: "conformance",
		Status:   status,
		Message:  message,
		Metadata: map[string]any{
			"project_key":        report.Project.Key,
			"conformance_status": report.Status,
			"mode":               report.Mode,
			"profile_id":         report.ProfileID,
			"adapter":            report.Adapter,
			"profile_hash":       report.ProfileHash,
			"stage_count":        report.StageCount,
			"gate_count":         report.GateCount,
			"checks":             len(report.Checks),
		},
	}
}

func releaseStatusFromPlan(status string) string {
	switch status {
	case "ready":
		return "ready"
	case "needs_attention":
		return "needs_attention"
	case "blocked":
		return "blocked"
	default:
		return "blocked"
	}
}

func releaseStatusFromPassFail(status string) string {
	switch status {
	case "ready", "pass":
		return "ready"
	case "warn", "needs_attention", "skipped":
		return "needs_attention"
	default:
		return "blocked"
	}
}

func releaseStatusFromPassWarnFail(status string) string {
	switch status {
	case "pass", "ready":
		return "ready"
	case "warn", "needs_attention", "skipped":
		return "needs_attention"
	default:
		return "blocked"
	}
}

func worseReleaseReadinessStatus(candidate string, current string) bool {
	return releaseReadinessStatusRank(candidate) > releaseReadinessStatusRank(current)
}

func releaseReadinessStatusRank(status string) int {
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
