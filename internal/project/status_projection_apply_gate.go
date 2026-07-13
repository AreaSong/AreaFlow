package project

import (
	"context"
	"fmt"
	"strings"
	"time"
)

const statusProjectionApplyGateMode = "status_projection_apply_gate_v1"
const StatusProjectionApplyRequiredApprovalReason = "授权执行 Package A，只允许写 AreaMatrix .areaflow/status.json"

type StatusProjectionApplyGateOptions struct {
	TargetURI                      string
	ExpectedBeforeExists           *bool
	ExpectedBeforeSHA256           string
	ExpectedBeforeSizeBytes        *int64
	SourceHash                     string
	SchemaURI                      string
	ValidatorPreflight             string
	ProtectedPathCheck             string
	ProtectedPathFingerprintSHA256 string
	RollbackAction                 string
	AcceptedPreimageSchemaStatus   string
	ExplicitApproval               bool
	ApprovalActor                  string
	ApprovalReason                 string
	GeneratedAt                    time.Time
}

type StatusProjectionApplyGate struct {
	Project                        Record
	Status                         string
	Mode                           string
	ClaimScope                     string
	NotReal100                     bool
	Decision                       string
	Message                        string
	TargetURI                      string
	TargetPath                     string
	Authorization                  StatusProjectionAuthorizationPreview
	Items                          []StatusProjectionApplyGateItem
	RequiredPacketFields           []string
	RequiredCapabilities           []string
	RequiredAuthorizationPhrase    string
	ProtectedPaths                 []string
	ForbiddenActions               []string
	SafetyFacts                    map[string]bool
	ApplyCommandEligible           bool
	ApplyCommandEligibleIsNotApply bool
	RequiresSeparateApplyCommand   bool
	ApprovalRequired               bool
	ApprovalStatus                 string
	ProjectWriteAttempted          bool
	ExecutionWriteAttempted        bool
	EngineCallAttempted            bool
	CommandRequestCreated          bool
	StatusProjectionWritten        bool
	GeneratedAt                    time.Time
}

type StatusProjectionApplyGateItem struct {
	Key              string
	Category         string
	Status           string
	Message          string
	Expected         string
	Actual           string
	RequiredEvidence []string
	BlockedBy        []string
}

func (s Store) StatusProjectionApplyGate(ctx context.Context, record Record, options StatusProjectionApplyGateOptions) (StatusProjectionApplyGate, error) {
	options = normalizeStatusProjectionApplyGateOptions(options)
	authorization, err := s.StatusProjectionAuthorizationPreview(ctx, record, StatusProjectionAuthorizationPreviewOptions{
		TargetURI:   options.TargetURI,
		GeneratedAt: options.GeneratedAt,
	})
	if err != nil {
		return StatusProjectionApplyGate{}, err
	}
	return BuildStatusProjectionApplyGate(authorization, options), nil
}

func normalizeStatusProjectionApplyGateOptions(options StatusProjectionApplyGateOptions) StatusProjectionApplyGateOptions {
	options.TargetURI = strings.TrimSpace(options.TargetURI)
	options.ExpectedBeforeSHA256 = strings.TrimSpace(options.ExpectedBeforeSHA256)
	options.SourceHash = strings.TrimSpace(options.SourceHash)
	options.SchemaURI = strings.TrimSpace(options.SchemaURI)
	options.ValidatorPreflight = strings.TrimSpace(options.ValidatorPreflight)
	options.ProtectedPathCheck = strings.TrimSpace(options.ProtectedPathCheck)
	options.ProtectedPathFingerprintSHA256 = strings.TrimSpace(options.ProtectedPathFingerprintSHA256)
	options.RollbackAction = strings.TrimSpace(options.RollbackAction)
	options.AcceptedPreimageSchemaStatus = strings.TrimSpace(options.AcceptedPreimageSchemaStatus)
	options.ApprovalActor = strings.TrimSpace(options.ApprovalActor)
	options.ApprovalReason = strings.TrimSpace(options.ApprovalReason)
	if options.TargetURI == "" {
		options.TargetURI = ".areaflow/status.json"
	}
	if options.GeneratedAt.IsZero() {
		options.GeneratedAt = time.Now().UTC()
	}
	return options
}

func BuildStatusProjectionApplyGate(authorization StatusProjectionAuthorizationPreview, options StatusProjectionApplyGateOptions) StatusProjectionApplyGate {
	options = normalizeStatusProjectionApplyGateOptions(options)
	approvalReasonItem := statusProjectionNonEmptyItem("approval_reason", "approval", options.ApprovalReason, "approval reason must be recorded")
	if requiredPhrase := statusProjectionRequiredAuthorizationPhrase(authorization); requiredPhrase != "" {
		approvalReasonItem = statusProjectionRequiredStringItem("approval_reason", "approval", options.ApprovalReason, requiredPhrase, "approval reason must match the exact Package A authorization phrase")
	}
	gate := StatusProjectionApplyGate{
		Project:                        authorization.Project,
		Status:                         "pass",
		Mode:                           statusProjectionApplyGateMode,
		ClaimScope:                     statusProjectionClaimScope,
		NotReal100:                     true,
		Decision:                       "go",
		Message:                        "status projection apply packet is ready for the protected apply command",
		TargetURI:                      authorization.TargetURI,
		TargetPath:                     authorization.TargetPath,
		Authorization:                  authorization,
		RequiredPacketFields:           append([]string{}, authorization.RequiredPacketFields...),
		RequiredCapabilities:           append([]string{}, authorization.RequiredCapabilities...),
		RequiredAuthorizationPhrase:    statusProjectionRequiredAuthorizationPhrase(authorization),
		ProtectedPaths:                 append([]string{}, authorization.ProtectedPaths...),
		ForbiddenActions:               append([]string{}, authorization.ForbiddenActions...),
		SafetyFacts:                    statusProjectionApplyGateSafetyFacts(),
		ApplyCommandEligibleIsNotApply: true,
		RequiresSeparateApplyCommand:   true,
		ApprovalRequired:               true,
		ApprovalStatus:                 "approved",
		GeneratedAt:                    options.GeneratedAt,
	}
	gate.Items = append(gate.Items,
		statusProjectionApplyGateItem("authorization_preview", "authorization", authorization.Status != "blocked", "authorization preview must not be blocked", "not blocked", authorization.Status, authorization.BlockedBy),
		statusProjectionApplyGateItem("target_supported", "target", authorization.TargetKind == "project_status_json", "target must be .areaflow/status.json stable projection", "project_status_json", authorization.TargetKind, []string{"unsupported_status_projection_target"}),
		statusProjectionApplyGateItem("permission_allowed", "permission", authorization.Permission.Allowed, "write_status capability and target path must be allowed", "allowed", authorization.Permission.Reason, []string{authorization.Permission.Reason}),
		statusProjectionRequiredStringItem("schema_uri", "packet", options.SchemaURI, authorization.SchemaURI, "schema URI must match authorization preview"),
		statusProjectionRequiredStringItem("source_snapshot_hash", "packet", options.SourceHash, authorization.SourceHash, "source snapshot hash must match latest imported snapshot"),
		statusProjectionRequiredStringItem("validator_preflight", "packet", options.ValidatorPreflight, authorization.ValidatorPreflight, "validator preflight must match authorization preview"),
		statusProjectionRequiredStringItem("protected_path_check", "packet", options.ProtectedPathCheck, statusProjectionProtectedPathCheck(authorization.Project), "protected path check must match authorization preview"),
		statusProjectionRequiredStringItem("protected_path_fingerprint_sha256", "packet", options.ProtectedPathFingerprintSHA256, authorization.ProtectedPathFingerprintSHA256, "protected path fingerprint must match authorization preview"),
		statusProjectionRequiredStringItem("accepted_preimage_schema_status", "packet", options.AcceptedPreimageSchemaStatus, authorization.Preimage.SchemaStatus, "packet must explicitly accept the current target schema status"),
		statusProjectionRequiredStringItem("rollback_action", "packet", options.RollbackAction, statusProjectionRollbackAction(authorization.Preimage), "rollback action must match authorization preview"),
		statusProjectionExpectedBoolItem("expected_before_exists", options.ExpectedBeforeExists, authorization.Preimage.Exists),
		statusProjectionExpectedInt64Item("expected_before_size_bytes", options.ExpectedBeforeSizeBytes, authorization.Preimage.SizeBytes),
		statusProjectionExpectedSHAItem("expected_before_sha256", options.ExpectedBeforeSHA256, authorization.Preimage),
		statusProjectionApplyGateItem("explicit_approval", "approval", options.ExplicitApproval, "explicit approval must be present before apply", "true", fmt.Sprintf("%t", options.ExplicitApproval), []string{"explicit_status_projection_apply_approval_missing"}),
		statusProjectionNonEmptyItem("approval_actor", "approval", options.ApprovalActor, "approval actor must be recorded"),
		approvalReasonItem,
	)
	for _, item := range gate.Items {
		if item.Status != "pass" {
			gate.Status = "blocked"
			gate.Decision = "no_go"
			gate.Message = "status projection apply packet is blocked"
			gate.ApprovalStatus = "missing_or_incomplete"
			break
		}
	}
	gate.ApplyCommandEligible = gate.Status == "pass"
	return gate
}

func statusProjectionRequiredAuthorizationPhrase(authorization StatusProjectionAuthorizationPreview) string {
	if statusProjectionApplyRequiresExactApprovalReason(authorization) {
		return StatusProjectionApplyRequiredApprovalReason
	}
	return ""
}

func statusProjectionApplyRequiresExactApprovalReason(authorization StatusProjectionAuthorizationPreview) bool {
	return authorization.Project.Key == completionAuditTargetProjectKey &&
		strings.TrimSpace(authorization.Project.RootPath) == completionAuditTargetProjectRoot &&
		strings.TrimSpace(authorization.TargetURI) == ".areaflow/status.json" &&
		authorization.TargetKind == "project_status_json"
}

func statusProjectionApplyGateItem(key string, category string, pass bool, message string, expected string, actual string, blockers []string) StatusProjectionApplyGateItem {
	status := "pass"
	if !pass {
		status = "blocked"
	}
	return StatusProjectionApplyGateItem{
		Key:      key,
		Category: category,
		Status:   status,
		Message:  message,
		Expected: expected,
		Actual:   actual,
		BlockedBy: func() []string {
			if pass {
				return nil
			}
			return uniqueStrings(blockers)
		}(),
	}
}

func statusProjectionRequiredStringItem(key string, category string, actual string, expected string, message string) StatusProjectionApplyGateItem {
	blockers := []string{key + "_missing_or_mismatch"}
	pass := actual != "" && actual == expected
	return statusProjectionApplyGateItem(key, category, pass, message, expected, actual, blockers)
}

func statusProjectionNonEmptyItem(key string, category string, actual string, message string) StatusProjectionApplyGateItem {
	return statusProjectionApplyGateItem(key, category, actual != "", message, "non-empty", actual, []string{key + "_missing"})
}

func statusProjectionExpectedBoolItem(key string, actual *bool, expected bool) StatusProjectionApplyGateItem {
	actualValue := "<missing>"
	pass := false
	if actual != nil {
		actualValue = fmt.Sprintf("%t", *actual)
		pass = *actual == expected
	}
	return statusProjectionApplyGateItem(key, "preimage", pass, "expected-before exists value must match current preimage", fmt.Sprintf("%t", expected), actualValue, []string{key + "_missing_or_mismatch"})
}

func statusProjectionExpectedInt64Item(key string, actual *int64, expected int64) StatusProjectionApplyGateItem {
	actualValue := "<missing>"
	pass := false
	if actual != nil {
		actualValue = fmt.Sprintf("%d", *actual)
		pass = *actual == expected
	}
	return statusProjectionApplyGateItem(key, "preimage", pass, "expected-before size must match current preimage", fmt.Sprintf("%d", expected), actualValue, []string{key + "_missing_or_mismatch"})
}

func statusProjectionExpectedSHAItem(key string, actual string, preimage StatusProjectionPreimage) StatusProjectionApplyGateItem {
	expected := preimage.SHA256
	if !preimage.Exists {
		return statusProjectionApplyGateItem(key, "preimage", actual == "", "expected-before sha256 must be empty when target is absent", "", actual, []string{key + "_must_be_empty_when_missing"})
	}
	return statusProjectionRequiredStringItem(key, "preimage", actual, expected, "expected-before sha256 must match current preimage")
}

func statusProjectionApplyGateSafetyFacts() map[string]bool {
	return map[string]bool{
		"read_only_gate":                      true,
		"apply_open":                          false,
		"apply_command_eligible_is_not_apply": true,
		"command_request_created":             false,
		"status_projection_written":           false,
		"project_write_attempted":             false,
		"execution_write_attempted":           false,
		"engine_call_attempted":               false,
		"commands_run":                        false,
		"worker_scheduled":                    false,
		"secrets_resolved":                    false,
		"network_used":                        false,
		"areamatrix_protected_paths_touched":  false,
	}
}
