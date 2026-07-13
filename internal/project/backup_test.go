package project

import (
	"testing"
	"time"
)

func TestBackupManifestHashIgnoresGeneratedAt(t *testing.T) {
	created := time.Date(2026, 6, 30, 10, 0, 0, 0, time.UTC)
	manifest := BackupManifest{
		Status:        "ready",
		Mode:          "read_only_manifest",
		SchemaVersion: 1,
		GeneratedAt:   created,
		TableCounts: []BackupTableCount{
			{Table: "projects", Rows: 1},
			{Table: "artifacts", Rows: 2},
		},
		Projects: []BackupProjectManifest{
			{
				Project:       Record{Key: "areamatrix"},
				Inventory:     ImportInventory{Versions: 2, Artifacts: 6},
				ArtifactCount: 2,
				Artifacts: []BackupArtifactSummary{
					{ID: 2, ArtifactType: "runner_preview_report", URI: "b.json", SHA256: "def", CreatedAt: created.Add(time.Minute)},
					{ID: 1, ArtifactType: "workflow_stage_artifact", URI: "a.md", SHA256: "abc", CreatedAt: created},
				},
			},
		},
	}

	first, err := backupManifestHash(manifest)
	if err != nil {
		t.Fatalf("backup manifest hash failed: %v", err)
	}
	manifest.GeneratedAt = created.Add(time.Hour)
	manifest.TableCounts[0], manifest.TableCounts[1] = manifest.TableCounts[1], manifest.TableCounts[0]
	manifest.Projects[0].Artifacts[0], manifest.Projects[0].Artifacts[1] = manifest.Projects[0].Artifacts[1], manifest.Projects[0].Artifacts[0]
	second, err := backupManifestHash(manifest)
	if err != nil {
		t.Fatalf("backup manifest hash failed: %v", err)
	}
	if first == "" || first != second {
		t.Fatalf("manifest hash should be stable, first=%q second=%q", first, second)
	}
}

func TestBackupManifestGuardrails(t *testing.T) {
	generated := time.Date(2026, 6, 30, 10, 0, 0, 0, time.UTC)
	manifest := BackupManifest{
		Status:        "ready",
		Mode:          "read_only_manifest",
		SchemaVersion: 1,
		GeneratedAt:   generated,
		Capabilities: []string{
			"export_postgres_metadata",
			"export_artifact_metadata",
			"verify_manifest_hash",
		},
		ForbiddenActions: []string{
			"read_artifact_contents",
			"write_project_files",
			"restore_database",
			"delete_existing_state",
			"resolve_secrets",
		},
	}

	if manifest.Status != "ready" || manifest.Mode != "read_only_manifest" {
		t.Fatalf("unexpected backup manifest mode: %+v", manifest)
	}
	if !containsString(manifest.Capabilities, "export_artifact_metadata") {
		t.Fatalf("missing backup capability: %+v", manifest.Capabilities)
	}
	if !containsString(manifest.ForbiddenActions, "restore_database") {
		t.Fatalf("missing backup forbidden action: %+v", manifest.ForbiddenActions)
	}
	if !manifest.GeneratedAt.Equal(generated) {
		t.Fatalf("generated_at = %s, want %s", manifest.GeneratedAt, generated)
	}
}

func TestBackupManifestHashReflectsScopedProjectSet(t *testing.T) {
	created := time.Date(2026, 7, 13, 12, 30, 0, 0, time.UTC)
	full := BackupManifest{
		Status:        "ready",
		Mode:          "read_only_manifest",
		Scope:         "platform",
		SchemaVersion: 1,
		GeneratedAt:   created,
		TableCounts:   []BackupTableCount{{Table: "projects", Rows: 2}},
		Projects: []BackupProjectManifest{
			{
				Project:       Record{ID: 1, Key: "areamatrix"},
				Inventory:     ImportInventory{Versions: 1, Artifacts: 1},
				ArtifactCount: 1,
				Artifacts:     []BackupArtifactSummary{{ID: 1, ArtifactType: "runner_preview_report", URI: "target.json", SHA256: "abc", CreatedAt: created}},
			},
			{
				Project:       Record{ID: 2, Key: "areamatrix-stale-fixture"},
				Inventory:     ImportInventory{Versions: 1, Artifacts: 1},
				ArtifactCount: 1,
				Artifacts:     []BackupArtifactSummary{{ID: 2, ArtifactType: "runner_preview_report", URI: "fixture.json", SHA256: "def", CreatedAt: created}},
			},
		},
	}
	scoped := full
	scoped.Scope = "project"
	scoped.ProjectKey = "areamatrix"
	scoped.Projects = full.Projects[:1]

	fullHash, err := backupManifestHash(full)
	if err != nil {
		t.Fatalf("full backup manifest hash failed: %v", err)
	}
	scopedHash, err := backupManifestHash(scoped)
	if err != nil {
		t.Fatalf("scoped backup manifest hash failed: %v", err)
	}
	if fullHash == "" || scopedHash == "" || fullHash == scopedHash {
		t.Fatalf("scoped manifest hash should differ from global manifest hash, full=%q scoped=%q", fullHash, scopedHash)
	}
}
