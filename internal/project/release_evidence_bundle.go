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

type ReleaseEvidenceBundleOptions struct {
	GeneratedAt time.Time
	ProjectID   int64
	ProjectKey  string
}

type ReleaseEvidenceBundleItem struct {
	Key         string
	Category    string
	Status      string
	Source      string
	Description string
	Metadata    map[string]any
}

type ReleaseEvidenceBundle struct {
	Real100Guardrail
	Status           string
	Mode             string
	Scope            string
	ProjectKey       string
	BundleHash       string
	FinalGate        ReleaseFinalGate
	Backup           BackupManifest
	AuditCoverage    AuditCoverage
	Items            []ReleaseEvidenceBundleItem
	Capabilities     []string
	ForbiddenActions []string
	GeneratedAt      time.Time
}

func (s Store) ReleaseEvidenceBundle(ctx context.Context, options ReleaseEvidenceBundleOptions) (ReleaseEvidenceBundle, error) {
	options = normalizeReleaseEvidenceBundleOptions(options)
	scope, err := s.resolveReleaseProjectScope(ctx, options.ProjectID, options.ProjectKey)
	if err != nil {
		return ReleaseEvidenceBundle{}, err
	}
	options.ProjectID = scope.ProjectID
	options.ProjectKey = scope.ProjectKey
	finalGate, err := s.ReleaseFinalGate(ctx, ReleaseFinalGateOptions{GeneratedAt: options.GeneratedAt, ProjectID: options.ProjectID, ProjectKey: options.ProjectKey})
	if err != nil {
		return ReleaseEvidenceBundle{}, err
	}
	backup, err := s.BackupManifest(ctx, BackupManifestOptions{GeneratedAt: options.GeneratedAt, ProjectID: options.ProjectID, ProjectKey: options.ProjectKey})
	if err != nil {
		return ReleaseEvidenceBundle{}, err
	}
	auditCoverage, err := s.AuditCoverage(ctx, AuditCoverageOptions{GeneratedAt: options.GeneratedAt, ProjectID: options.ProjectID, ProjectKey: options.ProjectKey})
	if err != nil {
		return ReleaseEvidenceBundle{}, err
	}
	return BuildReleaseEvidenceBundle(finalGate, backup, auditCoverage, options), nil
}

func normalizeReleaseEvidenceBundleOptions(options ReleaseEvidenceBundleOptions) ReleaseEvidenceBundleOptions {
	options.GeneratedAt, options.ProjectKey = normalizeReleaseScopeFields(options.GeneratedAt, options.ProjectKey)
	return options
}

func BuildReleaseEvidenceBundle(finalGate ReleaseFinalGate, backup BackupManifest, auditCoverage AuditCoverage, options ReleaseEvidenceBundleOptions) ReleaseEvidenceBundle {
	options = normalizeReleaseEvidenceBundleOptions(options)
	scope, projectKey := releaseScopeAndProjectKey(options.ProjectID, options.ProjectKey, finalGate.ProjectKey, backup.ProjectKey, auditCoverage.ProjectKey)
	bundle := ReleaseEvidenceBundle{
		Real100Guardrail: ReleasePreviewReal100Guardrail(),
		Status:           "ready",
		Mode:             "read_only_release_evidence_bundle",
		Scope:            scope,
		ProjectKey:       projectKey,
		FinalGate:        finalGate,
		Backup:           backup,
		AuditCoverage:    auditCoverage,
		Items:            []ReleaseEvidenceBundleItem{},
		Capabilities: []string{
			"read_release_final_gate",
			"read_backup_manifest",
			"read_audit_coverage",
			"assemble_release_evidence_index",
		},
		ForbiddenActions: []string{
			"create_release_package",
			"write_database",
			"write_project_files",
			"write_artifact_store",
			"read_artifact_contents",
			"create_approval",
			"run_migration",
			"insert_exception_record",
			"execute_commands",
			"start_worker",
			"apply_release",
		},
		GeneratedAt: options.GeneratedAt,
	}
	bundle.addItem(releaseEvidenceFinalGateItem(finalGate))
	bundle.addItem(releaseEvidenceBackupItem(backup))
	bundle.addItem(releaseEvidenceAuditItem(auditCoverage))
	for _, project := range backup.Projects {
		bundle.addItem(releaseEvidenceProjectInventoryItem(project))
	}
	if hash, err := releaseEvidenceBundleHash(bundle); err == nil {
		bundle.BundleHash = hash
	}
	return bundle
}

func ReleaseEvidenceBundleBindingMetadata(bundle ReleaseEvidenceBundle) map[string]any {
	inventoryKey := "evidence:project_inventory:" + bundle.ProjectKey
	inventoryPresent := false
	inventoryReady := false
	if strings.TrimSpace(bundle.ProjectKey) == "" {
		inventoryKey = ""
	}
	for _, item := range bundle.Items {
		if inventoryKey == "" || item.Key != inventoryKey {
			continue
		}
		inventoryPresent = true
		inventoryReady = item.Status == "ready"
	}
	return map[string]any{
		"release_evidence_bundle_hash":                      bundle.BundleHash,
		"release_evidence_bundle_status":                    bundle.Status,
		"release_evidence_bundle_mode":                      bundle.Mode,
		"release_evidence_bundle_scope":                     bundle.Scope,
		"release_evidence_bundle_project_key":               bundle.ProjectKey,
		"release_evidence_bundle_item_count":                len(bundle.Items),
		"release_evidence_bundle_project_inventory_key":     inventoryKey,
		"release_evidence_bundle_project_inventory_present": inventoryPresent,
		"release_evidence_bundle_project_inventory_ready":   inventoryReady,
		"release_evidence_bundle_ready": bundle.BundleHash != "" &&
			bundle.Status == "ready" &&
			bundle.Mode == "read_only_release_evidence_bundle" &&
			bundle.Scope == "project" &&
			strings.TrimSpace(bundle.ProjectKey) != "" &&
			inventoryPresent &&
			inventoryReady &&
			len(bundle.Items) > 0,
	}
}

func (b *ReleaseEvidenceBundle) addItem(item ReleaseEvidenceBundleItem) {
	if item.Metadata == nil {
		item.Metadata = map[string]any{}
	}
	b.Items = append(b.Items, item)
	if item.Status == "blocked" {
		b.Status = "blocked"
	}
	if item.Status == "needs_attention" && b.Status == "ready" {
		b.Status = "needs_attention"
	}
}

func releaseEvidenceFinalGateItem(finalGate ReleaseFinalGate) ReleaseEvidenceBundleItem {
	status := "ready"
	if finalGate.Status == "blocked" {
		status = "blocked"
	}
	return ReleaseEvidenceBundleItem{
		Key:         "evidence:release_final_gate",
		Category:    "release_gate",
		Status:      status,
		Source:      "release final-gate",
		Description: "release final go/no-go result",
		Metadata: map[string]any{
			"final_gate_status": finalGate.Status,
			"item_count":        len(finalGate.Items),
			"mode":              finalGate.Mode,
		},
	}
}

func releaseEvidenceBackupItem(backup BackupManifest) ReleaseEvidenceBundleItem {
	status := "ready"
	if backup.Status != "ready" {
		status = "blocked"
	}
	return ReleaseEvidenceBundleItem{
		Key:         "evidence:backup_manifest",
		Category:    "backup",
		Status:      status,
		Source:      "backup manifest",
		Description: "PostgreSQL and artifact metadata manifest",
		Metadata: map[string]any{
			"backup_status":  backup.Status,
			"mode":           backup.Mode,
			"manifest_hash":  backup.ManifestHash,
			"schema_version": backup.SchemaVersion,
			"project_count":  len(backup.Projects),
			"table_count":    len(backup.TableCounts),
		},
	}
}

func releaseEvidenceAuditItem(auditCoverage AuditCoverage) ReleaseEvidenceBundleItem {
	status := "ready"
	if auditCoverage.Status == "warn" {
		status = "needs_attention"
	}
	if auditCoverage.Status == "fail" {
		status = "blocked"
	}
	return ReleaseEvidenceBundleItem{
		Key:         "evidence:audit_coverage",
		Category:    "audit",
		Status:      status,
		Source:      "audit coverage",
		Description: "audit evidence coverage matrix",
		Metadata: map[string]any{
			"audit_status":         auditCoverage.Status,
			"scope":                auditCoverage.Scope,
			"covered_requirements": auditCoverage.CoveredRequirements,
			"gap_requirements":     auditCoverage.GapRequirements,
			"total_audit_events":   auditCoverage.TotalAuditEvents,
		},
	}
}

func releaseEvidenceProjectInventoryItem(project BackupProjectManifest) ReleaseEvidenceBundleItem {
	status := "ready"
	if project.Project.Key == "" {
		status = "blocked"
	}
	return ReleaseEvidenceBundleItem{
		Key:         "evidence:project_inventory:" + project.Project.Key,
		Category:    "project_inventory",
		Status:      status,
		Source:      "backup manifest project inventory",
		Description: "project inventory and artifact metadata index",
		Metadata: map[string]any{
			"project_key":      project.Project.Key,
			"project_kind":     project.Project.Kind,
			"adapter":          project.Project.Adapter,
			"workflow_profile": project.Project.WorkflowProfile,
			"default_branch":   project.Project.DefaultBranch,
			"root_path":        project.Project.RootPath,
			"versions":         project.Inventory.Versions,
			"residuals":        project.Inventory.Residuals,
			"artifacts":        project.ArtifactCount,
			"artifact_items":   len(project.Artifacts),
		},
	}
}

func releaseEvidenceBundleHash(bundle ReleaseEvidenceBundle) (string, error) {
	items := make([]ReleaseEvidenceBundleItem, 0, len(bundle.Items))
	for _, item := range bundle.Items {
		metadata := map[string]any{}
		for key, value := range item.Metadata {
			switch key {
			case "final_gate_status", "item_count", "mode", "backup_status", "schema_version", "project_count",
				"audit_status", "scope", "covered_requirements", "gap_requirements", "project_key",
				"project_kind", "adapter", "workflow_profile", "default_branch", "root_path",
				"versions", "residuals", "artifacts", "artifact_items":
				metadata[key] = value
			}
		}
		items = append(items, ReleaseEvidenceBundleItem{
			Key:         item.Key,
			Category:    item.Category,
			Status:      item.Status,
			Source:      item.Source,
			Description: item.Description,
			Metadata:    metadata,
		})
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].Key != items[j].Key {
			return items[i].Key < items[j].Key
		}
		return items[i].Category < items[j].Category
	})
	capabilities := append([]string{}, bundle.Capabilities...)
	forbiddenActions := append([]string{}, bundle.ForbiddenActions...)
	sort.Strings(capabilities)
	sort.Strings(forbiddenActions)
	payload, err := json.Marshal(map[string]any{
		"status":            bundle.Status,
		"mode":              bundle.Mode,
		"scope":             bundle.Scope,
		"project_key":       bundle.ProjectKey,
		"final_gate_status": bundle.FinalGate.Status,
		"backup_status":     bundle.Backup.Status,
		"backup_schema":     bundle.Backup.SchemaVersion,
		"audit_status":      bundle.AuditCoverage.Status,
		"audit_scope":       bundle.AuditCoverage.Scope,
		"items":             items,
		"capabilities":      capabilities,
		"forbidden_actions": forbiddenActions,
	})
	if err != nil {
		return "", fmt.Errorf("marshal release evidence bundle hash payload: %w", err)
	}
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:]), nil
}
