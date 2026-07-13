package project

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

var taskMatrixProofSourcePaths = []string{
	"tasks/backlog/0-100-platform-backlog.md",
	"docs/development/task-backlog-status-audit.md",
}

func TaskMatrixCurrentBinding() (map[string]any, error) {
	root, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("get task matrix binding root: %w", err)
	}
	return TaskMatrixCurrentBindingForRoot(root)
}

func TaskMatrixCurrentBindingForRoot(root string) (map[string]any, error) {
	backlogHash, err := taskMatrixFileSHA256(root, taskMatrixProofSourcePaths[0])
	if err != nil {
		return nil, err
	}
	statusAuditHash, err := taskMatrixFileSHA256(root, taskMatrixProofSourcePaths[1])
	if err != nil {
		return nil, err
	}
	return taskMatrixProofBindingMetadata(backlogHash, statusAuditHash, 0, 0, 0, true, nil), nil
}

func taskMatrixFileSHA256(root string, relativePath string) (string, error) {
	cleanPath := filepath.Clean(filepath.FromSlash(relativePath))
	if filepath.IsAbs(cleanPath) || cleanPath == ".." || strings.HasPrefix(cleanPath, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("task matrix binding path escapes root: %s", relativePath)
	}
	content, err := os.ReadFile(filepath.Join(root, cleanPath))
	if err != nil {
		return "", fmt.Errorf("read task matrix binding source %s: %w", relativePath, err)
	}
	sum := sha256.Sum256(content)
	return hex.EncodeToString(sum[:]), nil
}

func taskMatrixProofBindingMetadata(backlogHash string, statusAuditHash string, plannedCount int64, missingEvidenceCount int64, blockedCount int64, pass bool, blockers []string) map[string]any {
	status := "fail"
	if pass {
		status = "pass"
	}
	return map[string]any{
		"task_matrix_binding_status":              status,
		"task_matrix_binding_blockers":            uniqueStrings(blockers),
		"task_matrix_source_paths":                append([]string{}, taskMatrixProofSourcePaths...),
		"task_matrix_source_set_hash":             taskMatrixProofSourceSetHash(backlogHash, statusAuditHash),
		"task_backlog_hash":                       backlogHash,
		"task_status_audit_hash":                  statusAuditHash,
		"planned_v1_required_task_count":          plannedCount,
		"missing_evidence_v1_required_task_count": missingEvidenceCount,
		"blocked_v1_required_task_count":          blockedCount,
	}
}

func taskMatrixProofSourceSetHash(backlogHash string, statusAuditHash string) string {
	payload, err := json.Marshal(map[string]any{
		"source_paths":            taskMatrixProofSourcePaths,
		"task_backlog_hash":       backlogHash,
		"task_status_audit_hash":  statusAuditHash,
		"required_count_contract": "planned/missing_evidence/blocked_v1_required_counts_must_be_zero",
	})
	if err != nil {
		return ""
	}
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:])
}

func addTaskMatrixProofBindingMetadata(metadata map[string]any, options RecordTaskMatrixProofOptions) {
	blockers := taskMatrixProofOptionsBindingBlockers(options)
	pass := options.ProofStatus == "complete" && len(blockers) == 0
	binding := taskMatrixProofBindingMetadata(
		options.TaskBacklogHash,
		options.TaskStatusAuditHash,
		options.PlannedV1RequiredTaskCount,
		options.MissingEvidenceV1RequiredTaskCount,
		options.BlockedV1RequiredTaskCount,
		pass,
		blockers,
	)
	if options.ProofStatus != "complete" && len(blockers) == 0 {
		binding["task_matrix_binding_status"] = "not_required"
	}
	for key, value := range binding {
		metadata[key] = value
	}
}

func taskMatrixProofOptionsBindingBlockers(options RecordTaskMatrixProofOptions) []string {
	metadata := taskMatrixProofBindingMetadata(
		options.TaskBacklogHash,
		options.TaskStatusAuditHash,
		options.PlannedV1RequiredTaskCount,
		options.MissingEvidenceV1RequiredTaskCount,
		options.BlockedV1RequiredTaskCount,
		true,
		nil,
	)
	if options.TaskMatrixSourceSetHash != "" {
		metadata["task_matrix_source_set_hash"] = options.TaskMatrixSourceSetHash
	}
	if !options.PlannedV1RequiredTaskCountSet {
		metadata["planned_v1_required_task_count_missing"] = true
	}
	if !options.MissingEvidenceV1RequiredTaskCountSet {
		metadata["missing_evidence_v1_required_task_count_missing"] = true
	}
	if !options.BlockedV1RequiredTaskCountSet {
		metadata["blocked_v1_required_task_count_missing"] = true
	}
	return taskMatrixProofMetadataBindingBlockers(metadata)
}

func taskMatrixProofMetadataBindingBlockers(metadata map[string]any) []string {
	blockers := []string{}
	if metadataString(metadata, "task_matrix_binding_status") != "pass" {
		blockers = append(blockers, "task_matrix_binding_status_not_pass")
	}
	if !sameNormalizedStrings(metadataStringSlice(metadata, "task_matrix_source_paths"), taskMatrixProofSourcePaths) {
		blockers = append(blockers, "task_matrix_source_paths_missing_or_mismatch")
	}
	backlogHash := metadataString(metadata, "task_backlog_hash")
	if !looksLikeSHA256(backlogHash) {
		blockers = append(blockers, "task_backlog_hash_missing_or_invalid")
	}
	statusAuditHash := metadataString(metadata, "task_status_audit_hash")
	if !looksLikeSHA256(statusAuditHash) {
		blockers = append(blockers, "task_status_audit_hash_missing_or_invalid")
	}
	expectedSourceSetHash := taskMatrixProofSourceSetHash(backlogHash, statusAuditHash)
	if !looksLikeSHA256(metadataString(metadata, "task_matrix_source_set_hash")) ||
		metadataString(metadata, "task_matrix_source_set_hash") != expectedSourceSetHash {
		blockers = append(blockers, "task_matrix_source_set_hash_missing_or_mismatch")
	}
	if metadataBool(metadata, "planned_v1_required_task_count_missing") {
		blockers = append(blockers, "planned_v1_required_task_count_missing")
	} else if metadataInt64(metadata, "planned_v1_required_task_count") != 0 {
		blockers = append(blockers, "planned_v1_required_task_count_nonzero")
	}
	if metadataBool(metadata, "missing_evidence_v1_required_task_count_missing") {
		blockers = append(blockers, "missing_evidence_v1_required_task_count_missing")
	} else if metadataInt64(metadata, "missing_evidence_v1_required_task_count") != 0 {
		blockers = append(blockers, "missing_evidence_v1_required_task_count_nonzero")
	}
	if metadataBool(metadata, "blocked_v1_required_task_count_missing") {
		blockers = append(blockers, "blocked_v1_required_task_count_missing")
	} else if metadataInt64(metadata, "blocked_v1_required_task_count") != 0 {
		blockers = append(blockers, "blocked_v1_required_task_count_nonzero")
	}
	return uniqueStrings(blockers)
}

func taskMatrixProofCurrentBindingBlockers(proofMetadata map[string]any, currentBinding map[string]any) []string {
	blockers := taskMatrixProofMetadataBindingBlockers(proofMetadata)
	if len(blockers) > 0 {
		return blockers
	}
	for _, key := range []string{
		"task_matrix_source_set_hash",
		"task_backlog_hash",
		"task_status_audit_hash",
	} {
		if metadataString(proofMetadata, key) != metadataString(currentBinding, key) {
			blockers = append(blockers, key+"_current_mismatch")
		}
	}
	for _, key := range []string{
		"planned_v1_required_task_count",
		"missing_evidence_v1_required_task_count",
		"blocked_v1_required_task_count",
	} {
		if metadataInt64(proofMetadata, key) != metadataInt64(currentBinding, key) {
			blockers = append(blockers, key+"_current_mismatch")
		}
	}
	if !sameNormalizedStrings(metadataStringSlice(proofMetadata, "task_matrix_source_paths"), metadataStringSlice(currentBinding, "task_matrix_source_paths")) {
		blockers = append(blockers, "task_matrix_source_paths_current_mismatch")
	}
	return uniqueStrings(blockers)
}

func looksLikeSHA256(value string) bool {
	if len(value) != 64 {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}
