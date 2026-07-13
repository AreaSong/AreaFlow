package project

import (
	"testing"
	"time"
)

func TestBuildReleasePackagePreviewBlocksWhenEvidenceBundleBlocks(t *testing.T) {
	created := time.Date(2026, 6, 30, 12, 0, 0, 0, time.UTC)
	preview := BuildReleasePackagePreview(
		ReleaseEvidenceBundle{
			Status:     "blocked",
			Mode:       "read_only_release_evidence_bundle",
			Scope:      "project",
			ProjectKey: "areamatrix",
			Items: []ReleaseEvidenceBundleItem{
				{
					Key:         "evidence:release_final_gate",
					Category:    "release_gate",
					Status:      "blocked",
					Source:      "release final-gate",
					Description: "release final go/no-go result",
				},
			},
		},
		ReleasePackagePreviewOptions{GeneratedAt: created},
	)

	if preview.Status != "blocked" || preview.Mode != "read_only_release_package_preview" ||
		preview.Scope != "project" || preview.ProjectKey != "areamatrix" {
		t.Fatalf("unexpected package preview: %+v", preview)
	}
	if preview.PackageName == "" {
		t.Fatalf("package name missing: %+v", preview)
	}
	if len(preview.Items) != 2 {
		t.Fatalf("items = %d, want 2: %+v", len(preview.Items), preview.Items)
	}
	assertReleasePackagePreviewItem(t, preview, "package:manifest", "manifest", "blocked")
	assertReleasePackagePreviewItem(t, preview, "package:evidence:release_final_gate", "release_gate", "blocked")
	if preview.Items[0].Metadata["package_writable"] != false {
		t.Fatalf("package preview must remain read-only: %+v", preview.Items[0].Metadata)
	}
	if len(preview.ForbiddenActions) == 0 || preview.ForbiddenActions[0] != "create_release_package" || preview.ForbiddenActions[4] != "read_artifact_contents" {
		t.Fatalf("unexpected forbidden actions: %+v", preview.ForbiddenActions)
	}
	if !preview.GeneratedAt.Equal(created) {
		t.Fatalf("generated_at = %s, want %s", preview.GeneratedAt, created)
	}
}

func TestBuildReleasePackagePreviewReadyWhenEvidenceReady(t *testing.T) {
	preview := BuildReleasePackagePreview(
		ReleaseEvidenceBundle{
			Status: "ready",
			Items: []ReleaseEvidenceBundleItem{
				{Key: "evidence:backup_manifest", Category: "backup", Status: "ready", Source: "backup manifest", Description: "manifest"},
			},
		},
		ReleasePackagePreviewOptions{},
	)

	if preview.Status != "ready" {
		t.Fatalf("expected ready status: %+v", preview)
	}
	assertReleasePackagePreviewItem(t, preview, "package:manifest", "manifest", "ready")
	assertReleasePackagePreviewItem(t, preview, "package:evidence:backup_manifest", "backup", "ready")
}

func assertReleasePackagePreviewItem(t *testing.T, preview ReleasePackagePreview, key string, category string, status string) {
	t.Helper()
	for _, item := range preview.Items {
		if item.Key == key {
			if item.Category != category || item.Status != status {
				t.Fatalf("item %s = %+v, want category=%s status=%s", key, item, category, status)
			}
			if item.PackagePath == "" || item.Source == "" || item.Description == "" {
				t.Fatalf("item %s missing package path/source/description: %+v", key, item)
			}
			return
		}
	}
	t.Fatalf("item %s not found: %+v", key, preview.Items)
}
