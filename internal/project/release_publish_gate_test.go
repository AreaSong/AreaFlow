package project

import (
	"testing"
	"time"
)

func TestBuildReleasePublishGateBlocksWhenDistributionPreviewBlocks(t *testing.T) {
	created := time.Date(2026, 6, 30, 12, 0, 0, 0, time.UTC)
	gate := BuildReleasePublishGate(
		ReleaseDistributionPreview{
			Status: "blocked",
			Mode:   "read_only_release_distribution_preview",
			Items: []ReleaseDistributionPreviewItem{
				{
					Key:      "distribution:git_release",
					Category: "distribution",
					Status:   "blocked",
					Channel:  "git_release",
					Action:   "wait_for_package_preview",
					Owner:    "release-owner",
				},
			},
		},
		ReleasePublishGateOptions{GeneratedAt: created},
	)

	if gate.Status != "blocked" || gate.Mode != "read_only_release_publish_gate" {
		t.Fatalf("unexpected publish gate: %+v", gate)
	}
	if gate.DistributionPreview.Status != "blocked" {
		t.Fatalf("distribution preview not nested: %+v", gate.DistributionPreview)
	}
	if len(gate.Items) != 2 {
		t.Fatalf("items = %d, want 2: %+v", len(gate.Items), gate.Items)
	}
	assertReleasePublishGateItem(t, gate, "publish_gate:distribution_preview", "distribution_preview", "blocked", "all")
	assertReleasePublishGateItem(t, gate, "publish_gate:git_release", "distribution", "blocked", "git_release")
	if gate.Items[1].Metadata["publish_attempted"] != false || gate.Items[1].Metadata["publish_writable"] != false {
		t.Fatalf("publish gate must remain read-only: %+v", gate.Items[1].Metadata)
	}
	if len(gate.ForbiddenActions) == 0 || gate.ForbiddenActions[3] != "publish_release" || gate.ForbiddenActions[6] != "push_git" {
		t.Fatalf("unexpected forbidden actions: %+v", gate.ForbiddenActions)
	}
	if !gate.GeneratedAt.Equal(created) {
		t.Fatalf("generated_at = %s, want %s", gate.GeneratedAt, created)
	}
}

func TestBuildReleasePublishGatePassesWhenDistributionReady(t *testing.T) {
	gate := BuildReleasePublishGate(
		ReleaseDistributionPreview{
			Status: "ready",
			Items: []ReleaseDistributionPreviewItem{
				{Key: "distribution:local_archive", Category: "distribution", Status: "ready", Channel: "local_archive", Owner: "release-owner"},
				{Key: "distribution:artifact_registry", Category: "distribution", Status: "ready", Channel: "artifact_registry", Owner: "release-owner"},
			},
		},
		ReleasePublishGateOptions{},
	)

	if gate.Status != "pass" {
		t.Fatalf("expected pass status: %+v", gate)
	}
	assertReleasePublishGateItem(t, gate, "publish_gate:distribution_preview", "distribution_preview", "pass", "all")
	assertReleasePublishGateItem(t, gate, "publish_gate:local_archive", "distribution", "pass", "local_archive")
	assertReleasePublishGateItem(t, gate, "publish_gate:artifact_registry", "distribution", "pass", "artifact_registry")
}

func TestBuildReleasePublishGateBlocksWhenDistributionItemsMissing(t *testing.T) {
	gate := BuildReleasePublishGate(
		ReleaseDistributionPreview{Status: "ready"},
		ReleasePublishGateOptions{},
	)

	if gate.Status != "blocked" {
		t.Fatalf("expected blocked status when distribution items are missing: %+v", gate)
	}
	assertReleasePublishGateItem(t, gate, "publish_gate:distribution_items", "distribution", "blocked", "all")
}

func TestBuildReleasePublishGatePropagatesProjectScope(t *testing.T) {
	gate := BuildReleasePublishGate(
		ReleaseDistributionPreview{
			Status:     "ready",
			Scope:      "project",
			ProjectKey: "areamatrix",
		},
		ReleasePublishGateOptions{ProjectKey: "areamatrix"},
	)

	if gate.Scope != "project" || gate.ProjectKey != "areamatrix" {
		t.Fatalf("expected project-scoped publish gate: %+v", gate)
	}
	if gate.DistributionPreview.ProjectKey != "areamatrix" {
		t.Fatalf("expected nested distribution preview to keep project key: %+v", gate.DistributionPreview)
	}
}

func assertReleasePublishGateItem(t *testing.T, gate ReleasePublishGate, key string, category string, status string, channel string) {
	t.Helper()
	for _, item := range gate.Items {
		if item.Key == key {
			if item.Category != category || item.Status != status || item.Channel != channel {
				t.Fatalf("item %s = %+v, want category=%s status=%s channel=%s", key, item, category, status, channel)
			}
			if item.Message == "" || item.Owner == "" || len(item.RequiredEvidence) == 0 || item.NextCommand == "" {
				t.Fatalf("item %s missing message/owner/evidence/next command: %+v", key, item)
			}
			return
		}
	}
	t.Fatalf("item %s not found: %+v", key, gate.Items)
}
