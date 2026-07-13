package project

import (
	"context"
	"time"
)

type ReleasePublishGateOptions struct {
	GeneratedAt time.Time
	ProjectID   int64
	ProjectKey  string
}

type ReleasePublishGateItem struct {
	Key              string
	Category         string
	Status           string
	Channel          string
	Message          string
	Owner            string
	RequiredEvidence []string
	NextCommand      string
	Metadata         map[string]any
}

type ReleasePublishGate struct {
	Real100Guardrail
	Status              string
	Mode                string
	Scope               string
	ProjectKey          string
	DistributionPreview ReleaseDistributionPreview
	Items               []ReleasePublishGateItem
	Capabilities        []string
	ForbiddenActions    []string
	GeneratedAt         time.Time
}

func (s Store) ReleasePublishGate(ctx context.Context, options ReleasePublishGateOptions) (ReleasePublishGate, error) {
	options = normalizeReleasePublishGateOptions(options)
	distributionPreview, err := s.ReleaseDistributionPreview(ctx, ReleaseDistributionPreviewOptions{GeneratedAt: options.GeneratedAt, ProjectID: options.ProjectID, ProjectKey: options.ProjectKey})
	if err != nil {
		return ReleasePublishGate{}, err
	}
	return BuildReleasePublishGate(distributionPreview, options), nil
}

func normalizeReleasePublishGateOptions(options ReleasePublishGateOptions) ReleasePublishGateOptions {
	options.GeneratedAt, options.ProjectKey = normalizeReleaseScopeFields(options.GeneratedAt, options.ProjectKey)
	return options
}

func BuildReleasePublishGate(distributionPreview ReleaseDistributionPreview, options ReleasePublishGateOptions) ReleasePublishGate {
	options = normalizeReleasePublishGateOptions(options)
	scope, projectKey := releaseScopeAndProjectKey(options.ProjectID, options.ProjectKey, distributionPreview.ProjectKey)
	gate := ReleasePublishGate{
		Real100Guardrail:    ReleasePreviewReal100Guardrail(),
		Status:              "pass",
		Mode:                "read_only_release_publish_gate",
		Scope:               scope,
		ProjectKey:          projectKey,
		DistributionPreview: distributionPreview,
		Items:               []ReleasePublishGateItem{},
		Capabilities: []string{
			"read_release_distribution_preview",
			"evaluate_release_publish_gate",
			"report_release_publish_blockers",
		},
		ForbiddenActions: []string{
			"create_release_package",
			"write_release_manifest",
			"upload_release_artifacts",
			"publish_release",
			"create_git_tag",
			"sign_release",
			"push_git",
			"create_approval",
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
	gate.addItem(releasePublishDistributionItem(distributionPreview))
	for _, item := range distributionPreview.Items {
		if item.Category == "distribution" {
			gate.addItem(releasePublishChannelItem(item))
		}
	}
	if len(gate.Items) == 1 && len(distributionPreview.Items) == 0 {
		gate.addItem(ReleasePublishGateItem{
			Key:              "publish_gate:distribution_items",
			Category:         "distribution",
			Status:           "blocked",
			Channel:          "all",
			Message:          "release distribution preview produced no channel items",
			Owner:            "release-owner",
			RequiredEvidence: []string{"release distribution preview contains distribution channel items"},
			NextCommand:      "areaflow release distribution-preview --json",
			Metadata: map[string]any{
				"distribution_status": distributionPreview.Status,
				"item_count":          len(distributionPreview.Items),
			},
		})
	}
	return gate
}

func (g *ReleasePublishGate) addItem(item ReleasePublishGateItem) {
	if item.Metadata == nil {
		item.Metadata = map[string]any{}
	}
	g.Items = append(g.Items, item)
	if item.Status == "blocked" {
		g.Status = "blocked"
	}
}

func releasePublishDistributionItem(distributionPreview ReleaseDistributionPreview) ReleasePublishGateItem {
	status := "pass"
	message := "release distribution preview is ready"
	requiredEvidence := []string{"release distribution preview status remains ready"}
	nextCommand := "areaflow release distribution-preview --json"
	if distributionPreview.Status != "ready" {
		status = "blocked"
		message = "release distribution preview blocks publish"
		requiredEvidence = []string{"areaflow release distribution-preview --json returns status ready"}
	}
	return ReleasePublishGateItem{
		Key:              "publish_gate:distribution_preview",
		Category:         "distribution_preview",
		Status:           status,
		Channel:          "all",
		Message:          message,
		Owner:            "release-owner",
		RequiredEvidence: requiredEvidence,
		NextCommand:      nextCommand,
		Metadata: map[string]any{
			"distribution_status": distributionPreview.Status,
			"item_count":          len(distributionPreview.Items),
			"publish_writable":    false,
		},
	}
}

func releasePublishChannelItem(distribution ReleaseDistributionPreviewItem) ReleasePublishGateItem {
	status := "pass"
	message := "release distribution channel is ready for publish approval"
	requiredEvidence := []string{
		"release distribution channel remains ready",
		"publish approval recorded",
		"publish rollback plan reviewed",
	}
	nextCommand := "areaflow release publish-gate --json"
	if distribution.Status != "ready" {
		status = "blocked"
		message = "release distribution channel blocks publish"
		requiredEvidence = []string{"areaflow release distribution-preview --json returns this channel ready"}
		nextCommand = "areaflow release distribution-preview --json"
	}
	return ReleasePublishGateItem{
		Key:              "publish_gate:" + distribution.Channel,
		Category:         "distribution",
		Status:           status,
		Channel:          distribution.Channel,
		Message:          message,
		Owner:            distribution.Owner,
		RequiredEvidence: requiredEvidence,
		NextCommand:      nextCommand,
		Metadata: map[string]any{
			"distribution_key":    distribution.Key,
			"distribution_status": distribution.Status,
			"upload_attempted":    false,
			"publish_attempted":   false,
			"tag_attempted":       false,
			"sign_attempted":      false,
			"push_attempted":      false,
			"publish_writable":    false,
		},
	}
}
