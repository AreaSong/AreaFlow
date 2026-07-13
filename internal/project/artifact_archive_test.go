package project

import "testing"

func TestBuildArtifactArchivePreviewClassifiesRetention(t *testing.T) {
	record := Record{ID: 1, Key: "areamatrix"}
	result := buildArtifactArchivePreview(record, []ArtifactRecord{
		{ID: 1, ArtifactType: "runner_preview_report", StorageBackend: "local", Metadata: map[string]any{"dry_run": true}},
		{ID: 2, ArtifactType: "source_ref", StorageBackend: "project_reference", SourcePath: "workflow/file.md", Metadata: map[string]any{}},
		{ID: 3, ArtifactType: "release_evidence_bundle", StorageBackend: "local", Metadata: map[string]any{"retention_class": "release"}},
		{ID: 4, ArtifactType: "custom", StorageBackend: "object", Metadata: map[string]any{}},
	}, ArtifactArchivePreviewOptions{})

	if result.Status != "needs_attention" || result.Mode != "metadata_only_archive_preview" {
		t.Fatalf("unexpected preview status: %+v", result)
	}
	if result.Summary.TotalArtifacts != 4 || result.Summary.ArchiveCandidates != 1 || result.Summary.ExternalRefs != 1 || result.Summary.RetainedArtifacts != 1 || result.Summary.NeedsPolicy != 1 {
		t.Fatalf("unexpected summary: %+v", result.Summary)
	}
	if result.ProjectWriteAttempted || result.StorageWriteAttempted || result.ArtifactDeleteAttempted {
		t.Fatalf("preview must not attempt writes or deletes: %+v", result)
	}
	if result.Items[0].ArchiveState != "archive_candidate" || result.Items[1].ArchiveState != "metadata_only_reference" || result.Items[2].ArchiveState != "retained" || result.Items[3].ArchiveState != "needs_policy" {
		t.Fatalf("unexpected item states: %+v", result.Items)
	}
}

func TestFilterArtifactsByArchivePreviewRetentionClass(t *testing.T) {
	artifacts := []ArtifactRecord{
		{ID: 1, ArtifactType: "runner_preview_report", StorageBackend: "local", Metadata: map[string]any{"dry_run": true}},
		{ID: 2, ArtifactType: "release_evidence_bundle", StorageBackend: "local", Metadata: map[string]any{"retention_class": "release"}},
		{ID: 3, ArtifactType: "approval_audit", StorageBackend: "local", Metadata: map[string]any{}},
	}
	if got := filterArtifactsByArchivePreviewRetentionClass(artifacts, ""); len(got) != 3 {
		t.Fatalf("empty retention class should keep all artifacts: %+v", got)
	}
	filtered := filterArtifactsByArchivePreviewRetentionClass(artifacts, "release")
	if len(filtered) != 1 || filtered[0].ID != 2 {
		t.Fatalf("release retention filter mismatch: %+v", filtered)
	}
}

func TestArtifactArchivePreviewRequestHashAndKey(t *testing.T) {
	record := Record{ID: 1, Key: "areamatrix"}
	options := normalizeArtifactArchivePreviewOptions(ArtifactArchivePreviewOptions{
		RetentionClass: " ephemeral ",
		Limit:          10,
		Actor:          " local-user ",
		Reason:         " preview archive ",
	})
	if options.RetentionClass != "ephemeral" || options.Actor != "local-user" || options.Reason != "preview archive" {
		t.Fatalf("unexpected normalized options: %+v", options)
	}
	first, err := artifactArchivePreviewRequestHash(record, options)
	if err != nil {
		t.Fatalf("first hash failed: %v", err)
	}
	second, err := artifactArchivePreviewRequestHash(record, options)
	if err != nil {
		t.Fatalf("second hash failed: %v", err)
	}
	if first != second {
		t.Fatalf("hash should be stable: %s != %s", first, second)
	}
	key := artifactArchivePreviewIdempotencyKey(record, options, first)
	if want := "artifact.archive.preview:areamatrix:ephemeral:10:"; len(key) <= len(want) || key[:len(want)] != want {
		t.Fatalf("unexpected idempotency key: %s", key)
	}
	changed := options
	changed.Reason = "other"
	changedHash, err := artifactArchivePreviewRequestHash(record, changed)
	if err != nil {
		t.Fatalf("changed hash failed: %v", err)
	}
	if first == changedHash {
		t.Fatalf("hash should include reason")
	}
}

func TestArtifactArchivePreviewCommandResponseSafetyFacts(t *testing.T) {
	result := ArtifactArchivePreviewResult{
		Project: Record{ID: 1, Key: "areamatrix"},
		Status:  "needs_attention",
		Mode:    "metadata_only_archive_preview",
		Summary: ArtifactArchivePreviewSummary{TotalArtifacts: 1, ExternalRefs: 1},
		Items: []ArtifactArchivePreviewItem{{
			ArtifactID:     2,
			StorageBackend: "project_reference",
			RetentionClass: "external_ref",
			ArchiveState:   "metadata_only_reference",
			Action:         "keep_metadata_only",
			Decision:       "requires_archive_ownership_decision",
		}},
		EventID:      5,
		AuditEventID: 6,
	}
	response := artifactArchivePreviewCommandResponse(result)
	if response["project_write_attempted"] != false || response["storage_write_attempted"] != false || response["artifact_delete_attempted"] != false {
		t.Fatalf("preview response should record no writes/deletes: %+v", response)
	}
	if response["status"] != "needs_attention" || response["mode"] != "metadata_only_archive_preview" {
		t.Fatalf("unexpected response: %+v", response)
	}
}
