package project

import (
	"testing"
	"time"
)

func TestBuildReleaseRolloutPlanPreviewBlocksWhenPublishApprovalBlocks(t *testing.T) {
	created := time.Date(2026, 6, 30, 12, 0, 0, 0, time.UTC)
	preview := BuildReleaseRolloutPlanPreview(
		ReleasePublishApprovalPreview{
			Status: "blocked",
			Mode:   "read_only_release_publish_approval_preview",
			Items: []ReleasePublishApprovalPreviewItem{
				{
					Key:            "publish_approval:publish_gate",
					Category:       "publish_gate",
					Status:         "blocked",
					ApprovalStatus: "blocked",
					Channel:        "all",
				},
			},
		},
		ReleaseRolloutPlanPreviewOptions{GeneratedAt: created},
	)

	if preview.Status != "blocked" || preview.Mode != "read_only_release_rollout_plan_preview" {
		t.Fatalf("unexpected rollout plan preview: %+v", preview)
	}
	if preview.PublishApprovalPreview.Status != "blocked" {
		t.Fatalf("publish approval preview not nested: %+v", preview.PublishApprovalPreview)
	}
	if len(preview.Items) != 1 {
		t.Fatalf("items = %d, want 1: %+v", len(preview.Items), preview.Items)
	}
	assertReleaseRolloutPlanPreviewItem(t, preview, "rollout_plan:publish_approval", "publish_approval", "blocked", "preflight")
	if preview.Items[0].Metadata["rollout_writable"] != false || preview.Items[0].Metadata["publish_attempted"] != false {
		t.Fatalf("rollout preview must remain read-only: %+v", preview.Items[0].Metadata)
	}
	if len(preview.RolloutSteps) != 5 || preview.RolloutSteps[0].Action != "verify_publish_approval" {
		t.Fatalf("unexpected rollout steps: %+v", preview.RolloutSteps)
	}
	if len(preview.VerificationCheckpoints) != 5 || preview.VerificationCheckpoints[0].Action != "release_final_gate_pass" {
		t.Fatalf("unexpected verification checkpoints: %+v", preview.VerificationCheckpoints)
	}
	if len(preview.RollbackSteps) != 4 || preview.RollbackSteps[0].Action != "pause_distribution" {
		t.Fatalf("unexpected rollback steps: %+v", preview.RollbackSteps)
	}
	if len(preview.ForbiddenActions) == 0 || preview.ForbiddenActions[0] != "create_rollout" || preview.ForbiddenActions[8] != "publish_release" {
		t.Fatalf("unexpected forbidden actions: %+v", preview.ForbiddenActions)
	}
	if !preview.GeneratedAt.Equal(created) {
		t.Fatalf("generated_at = %s, want %s", preview.GeneratedAt, created)
	}
}

func TestBuildReleaseRolloutPlanPreviewNeedsApprovalWhenPublishApprovalNeedsApproval(t *testing.T) {
	preview := BuildReleaseRolloutPlanPreview(
		ReleasePublishApprovalPreview{
			Status: "needs_approval",
			Items: []ReleasePublishApprovalPreviewItem{
				{Key: "publish_approval:release_publish", Category: "approval", Status: "needs_approval", ApprovalStatus: "needs_approval"},
			},
		},
		ReleaseRolloutPlanPreviewOptions{},
	)

	if preview.Status != "needs_approval" {
		t.Fatalf("expected needs_approval status: %+v", preview)
	}
	assertReleaseRolloutPlanPreviewItem(t, preview, "rollout_plan:release_rollout", "approval", "needs_approval", "preflight")
	if preview.Items[0].Metadata["rollout_writable"] != false {
		t.Fatalf("rollout preview must remain read-only: %+v", preview.Items[0].Metadata)
	}
}

func TestBuildReleaseRolloutPlanPreviewReadyWhenPublishApprovalReady(t *testing.T) {
	preview := BuildReleaseRolloutPlanPreview(
		ReleasePublishApprovalPreview{Status: "ready"},
		ReleaseRolloutPlanPreviewOptions{},
	)

	if preview.Status != "ready" {
		t.Fatalf("expected ready status: %+v", preview)
	}
	assertReleaseRolloutPlanPreviewItem(t, preview, "rollout_plan:ready", "rollout", "ready", "preflight")
}

func TestBuildReleaseRolloutPlanPreviewPropagatesProjectScope(t *testing.T) {
	preview := BuildReleaseRolloutPlanPreview(
		ReleasePublishApprovalPreview{
			Status:     "ready",
			Scope:      "project",
			ProjectKey: "areamatrix",
		},
		ReleaseRolloutPlanPreviewOptions{ProjectKey: "areamatrix"},
	)

	if preview.Scope != "project" || preview.ProjectKey != "areamatrix" {
		t.Fatalf("expected project-scoped rollout plan preview: %+v", preview)
	}
	if preview.PublishApprovalPreview.ProjectKey != "areamatrix" {
		t.Fatalf("expected nested publish approval preview to keep project key: %+v", preview.PublishApprovalPreview)
	}
}

func assertReleaseRolloutPlanPreviewItem(t *testing.T, preview ReleaseRolloutPlanPreview, key string, category string, status string, stage string) {
	t.Helper()
	for _, item := range preview.Items {
		if item.Key == key {
			if item.Category != category || item.Status != status || item.Stage != stage {
				t.Fatalf("item %s = %+v, want category=%s status=%s stage=%s", key, item, category, status, stage)
			}
			if item.Action == "" || item.Message == "" || item.Owner == "" || len(item.RequiredEvidence) == 0 || item.NextCommand == "" {
				t.Fatalf("item %s missing action/message/owner/evidence/next command: %+v", key, item)
			}
			return
		}
	}
	t.Fatalf("item %s not found: %+v", key, preview.Items)
}
