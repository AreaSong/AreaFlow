package project

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"time"
)

const shimApplyGateMode = "shim_apply_gate_v1"

type ShimApplyGateOptions struct {
	AllowedFiles               []string
	ApprovalID                 string
	ApprovalScope              string
	AuthorizationSnapshotHash  string
	ExpectedAuthorizationMode  string
	StatusProjectionPacketID   string
	StatusProjectionGateID     string
	ReadOnlySmokeEvidenceID    string
	DirtyWorktreeReviewID      string
	ProtectedPathFingerprintID string
	RollbackPlanID             string
	IdempotencyKey             string
	AuditCorrelationID         string
	FailureMode                string
	ExplicitApproval           bool
	ApprovalActor              string
	ApprovalReason             string
	GeneratedAt                time.Time
}

type ShimApplyGate struct {
	Project                 Record
	Status                  string
	Mode                    string
	Decision                string
	Message                 string
	Authorization           ShimAuthorizationPacket
	Items                   []ShimApplyGateItem
	RequiredPacketFields    []string
	RequiredCapabilities    []string
	AllowedFiles            []string
	ForbiddenPaths          []string
	ForbiddenActions        []string
	RequiredPreflight       []string
	PostEditVerification    []string
	RollbackScope           []string
	RequiredProofFacts      []string
	SafetyFacts             map[string]bool
	ApprovalRequired        bool
	ApprovalStatus          string
	ApplyCommandEligible    bool
	ApplyOpen               bool
	CommandRequestCreated   bool
	ProjectWriteAttempted   bool
	ExecutionWriteAttempted bool
	EngineCallAttempted     bool
	TaskLoopRunForwarded    bool
	StatusProjectionWritten bool
	AreaMatrixFilesModified bool
	GeneratedAt             time.Time
}

type ShimApplyGateItem struct {
	Key              string
	Category         string
	Status           string
	Message          string
	Expected         string
	Actual           string
	RequiredEvidence []string
	BlockedBy        []string
}

func (s Store) ShimApplyGate(ctx context.Context, record Record, options ShimApplyGateOptions) (ShimApplyGate, error) {
	options = normalizeShimApplyGateOptions(options)
	authorization, err := s.ShimAuthorizationPacket(ctx, record)
	if err != nil {
		return ShimApplyGate{}, err
	}
	return BuildShimApplyGate(authorization, options), nil
}

func BuildShimApplyGate(authorization ShimAuthorizationPacket, options ShimApplyGateOptions) ShimApplyGate {
	options = normalizeShimApplyGateOptions(options)
	allowedFiles := shimAllowedFilePaths(authorization.AllowedFiles)
	blockedReadinessKeys := shimBlockedReadinessKeys(authorization.ReadinessItems)
	gate := ShimApplyGate{
		Project:              authorization.Project,
		Status:               "pass",
		Mode:                 shimApplyGateMode,
		Decision:             "go",
		Message:              "shim apply packet is ready for a future protected AreaMatrix shim edit command",
		Authorization:        authorization,
		RequiredPacketFields: shimApplyRequiredPacketFields(),
		RequiredCapabilities: []string{"project_shim_write"},
		AllowedFiles:         allowedFiles,
		ForbiddenPaths:       append([]string{}, authorization.ForbiddenPaths...),
		ForbiddenActions:     append([]string{}, authorization.ForbiddenActions...),
		RequiredPreflight:    append([]string{}, authorization.RequiredPreflight...),
		PostEditVerification: append([]string{}, authorization.PostEditVerification...),
		RollbackScope:        append([]string{}, authorization.RollbackScope...),
		RequiredProofFacts: []string{
			"real_areamatrix_readonly_smoke",
			"real_areamatrix_status_projection_schema",
			"areamatrix_dirty_worktree_review",
			"status_projection_apply_packet",
			"status_projection_apply_gate",
			"protected_path_fingerprint",
			"rollback_plan",
		},
		SafetyFacts:      shimApplyGateSafetyFacts(),
		ApprovalRequired: true,
		ApprovalStatus:   "approved",
		GeneratedAt:      options.GeneratedAt,
	}
	gate.Items = append(gate.Items,
		shimApplyGateItem("authorization_mode", "authorization", authorization.Mode == "read_only_authorization_packet", "authorization packet must be the read-only shim authorization packet", "read_only_authorization_packet", authorization.Mode, []string{"authorization_mode_mismatch"}),
		shimApplyGateItem("readiness_items_present", "readiness", len(authorization.ReadinessItems) > 0, "authorization packet must carry readiness item facts for machine review", "non-empty", fmt.Sprintf("%d", len(authorization.ReadinessItems)), []string{"readiness_items_missing"}),
		shimApplyGateItem("readiness_blockers", "readiness", shimOnlyExplicitApprovalBlocked(blockedReadinessKeys), "readiness blockers must be limited to explicit edit approval before the apply packet can pass", "none except explicit_edit_approval", strings.Join(blockedReadinessKeys, ","), []string{"shim_readiness_still_blocked"}),
		shimApplyGateItem("allowed_files", "packet", reflect.DeepEqual(normalizeStringList(options.AllowedFiles), normalizeStringList(allowedFiles)), "allowed files must exactly match the shim authorization packet", strings.Join(normalizeStringList(allowedFiles), ","), strings.Join(normalizeStringList(options.AllowedFiles), ","), []string{"allowed_files_missing_or_mismatch"}),
		shimApplyGateItem("forbidden_execution_paths", "policy", containsShimString(authorization.ForbiddenPaths, "workflow/versions/**/execution/**"), "authorization packet must keep execution paths forbidden", "workflow/versions/**/execution/**", strings.Join(authorization.ForbiddenPaths, ","), []string{"execution_paths_not_forbidden"}),
		shimApplyGateItem("task_loop_run_forbidden", "policy", containsShimString(authorization.ForbiddenActions, "task-loop run forwarding"), "task-loop run forwarding must remain forbidden", "task-loop run forwarding", strings.Join(authorization.ForbiddenActions, ","), []string{"task_loop_run_forwarding_not_forbidden"}),
		shimApplyGateItem("status_projection_packet_preflight", "preflight", containsShimString(authorization.RequiredPreflight, "areaflow project status-projection-apply-packet areamatrix --json"), "status projection apply packet preview must be part of the preflight", "present", strings.Join(authorization.RequiredPreflight, ","), []string{"status_projection_apply_packet_preflight_missing"}),
		shimApplyGateItem("status_projection_gate_preflight", "preflight", containsShimString(authorization.RequiredPreflight, "areaflow project status-projection-apply-gate areamatrix --json"), "status projection apply gate must be part of the preflight", "present", strings.Join(authorization.RequiredPreflight, ","), []string{"status_projection_apply_gate_preflight_missing"}),
		shimApplyGateItem("post_edit_task_loop_run_blocked_check", "verification", shimContainsFragment(authorization.PostEditVerification, "./task-loop run") && shimContainsFragment(authorization.PostEditVerification, "blocked"), "post-edit verification must prove task-loop run remains blocked", "./task-loop run blocked", strings.Join(authorization.PostEditVerification, ","), []string{"task_loop_run_blocked_check_missing"}),
		shimApplyGateItem("rollback_status_projection_preimage", "rollback", shimContainsFragment(authorization.RollbackScope, "restore the captured preimage bytes for .areaflow/status.json"), "rollback scope must include status projection preimage restore when projection apply is included", "status projection preimage restore", strings.Join(authorization.RollbackScope, ","), []string{"status_projection_rollback_missing"}),
		shimApplyGateRequiredStringItem("authorization_snapshot_hash", "packet", options.AuthorizationSnapshotHash, ShimAuthorizationSnapshotHash(authorization), "authorization snapshot hash must match the current packet"),
		shimApplyGateRequiredStringItem("expected_authorization_mode", "packet", options.ExpectedAuthorizationMode, "read_only_authorization_packet", "packet must explicitly target the read-only authorization mode"),
		shimApplyGateRequiredProofRefItem("status_projection_packet_id", options.StatusProjectionPacketID, authorization.Project.Key, "status_projection_apply_packet", "status projection apply packet review id must be project-scoped"),
		shimApplyGateRequiredProofRefItem("status_projection_gate_id", options.StatusProjectionGateID, authorization.Project.Key, "status_projection_apply_gate", "status projection apply gate review id must be project-scoped"),
		shimApplyGateRequiredProofRefItem("read_only_smoke_evidence_id", options.ReadOnlySmokeEvidenceID, authorization.Project.Key, "real_areamatrix_readonly_smoke", "real AreaMatrix read-only smoke evidence id must be project-scoped"),
		shimApplyGateRequiredProofRefItem("dirty_worktree_review_id", options.DirtyWorktreeReviewID, authorization.Project.Key, "areamatrix_dirty_worktree_review", "AreaMatrix dirty worktree review id must be project-scoped"),
		shimApplyGateRequiredProofRefItem("protected_path_fingerprint_id", options.ProtectedPathFingerprintID, authorization.Project.Key, "protected_path_fingerprint", "protected path fingerprint id must be project-scoped"),
		shimApplyGateRequiredProofRefItem("rollback_plan_id", options.RollbackPlanID, authorization.Project.Key, "rollback_plan", "rollback plan id must be project-scoped"),
		shimApplyGateRequiredStringItem("failure_mode", "packet", options.FailureMode, "fail_closed", "shim apply must fail closed"),
		shimApplyGateNonEmptyItem("approval_id", "approval", options.ApprovalID, "approval id must be recorded"),
		shimApplyGateRequiredStringItem("approval_scope", "approval", options.ApprovalScope, shimApplyApprovalScope(), "approval scope must match the minimal AreaMatrix shim edit policy"),
		shimApplyGateItem("explicit_approval", "approval", options.ExplicitApproval, "explicit shim edit approval must be present", "true", fmt.Sprintf("%t", options.ExplicitApproval), []string{"explicit_shim_apply_approval_missing"}),
		shimApplyGateNonEmptyItem("approval_actor", "approval", options.ApprovalActor, "approval actor must be recorded"),
		shimApplyGateNonEmptyItem("approval_reason", "approval", options.ApprovalReason, "approval reason must be recorded"),
		shimApplyGateNonEmptyItem("idempotency_key", "command_api", options.IdempotencyKey, "idempotency key must be recorded"),
		shimApplyGateNonEmptyItem("audit_correlation_id", "audit", options.AuditCorrelationID, "audit correlation id must be recorded"),
	)
	for _, item := range gate.Items {
		if item.Status != "pass" {
			gate.Status = "blocked"
			gate.Decision = "no_go"
			gate.Message = "shim apply packet is blocked"
			gate.ApprovalStatus = "missing_or_incomplete"
			break
		}
	}
	gate.ApplyCommandEligible = gate.Status == "pass"
	return gate
}

func normalizeShimApplyGateOptions(options ShimApplyGateOptions) ShimApplyGateOptions {
	options.AllowedFiles = normalizeStringList(options.AllowedFiles)
	options.ApprovalID = strings.TrimSpace(options.ApprovalID)
	options.ApprovalScope = strings.TrimSpace(options.ApprovalScope)
	options.AuthorizationSnapshotHash = strings.TrimSpace(options.AuthorizationSnapshotHash)
	options.ExpectedAuthorizationMode = strings.TrimSpace(options.ExpectedAuthorizationMode)
	options.StatusProjectionPacketID = strings.TrimSpace(options.StatusProjectionPacketID)
	options.StatusProjectionGateID = strings.TrimSpace(options.StatusProjectionGateID)
	options.ReadOnlySmokeEvidenceID = strings.TrimSpace(options.ReadOnlySmokeEvidenceID)
	options.DirtyWorktreeReviewID = strings.TrimSpace(options.DirtyWorktreeReviewID)
	options.ProtectedPathFingerprintID = strings.TrimSpace(options.ProtectedPathFingerprintID)
	options.RollbackPlanID = strings.TrimSpace(options.RollbackPlanID)
	options.IdempotencyKey = strings.TrimSpace(options.IdempotencyKey)
	options.AuditCorrelationID = strings.TrimSpace(options.AuditCorrelationID)
	options.FailureMode = strings.TrimSpace(options.FailureMode)
	options.ApprovalActor = strings.TrimSpace(options.ApprovalActor)
	options.ApprovalReason = strings.TrimSpace(options.ApprovalReason)
	if options.GeneratedAt.IsZero() {
		options.GeneratedAt = time.Now().UTC()
	}
	return options
}

func ShimAuthorizationSnapshotHash(authorization ShimAuthorizationPacket) string {
	payload, err := json.Marshal(map[string]any{
		"project_key":            authorization.Project.Key,
		"mode":                   authorization.Mode,
		"readiness_status":       authorization.ReadinessStatus,
		"readiness_items":        shimReadinessHashItems(authorization.ReadinessItems),
		"allowed_files":          shimAllowedFileHashItems(authorization.AllowedFiles),
		"forbidden_paths":        normalizeStringList(authorization.ForbiddenPaths),
		"forbidden_actions":      normalizeStringList(authorization.ForbiddenActions),
		"required_preflight":     normalizeStringList(authorization.RequiredPreflight),
		"post_edit_verification": normalizeStringList(authorization.PostEditVerification),
		"rollback_scope":         normalizeStringList(authorization.RollbackScope),
		"next_required_approval": authorization.NextRequiredApproval,
	})
	if err != nil {
		return ""
	}
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:])
}

func shimApplyRequiredPacketFields() []string {
	return []string{
		"allowed_files",
		"authorization_snapshot_hash",
		"expected_authorization_mode",
		"status_projection_packet_id",
		"status_projection_gate_id",
		"read_only_smoke_evidence_id",
		"dirty_worktree_review_id",
		"protected_path_fingerprint_id",
		"rollback_plan_id",
		"failure_mode",
		"approval_id",
		"approval_scope",
		"explicit_approval",
		"approval_actor",
		"approval_reason",
		"idempotency_key",
		"audit_correlation_id",
	}
}

func shimApplyApprovalScope() string {
	return "areamatrix_compatibility_shim_files_only_no_execution_cutover"
}

func shimAllowedFilePaths(files []ShimFilePlan) []string {
	values := make([]string, 0, len(files))
	for _, file := range files {
		values = append(values, file.Path)
	}
	return normalizeStringList(values)
}

func shimBlockedReadinessKeys(items []ShimReadinessItem) []string {
	keys := []string{}
	for _, item := range items {
		if item.Status == "blocked" {
			keys = append(keys, item.Key)
		}
	}
	return normalizeStringList(keys)
}

func shimOnlyExplicitApprovalBlocked(keys []string) bool {
	keys = normalizeStringList(keys)
	return len(keys) == 1 && keys[0] == "explicit_edit_approval"
}

func shimContainsFragment(values []string, fragment string) bool {
	for _, value := range values {
		if strings.Contains(value, fragment) {
			return true
		}
	}
	return false
}

func shimReadinessHashItems(items []ShimReadinessItem) []map[string]string {
	out := make([]map[string]string, 0, len(items))
	for _, item := range items {
		out = append(out, map[string]string{
			"key":     item.Key,
			"status":  item.Status,
			"message": item.Message,
		})
	}
	return out
}

func shimAllowedFileHashItems(files []ShimFilePlan) []map[string]any {
	out := make([]map[string]any, 0, len(files))
	for _, file := range files {
		out = append(out, map[string]any{
			"path":     file.Path,
			"action":   file.Action,
			"required": file.Required,
			"boundary": file.Boundary,
		})
	}
	return out
}

func shimApplyGateItem(key string, category string, pass bool, message string, expected string, actual string, blockers []string) ShimApplyGateItem {
	status := "pass"
	if !pass {
		status = "blocked"
	}
	item := ShimApplyGateItem{
		Key:      key,
		Category: category,
		Status:   status,
		Message:  message,
		Expected: expected,
		Actual:   actual,
	}
	if !pass {
		item.BlockedBy = uniqueStrings(blockers)
	}
	return item
}

func shimApplyGateRequiredStringItem(key string, category string, actual string, expected string, message string) ShimApplyGateItem {
	return shimApplyGateItem(key, category, actual != "" && actual == expected, message, expected, actual, []string{key + "_missing_or_mismatch"})
}

func shimApplyGateNonEmptyItem(key string, category string, actual string, message string) ShimApplyGateItem {
	return shimApplyGateItem(key, category, actual != "", message, "non-empty", actual, []string{key + "_missing"})
}

func shimApplyGateRequiredProofRefItem(key string, actual string, projectKey string, evidenceKind string, message string) ShimApplyGateItem {
	return shimApplyGateItem(
		key,
		"proof",
		shimScopedProofRefMatches(actual, projectKey, evidenceKind),
		message,
		shimScopedProofRefPrefix(projectKey, evidenceKind)+"<id>",
		actual,
		[]string{key + "_missing_or_unscoped"},
	)
}

func shimScopedProofRefMatches(value string, projectKey string, evidenceKind string) bool {
	prefix := shimScopedProofRefPrefix(projectKey, evidenceKind)
	return value != "" && strings.HasPrefix(value, prefix) && strings.TrimSpace(strings.TrimPrefix(value, prefix)) != ""
}

func shimScopedProofRefPrefix(projectKey string, evidenceKind string) string {
	return strings.TrimSpace(projectKey) + ":" + evidenceKind + ":"
}

func shimApplyGateSafetyFacts() map[string]bool {
	return map[string]bool{
		"read_only_gate":                      true,
		"apply_open":                          false,
		"apply_command_eligible_is_not_apply": true,
		"command_request_created":             false,
		"project_write_attempted":             false,
		"execution_write_attempted":           false,
		"task_loop_run_forwarded":             false,
		"status_projection_written":           false,
		"area_matrix_files_modified":          false,
		"engine_call_attempted":               false,
		"commands_run":                        false,
		"worker_scheduled":                    false,
		"secrets_resolved":                    false,
		"network_used":                        false,
		"source_write_open":                   false,
		"generated_retained_write_open":       false,
		"repair_apply_open":                   false,
		"checkpoint_apply_open":               false,
		"publish_apply_open":                  false,
		"restore_apply_open":                  false,
		"areamatrix_protected_paths_touched":  false,
	}
}
