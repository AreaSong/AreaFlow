package project

import (
	"context"
	"time"
)

type ReleasePackagePreviewOptions struct {
	GeneratedAt time.Time
	ProjectID   int64
	ProjectKey  string
}

type ReleasePackagePreviewItem struct {
	Key         string
	Category    string
	Status      string
	PackagePath string
	Source      string
	Description string
	Metadata    map[string]any
}

type ReleasePackagePreview struct {
	Real100Guardrail
	Status           string
	Mode             string
	Scope            string
	ProjectKey       string
	EvidenceBundle   ReleaseEvidenceBundle
	PackageName      string
	Items            []ReleasePackagePreviewItem
	Capabilities     []string
	ForbiddenActions []string
	GeneratedAt      time.Time
}

func (s Store) ReleasePackagePreview(ctx context.Context, options ReleasePackagePreviewOptions) (ReleasePackagePreview, error) {
	options = normalizeReleasePackagePreviewOptions(options)
	evidenceBundle, err := s.ReleaseEvidenceBundle(ctx, ReleaseEvidenceBundleOptions{GeneratedAt: options.GeneratedAt, ProjectID: options.ProjectID, ProjectKey: options.ProjectKey})
	if err != nil {
		return ReleasePackagePreview{}, err
	}
	return BuildReleasePackagePreview(evidenceBundle, options), nil
}

func normalizeReleasePackagePreviewOptions(options ReleasePackagePreviewOptions) ReleasePackagePreviewOptions {
	options.GeneratedAt, options.ProjectKey = normalizeReleaseScopeFields(options.GeneratedAt, options.ProjectKey)
	return options
}

func BuildReleasePackagePreview(evidenceBundle ReleaseEvidenceBundle, options ReleasePackagePreviewOptions) ReleasePackagePreview {
	options = normalizeReleasePackagePreviewOptions(options)
	scope, projectKey := releaseScopeAndProjectKey(options.ProjectID, options.ProjectKey, evidenceBundle.ProjectKey)
	preview := ReleasePackagePreview{
		Real100Guardrail: ReleasePreviewReal100Guardrail(),
		Status:           "ready",
		Mode:             "read_only_release_package_preview",
		Scope:            scope,
		ProjectKey:       projectKey,
		EvidenceBundle:   evidenceBundle,
		PackageName:      "areaflow-v1.0-release-evidence-preview",
		Items:            []ReleasePackagePreviewItem{},
		Capabilities: []string{
			"read_release_evidence_bundle",
			"preview_release_package_manifest",
			"report_release_package_blockers",
		},
		ForbiddenActions: []string{
			"create_release_package",
			"write_database",
			"write_project_files",
			"write_artifact_store",
			"read_artifact_contents",
			"compress_artifacts",
			"create_approval",
			"run_migration",
			"insert_exception_record",
			"execute_commands",
			"start_worker",
			"apply_release",
		},
		GeneratedAt: options.GeneratedAt,
	}
	preview.addItem(releasePackageManifestItem(evidenceBundle))
	for _, evidence := range evidenceBundle.Items {
		preview.addItem(releasePackageItemForEvidence(evidence))
	}
	return preview
}

func (p *ReleasePackagePreview) addItem(item ReleasePackagePreviewItem) {
	if item.Metadata == nil {
		item.Metadata = map[string]any{}
	}
	p.Items = append(p.Items, item)
	if item.Status == "blocked" {
		p.Status = "blocked"
	}
	if item.Status == "needs_attention" && p.Status == "ready" {
		p.Status = "needs_attention"
	}
}

func releasePackageManifestItem(evidenceBundle ReleaseEvidenceBundle) ReleasePackagePreviewItem {
	status := "ready"
	if evidenceBundle.Status == "blocked" {
		status = "blocked"
	}
	if evidenceBundle.Status == "needs_attention" {
		status = "needs_attention"
	}
	return ReleasePackagePreviewItem{
		Key:         "package:manifest",
		Category:    "manifest",
		Status:      status,
		PackagePath: "release/manifest.json",
		Source:      "release evidence-bundle",
		Description: "release package manifest preview",
		Metadata: map[string]any{
			"evidence_bundle_status": evidenceBundle.Status,
			"evidence_item_count":    len(evidenceBundle.Items),
			"package_writable":       false,
		},
	}
}

func releasePackageItemForEvidence(evidence ReleaseEvidenceBundleItem) ReleasePackagePreviewItem {
	return ReleasePackagePreviewItem{
		Key:         "package:" + evidence.Key,
		Category:    evidence.Category,
		Status:      evidence.Status,
		PackagePath: "release/evidence/" + evidence.Key + ".json",
		Source:      evidence.Source,
		Description: evidence.Description,
		Metadata: map[string]any{
			"evidence_key":    evidence.Key,
			"evidence_status": evidence.Status,
			"read_contents":   false,
		},
	}
}
