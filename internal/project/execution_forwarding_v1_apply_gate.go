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

const executionForwardingV1ApplyGateMode = "execution_forwarding_v1_apply_gate_v1"

type ExecutionForwardingV1ApplyGateOptions struct {
	AllowedTaskTypes           []string
	ApprovalID                 string
	ApprovalScope              string
	ReadinessSnapshotHash      string
	ExpectedShimLifecycleState string
	LegacyNonWriteProofID      string
	RollbackPlanID             string
	ProtectedPathFingerprintID string
	IdempotencyKey             string
	AuditCorrelationID         string
	FailureMode                string
	ExplicitApproval           bool
	ApprovalActor              string
	ApprovalReason             string
	GeneratedAt                time.Time
}

type ExecutionForwardingV1ApplyGate struct {
	Project                 Record
	Status                  string
	Mode                    string
	Decision                string
	Message                 string
	ApplyPreview            ExecutionForwardingV1ApplyPreview
	Items                   []ExecutionForwardingV1ApplyGateItem
	RequiredPacketFields    []string
	RequiredCapabilities    []string
	AllowedTaskTypes        []string
	TargetCommandTypes      []string
	BlockedTaskTypes        []string
	ForbiddenActions        []string
	FailClosedFields        []string
	RequiredProofFacts      []string
	SafetyFacts             map[string]bool
	ApprovalRequired        bool
	ApprovalStatus          string
	ApplyCommandEligible    bool
	ApplyOpen               bool
	CommandRequestCreated   bool
	AreaFlowRunCreated      bool
	TaskLoopRunForwarded    bool
	ProjectWriteAttempted   bool
	ExecutionWriteAttempted bool
	EngineCallAttempted     bool
	GeneratedAt             time.Time
}

type ExecutionForwardingV1ApplyGateItem struct {
	Key              string
	Category         string
	Status           string
	Message          string
	Expected         string
	Actual           string
	RequiredEvidence []string
	BlockedBy        []string
}

func (s Store) ExecutionForwardingV1ApplyGate(ctx context.Context, record Record, options ExecutionForwardingV1ApplyGateOptions) (ExecutionForwardingV1ApplyGate, error) {
	options = normalizeExecutionForwardingV1ApplyGateOptions(options)
	applyPreview, err := s.ExecutionForwardingV1ApplyPreview(ctx, record, ExecutionForwardingV1ApplyPreviewOptions{
		GeneratedAt: options.GeneratedAt,
	})
	if err != nil {
		return ExecutionForwardingV1ApplyGate{}, err
	}
	return BuildExecutionForwardingV1ApplyGate(applyPreview, options), nil
}

func BuildExecutionForwardingV1ApplyGate(applyPreview ExecutionForwardingV1ApplyPreview, options ExecutionForwardingV1ApplyGateOptions) ExecutionForwardingV1ApplyGate {
	options = normalizeExecutionForwardingV1ApplyGateOptions(options)
	expectedHash := ExecutionForwardingV1ReadinessSnapshotHash(applyPreview)
	expectedLegacyNonWriteProofID := executionForwardingV1ExpectedLegacyNonWriteProofID(applyPreview.Readiness, applyPreview.Project.Key)
	expectedRollbackPlanID := executionForwardingV1ExpectedRollbackPlanID(applyPreview.Readiness, applyPreview.Project.Key)
	expectedProtectedPathFingerprintID := executionForwardingV1ExpectedProtectedPathFingerprintID(applyPreview.Readiness, applyPreview.Project.Key)
	allowedTaskTypes := append([]string{}, applyPreview.AllowedTaskTypes...)
	targetCommandTypes := executionForwardingV1TargetCommandTypes(applyPreview.ForwardingTargets)
	blockedTaskTypes := executionForwardingV1BlockedTaskTypes(applyPreview.BlockedTargets)
	gate := ExecutionForwardingV1ApplyGate{
		Project:              applyPreview.Project,
		Status:               "pass",
		Mode:                 executionForwardingV1ApplyGateMode,
		Decision:             "go",
		Message:              "execution forwarding v1 packet is ready for a future protected apply command",
		ApplyPreview:         applyPreview,
		RequiredPacketFields: append([]string{}, applyPreview.ApplyPacketFields...),
		RequiredCapabilities: append([]string{}, applyPreview.RequiredCapabilities...),
		AllowedTaskTypes:     allowedTaskTypes,
		TargetCommandTypes:   targetCommandTypes,
		BlockedTaskTypes:     blockedTaskTypes,
		ForbiddenActions:     append([]string{}, applyPreview.ForbiddenActions...),
		FailClosedFields:     append([]string{}, applyPreview.FailClosedFields...),
		RequiredProofFacts:   append([]string{}, applyPreview.RequiredProofFacts...),
		SafetyFacts:          executionForwardingV1ApplyGateSafetyFacts(),
		ApprovalRequired:     true,
		ApprovalStatus:       "approved",
		GeneratedAt:          options.GeneratedAt,
	}
	gate.Items = append(gate.Items,
		executionForwardingV1ApplyGateItem("readiness_status", "readiness", applyPreview.Readiness.Status == "pass", "overall execution forwarding v1 readiness must pass before apply gate eligibility", "pass", applyPreview.Readiness.Status, []string{"execution_forwarding_v1_readiness_not_pass"}),
		executionForwardingV1ApplyGateItem("allowed_task_types", "packet", reflect.DeepEqual(options.AllowedTaskTypes, normalizeStringList(allowedTaskTypes)), "allowed task types must match the apply preview target policy", strings.Join(allowedTaskTypes, ","), strings.Join(options.AllowedTaskTypes, ","), []string{"allowed_task_types_missing_or_mismatch"}),
		executionForwardingV1ApplyGateItem("target_policy", "policy", executionForwardingV1ReadinessItemStatus(applyPreview.Readiness, "allowed_task_scope") == "pass", "allowed task scope must remain read-only/evidence only", "pass", executionForwardingV1ReadinessItemStatus(applyPreview.Readiness, "allowed_task_scope"), []string{"allowed_task_scope_not_pass"}),
		executionForwardingV1ApplyGateItem("forbidden_targets_fail_closed", "policy", executionForwardingV1AllBlockedTargetsFailClosed(applyPreview.BlockedTargets), "blocked target matrix must fail closed", "fail_closed", executionForwardingV1BlockedTargetsFailureModes(applyPreview.BlockedTargets), []string{"blocked_targets_not_fail_closed"}),
		executionForwardingV1ApplyGateRequiredStringItem("readiness_snapshot_hash", "packet", options.ReadinessSnapshotHash, expectedHash, "readiness snapshot hash must match the current apply preview"),
		executionForwardingV1ApplyGateRequiredStringItem("expected_shim_lifecycle_state", "packet", options.ExpectedShimLifecycleState, "read_only_shim", "packet must explicitly target the read-only shim lifecycle"),
		executionForwardingV1ApplyGateItem("read_only_shim", "readiness", executionForwardingV1ReadinessItemStatus(applyPreview.Readiness, "read_only_shim") == "pass", "read-only shim must be landed before forwarding can open", "pass", executionForwardingV1ReadinessItemStatus(applyPreview.Readiness, "read_only_shim"), []string{"read_only_shim_not_pass"}),
		executionForwardingV1ApplyGateItem("read_only_verify_evidence", "readiness", executionForwardingV1ReadinessItemStatus(applyPreview.Readiness, "read_only_verify_evidence") == "pass", "read-only verify evidence must pass before forwarding can open", "pass", executionForwardingV1ReadinessItemStatus(applyPreview.Readiness, "read_only_verify_evidence"), []string{"read_only_verify_evidence_not_pass"}),
		executionForwardingV1ApplyGateItem("artifact_evidence", "readiness", executionForwardingV1ReadinessItemStatus(applyPreview.Readiness, "artifact_evidence") == "pass", "AreaFlow artifact evidence must pass before forwarding can open", "pass", executionForwardingV1ReadinessItemStatus(applyPreview.Readiness, "artifact_evidence"), []string{"artifact_evidence_not_pass"}),
		executionForwardingV1ApplyGateItem("legacy_non_write_proof", "proof", executionForwardingV1ReadinessItemStatus(applyPreview.Readiness, "legacy_non_write_proof") == "pass", "legacy runner/progress/log/checkpoint non-write proof must pass", "pass", executionForwardingV1ReadinessItemStatus(applyPreview.Readiness, "legacy_non_write_proof"), []string{"legacy_non_write_proof_not_pass"}),
		executionForwardingV1ApplyGateRequiredCurrentProofRefItem("legacy_non_write_proof_id", options.LegacyNonWriteProofID, expectedLegacyNonWriteProofID, "legacy non-write proof id must match the current readiness proof event"),
		executionForwardingV1ApplyGateRequiredCurrentProofRefItem("rollback_plan_id", options.RollbackPlanID, expectedRollbackPlanID, "rollback plan id must match the current rollback proof event"),
		executionForwardingV1ApplyGateItem("rollback_to_read_only_shim", "rollback", executionForwardingV1ReadinessItemStatus(applyPreview.Readiness, "rollback_to_read_only_shim") == "pass", "rollback-to-read-only-shim proof must pass before forwarding can open", "pass", executionForwardingV1ReadinessItemStatus(applyPreview.Readiness, "rollback_to_read_only_shim"), []string{"rollback_to_read_only_shim_not_pass"}),
		executionForwardingV1ApplyGateRequiredCurrentProofRefItem("protected_path_fingerprint_id", options.ProtectedPathFingerprintID, expectedProtectedPathFingerprintID, "protected path fingerprint id must match the current protected path set hash"),
		executionForwardingV1ApplyGateRequiredStringItem("failure_mode", "packet", options.FailureMode, "fail_closed", "forwarding v1 must fail closed"),
		executionForwardingV1ApplyGateNonEmptyItem("approval_id", "approval", options.ApprovalID, "approval id must be recorded"),
		executionForwardingV1ApplyGateRequiredStringItem("approval_scope", "approval", options.ApprovalScope, executionForwardingV1ApprovalScope(), "approval scope must match forwarding v1 read-only/evidence-only policy"),
		executionForwardingV1ApplyGateItem("explicit_approval", "approval", options.ExplicitApproval, "explicit execution forwarding approval must be present", "true", fmt.Sprintf("%t", options.ExplicitApproval), []string{"explicit_execution_forwarding_v1_approval_missing"}),
		executionForwardingV1ApplyGateNonEmptyItem("approval_actor", "approval", options.ApprovalActor, "approval actor must be recorded"),
		executionForwardingV1ApplyGateNonEmptyItem("approval_reason", "approval", options.ApprovalReason, "approval reason must be recorded"),
		executionForwardingV1ApplyGateNonEmptyItem("idempotency_key", "command_api", options.IdempotencyKey, "idempotency key must be recorded"),
		executionForwardingV1ApplyGateNonEmptyItem("audit_correlation_id", "audit", options.AuditCorrelationID, "audit correlation id must be recorded"),
	)
	for _, item := range gate.Items {
		if item.Status != "pass" {
			gate.Status = "blocked"
			gate.Decision = "no_go"
			gate.Message = "execution forwarding v1 apply packet is blocked"
			gate.ApprovalStatus = "missing_or_incomplete"
			break
		}
	}
	gate.ApplyCommandEligible = gate.Status == "pass"
	return gate
}

func normalizeExecutionForwardingV1ApplyGateOptions(options ExecutionForwardingV1ApplyGateOptions) ExecutionForwardingV1ApplyGateOptions {
	options.AllowedTaskTypes = normalizeStringList(options.AllowedTaskTypes)
	options.ApprovalID = strings.TrimSpace(options.ApprovalID)
	options.ApprovalScope = strings.TrimSpace(options.ApprovalScope)
	options.ReadinessSnapshotHash = strings.TrimSpace(options.ReadinessSnapshotHash)
	options.ExpectedShimLifecycleState = strings.TrimSpace(options.ExpectedShimLifecycleState)
	options.LegacyNonWriteProofID = strings.TrimSpace(options.LegacyNonWriteProofID)
	options.RollbackPlanID = strings.TrimSpace(options.RollbackPlanID)
	options.ProtectedPathFingerprintID = strings.TrimSpace(options.ProtectedPathFingerprintID)
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

func ExecutionForwardingV1ReadinessSnapshotHash(applyPreview ExecutionForwardingV1ApplyPreview) string {
	payload, err := json.Marshal(map[string]any{
		"project_key":                applyPreview.Project.Key,
		"readiness_status":           applyPreview.Readiness.Status,
		"readiness_mode":             applyPreview.Readiness.Mode,
		"allowed_task_types":         applyPreview.AllowedTaskTypes,
		"target_command_types":       executionForwardingV1TargetCommandTypes(applyPreview.ForwardingTargets),
		"blocked_task_types":         executionForwardingV1BlockedTaskTypes(applyPreview.BlockedTargets),
		"required_capabilities":      applyPreview.RequiredCapabilities,
		"required_proof_facts":       applyPreview.RequiredProofFacts,
		"forbidden_actions":          applyPreview.ForbiddenActions,
		"read_only_shim":             executionForwardingV1ReadinessItemStatus(applyPreview.Readiness, "read_only_shim"),
		"read_only_verify_evidence":  executionForwardingV1ReadinessItemStatus(applyPreview.Readiness, "read_only_verify_evidence"),
		"artifact_evidence":          executionForwardingV1ReadinessItemStatus(applyPreview.Readiness, "artifact_evidence"),
		"legacy_non_write_proof":     executionForwardingV1ReadinessItemStatus(applyPreview.Readiness, "legacy_non_write_proof"),
		"rollback_to_read_only_shim": executionForwardingV1ReadinessItemStatus(applyPreview.Readiness, "rollback_to_read_only_shim"),
		"legacy_non_write_proof_id":  executionForwardingV1ExpectedLegacyNonWriteProofID(applyPreview.Readiness, applyPreview.Project.Key),
		"rollback_plan_id":           executionForwardingV1ExpectedRollbackPlanID(applyPreview.Readiness, applyPreview.Project.Key),
		"protected_path_fingerprint_id": executionForwardingV1ExpectedProtectedPathFingerprintID(
			applyPreview.Readiness,
			applyPreview.Project.Key,
		),
		"rollback_target": applyPreview.RollbackTarget,
	})
	if err != nil {
		return ""
	}
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:])
}

func executionForwardingV1ApplyGateItem(key string, category string, pass bool, message string, expected string, actual string, blockers []string) ExecutionForwardingV1ApplyGateItem {
	status := "pass"
	if !pass {
		status = "blocked"
	}
	item := ExecutionForwardingV1ApplyGateItem{
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

func executionForwardingV1ApplyGateRequiredStringItem(key string, category string, actual string, expected string, message string) ExecutionForwardingV1ApplyGateItem {
	return executionForwardingV1ApplyGateItem(key, category, actual != "" && actual == expected, message, expected, actual, []string{key + "_missing_or_mismatch"})
}

func executionForwardingV1ApplyGateNonEmptyItem(key string, category string, actual string, message string) ExecutionForwardingV1ApplyGateItem {
	return executionForwardingV1ApplyGateItem(key, category, actual != "", message, "non-empty", actual, []string{key + "_missing"})
}

func executionForwardingV1ApplyGateRequiredCurrentProofRefItem(key string, actual string, expected string, message string) ExecutionForwardingV1ApplyGateItem {
	if expected == "" {
		expected = "current readiness proof ref"
	}
	return executionForwardingV1ApplyGateItem(
		key,
		"proof",
		actual != "" && actual == expected,
		message,
		expected,
		actual,
		[]string{key + "_missing_or_mismatch"},
	)
}

func executionForwardingV1ExpectedLegacyNonWriteProofID(readiness ExecutionForwardingV1Readiness, projectKey string) string {
	eventID := executionForwardingV1ReadinessItemMetadataInt64(readiness, "legacy_non_write_proof", "proof_event_id")
	if eventID == 0 {
		return ""
	}
	return executionForwardingV1ScopedProofRef(projectKey, "legacy_non_write_proof", fmt.Sprintf("%d", eventID))
}

func executionForwardingV1ExpectedRollbackPlanID(readiness ExecutionForwardingV1Readiness, projectKey string) string {
	eventID := executionForwardingV1ReadinessItemMetadataInt64(readiness, "rollback_to_read_only_shim", "proof_event_id")
	if eventID == 0 {
		return ""
	}
	return executionForwardingV1ScopedProofRef(projectKey, "rollback_to_read_only_shim", fmt.Sprintf("%d", eventID))
}

func executionForwardingV1ExpectedProtectedPathFingerprintID(readiness ExecutionForwardingV1Readiness, projectKey string) string {
	protectedPathSetHash := executionForwardingV1ReadinessItemMetadataString(readiness, "legacy_non_write_proof", "protected_path_set_hash")
	if protectedPathSetHash == "" {
		return ""
	}
	return executionForwardingV1ScopedProofRef(projectKey, "protected_path_fingerprint", protectedPathSetHash)
}

func executionForwardingV1ScopedProofRef(projectKey string, evidenceKind string, id string) string {
	projectKey = strings.TrimSpace(projectKey)
	evidenceKind = strings.TrimSpace(evidenceKind)
	id = strings.TrimSpace(id)
	if projectKey == "" || evidenceKind == "" || id == "" {
		return ""
	}
	return projectKey + ":" + evidenceKind + ":" + id
}

func executionForwardingV1ReadinessItemMetadataString(readiness ExecutionForwardingV1Readiness, itemKey string, metadataKey string) string {
	return metadataString(executionForwardingV1ReadinessItemMetadata(readiness, itemKey), metadataKey)
}

func executionForwardingV1ReadinessItemMetadataInt64(readiness ExecutionForwardingV1Readiness, itemKey string, metadataKey string) int64 {
	return metadataInt64(executionForwardingV1ReadinessItemMetadata(readiness, itemKey), metadataKey)
}

func executionForwardingV1ReadinessItemMetadata(readiness ExecutionForwardingV1Readiness, itemKey string) map[string]any {
	for _, item := range readiness.Items {
		if item.Key == itemKey {
			return item.Metadata
		}
	}
	return nil
}

func executionForwardingV1ApplyGateSafetyFacts() map[string]bool {
	return map[string]bool{
		"read_only_gate":                      true,
		"apply_open":                          false,
		"apply_command_eligible_is_not_apply": true,
		"forwarding_v1_apply_open":            false,
		"task_loop_run_forwarded":             false,
		"legacy_task_loop_started":            false,
		"legacy_progress_written":             false,
		"legacy_logs_written":                 false,
		"legacy_checkpoint_written":           false,
		"command_request_created":             false,
		"area_flow_run_created":               false,
		"worker_scheduled":                    false,
		"project_write_attempted":             false,
		"execution_write_attempted":           false,
		"engine_call_attempted":               false,
		"commands_run":                        false,
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

func executionForwardingV1TargetCommandTypes(targets []ExecutionForwardingV1ForwardingTarget) []string {
	values := make([]string, 0, len(targets))
	for _, target := range targets {
		values = append(values, target.TargetCommandType)
	}
	return values
}

func executionForwardingV1BlockedTaskTypes(targets []ExecutionForwardingV1BlockedTarget) []string {
	values := make([]string, 0, len(targets))
	for _, target := range targets {
		values = append(values, target.TaskType)
	}
	return values
}

func executionForwardingV1AllBlockedTargetsFailClosed(targets []ExecutionForwardingV1BlockedTarget) bool {
	for _, target := range targets {
		if target.FailureMode != "fail_closed" {
			return false
		}
	}
	return true
}

func executionForwardingV1BlockedTargetsFailureModes(targets []ExecutionForwardingV1BlockedTarget) string {
	values := make([]string, 0, len(targets))
	for _, target := range targets {
		values = append(values, target.TaskType+"="+target.FailureMode)
	}
	return strings.Join(values, ",")
}

func executionForwardingV1ApprovalScope() string {
	return "execution_forwarding_v1_read_only_evidence_only"
}
