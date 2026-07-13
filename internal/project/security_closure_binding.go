package project

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"
)

type SecurityClosureCurrentBindingOptions struct {
	GeneratedAt time.Time
}

type SecurityClosureCurrentBinding struct {
	Project                   Record
	SecurityBoundaryReadiness SecurityBoundaryReadiness
	PermissionDoctor          PermissionPolicyDoctor
	AuditCoverage             AuditCoverage
	Metadata                  map[string]any
}

var securityClosureBindingComparisonKeys = []string{
	"security_closure_binding_hash",
	"security_boundary_status",
	"security_boundary_mode",
	"security_boundary_capabilities_hash",
	"security_boundary_capabilities_count",
	"security_boundary_forbidden_actions_hash",
	"security_boundary_forbidden_actions_count",
	"auth_enforcement_open",
	"team_permission_enforcement_open",
	"api_token_issuance_open",
	"api_token_enforcement_open",
	"secret_resolve_open",
	"remote_worker_credentials_open",
	"budget_enforcement_open",
	"quota_decrement_open",
	"usage_charge_written",
	"webhook_delivery_open",
	"inbound_callback_open",
	"external_api_call_open",
	"authorization_changed",
	"secret_plaintext_read",
	"remote_worker_direct_pg_allowed",
	"team_console_command_open",
	"remote_ops_control_open",
	"managed_upgrade_open",
	"support_bundle_export_open",
	"default_remote_telemetry_open",
	"permission_doctor_status",
	"permission_doctor_mode",
	"permission_doctor_check_count",
	"permission_doctor_fail_count",
	"permission_doctor_warn_count",
	"permission_doctor_run_commands_known",
	"permission_doctor_run_commands",
	"permission_doctor_use_secrets_known",
	"permission_doctor_use_secrets",
	"permission_doctor_manage_workers_known",
	"permission_doctor_manage_workers",
	"audit_coverage_status",
	"audit_coverage_mode",
	"audit_coverage_scope",
	"audit_coverage_requirement_count",
	"audit_coverage_covered_requirements",
	"audit_coverage_gap_requirements",
	"audit_coverage_missing_action_count",
	"audit_coverage_missing_actions_hash",
	"audit_coverage_enabled_status",
	"audit_coverage_enabled_missing_action_count",
	"audit_coverage_enabled_missing_actions_hash",
	"audit_coverage_future_only_missing_action_count",
	"audit_coverage_future_only_missing_actions_hash",
}

func (s Store) SecurityClosureCurrentBinding(ctx context.Context, record Record, options SecurityClosureCurrentBindingOptions) (SecurityClosureCurrentBinding, error) {
	options = normalizeSecurityClosureCurrentBindingOptions(options)
	readiness, err := s.SecurityBoundaryReadiness(ctx, SecurityBoundaryReadinessOptions{GeneratedAt: options.GeneratedAt})
	if err != nil {
		return SecurityClosureCurrentBinding{}, fmt.Errorf("security boundary readiness: %w", err)
	}
	doctor, err := s.PermissionPolicyDoctor(ctx, record, PermissionPolicyDoctorOptions{GeneratedAt: options.GeneratedAt})
	if err != nil {
		return SecurityClosureCurrentBinding{}, fmt.Errorf("permission doctor: %w", err)
	}
	coverage, err := s.AuditCoverage(ctx, AuditCoverageOptions{ProjectID: record.ID, ProjectKey: record.Key, GeneratedAt: options.GeneratedAt})
	if err != nil {
		return SecurityClosureCurrentBinding{}, fmt.Errorf("audit coverage: %w", err)
	}
	return BuildSecurityClosureCurrentBinding(record, readiness, doctor, coverage, options), nil
}

func BuildSecurityClosureCurrentBinding(record Record, readiness SecurityBoundaryReadiness, doctor PermissionPolicyDoctor, coverage AuditCoverage, options SecurityClosureCurrentBindingOptions) SecurityClosureCurrentBinding {
	options = normalizeSecurityClosureCurrentBindingOptions(options)
	metadata := securityClosureBindingMetadata(readiness, doctor, coverage, true, nil)
	return SecurityClosureCurrentBinding{
		Project:                   record,
		SecurityBoundaryReadiness: readiness,
		PermissionDoctor:          doctor,
		AuditCoverage:             coverage,
		Metadata:                  metadata,
	}
}

func normalizeSecurityClosureCurrentBindingOptions(options SecurityClosureCurrentBindingOptions) SecurityClosureCurrentBindingOptions {
	if options.GeneratedAt.IsZero() {
		options.GeneratedAt = time.Now().UTC()
	}
	return options
}

func addSecurityClosureProofBindingMetadata(metadata map[string]any, options RecordSecurityClosureProofOptions) {
	binding := map[string]any{}
	for _, key := range securityClosureBindingComparisonKeys {
		if value, ok := options.SecurityClosureBinding[key]; ok {
			binding[key] = value
		}
	}
	if status, ok := options.SecurityClosureBinding["security_closure_binding_status"]; ok {
		binding["security_closure_binding_status"] = status
	}
	blockers := securityClosureProofOptionsBindingBlockers(options)
	if len(blockers) > 0 {
		binding["security_closure_binding_status"] = "fail"
		binding["security_closure_binding_blockers"] = blockers
	} else if options.ProofStatus == "complete" {
		binding["security_closure_binding_status"] = "pass"
		binding["security_closure_binding_blockers"] = []string{}
	} else {
		binding["security_closure_binding_status"] = "not_required"
		binding["security_closure_binding_blockers"] = []string{}
	}
	for key, value := range binding {
		metadata[key] = value
	}
}

func securityClosureProofOptionsBindingBlockers(options RecordSecurityClosureProofOptions) []string {
	if len(options.SecurityClosureBinding) == 0 {
		return []string{"security_closure_binding_missing"}
	}
	return securityClosureProofMetadataBindingBlockers(options.SecurityClosureBinding)
}

func securityClosureBindingMetadata(readiness SecurityBoundaryReadiness, doctor PermissionPolicyDoctor, coverage AuditCoverage, pass bool, blockers []string) map[string]any {
	status := "fail"
	if pass {
		status = "pass"
	}
	permissionCounts := permissionDoctorStatusCounts(doctor)
	missingActions := auditCoverageMissingActions(coverage)
	runCommands, runCommandsKnown := securityClosurePermissionDoctorBool(doctor, "command_policy", "run_commands")
	useSecrets, useSecretsKnown := securityClosurePermissionDoctorBool(doctor, "secret_policy", "use_secrets")
	manageWorkers, manageWorkersKnown := securityClosurePermissionDoctorBool(doctor, "worker_capability_policy", "manage_workers")
	enabledMissingActions, futureOnlyMissingActions := securityClosureAuditCoverageMissingActions(readiness, doctor, missingActions)
	enabledCoverageStatus := "pass"
	if len(enabledMissingActions) > 0 {
		enabledCoverageStatus = "warn"
	}
	metadata := map[string]any{
		"security_closure_binding_status":             status,
		"security_closure_binding_blockers":           uniqueStrings(blockers),
		"security_boundary_status":                    readiness.Status,
		"security_boundary_mode":                      readiness.Mode,
		"security_boundary_capabilities_hash":         securityClosureStringSetHash("security_boundary_capabilities", readiness.Capabilities),
		"security_boundary_capabilities_count":        int64(len(normalizeStringList(readiness.Capabilities))),
		"security_boundary_forbidden_actions_hash":    securityClosureStringSetHash("security_boundary_forbidden_actions", readiness.ForbiddenActions),
		"security_boundary_forbidden_actions_count":   int64(len(normalizeStringList(readiness.ForbiddenActions))),
		"auth_enforcement_open":                       readiness.AuthEnforcementOpen,
		"team_permission_enforcement_open":            readiness.TeamPermissionEnforcementOpen,
		"api_token_issuance_open":                     readiness.APITokenIssuanceOpen,
		"api_token_enforcement_open":                  readiness.APITokenEnforcementOpen,
		"secret_resolve_open":                         readiness.SecretResolveOpen,
		"remote_worker_credentials_open":              readiness.RemoteWorkerCredentialsOpen,
		"budget_enforcement_open":                     readiness.BudgetEnforcementOpen,
		"quota_decrement_open":                        readiness.QuotaDecrementOpen,
		"usage_charge_written":                        readiness.UsageChargeWritten,
		"webhook_delivery_open":                       readiness.WebhookDeliveryOpen,
		"inbound_callback_open":                       readiness.InboundCallbackOpen,
		"external_api_call_open":                      readiness.ExternalAPICallOpen,
		"authorization_changed":                       readiness.AuthorizationChanged,
		"secret_plaintext_read":                       readiness.SecretPlaintextRead,
		"remote_worker_direct_pg_allowed":             readiness.RemoteWorkerDirectPGAllowed,
		"team_console_command_open":                   readiness.TeamConsoleCommandOpen,
		"remote_ops_control_open":                     readiness.RemoteOpsControlOpen,
		"managed_upgrade_open":                        readiness.ManagedUpgradeOpen,
		"support_bundle_export_open":                  readiness.SupportBundleExportOpen,
		"default_remote_telemetry_open":               readiness.DefaultRemoteTelemetryOpen,
		"permission_doctor_status":                    doctor.Status,
		"permission_doctor_mode":                      doctor.Mode,
		"permission_doctor_check_count":               int64(len(doctor.Checks)),
		"permission_doctor_fail_count":                permissionCounts["fail"],
		"permission_doctor_warn_count":                permissionCounts["warn"],
		"permission_doctor_run_commands_known":        runCommandsKnown,
		"permission_doctor_run_commands":              runCommands,
		"permission_doctor_use_secrets_known":         useSecretsKnown,
		"permission_doctor_use_secrets":               useSecrets,
		"permission_doctor_manage_workers_known":      manageWorkersKnown,
		"permission_doctor_manage_workers":            manageWorkers,
		"audit_coverage_status":                       coverage.Status,
		"audit_coverage_mode":                         coverage.Mode,
		"audit_coverage_scope":                        coverage.Scope,
		"audit_coverage_requirement_count":            int64(len(coverage.Requirements)),
		"audit_coverage_covered_requirements":         int64(coverage.CoveredRequirements),
		"audit_coverage_gap_requirements":             int64(coverage.GapRequirements),
		"audit_coverage_missing_action_count":         int64(len(missingActions)),
		"audit_coverage_missing_actions_hash":         securityClosureStringSetHash("audit_coverage_missing_actions", missingActions),
		"audit_coverage_enabled_status":               enabledCoverageStatus,
		"audit_coverage_enabled_missing_action_count": int64(len(enabledMissingActions)),
		"audit_coverage_enabled_missing_actions_hash": securityClosureStringSetHash(
			"audit_coverage_enabled_missing_actions",
			enabledMissingActions,
		),
		"audit_coverage_future_only_missing_action_count": int64(len(futureOnlyMissingActions)),
		"audit_coverage_future_only_missing_actions_hash": securityClosureStringSetHash(
			"audit_coverage_future_only_missing_actions",
			futureOnlyMissingActions,
		),
	}
	metadata["security_closure_binding_hash"] = securityClosureBindingHash(metadata)
	return metadata
}

func securityClosureProofMetadataBindingBlockers(metadata map[string]any) []string {
	blockers := []string{}
	if metadataString(metadata, "security_closure_binding_status") != "pass" {
		blockers = append(blockers, "security_closure_binding_status_not_pass")
	}
	if metadataString(metadata, "security_boundary_status") != "ready" {
		blockers = append(blockers, "security_boundary_status_not_ready")
	}
	if metadataString(metadata, "security_boundary_mode") != "read_only_security_boundary_readiness" {
		blockers = append(blockers, "security_boundary_mode_missing_or_mismatch")
	}
	if !looksLikeSHA256(metadataString(metadata, "security_boundary_capabilities_hash")) || metadataInt64(metadata, "security_boundary_capabilities_count") == 0 {
		blockers = append(blockers, "security_boundary_capabilities_binding_missing_or_invalid")
	}
	if !looksLikeSHA256(metadataString(metadata, "security_boundary_forbidden_actions_hash")) || metadataInt64(metadata, "security_boundary_forbidden_actions_count") == 0 {
		blockers = append(blockers, "security_boundary_forbidden_actions_binding_missing_or_invalid")
	}
	blockers = append(blockers, securityClosureOpenCapabilityBlockers(metadata)...)
	if metadataString(metadata, "permission_doctor_status") != "pass" {
		blockers = append(blockers, "permission_doctor_status_not_pass")
	}
	if metadataString(metadata, "permission_doctor_mode") != "read_only_permission_policy_doctor" {
		blockers = append(blockers, "permission_doctor_mode_missing_or_mismatch")
	}
	if metadataInt64(metadata, "permission_doctor_check_count") == 0 {
		blockers = append(blockers, "permission_doctor_check_count_missing")
	}
	if metadataInt64(metadata, "permission_doctor_fail_count") != 0 {
		blockers = append(blockers, "permission_doctor_fail_count_nonzero")
	}
	if metadataInt64(metadata, "permission_doctor_warn_count") != 0 {
		blockers = append(blockers, "permission_doctor_warn_count_nonzero")
	}
	if !metadataBool(metadata, "permission_doctor_run_commands_known") {
		blockers = append(blockers, "permission_doctor_run_commands_binding_missing")
	}
	if !metadataBool(metadata, "permission_doctor_use_secrets_known") {
		blockers = append(blockers, "permission_doctor_use_secrets_binding_missing")
	}
	if !metadataBool(metadata, "permission_doctor_manage_workers_known") {
		blockers = append(blockers, "permission_doctor_manage_workers_binding_missing")
	}
	if metadataString(metadata, "audit_coverage_mode") != "read_only_audit_coverage" {
		blockers = append(blockers, "audit_coverage_mode_missing_or_mismatch")
	}
	if metadataString(metadata, "audit_coverage_scope") != "project" {
		blockers = append(blockers, "audit_coverage_scope_not_project")
	}
	if metadataInt64(metadata, "audit_coverage_requirement_count") == 0 {
		blockers = append(blockers, "audit_coverage_requirement_count_missing")
	}
	if metadataString(metadata, "audit_coverage_enabled_status") != "pass" {
		blockers = append(blockers, "audit_coverage_enabled_status_not_pass")
	}
	if metadataInt64(metadata, "audit_coverage_enabled_missing_action_count") != 0 {
		blockers = append(blockers, "audit_coverage_enabled_missing_action_count_nonzero")
	}
	if !looksLikeSHA256(metadataString(metadata, "audit_coverage_enabled_missing_actions_hash")) {
		blockers = append(blockers, "audit_coverage_enabled_missing_actions_hash_missing_or_invalid")
	}
	if !looksLikeSHA256(metadataString(metadata, "audit_coverage_future_only_missing_actions_hash")) {
		blockers = append(blockers, "audit_coverage_future_only_missing_actions_hash_missing_or_invalid")
	}
	if !looksLikeSHA256(metadataString(metadata, "security_closure_binding_hash")) ||
		metadataString(metadata, "security_closure_binding_hash") != securityClosureBindingHash(metadata) {
		blockers = append(blockers, "security_closure_binding_hash_missing_or_mismatch")
	}
	return uniqueStrings(blockers)
}

func securityClosureProofCurrentBindingBlockers(proofMetadata map[string]any, currentBinding map[string]any) []string {
	blockers := securityClosureProofMetadataBindingBlockers(proofMetadata)
	if len(blockers) > 0 {
		return blockers
	}
	currentBlockers := securityClosureProofMetadataBindingBlockers(currentBinding)
	if len(currentBlockers) > 0 {
		for _, blocker := range currentBlockers {
			blockers = append(blockers, "current_"+blocker)
		}
		return uniqueStrings(blockers)
	}
	for _, key := range securityClosureBindingComparisonKeys {
		if !securityClosureMetadataValuesEqual(proofMetadata, currentBinding, key) {
			blockers = append(blockers, key+"_current_mismatch")
		}
	}
	return uniqueStrings(blockers)
}

func securityClosureOpenCapabilityBlockers(metadata map[string]any) []string {
	blockers := []string{}
	for key, blocker := range map[string]string{
		"auth_enforcement_open":            "auth_enforcement_open",
		"team_permission_enforcement_open": "team_permission_enforcement_open",
		"api_token_issuance_open":          "api_token_issuance_open",
		"api_token_enforcement_open":       "api_token_enforcement_open",
		"secret_resolve_open":              "secret_resolve_open",
		"remote_worker_credentials_open":   "remote_worker_credentials_open",
		"budget_enforcement_open":          "budget_enforcement_open",
		"quota_decrement_open":             "quota_decrement_open",
		"usage_charge_written":             "usage_charge_written",
		"webhook_delivery_open":            "webhook_delivery_open",
		"inbound_callback_open":            "inbound_callback_open",
		"external_api_call_open":           "external_api_call_open",
		"authorization_changed":            "authorization_changed",
		"secret_plaintext_read":            "secret_plaintext_read",
		"remote_worker_direct_pg_allowed":  "remote_worker_direct_pg_allowed",
		"team_console_command_open":        "team_console_command_open",
		"remote_ops_control_open":          "remote_ops_control_open",
		"managed_upgrade_open":             "managed_upgrade_open",
		"support_bundle_export_open":       "support_bundle_export_open",
		"default_remote_telemetry_open":    "default_remote_telemetry_open",
	} {
		if metadataBool(metadata, key) {
			blockers = append(blockers, blocker)
		}
	}
	return blockers
}

func securityClosureBindingHash(metadata map[string]any) string {
	payload := map[string]any{}
	for _, key := range securityClosureBindingComparisonKeys {
		if key == "security_closure_binding_hash" {
			continue
		}
		payload[key] = metadata[key]
	}
	content, err := json.Marshal(payload)
	if err != nil {
		return ""
	}
	sum := sha256.Sum256(content)
	return hex.EncodeToString(sum[:])
}

func securityClosureStringSetHash(kind string, values []string) string {
	payload, err := json.Marshal(map[string]any{
		"kind":   kind,
		"values": normalizeStringList(values),
	})
	if err != nil {
		return ""
	}
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:])
}

func permissionDoctorStatusCounts(doctor PermissionPolicyDoctor) map[string]int64 {
	counts := map[string]int64{"pass": 0, "warn": 0, "fail": 0}
	for _, check := range doctor.Checks {
		counts[check.Status]++
	}
	return counts
}

func auditCoverageMissingActions(coverage AuditCoverage) []string {
	actions := []string{}
	for _, requirement := range coverage.Requirements {
		actions = append(actions, requirement.MissingActions...)
	}
	sort.Strings(actions)
	return uniqueStrings(actions)
}

func securityClosureAuditCoverageMissingActions(readiness SecurityBoundaryReadiness, doctor PermissionPolicyDoctor, missingActions []string) ([]string, []string) {
	enabledMissingActions := []string{}
	futureOnlyMissingActions := []string{}
	for _, action := range missingActions {
		if securityClosureFutureOnlyAuditGap(readiness, doctor, action) {
			futureOnlyMissingActions = append(futureOnlyMissingActions, action)
			continue
		}
		enabledMissingActions = append(enabledMissingActions, action)
	}
	sort.Strings(enabledMissingActions)
	sort.Strings(futureOnlyMissingActions)
	return uniqueStrings(enabledMissingActions), uniqueStrings(futureOnlyMissingActions)
}

func securityClosureFutureOnlyAuditGap(readiness SecurityBoundaryReadiness, doctor PermissionPolicyDoctor, action string) bool {
	switch securityClosureAuditGapAction(action) {
	case "command.execute":
		runCommands, ok := securityClosurePermissionDoctorBool(doctor, "command_policy", "run_commands")
		return ok && !runCommands
	case "secret.resolve":
		useSecrets, ok := securityClosurePermissionDoctorBool(doctor, "secret_policy", "use_secrets")
		return ok && !useSecrets && !readiness.SecretResolveOpen
	case "permission.change":
		return !readiness.TeamPermissionEnforcementOpen && !readiness.AuthorizationChanged
	case "lease.acquire", "lease.release", "lease.recover":
		manageWorkers, ok := securityClosurePermissionDoctorBool(doctor, "worker_capability_policy", "manage_workers")
		return ok && !manageWorkers
	default:
		return false
	}
}

func securityClosureAuditGapAction(action string) string {
	action = strings.TrimSpace(action)
	if before, _, found := strings.Cut(action, ":"); found {
		return strings.TrimSpace(before)
	}
	return action
}

func securityClosurePermissionDoctorBool(doctor PermissionPolicyDoctor, checkKey string, metadataKey string) (bool, bool) {
	for _, check := range doctor.Checks {
		if check.Key != checkKey {
			continue
		}
		if check.Metadata == nil {
			return false, false
		}
		value, ok := check.Metadata[metadataKey]
		if !ok {
			return false, false
		}
		switch typed := value.(type) {
		case bool:
			return typed, true
		case string:
			switch typed {
			case "true":
				return true, true
			case "false":
				return false, true
			default:
				return false, false
			}
		default:
			return false, false
		}
	}
	return false, false
}

func securityClosureMetadataValuesEqual(left map[string]any, right map[string]any, key string) bool {
	switch key {
	case "security_boundary_capabilities_count",
		"security_boundary_forbidden_actions_count",
		"permission_doctor_check_count",
		"permission_doctor_fail_count",
		"permission_doctor_warn_count",
		"audit_coverage_requirement_count",
		"audit_coverage_covered_requirements",
		"audit_coverage_gap_requirements",
		"audit_coverage_missing_action_count",
		"audit_coverage_enabled_missing_action_count",
		"audit_coverage_future_only_missing_action_count":
		return metadataInt64(left, key) == metadataInt64(right, key)
	case "auth_enforcement_open",
		"team_permission_enforcement_open",
		"api_token_issuance_open",
		"api_token_enforcement_open",
		"secret_resolve_open",
		"remote_worker_credentials_open",
		"budget_enforcement_open",
		"quota_decrement_open",
		"usage_charge_written",
		"webhook_delivery_open",
		"inbound_callback_open",
		"external_api_call_open",
		"authorization_changed",
		"secret_plaintext_read",
		"remote_worker_direct_pg_allowed",
		"team_console_command_open",
		"remote_ops_control_open",
		"managed_upgrade_open",
		"support_bundle_export_open",
		"default_remote_telemetry_open",
		"permission_doctor_run_commands_known",
		"permission_doctor_run_commands",
		"permission_doctor_use_secrets_known",
		"permission_doctor_use_secrets",
		"permission_doctor_manage_workers_known",
		"permission_doctor_manage_workers":
		return metadataBool(left, key) == metadataBool(right, key)
	default:
		return metadataString(left, key) == metadataString(right, key)
	}
}
