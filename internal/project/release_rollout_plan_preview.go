package project

import (
	"context"
	"time"
)

type ReleaseRolloutPlanPreviewOptions struct {
	GeneratedAt time.Time
	ProjectID   int64
	ProjectKey  string
}

type ReleaseRolloutPlanPreviewItem struct {
	Key              string
	Category         string
	Status           string
	Stage            string
	Action           string
	Message          string
	Owner            string
	RequiredEvidence []string
	NextCommand      string
	Metadata         map[string]any
}

type ReleaseRolloutPlanPreviewStep struct {
	Order       int
	Stage       string
	Action      string
	Description string
	BlockedBy   []string
}

type ReleaseRolloutPlanPreview struct {
	Real100Guardrail
	Status                  string
	Mode                    string
	Scope                   string
	ProjectKey              string
	PublishApprovalPreview  ReleasePublishApprovalPreview
	Items                   []ReleaseRolloutPlanPreviewItem
	RolloutSteps            []ReleaseRolloutPlanPreviewStep
	VerificationCheckpoints []ReleaseRolloutPlanPreviewStep
	RollbackSteps           []ReleaseRolloutPlanPreviewStep
	Capabilities            []string
	ForbiddenActions        []string
	GeneratedAt             time.Time
}

func (s Store) ReleaseRolloutPlanPreview(ctx context.Context, options ReleaseRolloutPlanPreviewOptions) (ReleaseRolloutPlanPreview, error) {
	options = normalizeReleaseRolloutPlanPreviewOptions(options)
	publishApproval, err := s.ReleasePublishApprovalPreview(ctx, ReleasePublishApprovalPreviewOptions{GeneratedAt: options.GeneratedAt, ProjectID: options.ProjectID, ProjectKey: options.ProjectKey})
	if err != nil {
		return ReleaseRolloutPlanPreview{}, err
	}
	return BuildReleaseRolloutPlanPreview(publishApproval, options), nil
}

func normalizeReleaseRolloutPlanPreviewOptions(options ReleaseRolloutPlanPreviewOptions) ReleaseRolloutPlanPreviewOptions {
	options.GeneratedAt, options.ProjectKey = normalizeReleaseScopeFields(options.GeneratedAt, options.ProjectKey)
	return options
}

func BuildReleaseRolloutPlanPreview(publishApproval ReleasePublishApprovalPreview, options ReleaseRolloutPlanPreviewOptions) ReleaseRolloutPlanPreview {
	options = normalizeReleaseRolloutPlanPreviewOptions(options)
	scope, projectKey := releaseScopeAndProjectKey(options.ProjectID, options.ProjectKey, publishApproval.ProjectKey)
	preview := ReleaseRolloutPlanPreview{
		Real100Guardrail:       ReleasePreviewReal100Guardrail(),
		Status:                 "ready",
		Mode:                   "read_only_release_rollout_plan_preview",
		Scope:                  scope,
		ProjectKey:             projectKey,
		PublishApprovalPreview: publishApproval,
		Items:                  []ReleaseRolloutPlanPreviewItem{},
		RolloutSteps: []ReleaseRolloutPlanPreviewStep{
			{Order: 1, Stage: "preflight", Action: "verify_publish_approval", Description: "confirm release publish approval remains valid", BlockedBy: []string{"publish_approval:release_publish"}},
			{Order: 2, Stage: "prepare", Action: "prepare_release_artifacts", Description: "prepare release artifacts after explicit publish approval", BlockedBy: []string{"publish_approval:release_publish"}},
			{Order: 3, Stage: "publish", Action: "publish_release_channels", Description: "publish approved release channels in the selected order", BlockedBy: []string{"publish_approval:release_publish"}},
			{Order: 4, Stage: "observe", Action: "observe_release_health", Description: "observe release health before marking rollout complete", BlockedBy: []string{"rollout_verification"}},
			{Order: 5, Stage: "closeout", Action: "record_rollout_closeout", Description: "record rollout closeout evidence after verification passes", BlockedBy: []string{"rollout_verification"}},
		},
		VerificationCheckpoints: []ReleaseRolloutPlanPreviewStep{
			{Order: 1, Stage: "preflight", Action: "release_final_gate_pass", Description: "release final gate remains pass", BlockedBy: []string{"final_gate:release_readiness", "final_gate:release_acceptance", "final_gate:release_exception_apply"}},
			{Order: 2, Stage: "package", Action: "release_package_ready", Description: "release package preview remains ready", BlockedBy: []string{"package:manifest"}},
			{Order: 3, Stage: "distribution", Action: "distribution_channels_ready", Description: "distribution preview and publish gate remain ready", BlockedBy: []string{"publish_gate:distribution_preview"}},
			{Order: 4, Stage: "approval", Action: "publish_approval_recorded", Description: "release owner publish approval is recorded", BlockedBy: []string{"publish_approval:release_publish"}},
			{Order: 5, Stage: "post_publish", Action: "post_publish_smoke_ready", Description: "post-publish smoke checks and rollback window are defined", BlockedBy: []string{"rollout_approval"}},
		},
		RollbackSteps: []ReleaseRolloutPlanPreviewStep{
			{Order: 1, Stage: "pause", Action: "pause_distribution", Description: "pause further distribution channels", BlockedBy: []string{}},
			{Order: 2, Stage: "revoke", Action: "revoke_or_hide_release", Description: "revoke, hide or mark the published release according to channel policy", BlockedBy: []string{}},
			{Order: 3, Stage: "restore", Action: "restore_previous_release_pointer", Description: "restore previous release pointer or compatibility status", BlockedBy: []string{}},
			{Order: 4, Stage: "audit", Action: "record_rollback_audit", Description: "record rollback decision, owner and evidence", BlockedBy: []string{}},
		},
		Capabilities: []string{
			"read_release_publish_approval_preview",
			"preview_release_rollout_plan",
			"report_release_rollout_blockers",
		},
		ForbiddenActions: []string{
			"create_rollout",
			"write_release_state",
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
	if publishApproval.Status == "blocked" {
		preview.addItem(releaseRolloutBlockedByPublishApprovalItem(publishApproval))
		return preview
	}
	if publishApproval.Status == "needs_approval" {
		preview.addItem(releaseRolloutNeedsPublishApprovalItem(publishApproval))
		return preview
	}
	preview.addItem(releaseRolloutReadyItem(publishApproval))
	return preview
}

func (p *ReleaseRolloutPlanPreview) addItem(item ReleaseRolloutPlanPreviewItem) {
	if item.Metadata == nil {
		item.Metadata = map[string]any{}
	}
	p.Items = append(p.Items, item)
	if item.Status == "blocked" {
		p.Status = "blocked"
	}
	if item.Status == "needs_approval" && p.Status == "ready" {
		p.Status = "needs_approval"
	}
}

func releaseRolloutBlockedByPublishApprovalItem(publishApproval ReleasePublishApprovalPreview) ReleaseRolloutPlanPreviewItem {
	return ReleaseRolloutPlanPreviewItem{
		Key:      "rollout_plan:publish_approval",
		Category: "publish_approval",
		Status:   "blocked",
		Stage:    "preflight",
		Action:   "wait_for_publish_approval_preview",
		Message:  "release rollout plan is blocked until publish approval preview is no longer blocked",
		Owner:    "release-owner",
		RequiredEvidence: []string{
			"areaflow release publish-approval-preview --json returns needs_approval or ready",
			"release publish gate blockers are closed",
		},
		NextCommand: "areaflow release publish-approval-preview --json",
		Metadata: map[string]any{
			"publish_approval_status": publishApproval.Status,
			"rollout_writable":        false,
			"publish_attempted":       false,
		},
	}
}

func releaseRolloutNeedsPublishApprovalItem(publishApproval ReleasePublishApprovalPreview) ReleaseRolloutPlanPreviewItem {
	return ReleaseRolloutPlanPreviewItem{
		Key:      "rollout_plan:release_rollout",
		Category: "approval",
		Status:   "needs_approval",
		Stage:    "preflight",
		Action:   "wait_for_release_owner_approval",
		Message:  "release rollout requires explicit release owner approval before any publish action",
		Owner:    "release-owner",
		RequiredEvidence: []string{
			"release publish approval recorded",
			"rollout verification checkpoints reviewed",
			"rollback plan reviewed",
		},
		NextCommand: "areaflow release rollout-plan-preview --json",
		Metadata: map[string]any{
			"publish_approval_status": publishApproval.Status,
			"rollout_writable":        false,
			"publish_attempted":       false,
		},
	}
}

func releaseRolloutReadyItem(publishApproval ReleasePublishApprovalPreview) ReleaseRolloutPlanPreviewItem {
	return ReleaseRolloutPlanPreviewItem{
		Key:      "rollout_plan:ready",
		Category: "rollout",
		Status:   "ready",
		Stage:    "preflight",
		Action:   "preview_rollout",
		Message:  "release rollout plan is ready for explicit rollout approval",
		Owner:    "release-owner",
		RequiredEvidence: []string{
			"release publish approval remains ready",
			"rollout approval recorded",
			"rollback plan reviewed",
		},
		NextCommand: "areaflow release rollout-plan-preview --json",
		Metadata: map[string]any{
			"publish_approval_status": publishApproval.Status,
			"rollout_writable":        false,
			"publish_attempted":       false,
		},
	}
}
