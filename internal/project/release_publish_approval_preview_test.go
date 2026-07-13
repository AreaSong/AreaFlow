package project

import (
	"testing"
	"time"
)

func TestBuildReleasePublishApprovalPreviewBlocksWhenPublishGateBlocks(t *testing.T) {
	created := time.Date(2026, 6, 30, 12, 0, 0, 0, time.UTC)
	preview := BuildReleasePublishApprovalPreview(
		ReleasePublishGate{
			Status: "blocked",
			Mode:   "read_only_release_publish_gate",
			Items: []ReleasePublishGateItem{
				{
					Key:      "publish_gate:distribution_preview",
					Category: "distribution_preview",
					Status:   "blocked",
					Channel:  "all",
					Owner:    "release-owner",
				},
			},
		},
		ReleasePublishApprovalPreviewOptions{GeneratedAt: created},
	)

	if preview.Status != "blocked" || preview.Mode != "read_only_release_publish_approval_preview" {
		t.Fatalf("unexpected publish approval preview: %+v", preview)
	}
	if preview.PublishGate.Status != "blocked" {
		t.Fatalf("publish gate not nested: %+v", preview.PublishGate)
	}
	if len(preview.Items) != 1 {
		t.Fatalf("items = %d, want 1: %+v", len(preview.Items), preview.Items)
	}
	assertReleasePublishApprovalPreviewItem(t, preview, "publish_approval:publish_gate", "publish_gate", "blocked", "blocked", "all")
	if preview.Items[0].Metadata["approval_writable"] != false || preview.Items[0].Metadata["publish_writable"] != false {
		t.Fatalf("publish approval preview must remain read-only: %+v", preview.Items[0].Metadata)
	}
	if len(preview.ForbiddenActions) == 0 || preview.ForbiddenActions[0] != "create_approval" || preview.ForbiddenActions[8] != "publish_release" {
		t.Fatalf("unexpected forbidden actions: %+v", preview.ForbiddenActions)
	}
	if !preview.GeneratedAt.Equal(created) {
		t.Fatalf("generated_at = %s, want %s", preview.GeneratedAt, created)
	}
}

func TestBuildReleasePublishApprovalPreviewNeedsApprovalWhenPublishGatePasses(t *testing.T) {
	preview := BuildReleasePublishApprovalPreview(
		ReleasePublishGate{
			Status: "pass",
			Items: []ReleasePublishGateItem{
				{Key: "publish_gate:local_archive", Category: "distribution", Status: "pass", Channel: "local_archive", Owner: "release-owner"},
				{Key: "publish_gate:git_release", Category: "distribution", Status: "pass", Channel: "git_release", Owner: "release-owner"},
			},
		},
		ReleasePublishApprovalPreviewOptions{},
	)

	if preview.Status != "needs_approval" {
		t.Fatalf("expected needs_approval status: %+v", preview)
	}
	if len(preview.Items) != 3 {
		t.Fatalf("items = %d, want 3: %+v", len(preview.Items), preview.Items)
	}
	assertReleasePublishApprovalPreviewItem(t, preview, "publish_approval:release_publish", "approval", "needs_approval", "needs_approval", "all")
	assertReleasePublishApprovalPreviewItem(t, preview, "publish_approval:local_archive", "distribution", "needs_approval", "needs_approval", "local_archive")
	assertReleasePublishApprovalPreviewItem(t, preview, "publish_approval:git_release", "distribution", "needs_approval", "needs_approval", "git_release")
}

func TestBuildReleasePublishApprovalPreviewPropagatesProjectScope(t *testing.T) {
	preview := BuildReleasePublishApprovalPreview(
		ReleasePublishGate{
			Status:     "pass",
			Scope:      "project",
			ProjectKey: "areamatrix",
		},
		ReleasePublishApprovalPreviewOptions{ProjectKey: "areamatrix"},
	)

	if preview.Scope != "project" || preview.ProjectKey != "areamatrix" {
		t.Fatalf("expected project-scoped publish approval preview: %+v", preview)
	}
	if preview.PublishGate.ProjectKey != "areamatrix" {
		t.Fatalf("expected nested publish gate to keep project key: %+v", preview.PublishGate)
	}
}

func assertReleasePublishApprovalPreviewItem(t *testing.T, preview ReleasePublishApprovalPreview, key string, category string, status string, approvalStatus string, channel string) {
	t.Helper()
	for _, item := range preview.Items {
		if item.Key == key {
			if item.Category != category || item.Status != status || item.ApprovalStatus != approvalStatus || item.Channel != channel {
				t.Fatalf("item %s = %+v, want category=%s status=%s approval=%s channel=%s", key, item, category, status, approvalStatus, channel)
			}
			if item.Message == "" || item.Owner == "" || len(item.RequiredEvidence) == 0 || item.NextCommand == "" {
				t.Fatalf("item %s missing message/owner/evidence/next command: %+v", key, item)
			}
			return
		}
	}
	t.Fatalf("item %s not found: %+v", key, preview.Items)
}
