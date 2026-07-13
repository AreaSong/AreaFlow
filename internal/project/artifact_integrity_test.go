package project

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestBuildArtifactIntegrityReportWarnsForReferencedArtifacts(t *testing.T) {
	created := time.Date(2026, 6, 30, 10, 0, 0, 0, time.UTC)
	report := BuildArtifactIntegrityReport(
		Record{ID: 1, Key: "areamatrix"},
		[]ArtifactRecord{
			{
				ID:             7,
				ProjectID:      1,
				ArtifactType:   "source_ref",
				StorageBackend: "external_project",
				URI:            "workflow/versions/v1-mvp/execution/progress.json",
				SourcePath:     "workflow/versions/v1-mvp/execution/progress.json",
				SHA256:         "abc123",
				SizeBytes:      42,
				ContentType:    "application/json",
			},
		},
		ArtifactIntegrityOptions{GeneratedAt: created},
	)

	if report.Status != "warn" || report.Mode != "read_only_artifact_integrity" {
		t.Fatalf("unexpected report: %+v", report)
	}
	if report.CheckedArtifacts != 1 || report.SkippedArtifacts != 1 {
		t.Fatalf("unexpected counters: %+v", report)
	}
	if report.Checks[0].Status != "skipped" {
		t.Fatalf("referenced artifact check = %+v, want skipped", report.Checks[0])
	}
	if !report.GeneratedAt.Equal(created) {
		t.Fatalf("generated_at = %s, want %s", report.GeneratedAt, created)
	}
}

func TestBuildArtifactIntegrityReportPassesLocalArtifact(t *testing.T) {
	path, sum, size := writeIntegrityTestFile(t, "report")
	report := BuildArtifactIntegrityReport(
		Record{ID: 1, Key: "areamatrix"},
		[]ArtifactRecord{
			{
				ID:             8,
				ProjectID:      1,
				ArtifactType:   "runner_preview_report",
				StorageBackend: "local",
				URI:            path,
				SourcePath:     "v2/runner-preview/report.json",
				SHA256:         sum,
				SizeBytes:      size,
				ContentType:    "application/json",
			},
		},
		ArtifactIntegrityOptions{},
	)

	if report.Status != "pass" || report.PassedArtifacts != 1 || report.FailedArtifacts != 0 {
		t.Fatalf("unexpected report: %+v", report)
	}
	if report.Checks[0].Metadata["actual_sha256"] != sum {
		t.Fatalf("unexpected metadata: %+v", report.Checks[0].Metadata)
	}
}

func TestBuildArtifactIntegrityReportFailsLocalHashMismatch(t *testing.T) {
	path, _, size := writeIntegrityTestFile(t, "report")
	report := BuildArtifactIntegrityReport(
		Record{ID: 1, Key: "areamatrix"},
		[]ArtifactRecord{
			{
				ID:             9,
				ProjectID:      1,
				ArtifactType:   "runner_preview_report",
				StorageBackend: "local",
				URI:            path,
				SourcePath:     "v2/runner-preview/report.json",
				SHA256:         "not-the-hash",
				SizeBytes:      size,
				ContentType:    "application/json",
			},
		},
		ArtifactIntegrityOptions{},
	)

	if report.Status != "fail" || report.FailedArtifacts != 1 {
		t.Fatalf("unexpected report: %+v", report)
	}
	if report.Checks[0].Status != "fail" {
		t.Fatalf("unexpected check: %+v", report.Checks[0])
	}
}

func TestReadArtifactContentPassesLocalArtifact(t *testing.T) {
	path, sum, size := writeIntegrityTestFile(t, `{"ok":true}`)
	content, err := ReadArtifactContent(ArtifactRecord{
		ID:             10,
		ArtifactType:   "runner_preview_report",
		StorageBackend: "local",
		URI:            path,
		SHA256:         sum,
		SizeBytes:      size,
		ContentType:    "application/json",
	})
	if err != nil {
		t.Fatalf("read artifact content: %v", err)
	}
	if string(content.Content) != `{"ok":true}` || content.ContentType != "application/json" {
		t.Fatalf("unexpected content: %+v", content)
	}
	if content.Artifact.ID != 10 {
		t.Fatalf("artifact id = %d, want 10", content.Artifact.ID)
	}
}

func TestReadArtifactContentRejectsProjectReference(t *testing.T) {
	_, err := ReadArtifactContent(ArtifactRecord{
		ID:             11,
		ArtifactType:   "source_ref",
		StorageBackend: "project_reference",
		URI:            "workflow/file.md",
		SourcePath:     "workflow/file.md",
	})
	if !errors.Is(err, ErrArtifactContentUnavailable) {
		t.Fatalf("error = %v, want ErrArtifactContentUnavailable", err)
	}
}

func TestReadArtifactContentDetectsHashMismatch(t *testing.T) {
	path, _, size := writeIntegrityTestFile(t, "report")
	_, err := ReadArtifactContent(ArtifactRecord{
		ID:             12,
		ArtifactType:   "runner_preview_report",
		StorageBackend: "local",
		URI:            path,
		SHA256:         "not-the-hash",
		SizeBytes:      size,
		ContentType:    "application/json",
	})
	if !errors.Is(err, ErrArtifactContentMismatch) {
		t.Fatalf("error = %v, want ErrArtifactContentMismatch", err)
	}
}

func writeIntegrityTestFile(t *testing.T, content string) (string, string, int64) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "artifact.json")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write test artifact: %v", err)
	}
	sum := sha256.Sum256([]byte(content))
	return path, hex.EncodeToString(sum[:]), int64(len(content))
}
