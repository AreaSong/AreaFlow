package project

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"
)

type ArtifactIntegrityOptions struct {
	GeneratedAt time.Time
}

type ArtifactIntegrityCheck struct {
	Artifact ArtifactRecord
	Status   string
	Message  string
	Metadata map[string]any
}

type ArtifactIntegrityReport struct {
	Status           string
	Mode             string
	Project          Record
	CheckedArtifacts int
	PassedArtifacts  int
	WarnArtifacts    int
	FailedArtifacts  int
	SkippedArtifacts int
	Checks           []ArtifactIntegrityCheck
	GeneratedAt      time.Time
}

func (s Store) ArtifactIntegrity(ctx context.Context, record Record, options ArtifactIntegrityOptions) (ArtifactIntegrityReport, error) {
	options = normalizeArtifactIntegrityOptions(options)
	artifacts, err := s.listAllProjectArtifacts(ctx, record.ID)
	if err != nil {
		return ArtifactIntegrityReport{}, err
	}
	return BuildArtifactIntegrityReport(record, artifacts, options), nil
}

func normalizeArtifactIntegrityOptions(options ArtifactIntegrityOptions) ArtifactIntegrityOptions {
	if options.GeneratedAt.IsZero() {
		options.GeneratedAt = time.Now().UTC()
	}
	return options
}

func (s Store) listAllProjectArtifacts(ctx context.Context, projectID int64) ([]ArtifactRecord, error) {
	if projectID <= 0 {
		return nil, fmt.Errorf("project id is required")
	}
	rows, err := s.pool.Query(ctx, `
SELECT id, project_id, COALESCE(workflow_version_id, 0), COALESCE(workflow_item_id, 0),
       artifact_type, storage_backend, uri, COALESCE(source_path, ''), COALESCE(sha256, ''),
       COALESCE(size_bytes, 0), COALESCE(content_type, ''), metadata, created_at
FROM artifacts
WHERE project_id = $1
ORDER BY created_at DESC, id DESC`,
		projectID,
	)
	if err != nil {
		return nil, fmt.Errorf("list artifacts for integrity check: %w", err)
	}
	defer rows.Close()
	artifacts := []ArtifactRecord{}
	for rows.Next() {
		record, err := scanArtifactRecord(rows)
		if err != nil {
			return nil, err
		}
		artifacts = append(artifacts, record)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate artifacts for integrity check: %w", err)
	}
	return artifacts, nil
}

func BuildArtifactIntegrityReport(record Record, artifacts []ArtifactRecord, options ArtifactIntegrityOptions) ArtifactIntegrityReport {
	options = normalizeArtifactIntegrityOptions(options)
	report := ArtifactIntegrityReport{
		Status:      "pass",
		Mode:        "read_only_artifact_integrity",
		Project:     record,
		Checks:      make([]ArtifactIntegrityCheck, 0, len(artifacts)),
		GeneratedAt: options.GeneratedAt,
	}
	for _, artifact := range artifacts {
		check := checkArtifactIntegrity(artifact)
		report.Checks = append(report.Checks, check)
		report.CheckedArtifacts++
		switch check.Status {
		case "pass":
			report.PassedArtifacts++
		case "warn":
			report.WarnArtifacts++
		case "fail":
			report.FailedArtifacts++
		case "skipped":
			report.SkippedArtifacts++
		}
		reportStatus := check.Status
		if reportStatus == "skipped" {
			reportStatus = "warn"
		}
		if worseArtifactIntegrityStatus(reportStatus, report.Status) {
			report.Status = reportStatus
		}
	}
	return report
}

func checkArtifactIntegrity(artifact ArtifactRecord) ArtifactIntegrityCheck {
	switch artifact.StorageBackend {
	case "local":
		return checkLocalArtifactIntegrity(artifact)
	case "external_project", "project_reference":
		return checkReferencedArtifactIntegrity(artifact)
	case "object":
		return ArtifactIntegrityCheck{
			Artifact: artifact,
			Status:   "skipped",
			Message:  "object storage artifact integrity is not checked by local doctor",
			Metadata: map[string]any{"storage_backend": artifact.StorageBackend, "read_contents": false},
		}
	default:
		return ArtifactIntegrityCheck{
			Artifact: artifact,
			Status:   "warn",
			Message:  "artifact uses an unknown storage backend",
			Metadata: map[string]any{"storage_backend": artifact.StorageBackend, "read_contents": false},
		}
	}
}

func checkLocalArtifactIntegrity(artifact ArtifactRecord) ArtifactIntegrityCheck {
	if strings.TrimSpace(artifact.URI) == "" {
		return ArtifactIntegrityCheck{
			Artifact: artifact,
			Status:   "fail",
			Message:  "local artifact URI is missing",
			Metadata: map[string]any{"read_contents": false},
		}
	}
	file, err := os.Open(artifact.URI)
	if err != nil {
		status := "fail"
		message := "local artifact is not readable"
		if errors.Is(err, os.ErrNotExist) {
			message = "local artifact file is missing"
		}
		return ArtifactIntegrityCheck{
			Artifact: artifact,
			Status:   status,
			Message:  message,
			Metadata: map[string]any{"path": artifact.URI, "error": err.Error(), "read_contents": false},
		}
	}
	defer file.Close()

	hasher := sha256.New()
	size, err := io.Copy(hasher, file)
	if err != nil {
		return ArtifactIntegrityCheck{
			Artifact: artifact,
			Status:   "fail",
			Message:  "local artifact could not be hashed",
			Metadata: map[string]any{"path": artifact.URI, "error": err.Error(), "read_contents": true},
		}
	}
	actualSHA := hex.EncodeToString(hasher.Sum(nil))
	failures := []string{}
	if artifact.SHA256 == "" {
		failures = append(failures, "missing_sha256")
	} else if artifact.SHA256 != actualSHA {
		failures = append(failures, "sha256_mismatch")
	}
	if artifact.SizeBytes <= 0 {
		failures = append(failures, "missing_size_bytes")
	} else if artifact.SizeBytes != size {
		failures = append(failures, "size_mismatch")
	}
	metadata := map[string]any{
		"path":                artifact.URI,
		"expected_sha256":     artifact.SHA256,
		"actual_sha256":       actualSHA,
		"expected_size":       artifact.SizeBytes,
		"actual_size":         size,
		"read_contents":       true,
		"storage_backend":     artifact.StorageBackend,
		"artifact_type":       artifact.ArtifactType,
		"workflow_item_id":    artifact.WorkflowItemID,
		"workflow_version_id": artifact.WorkflowVersionID,
	}
	if len(failures) > 0 {
		metadata["failures"] = failures
		return ArtifactIntegrityCheck{
			Artifact: artifact,
			Status:   "fail",
			Message:  "local artifact metadata does not match stored content",
			Metadata: metadata,
		}
	}
	return ArtifactIntegrityCheck{
		Artifact: artifact,
		Status:   "pass",
		Message:  "local artifact hash and size match metadata",
		Metadata: metadata,
	}
}

func checkReferencedArtifactIntegrity(artifact ArtifactRecord) ArtifactIntegrityCheck {
	missing := []string{}
	if strings.TrimSpace(artifact.URI) == "" {
		missing = append(missing, "uri")
	}
	if strings.TrimSpace(artifact.SourcePath) == "" {
		missing = append(missing, "source_path")
	}
	if strings.TrimSpace(artifact.SHA256) == "" {
		missing = append(missing, "sha256")
	}
	if artifact.SizeBytes <= 0 {
		missing = append(missing, "size_bytes")
	}
	if len(missing) > 0 {
		return ArtifactIntegrityCheck{
			Artifact: artifact,
			Status:   "warn",
			Message:  "referenced project artifact metadata is incomplete",
			Metadata: map[string]any{
				"storage_backend": artifact.StorageBackend,
				"missing":         missing,
				"read_contents":   false,
			},
		}
	}
	return ArtifactIntegrityCheck{
		Artifact: artifact,
		Status:   "skipped",
		Message:  "referenced project artifact content remains in managed project",
		Metadata: map[string]any{
			"storage_backend": artifact.StorageBackend,
			"source_path":     artifact.SourcePath,
			"read_contents":   false,
			"reason":          "project_reference_metadata_only",
		},
	}
}

func worseArtifactIntegrityStatus(candidate string, current string) bool {
	return artifactIntegrityStatusRank(candidate) > artifactIntegrityStatusRank(current)
}

func artifactIntegrityStatusRank(status string) int {
	switch status {
	case "fail":
		return 4
	case "warn":
		return 3
	case "skipped":
		return 2
	case "pass":
		return 1
	default:
		return 0
	}
}
