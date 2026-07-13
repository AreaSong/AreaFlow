package status

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/areasong/areaflow/internal/project"
)

func TestWrite(t *testing.T) {
	root := t.TempDir()
	record := project.Record{Key: "demo", Name: "Demo Project", RootPath: root}
	snapshot := project.Snapshot{
		SourceHash: "abc",
		Summary: map[string]any{
			"area_flow_url":  "http://127.0.0.1:3847/projects/demo",
			"cutover_phase":  "read_only_shim",
			"version_count":  float64(1),
			"residual_count": float64(2),
			"versions": []any{
				map[string]any{
					"display_label":    "v1-mvp",
					"version_kind":     "workflow_version",
					"lifecycle_status": "mixed-blocked",
				},
			},
			"v1_execution": map[string]any{
				"done":  float64(86),
				"total": float64(100),
			},
		},
	}

	target, err := Write(record, snapshot, ".areaflow/status.json")
	if err != nil {
		t.Fatalf("write status: %v", err)
	}
	if target != filepath.Join(root, ".areaflow", "status.json") {
		t.Fatalf("target = %s", target)
	}
	content, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("read status: %v", err)
	}
	var doc ExportDocument
	if err := json.Unmarshal(content, &doc); err != nil {
		t.Fatalf("parse status: %v", err)
	}
	if doc.SchemaVersion != 1 || doc.ProjectID != "demo" || doc.ProjectName != "Demo Project" || doc.SourceSnapshotHash != "abc" {
		t.Fatalf("unexpected status doc: %+v", doc)
	}
	if doc.AreaFlowURL != "http://127.0.0.1:3847/projects/demo" || doc.CutoverPhase != "read_only_shim" {
		t.Fatalf("unexpected status routing: %+v", doc)
	}
	if len(doc.ActiveVersions) != 1 || doc.ActiveVersions[0].DisplayLabel != "v1-mvp" {
		t.Fatalf("unexpected active versions: %+v", doc.ActiveVersions)
	}
	progress := doc.ActiveVersions[0].RoughProgress
	if progress.Percent != 86 || progress.Label != "86/100 v1 execution tasks completed" || !progress.Blocked {
		t.Fatalf("unexpected rough progress: %+v", progress)
	}
	if doc.Compatibility.ShimLifecycleState != "read_only_shim" || len(doc.Compatibility.BlockedCommands) == 0 {
		t.Fatalf("unexpected compatibility summary: %+v", doc.Compatibility)
	}
	var raw map[string]any
	if err := json.Unmarshal(content, &raw); err != nil {
		t.Fatalf("parse raw status: %v", err)
	}
	forbidden := []string{"summary", "generated_at", "source", "source_hash"}
	for _, key := range forbidden {
		if _, ok := raw[key]; ok {
			t.Fatalf("status projection should not expose legacy broad field %q: %s", key, content)
		}
	}
}

func TestWriteDoesNotRewriteSemanticallyIdenticalDocument(t *testing.T) {
	root := t.TempDir()
	record := project.Record{Key: "demo", RootPath: root}
	snapshot := project.Snapshot{
		SourceHash: "abc",
		Summary: map[string]any{
			"active": float64(0),
		},
	}

	target, err := Write(record, snapshot, ".areaflow/status.json")
	if err != nil {
		t.Fatalf("write status: %v", err)
	}
	original, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("read original status: %v", err)
	}

	if _, err := Write(record, snapshot, ".areaflow/status.json"); err != nil {
		t.Fatalf("rewrite status: %v", err)
	}
	current, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("read current status: %v", err)
	}
	if string(current) != string(original) {
		t.Fatalf("status export should not rewrite when only generated_at would change\noriginal:\n%s\ncurrent:\n%s", original, current)
	}
}

func TestWriteRejectsInvalidStableProjectionDocument(t *testing.T) {
	root := t.TempDir()
	_, err := Write(project.Record{Key: "demo", RootPath: root}, project.Snapshot{}, ".areaflow/status.json")
	if err == nil {
		t.Fatalf("expected missing source snapshot hash to be rejected")
	}
	if _, statErr := os.Stat(filepath.Join(root, ".areaflow", "status.json")); !os.IsNotExist(statErr) {
		t.Fatalf("invalid projection should not be written: %v", statErr)
	}
}

func TestMarshalStableExportDocumentRejectsInvalidProgress(t *testing.T) {
	document := ExportDocument{
		SchemaVersion:      1,
		ProjectID:          "demo",
		ProjectName:        "Demo",
		AreaFlowURL:        "http://127.0.0.1:3847/projects/demo",
		CutoverPhase:       "import_mirror",
		LastSyncedAt:       "2026-07-04T00:00:00Z",
		SourceSnapshotHash: "hash",
		ActiveVersions: []ActiveVersion{
			{
				DisplayLabel:    "v1",
				VersionKind:     "workflow_version",
				LifecycleStatus: "imported",
				RoughProgress:   RoughProgress{Percent: 101, Label: "too high"},
			},
		},
		Compatibility: CompatibilitySummary{
			ShimLifecycleState: "not_installed",
			OfflineSource:      ".areaflow/status.json",
			BlockedCommands:    []string{"./task-loop run"},
		},
	}

	if _, err := marshalStableExportDocument(document); err == nil {
		t.Fatalf("expected invalid rough progress percent to be rejected")
	}
}

func TestValidateStableExportJSONRejectsLegacyFields(t *testing.T) {
	content := []byte(`{
  "schema_version": 1,
  "project_id": "demo",
  "project_name": "Demo",
  "area_flow_url": "http://127.0.0.1:3847/projects/demo",
  "cutover_phase": "import_mirror",
  "active_versions": [],
  "last_synced_at": "2026-07-04T00:00:00Z",
  "source_snapshot_hash": "hash",
  "summary": {},
  "compatibility": {
    "shim_lifecycle_state": "not_installed",
    "offline_source": ".areaflow/status.json",
    "blocked_commands": ["./task-loop run"]
  }
}`)

	if err := validateStableExportJSON(content); err == nil {
		t.Fatalf("expected legacy summary field to be rejected")
	}
}

func TestWriteReplacesStatusFileAtomicallyWithoutTempResidue(t *testing.T) {
	root := t.TempDir()
	record := project.Record{Key: "demo", RootPath: root}
	snapshot := project.Snapshot{
		SourceHash: "abc",
		Summary: map[string]any{
			"v1_execution": map[string]any{
				"done":  float64(1),
				"total": float64(2),
			},
		},
	}
	target := filepath.Join(root, ".areaflow", "status.json")
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		t.Fatalf("mkdir status dir: %v", err)
	}
	if err := os.WriteFile(target, []byte(`{"version":1,"summary":{}}`), 0o644); err != nil {
		t.Fatalf("write legacy status: %v", err)
	}

	result, err := WriteWithResult(record, snapshot, ".areaflow/status.json")
	if err != nil {
		t.Fatalf("replace status: %v", err)
	}

	if result.Target != target || result.Hash == "" || result.Size == 0 {
		t.Fatalf("unexpected write result: %+v", result)
	}
	if !result.RootContained || !result.StableProjectionValidated || !result.AtomicReplaceUsed {
		t.Fatalf("expected guarded write result facts: %+v", result)
	}
	content, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("read status: %v", err)
	}
	var doc ExportDocument
	if err := json.Unmarshal(content, &doc); err != nil {
		t.Fatalf("parse status: %v\n%s", err, content)
	}
	if doc.SchemaVersion != 1 || doc.SourceSnapshotHash != "abc" {
		t.Fatalf("unexpected status document: %+v", doc)
	}
	matches, err := filepath.Glob(filepath.Join(filepath.Dir(target), ".status.json.tmp-*"))
	if err != nil {
		t.Fatalf("glob temp files: %v", err)
	}
	if len(matches) != 0 {
		t.Fatalf("status export left temp files: %+v", matches)
	}
}

func TestWriteRejectsPathEscape(t *testing.T) {
	root := t.TempDir()
	outside := filepath.Join(root, "..", "outside-status.json")
	_, err := Write(project.Record{Key: "demo", RootPath: root}, project.Snapshot{SourceHash: "abc"}, "../outside-status.json")
	if err == nil {
		t.Fatalf("expected path escape to be rejected")
	}
	if _, statErr := os.Stat(outside); !os.IsNotExist(statErr) {
		t.Fatalf("escaped status file should not exist: %v", statErr)
	}
}

func TestWriteRejectsAbsolutePath(t *testing.T) {
	root := t.TempDir()
	outside := filepath.Join(t.TempDir(), "status.json")
	_, err := Write(project.Record{Key: "demo", RootPath: root}, project.Snapshot{SourceHash: "abc"}, outside)
	if err == nil {
		t.Fatalf("expected absolute target path to be rejected")
	}
	if _, statErr := os.Stat(outside); !os.IsNotExist(statErr) {
		t.Fatalf("absolute status file should not exist: %v", statErr)
	}
}

func TestWriteRejectsSymlinkedParentEscape(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	link := filepath.Join(root, ".areaflow")
	if err := os.Symlink(outside, link); err != nil {
		t.Skipf("symlink not available: %v", err)
	}

	_, err := Write(project.Record{Key: "demo", RootPath: root}, project.Snapshot{SourceHash: "abc"}, ".areaflow/status.json")
	if err == nil {
		t.Fatalf("expected symlinked parent escape to be rejected")
	}
	if _, statErr := os.Stat(filepath.Join(outside, "status.json")); !os.IsNotExist(statErr) {
		t.Fatalf("symlink escape should not write outside status: %v", statErr)
	}
}
