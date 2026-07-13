package project

import (
	"context"
	"time"
)

type ReleaseDistributionPreviewOptions struct {
	GeneratedAt time.Time
	ProjectID   int64
	ProjectKey  string
}

type ReleaseDistributionPreviewItem struct {
	Key              string
	Category         string
	Status           string
	Channel          string
	Action           string
	Message          string
	Owner            string
	RequiredEvidence []string
	NextCommand      string
	Metadata         map[string]any
}

type ReleaseDistributionPreview struct {
	Real100Guardrail
	Status           string
	Mode             string
	Scope            string
	ProjectKey       string
	PackagePreview   ReleasePackagePreview
	Items            []ReleaseDistributionPreviewItem
	Capabilities     []string
	ForbiddenActions []string
	GeneratedAt      time.Time
}

func (s Store) ReleaseDistributionPreview(ctx context.Context, options ReleaseDistributionPreviewOptions) (ReleaseDistributionPreview, error) {
	options = normalizeReleaseDistributionPreviewOptions(options)
	packagePreview, err := s.ReleasePackagePreview(ctx, ReleasePackagePreviewOptions{GeneratedAt: options.GeneratedAt, ProjectID: options.ProjectID, ProjectKey: options.ProjectKey})
	if err != nil {
		return ReleaseDistributionPreview{}, err
	}
	return BuildReleaseDistributionPreview(packagePreview, options), nil
}

func normalizeReleaseDistributionPreviewOptions(options ReleaseDistributionPreviewOptions) ReleaseDistributionPreviewOptions {
	options.GeneratedAt, options.ProjectKey = normalizeReleaseScopeFields(options.GeneratedAt, options.ProjectKey)
	return options
}

func BuildReleaseDistributionPreview(packagePreview ReleasePackagePreview, options ReleaseDistributionPreviewOptions) ReleaseDistributionPreview {
	options = normalizeReleaseDistributionPreviewOptions(options)
	scope, projectKey := releaseScopeAndProjectKey(options.ProjectID, options.ProjectKey, packagePreview.ProjectKey)
	preview := ReleaseDistributionPreview{
		Real100Guardrail: ReleasePreviewReal100Guardrail(),
		Status:           "ready",
		Mode:             "read_only_release_distribution_preview",
		Scope:            scope,
		ProjectKey:       projectKey,
		PackagePreview:   packagePreview,
		Items:            []ReleaseDistributionPreviewItem{},
		Capabilities: []string{
			"read_release_package_preview",
			"preview_release_distribution_channels",
			"report_release_distribution_blockers",
		},
		ForbiddenActions: []string{
			"create_release_package",
			"write_release_manifest",
			"upload_release_artifacts",
			"publish_release",
			"create_git_tag",
			"sign_release",
			"push_git",
			"write_database",
			"write_project_files",
			"write_artifact_store",
			"read_artifact_contents",
			"execute_commands",
			"start_worker",
			"apply_release",
		},
		GeneratedAt: options.GeneratedAt,
	}
	preview.addItem(releaseDistributionPackageItem(packagePreview))
	preview.addItem(releaseDistributionChannelItem(packagePreview, "distribution:local_archive", "local_archive", "Local archive distribution preview"))
	preview.addItem(releaseDistributionChannelItem(packagePreview, "distribution:git_release", "git_release", "Git release distribution preview"))
	preview.addItem(releaseDistributionChannelItem(packagePreview, "distribution:artifact_registry", "artifact_registry", "Artifact registry distribution preview"))
	return preview
}

func (p *ReleaseDistributionPreview) addItem(item ReleaseDistributionPreviewItem) {
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

func releaseDistributionPackageItem(packagePreview ReleasePackagePreview) ReleaseDistributionPreviewItem {
	status := distributionStatusFromPackagePreview(packagePreview.Status)
	action := "package_preview_ready"
	message := "release package preview is ready for distribution planning"
	nextCommand := "areaflow release distribution-preview --json"
	if status == "blocked" {
		action = "wait_for_package_preview"
		message = "release distribution is blocked until package preview is ready"
		nextCommand = "areaflow release package-preview --json"
	}
	if status == "needs_attention" {
		action = "review_package_preview"
		message = "release distribution needs package preview attention before publish planning"
		nextCommand = "areaflow release package-preview --json"
	}
	return ReleaseDistributionPreviewItem{
		Key:         "distribution:package_preview",
		Category:    "package",
		Status:      status,
		Channel:     "release_package",
		Action:      action,
		Message:     message,
		Owner:       "release-owner",
		NextCommand: nextCommand,
		RequiredEvidence: []string{
			"release package preview ready",
			"release evidence bundle ready",
		},
		Metadata: map[string]any{
			"package_status":   packagePreview.Status,
			"package_name":     packagePreview.PackageName,
			"package_writable": false,
		},
	}
}

func releaseDistributionChannelItem(packagePreview ReleasePackagePreview, key string, channel string, message string) ReleaseDistributionPreviewItem {
	status := distributionStatusFromPackagePreview(packagePreview.Status)
	action := "preview_distribution_channel"
	nextCommand := "areaflow release distribution-preview --json"
	if status == "blocked" {
		action = "wait_for_package_preview"
		message = message + " is blocked until package preview is ready"
		nextCommand = "areaflow release package-preview --json"
	}
	if status == "needs_attention" {
		action = "review_package_preview"
		message = message + " needs package preview attention"
		nextCommand = "areaflow release package-preview --json"
	}
	return ReleaseDistributionPreviewItem{
		Key:         key,
		Category:    "distribution",
		Status:      status,
		Channel:     channel,
		Action:      action,
		Message:     message,
		Owner:       "release-owner",
		NextCommand: nextCommand,
		RequiredEvidence: []string{
			"release package preview ready",
			"distribution approval recorded",
			"publish rollback plan reviewed",
		},
		Metadata: map[string]any{
			"package_status":        packagePreview.Status,
			"upload_attempted":      false,
			"publish_attempted":     false,
			"tag_attempted":         false,
			"sign_attempted":        false,
			"release_write_allowed": false,
		},
	}
}

func distributionStatusFromPackagePreview(status string) string {
	switch status {
	case "blocked":
		return "blocked"
	case "needs_attention":
		return "needs_attention"
	default:
		return "ready"
	}
}
