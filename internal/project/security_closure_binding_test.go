package project

import (
	"testing"
	"time"
)

func TestSecurityClosureBindingTreatsClosedFutureOnlyAuditGapsAsNonBlocking(t *testing.T) {
	generated := time.Date(2026, 7, 13, 10, 0, 0, 0, time.UTC)
	record := Record{ID: 1, Key: "areamatrix"}
	readiness := BuildSecurityBoundaryReadiness(SecurityBoundaryReadinessOptions{GeneratedAt: generated})
	doctor := testSecurityClosurePermissionDoctor(record, generated)
	coverage := BuildAuditCoverage(AuditCoverageOptions{
		ProjectID:   record.ID,
		ProjectKey:  record.Key,
		GeneratedAt: generated,
	}, 11, []auditActionCount{
		{Action: "project.upsert", Decision: "allowed", Count: 1, LastAuditAt: generated},
		{Action: "status.export", Decision: "allowed", Count: 1, LastAuditAt: generated},
		{Action: "workflow.version.create", Decision: "allowed", Count: 1, LastAuditAt: generated},
		{Action: "workflow.stage_skeleton.create", Decision: "allowed", Count: 1, LastAuditAt: generated},
		{Action: "workflow.item.mark_ready", Decision: "allowed", Count: 1, LastAuditAt: generated},
		{Action: "workflow.approval.record", Decision: "approved", Count: 1, LastAuditAt: generated},
		{Action: "runner.preview", Decision: "allowed", Count: 1, LastAuditAt: generated},
		{Action: "worker.register", Decision: "allowed", Count: 1, LastAuditAt: generated},
		{Action: "worker.run_once", Decision: "denied", Count: 1, LastAuditAt: generated},
		{Action: "lease.acquire", Decision: "allowed", Count: 1, LastAuditAt: generated},
		{Action: "lease.release", Decision: "allowed", Count: 1, LastAuditAt: generated},
		{Action: "lease.recover", Decision: "allowed", Count: 1, LastAuditAt: generated},
	})

	binding := BuildSecurityClosureCurrentBinding(
		record,
		readiness,
		doctor,
		coverage,
		SecurityClosureCurrentBindingOptions{GeneratedAt: generated},
	)

	if binding.Metadata["audit_coverage_status"] != "warn" {
		t.Fatalf("raw audit coverage should still expose future-only gaps: %+v", binding.Metadata)
	}
	if binding.Metadata["audit_coverage_enabled_status"] != "pass" ||
		binding.Metadata["audit_coverage_enabled_missing_action_count"] != int64(0) ||
		binding.Metadata["audit_coverage_future_only_missing_action_count"] != int64(3) {
		t.Fatalf("unexpected enabled audit coverage metadata: %+v", binding.Metadata)
	}
	if blockers := securityClosureProofMetadataBindingBlockers(binding.Metadata); len(blockers) != 0 {
		t.Fatalf("closed future-only gaps must not block E8 security closure binding: %v", blockers)
	}
}

func TestSecurityClosureBindingKeepsEnabledAuditGapsBlocking(t *testing.T) {
	generated := time.Date(2026, 7, 13, 10, 30, 0, 0, time.UTC)
	record := Record{ID: 1, Key: "areamatrix"}
	readiness := BuildSecurityBoundaryReadiness(SecurityBoundaryReadinessOptions{GeneratedAt: generated})
	doctor := testSecurityClosurePermissionDoctor(record, generated)
	coverage := testSecurityClosureAuditCoverage(record, generated, map[string]bool{"status.export": true})

	binding := BuildSecurityClosureCurrentBinding(
		record,
		readiness,
		doctor,
		coverage,
		SecurityClosureCurrentBindingOptions{GeneratedAt: generated},
	)

	blockers := securityClosureProofMetadataBindingBlockers(binding.Metadata)
	if !containsString(blockers, "audit_coverage_enabled_status_not_pass") ||
		!containsString(blockers, "audit_coverage_enabled_missing_action_count_nonzero") {
		t.Fatalf("enabled audit gap must block E8 security closure binding: %+v metadata=%+v", blockers, binding.Metadata)
	}
}

func TestSecurityClosureBindingTreatsWorkerLeaseGapsAsFutureOnlyWhenManageWorkersDisabled(t *testing.T) {
	generated := time.Date(2026, 7, 13, 10, 45, 0, 0, time.UTC)
	record := Record{ID: 1, Key: "areamatrix"}
	readiness := BuildSecurityBoundaryReadiness(SecurityBoundaryReadinessOptions{GeneratedAt: generated})
	doctor := testSecurityClosurePermissionDoctor(record, generated)
	coverage := testSecurityClosureAuditCoverage(record, generated, map[string]bool{
		"lease.acquire": true,
		"lease.release": true,
		"lease.recover": true,
	})

	binding := BuildSecurityClosureCurrentBinding(
		record,
		readiness,
		doctor,
		coverage,
		SecurityClosureCurrentBindingOptions{GeneratedAt: generated},
	)

	if binding.Metadata["audit_coverage_enabled_missing_action_count"] != int64(0) ||
		binding.Metadata["audit_coverage_future_only_missing_action_count"] != int64(3) {
		t.Fatalf("lease gaps must be future-only while manage_workers is disabled: %+v", binding.Metadata)
	}
	if blockers := securityClosureProofMetadataBindingBlockers(binding.Metadata); len(blockers) != 0 {
		t.Fatalf("disabled worker lease gaps must not block proof binding: %v metadata=%+v", blockers, binding.Metadata)
	}
}

func TestSecurityClosureBindingKeepsCommandGapBlockingWhenRunCommandsEnabled(t *testing.T) {
	generated := time.Date(2026, 7, 13, 11, 0, 0, 0, time.UTC)
	record := Record{ID: 1, Key: "areamatrix"}
	readiness := BuildSecurityBoundaryReadiness(SecurityBoundaryReadinessOptions{GeneratedAt: generated})
	doctor := testSecurityClosurePermissionDoctor(record, generated)
	setPermissionDoctorCheckMetadataBool(&doctor, "command_policy", "run_commands", true)
	coverage := testSecurityClosureAuditCoverage(record, generated, map[string]bool{"command.execute": true})

	binding := BuildSecurityClosureCurrentBinding(
		record,
		readiness,
		doctor,
		coverage,
		SecurityClosureCurrentBindingOptions{GeneratedAt: generated},
	)

	if binding.Metadata["audit_coverage_enabled_missing_action_count"] != int64(1) ||
		binding.Metadata["audit_coverage_future_only_missing_action_count"] != int64(0) {
		t.Fatalf("command.execute must be enabled gap when run_commands is enabled: %+v", binding.Metadata)
	}
	blockers := securityClosureProofMetadataBindingBlockers(binding.Metadata)
	if !containsString(blockers, "audit_coverage_enabled_status_not_pass") {
		t.Fatalf("enabled command gap must block proof binding: %v metadata=%+v", blockers, binding.Metadata)
	}
}

func TestSecurityClosureBindingKeepsSecretGapBlockingWhenUseSecretsEnabled(t *testing.T) {
	generated := time.Date(2026, 7, 13, 11, 30, 0, 0, time.UTC)
	record := Record{ID: 1, Key: "areamatrix"}
	readiness := BuildSecurityBoundaryReadiness(SecurityBoundaryReadinessOptions{GeneratedAt: generated})
	doctor := testSecurityClosurePermissionDoctor(record, generated)
	setPermissionDoctorCheckMetadataBool(&doctor, "secret_policy", "use_secrets", true)
	coverage := testSecurityClosureAuditCoverage(record, generated, map[string]bool{"secret.resolve": true})

	binding := BuildSecurityClosureCurrentBinding(
		record,
		readiness,
		doctor,
		coverage,
		SecurityClosureCurrentBindingOptions{GeneratedAt: generated},
	)

	if binding.Metadata["audit_coverage_enabled_missing_action_count"] != int64(1) ||
		binding.Metadata["audit_coverage_future_only_missing_action_count"] != int64(0) {
		t.Fatalf("secret.resolve must be enabled gap when use_secrets is enabled: %+v", binding.Metadata)
	}
	blockers := securityClosureProofMetadataBindingBlockers(binding.Metadata)
	if !containsString(blockers, "audit_coverage_enabled_status_not_pass") {
		t.Fatalf("enabled secret gap must block proof binding: %v metadata=%+v", blockers, binding.Metadata)
	}
}

func TestSecurityClosureBindingKeepsWorkerLeaseGapBlockingWhenManageWorkersEnabled(t *testing.T) {
	generated := time.Date(2026, 7, 13, 11, 45, 0, 0, time.UTC)
	record := Record{ID: 1, Key: "areamatrix"}
	readiness := BuildSecurityBoundaryReadiness(SecurityBoundaryReadinessOptions{GeneratedAt: generated})
	doctor := testSecurityClosurePermissionDoctor(record, generated)
	setPermissionDoctorCheckMetadataBool(&doctor, "worker_capability_policy", "manage_workers", true)
	coverage := testSecurityClosureAuditCoverage(record, generated, map[string]bool{"lease.acquire": true})

	binding := BuildSecurityClosureCurrentBinding(
		record,
		readiness,
		doctor,
		coverage,
		SecurityClosureCurrentBindingOptions{GeneratedAt: generated},
	)

	if binding.Metadata["audit_coverage_enabled_missing_action_count"] != int64(1) ||
		binding.Metadata["audit_coverage_future_only_missing_action_count"] != int64(0) {
		t.Fatalf("lease.acquire must be enabled gap when manage_workers is enabled: %+v", binding.Metadata)
	}
	blockers := securityClosureProofMetadataBindingBlockers(binding.Metadata)
	if !containsString(blockers, "audit_coverage_enabled_status_not_pass") {
		t.Fatalf("enabled worker lease gap must block proof binding: %v metadata=%+v", blockers, binding.Metadata)
	}
}

func TestSecurityClosureBindingBlocksWhenPermissionDoctorCapabilityEvidenceMissing(t *testing.T) {
	generated := time.Date(2026, 7, 13, 12, 0, 0, 0, time.UTC)
	record := Record{ID: 1, Key: "areamatrix"}
	readiness := BuildSecurityBoundaryReadiness(SecurityBoundaryReadinessOptions{GeneratedAt: generated})
	doctor := PermissionPolicyDoctor{
		Status: "pass",
		Mode:   "read_only_permission_policy_doctor",
		Checks: []PermissionPolicyCheck{{Key: "placeholder", Status: "pass"}},
	}
	coverage := testSecurityClosureAuditCoverage(record, generated, map[string]bool{"command.execute": true})

	binding := BuildSecurityClosureCurrentBinding(
		record,
		readiness,
		doctor,
		coverage,
		SecurityClosureCurrentBindingOptions{GeneratedAt: generated},
	)

	blockers := securityClosureProofMetadataBindingBlockers(binding.Metadata)
	if !containsString(blockers, "permission_doctor_run_commands_binding_missing") ||
		!containsString(blockers, "permission_doctor_use_secrets_binding_missing") ||
		!containsString(blockers, "permission_doctor_manage_workers_binding_missing") {
		t.Fatalf("missing capability evidence must conservatively block proof binding: %v metadata=%+v", blockers, binding.Metadata)
	}
}

func testSecurityClosurePermissionDoctor(record Record, generated time.Time) PermissionPolicyDoctor {
	return BuildPermissionPolicyDoctor(
		record,
		testPermissionProjectConfig(generated),
		true,
		testPermissionRows(),
		PermissionPolicyDoctorOptions{GeneratedAt: generated},
	)
}

func testSecurityClosureAuditCoverage(record Record, generated time.Time, missingActions map[string]bool) AuditCoverage {
	counts := []auditActionCount{}
	for _, spec := range auditCoverageRequirementSpecs {
		for _, action := range spec.Actions {
			if missingActions[action.Action] {
				continue
			}
			decision := action.Decision
			if decision == "" {
				decision = "allowed"
			}
			counts = append(counts, auditActionCount{
				Action:      action.Action,
				Decision:    decision,
				Count:       1,
				LastAuditAt: generated,
			})
		}
	}
	return BuildAuditCoverage(
		AuditCoverageOptions{ProjectID: record.ID, ProjectKey: record.Key, GeneratedAt: generated},
		int64(len(counts)),
		counts,
	)
}

func setPermissionDoctorCheckMetadataBool(doctor *PermissionPolicyDoctor, checkKey string, metadataKey string, value bool) {
	for index := range doctor.Checks {
		if doctor.Checks[index].Key != checkKey {
			continue
		}
		if doctor.Checks[index].Metadata == nil {
			doctor.Checks[index].Metadata = map[string]any{}
		}
		doctor.Checks[index].Metadata[metadataKey] = value
		return
	}
}
