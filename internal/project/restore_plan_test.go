package project

import (
	"testing"
	"time"
)

func TestBuildRestorePlanNeedsAttentionForProjectReferences(t *testing.T) {
	created := time.Date(2026, 6, 30, 12, 0, 0, 0, time.UTC)
	plan := BuildRestorePlan(BackupManifest{
		Status:        "ready",
		Mode:          "read_only_manifest",
		SchemaVersion: 1,
		ManifestHash:  "hash-1",
		Projects: []BackupProjectManifest{
			{
				Project: projectRecordForRestoreTest(),
				Inventory: ImportInventory{
					Versions:  2,
					Artifacts: 2,
				},
				ArtifactCount: 2,
				Artifacts: []BackupArtifactSummary{
					{ID: 1, StorageBackend: "local", ArtifactType: "runner_preview_report", SHA256: "abc", SizeBytes: 10},
					{ID: 2, StorageBackend: "external_project", ArtifactType: "source_ref", SHA256: "def", SizeBytes: 20},
				},
			},
		},
		ForbiddenActions: []string{"restore_database", "write_project_files", "delete_existing_state", "resolve_secrets"},
	}, RestorePlanOptions{GeneratedAt: created})

	if plan.Status != "needs_attention" || plan.Mode != "read_only_restore_plan" {
		t.Fatalf("unexpected restore plan: %+v", plan)
	}
	if plan.Scope != "platform" || plan.ProjectKey != "" {
		t.Fatalf("unexpected restore scope: %+v", plan)
	}
	if plan.SchemaVersion != 1 || plan.ManifestHash != "hash-1" || len(plan.Projects) != 1 {
		t.Fatalf("unexpected manifest data: %+v", plan)
	}
	assertRestorePlanItem(t, plan, "manifest_shape", "ready")
	assertRestorePlanItem(t, plan, "project_inventory", "ready")
	assertRestorePlanItem(t, plan, "artifact_inventory", "needs_attention")
	assertRestorePlanItem(t, plan, "dry_run_guardrails", "ready")
	if !plan.GeneratedAt.Equal(created) {
		t.Fatalf("generated_at = %s, want %s", plan.GeneratedAt, created)
	}
}

func TestBuildRestorePlanScopedToTargetProject(t *testing.T) {
	created := time.Date(2026, 7, 13, 12, 0, 0, 0, time.UTC)
	target := projectRecordForRestoreTest()
	fixture := target
	fixture.ID = 2
	fixture.Key = "areamatrix-stale-fixture"
	plan := BuildRestorePlan(BackupManifest{
		Status:        "ready",
		Mode:          "read_only_manifest",
		SchemaVersion: 1,
		ManifestHash:  "hash-1",
		Projects: []BackupProjectManifest{
			{
				Project:       target,
				ArtifactCount: 1,
				Artifacts: []BackupArtifactSummary{
					{ID: 1, StorageBackend: "local", ArtifactType: "runner_preview_report", SHA256: "abc", SizeBytes: 10},
				},
			},
			{
				Project:       fixture,
				ArtifactCount: 1,
				Artifacts: []BackupArtifactSummary{
					{ID: 2, StorageBackend: "external_project", ArtifactType: "source_ref", SHA256: "def", SizeBytes: 20},
				},
			},
		},
		ForbiddenActions: []string{"restore_database", "write_project_files", "delete_existing_state", "resolve_secrets"},
	}, RestorePlanOptions{GeneratedAt: created, ProjectKey: target.Key})

	if plan.Status != "ready" || plan.Scope != "project" || plan.ProjectKey != target.Key {
		t.Fatalf("target scoped restore plan should ignore non-target fixture warnings: %+v", plan)
	}
	if len(plan.Projects) != 1 || plan.Projects[0].Key != target.Key {
		t.Fatalf("unexpected scoped projects: %+v", plan.Projects)
	}
	assertRestorePlanItem(t, plan, "artifact_inventory", "ready")
}

func TestBuildRestorePlanBlocksMissingManifestHash(t *testing.T) {
	plan := BuildRestorePlan(BackupManifest{
		SchemaVersion:    1,
		Projects:         []BackupProjectManifest{{Project: projectRecordForRestoreTest()}},
		ForbiddenActions: []string{"restore_database", "write_project_files", "delete_existing_state", "resolve_secrets"},
	}, RestorePlanOptions{})

	if plan.Status != "blocked" {
		t.Fatalf("expected blocked plan: %+v", plan)
	}
	assertRestorePlanItem(t, plan, "manifest_shape", "blocked")
}

func TestBuildRestorePlanBlocksMissingGuardrails(t *testing.T) {
	plan := BuildRestorePlan(BackupManifest{
		SchemaVersion: 1,
		ManifestHash:  "hash-1",
		Projects:      []BackupProjectManifest{{Project: projectRecordForRestoreTest()}},
	}, RestorePlanOptions{})

	if plan.Status != "blocked" {
		t.Fatalf("expected blocked plan: %+v", plan)
	}
	assertRestorePlanItem(t, plan, "dry_run_guardrails", "blocked")
}

func TestRestorePlanItemFromArtifactIntegrity(t *testing.T) {
	item := restorePlanItemFromArtifactIntegrity(ArtifactIntegrityReport{
		Status:           "warn",
		Project:          projectRecordForRestoreTest(),
		CheckedArtifacts: 2,
		PassedArtifacts:  1,
		SkippedArtifacts: 1,
	})

	if item.Status != "needs_attention" || item.Key != "artifact_integrity:areamatrix" {
		t.Fatalf("unexpected integrity item: %+v", item)
	}
	if item.Metadata["skipped_artifacts"] != 1 {
		t.Fatalf("unexpected metadata: %+v", item.Metadata)
	}
}

func projectRecordForRestoreTest() Record {
	return Record{
		ID:              1,
		Key:             "areamatrix",
		Name:            "AreaMatrix",
		Kind:            "product-repo",
		Adapter:         "areamatrix",
		WorkflowProfile: "areamatrix",
	}
}

func assertRestorePlanItem(t *testing.T, plan RestorePlan, key string, status string) {
	t.Helper()
	for _, item := range plan.Items {
		if item.Key == key {
			if item.Status != status {
				t.Fatalf("item %s status = %s, want %s: %+v", key, item.Status, status, item)
			}
			return
		}
	}
	t.Fatalf("restore plan item %s not found: %+v", key, plan.Items)
}
