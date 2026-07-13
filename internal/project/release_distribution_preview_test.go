package project

import (
	"testing"
	"time"
)

func TestBuildReleaseDistributionPreviewBlocksWhenPackagePreviewBlocks(t *testing.T) {
	created := time.Date(2026, 6, 30, 12, 0, 0, 0, time.UTC)
	preview := BuildReleaseDistributionPreview(
		ReleasePackagePreview{
			Status:      "blocked",
			Mode:        "read_only_release_package_preview",
			PackageName: "areaflow-v1.0-release-evidence-preview",
			Items: []ReleasePackagePreviewItem{
				{
					Key:         "package:manifest",
					Category:    "manifest",
					Status:      "blocked",
					PackagePath: "release/manifest.json",
					Source:      "release evidence-bundle",
					Description: "release package manifest preview",
				},
			},
		},
		ReleaseDistributionPreviewOptions{GeneratedAt: created},
	)

	if preview.Status != "blocked" || preview.Mode != "read_only_release_distribution_preview" {
		t.Fatalf("unexpected distribution preview: %+v", preview)
	}
	if preview.PackagePreview.Status != "blocked" {
		t.Fatalf("package preview not nested: %+v", preview.PackagePreview)
	}
	if len(preview.Items) != 4 {
		t.Fatalf("items = %d, want 4: %+v", len(preview.Items), preview.Items)
	}
	assertReleaseDistributionPreviewItem(t, preview, "distribution:package_preview", "package", "blocked", "release_package")
	assertReleaseDistributionPreviewItem(t, preview, "distribution:local_archive", "distribution", "blocked", "local_archive")
	assertReleaseDistributionPreviewItem(t, preview, "distribution:git_release", "distribution", "blocked", "git_release")
	assertReleaseDistributionPreviewItem(t, preview, "distribution:artifact_registry", "distribution", "blocked", "artifact_registry")
	if preview.Items[1].Metadata["publish_attempted"] != false || preview.Items[1].Metadata["release_write_allowed"] != false {
		t.Fatalf("distribution preview must remain read-only: %+v", preview.Items[1].Metadata)
	}
	if len(preview.ForbiddenActions) == 0 || preview.ForbiddenActions[0] != "create_release_package" || preview.ForbiddenActions[3] != "publish_release" {
		t.Fatalf("unexpected forbidden actions: %+v", preview.ForbiddenActions)
	}
	if !preview.GeneratedAt.Equal(created) {
		t.Fatalf("generated_at = %s, want %s", preview.GeneratedAt, created)
	}
}

func TestBuildReleaseDistributionPreviewNeedsAttentionWhenPackageNeedsAttention(t *testing.T) {
	preview := BuildReleaseDistributionPreview(
		ReleasePackagePreview{Status: "needs_attention", PackageName: "pkg"},
		ReleaseDistributionPreviewOptions{},
	)

	if preview.Status != "needs_attention" {
		t.Fatalf("expected needs_attention status: %+v", preview)
	}
	assertReleaseDistributionPreviewItem(t, preview, "distribution:package_preview", "package", "needs_attention", "release_package")
	assertReleaseDistributionPreviewItem(t, preview, "distribution:local_archive", "distribution", "needs_attention", "local_archive")
}

func TestBuildReleaseDistributionPreviewReadyWhenPackageReady(t *testing.T) {
	preview := BuildReleaseDistributionPreview(
		ReleasePackagePreview{Status: "ready", PackageName: "pkg"},
		ReleaseDistributionPreviewOptions{},
	)

	if preview.Status != "ready" {
		t.Fatalf("expected ready status: %+v", preview)
	}
	assertReleaseDistributionPreviewItem(t, preview, "distribution:package_preview", "package", "ready", "release_package")
	assertReleaseDistributionPreviewItem(t, preview, "distribution:git_release", "distribution", "ready", "git_release")
}

func TestBuildReleaseDistributionPreviewPropagatesProjectScope(t *testing.T) {
	preview := BuildReleaseDistributionPreview(
		ReleasePackagePreview{
			Status:     "ready",
			Scope:      "project",
			ProjectKey: "areamatrix",
		},
		ReleaseDistributionPreviewOptions{ProjectKey: "areamatrix"},
	)

	if preview.Scope != "project" || preview.ProjectKey != "areamatrix" {
		t.Fatalf("expected project-scoped distribution preview: %+v", preview)
	}
	if preview.PackagePreview.ProjectKey != "areamatrix" {
		t.Fatalf("expected nested package preview to keep project key: %+v", preview.PackagePreview)
	}
}

func assertReleaseDistributionPreviewItem(t *testing.T, preview ReleaseDistributionPreview, key string, category string, status string, channel string) {
	t.Helper()
	for _, item := range preview.Items {
		if item.Key == key {
			if item.Category != category || item.Status != status || item.Channel != channel {
				t.Fatalf("item %s = %+v, want category=%s status=%s channel=%s", key, item, category, status, channel)
			}
			if item.Action == "" || item.Message == "" || item.Owner == "" || len(item.RequiredEvidence) == 0 || item.NextCommand == "" {
				t.Fatalf("item %s missing action/message/owner/evidence/next command: %+v", key, item)
			}
			return
		}
	}
	t.Fatalf("item %s not found: %+v", key, preview.Items)
}
