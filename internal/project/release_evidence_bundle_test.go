package project

import (
	"testing"
	"time"
)

func TestBuildReleaseEvidenceBundleBlocksWhenFinalGateBlocks(t *testing.T) {
	created := time.Date(2026, 6, 30, 12, 0, 0, 0, time.UTC)
	bundle := BuildReleaseEvidenceBundle(
		ReleaseFinalGate{Status: "blocked", Mode: "read_only_release_final_gate", Items: []ReleaseFinalGateItem{{Key: "final_gate:release_readiness"}}},
		BackupManifest{
			Status:        "ready",
			Mode:          "read_only_manifest",
			Scope:         "project",
			ProjectKey:    "areamatrix",
			SchemaVersion: 1,
			ManifestHash:  "abc123",
			Projects: []BackupProjectManifest{
				{
					Project:       Record{Key: "areamatrix"},
					Inventory:     ImportInventory{Versions: 2, Residuals: 10},
					ArtifactCount: 6,
					Artifacts:     []BackupArtifactSummary{{ID: 1}},
				},
			},
		},
		AuditCoverage{Status: "warn", Scope: "platform", GapRequirements: 2, TotalAuditEvents: 4},
		ReleaseEvidenceBundleOptions{GeneratedAt: created, ProjectKey: "areamatrix"},
	)

	if bundle.Status != "blocked" || bundle.Mode != "read_only_release_evidence_bundle" ||
		bundle.Scope != "project" || bundle.ProjectKey != "areamatrix" {
		t.Fatalf("unexpected evidence bundle: %+v", bundle)
	}
	if len(bundle.Items) != 4 {
		t.Fatalf("items = %d, want 4: %+v", len(bundle.Items), bundle.Items)
	}
	assertReleaseEvidenceBundleItem(t, bundle, "evidence:release_final_gate", "release_gate", "blocked")
	assertReleaseEvidenceBundleItem(t, bundle, "evidence:backup_manifest", "backup", "ready")
	assertReleaseEvidenceBundleItem(t, bundle, "evidence:audit_coverage", "audit", "needs_attention")
	assertReleaseEvidenceBundleItem(t, bundle, "evidence:project_inventory:areamatrix", "project_inventory", "ready")
	if len(bundle.ForbiddenActions) == 0 || bundle.ForbiddenActions[0] != "create_release_package" || bundle.ForbiddenActions[4] != "read_artifact_contents" {
		t.Fatalf("unexpected forbidden actions: %+v", bundle.ForbiddenActions)
	}
	if !bundle.GeneratedAt.Equal(created) {
		t.Fatalf("generated_at = %s, want %s", bundle.GeneratedAt, created)
	}
}

func TestBuildReleaseEvidenceBundleReadyWhenInputsReady(t *testing.T) {
	bundle := BuildReleaseEvidenceBundle(
		ReleaseFinalGate{Status: "pass"},
		BackupManifest{Status: "ready", ManifestHash: "abc123"},
		AuditCoverage{Status: "pass"},
		ReleaseEvidenceBundleOptions{},
	)

	if bundle.Status != "ready" {
		t.Fatalf("expected ready status: %+v", bundle)
	}
	assertReleaseEvidenceBundleItem(t, bundle, "evidence:release_final_gate", "release_gate", "ready")
	assertReleaseEvidenceBundleItem(t, bundle, "evidence:backup_manifest", "backup", "ready")
	assertReleaseEvidenceBundleItem(t, bundle, "evidence:audit_coverage", "audit", "ready")
}

func TestReleaseEvidenceBundleHashIgnoresMutableEvidenceCounts(t *testing.T) {
	created := time.Date(2026, 6, 30, 12, 0, 0, 0, time.UTC)
	areaMatrixRecord := Record{
		Key:             "areamatrix",
		Kind:            "product-repo",
		Adapter:         "areamatrix",
		WorkflowProfile: "areamatrix",
		DefaultBranch:   "main",
		RootPath:        "/Users/as/Ai-Project/project/AreaMatrix",
	}
	base := BuildReleaseEvidenceBundle(
		ReleaseFinalGate{Status: "pass", Mode: "read_only_release_final_gate"},
		BackupManifest{
			Status:        "ready",
			Mode:          "read_only_manifest",
			SchemaVersion: 1,
			ManifestHash:  "backup-hash-1",
			TableCounts:   []BackupTableCount{{Table: "events", Rows: 10}},
			Projects: []BackupProjectManifest{
				{
					Project:       areaMatrixRecord,
					Inventory:     ImportInventory{Versions: 2, Residuals: 10},
					ArtifactCount: 6,
					Artifacts:     []BackupArtifactSummary{{ID: 1}},
				},
			},
		},
		AuditCoverage{Status: "pass", Scope: "platform", CoveredRequirements: 9, TotalAuditEvents: 4},
		ReleaseEvidenceBundleOptions{GeneratedAt: created},
	)
	mutableOnly := BuildReleaseEvidenceBundle(
		ReleaseFinalGate{Status: "pass", Mode: "read_only_release_final_gate"},
		BackupManifest{
			Status:        "ready",
			Mode:          "read_only_manifest",
			SchemaVersion: 1,
			ManifestHash:  "backup-hash-2",
			TableCounts:   []BackupTableCount{{Table: "events", Rows: 99}, {Table: "audit_events", Rows: 88}},
			Projects: []BackupProjectManifest{
				{
					Project:       areaMatrixRecord,
					Inventory:     ImportInventory{Versions: 2, Residuals: 10},
					ArtifactCount: 6,
					Artifacts:     []BackupArtifactSummary{{ID: 1}},
				},
			},
		},
		AuditCoverage{Status: "pass", Scope: "platform", CoveredRequirements: 9, TotalAuditEvents: 400},
		ReleaseEvidenceBundleOptions{GeneratedAt: created.Add(2 * time.Hour)},
	)

	if base.BundleHash == "" {
		t.Fatalf("bundle hash should be populated: %+v", base)
	}
	if mutableOnly.BundleHash != base.BundleHash {
		t.Fatalf("mutable evidence counts should not change bundle hash: base=%s mutable=%s", base.BundleHash, mutableOnly.BundleHash)
	}
	projectItem := releaseEvidenceBundleItem(t, base, "evidence:project_inventory:areamatrix")
	if projectItem.Metadata["root_path"] != "/Users/as/Ai-Project/project/AreaMatrix" ||
		projectItem.Metadata["project_kind"] != "product-repo" ||
		projectItem.Metadata["adapter"] != "areamatrix" ||
		projectItem.Metadata["workflow_profile"] != "areamatrix" {
		t.Fatalf("project inventory item should bind real project identity metadata: %+v", projectItem.Metadata)
	}

	changedInventory := BuildReleaseEvidenceBundle(
		ReleaseFinalGate{Status: "pass", Mode: "read_only_release_final_gate"},
		BackupManifest{
			Status:        "ready",
			Mode:          "read_only_manifest",
			SchemaVersion: 1,
			ManifestHash:  "backup-hash-2",
			TableCounts:   []BackupTableCount{{Table: "events", Rows: 99}},
			Projects: []BackupProjectManifest{
				{
					Project:       areaMatrixRecord,
					Inventory:     ImportInventory{Versions: 3, Residuals: 10},
					ArtifactCount: 6,
					Artifacts:     []BackupArtifactSummary{{ID: 1}},
				},
			},
		},
		AuditCoverage{Status: "pass", Scope: "platform", CoveredRequirements: 9, TotalAuditEvents: 400},
		ReleaseEvidenceBundleOptions{GeneratedAt: created.Add(2 * time.Hour)},
	)
	if changedInventory.BundleHash == base.BundleHash {
		t.Fatalf("stable inventory changes should change bundle hash: %s", changedInventory.BundleHash)
	}

	changedIdentityRecord := areaMatrixRecord
	changedIdentityRecord.RootPath = "/tmp/areaflow-completion-audit-rc.fake/areamatrix-root"
	changedIdentity := BuildReleaseEvidenceBundle(
		ReleaseFinalGate{Status: "pass", Mode: "read_only_release_final_gate"},
		BackupManifest{
			Status:        "ready",
			Mode:          "read_only_manifest",
			SchemaVersion: 1,
			ManifestHash:  "backup-hash-2",
			TableCounts:   []BackupTableCount{{Table: "events", Rows: 99}},
			Projects: []BackupProjectManifest{
				{
					Project:       changedIdentityRecord,
					Inventory:     ImportInventory{Versions: 2, Residuals: 10},
					ArtifactCount: 6,
					Artifacts:     []BackupArtifactSummary{{ID: 1}},
				},
			},
		},
		AuditCoverage{Status: "pass", Scope: "platform", CoveredRequirements: 9, TotalAuditEvents: 400},
		ReleaseEvidenceBundleOptions{GeneratedAt: created.Add(2 * time.Hour)},
	)
	if changedIdentity.BundleHash == base.BundleHash {
		t.Fatalf("project identity changes should change bundle hash: %s", changedIdentity.BundleHash)
	}
}

func releaseEvidenceBundleItem(t *testing.T, bundle ReleaseEvidenceBundle, key string) ReleaseEvidenceBundleItem {
	t.Helper()
	for _, item := range bundle.Items {
		if item.Key == key {
			return item
		}
	}
	t.Fatalf("item %s not found: %+v", key, bundle.Items)
	return ReleaseEvidenceBundleItem{}
}

func assertReleaseEvidenceBundleItem(t *testing.T, bundle ReleaseEvidenceBundle, key string, category string, status string) {
	t.Helper()
	for _, item := range bundle.Items {
		if item.Key == key {
			if item.Category != category || item.Status != status {
				t.Fatalf("item %s = %+v, want category=%s status=%s", key, item, category, status)
			}
			if item.Source == "" || item.Description == "" {
				t.Fatalf("item %s missing source/description: %+v", key, item)
			}
			return
		}
	}
	t.Fatalf("item %s not found: %+v", key, bundle.Items)
}
