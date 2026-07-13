package project

import (
	"context"
	"time"
)

type ReleasePublishApprovalPreviewOptions struct {
	GeneratedAt time.Time
	ProjectID   int64
	ProjectKey  string
}

type ReleasePublishApprovalPreviewItem struct {
	Key              string
	Category         string
	Status           string
	ApprovalStatus   string
	Channel          string
	Message          string
	Owner            string
	RequiredEvidence []string
	NextCommand      string
	Metadata         map[string]any
}

type ReleasePublishApprovalPreview struct {
	Real100Guardrail
	Status           string
	Mode             string
	Scope            string
	ProjectKey       string
	PublishGate      ReleasePublishGate
	Items            []ReleasePublishApprovalPreviewItem
	Capabilities     []string
	ForbiddenActions []string
	GeneratedAt      time.Time
}

func (s Store) ReleasePublishApprovalPreview(ctx context.Context, options ReleasePublishApprovalPreviewOptions) (ReleasePublishApprovalPreview, error) {
	options = normalizeReleasePublishApprovalPreviewOptions(options)
	publishGate, err := s.ReleasePublishGate(ctx, ReleasePublishGateOptions{GeneratedAt: options.GeneratedAt, ProjectID: options.ProjectID, ProjectKey: options.ProjectKey})
	if err != nil {
		return ReleasePublishApprovalPreview{}, err
	}
	return BuildReleasePublishApprovalPreview(publishGate, options), nil
}

func normalizeReleasePublishApprovalPreviewOptions(options ReleasePublishApprovalPreviewOptions) ReleasePublishApprovalPreviewOptions {
	options.GeneratedAt, options.ProjectKey = normalizeReleaseScopeFields(options.GeneratedAt, options.ProjectKey)
	return options
}

func BuildReleasePublishApprovalPreview(publishGate ReleasePublishGate, options ReleasePublishApprovalPreviewOptions) ReleasePublishApprovalPreview {
	options = normalizeReleasePublishApprovalPreviewOptions(options)
	scope, projectKey := releaseScopeAndProjectKey(options.ProjectID, options.ProjectKey, publishGate.ProjectKey)
	preview := ReleasePublishApprovalPreview{
		Real100Guardrail: ReleasePreviewReal100Guardrail(),
		Status:           "needs_approval",
		Mode:             "read_only_release_publish_approval_preview",
		Scope:            scope,
		ProjectKey:       projectKey,
		PublishGate:      publishGate,
		Items:            []ReleasePublishApprovalPreviewItem{},
		Capabilities: []string{
			"read_release_publish_gate",
			"preview_release_publish_approval",
			"report_release_publish_approval_blockers",
		},
		ForbiddenActions: []string{
			"create_approval",
			"approve_release",
			"write_database",
			"write_project_files",
			"write_artifact_store",
			"create_release_package",
			"write_release_manifest",
			"upload_release_artifacts",
			"publish_release",
			"create_git_tag",
			"sign_release",
			"push_git",
			"insert_audit_event",
			"execute_commands",
			"start_worker",
			"apply_release",
		},
		GeneratedAt: options.GeneratedAt,
	}
	if publishGate.Status != "pass" {
		preview.addItem(releasePublishApprovalBlockedItem(publishGate))
		return preview
	}
	preview.addItem(releasePublishApprovalRequiredItem(publishGate))
	for _, item := range publishGate.Items {
		if item.Category == "distribution" {
			preview.addItem(releasePublishApprovalChannelItem(item))
		}
	}
	return preview
}

func (p *ReleasePublishApprovalPreview) addItem(item ReleasePublishApprovalPreviewItem) {
	if item.Metadata == nil {
		item.Metadata = map[string]any{}
	}
	p.Items = append(p.Items, item)
	if item.Status == "blocked" {
		p.Status = "blocked"
	}
}

func releasePublishApprovalBlockedItem(publishGate ReleasePublishGate) ReleasePublishApprovalPreviewItem {
	return ReleasePublishApprovalPreviewItem{
		Key:            "publish_approval:publish_gate",
		Category:       "publish_gate",
		Status:         "blocked",
		ApprovalStatus: "blocked",
		Channel:        "all",
		Message:        "release publish approval cannot be requested until publish gate passes",
		Owner:          "release-owner",
		RequiredEvidence: []string{
			"areaflow release publish-gate --json returns status pass",
			"all distribution channels are ready",
		},
		NextCommand: "areaflow release publish-gate --json",
		Metadata: map[string]any{
			"publish_gate_status": publishGate.Status,
			"approval_writable":   false,
			"publish_writable":    false,
		},
	}
}

func releasePublishApprovalRequiredItem(publishGate ReleasePublishGate) ReleasePublishApprovalPreviewItem {
	return ReleasePublishApprovalPreviewItem{
		Key:            "publish_approval:release_publish",
		Category:       "approval",
		Status:         "needs_approval",
		ApprovalStatus: "needs_approval",
		Channel:        "all",
		Message:        "explicit release publish approval is required before publish can proceed",
		Owner:          "release-owner",
		RequiredEvidence: []string{
			"release publish gate remains pass",
			"release distribution preview remains ready",
			"publish rollback plan reviewed",
			"release owner approval recorded",
		},
		NextCommand: "areaflow release publish-approval-preview --json",
		Metadata: map[string]any{
			"publish_gate_status": publishGate.Status,
			"approval_writable":   false,
			"publish_writable":    false,
		},
	}
}

func releasePublishApprovalChannelItem(gateItem ReleasePublishGateItem) ReleasePublishApprovalPreviewItem {
	status := "needs_approval"
	approvalStatus := "needs_approval"
	message := "explicit publish approval must include this distribution channel"
	nextCommand := "areaflow release publish-approval-preview --json"
	if gateItem.Status != "pass" {
		status = "blocked"
		approvalStatus = "blocked"
		message = "distribution channel gate blocks publish approval"
		nextCommand = "areaflow release publish-gate --json"
	}
	return ReleasePublishApprovalPreviewItem{
		Key:            "publish_approval:" + gateItem.Channel,
		Category:       "distribution",
		Status:         status,
		ApprovalStatus: approvalStatus,
		Channel:        gateItem.Channel,
		Message:        message,
		Owner:          gateItem.Owner,
		RequiredEvidence: []string{
			"publish gate channel remains pass",
			"channel-specific rollback plan reviewed",
			"release owner approval includes channel " + gateItem.Channel,
		},
		NextCommand: nextCommand,
		Metadata: map[string]any{
			"publish_gate_item":   gateItem.Key,
			"publish_gate_status": gateItem.Status,
			"approval_writable":   false,
			"upload_attempted":    false,
			"publish_attempted":   false,
			"tag_attempted":       false,
			"sign_attempted":      false,
			"push_attempted":      false,
		},
	}
}
