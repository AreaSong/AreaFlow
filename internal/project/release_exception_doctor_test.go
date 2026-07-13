package project

import (
	"testing"
	"time"
)

func TestBuildReleaseExceptionDoctorWarnsForPendingExceptions(t *testing.T) {
	created := time.Date(2026, 6, 30, 12, 0, 0, 0, time.UTC)
	gate := ReleaseAcceptanceGate{
		Status: "blocked",
		Mode:   "read_only_release_acceptance_gate",
		Items: []ReleaseAcceptanceGateItem{
			{
				Key:              "gate:accept:restore_plan",
				Category:         "restore",
				Status:           "blocked",
				DecisionStatus:   "needs_decision",
				AcceptanceType:   "metadata_only_history",
				Owner:            "release_owner",
				RequiredEvidence: []string{"release notes state metadata-only artifacts"},
				NextCommand:      "areaflow backup restore-plan --json",
			},
			{
				Key:              "gate:accept:audit_coverage",
				Category:         "audit",
				Status:           "blocked",
				DecisionStatus:   "needs_decision",
				AcceptanceType:   "future_only_gap",
				Owner:            "platform_owner",
				RequiredEvidence: []string{"audit coverage lists missing actions"},
				NextCommand:      "areaflow audit coverage --json",
			},
		},
	}

	doctor := BuildReleaseExceptionDoctor(gate, ReleaseExceptionDoctorOptions{GeneratedAt: created})

	if doctor.Status != "warn" || doctor.Mode != "read_only_release_exception_doctor" {
		t.Fatalf("unexpected release exception doctor: %+v", doctor)
	}
	assertReleaseExceptionCheck(t, doctor, "exception_record_schema", "schema", "warn")
	assertReleaseExceptionCheck(t, doctor, "exception_audit_contract", "audit", "warn")
	assertReleaseExceptionCheck(t, doctor, "exception_write_guardrails", "safety", "pass")
	assertReleaseExceptionCheck(t, doctor, "exception:gate:accept:restore_plan", "restore", "warn")
	assertReleaseExceptionCheck(t, doctor, "exception:gate:accept:audit_coverage", "audit", "warn")
	if len(doctor.ForbiddenActions) == 0 || doctor.ForbiddenActions[0] != "write_database" {
		t.Fatalf("unexpected forbidden actions: %+v", doctor.ForbiddenActions)
	}
	if !doctor.GeneratedAt.Equal(created) {
		t.Fatalf("generated_at = %s, want %s", doctor.GeneratedAt, created)
	}
}

func TestBuildReleaseExceptionDoctorFailsNotAcceptableItems(t *testing.T) {
	doctor := BuildReleaseExceptionDoctor(
		ReleaseAcceptanceGate{
			Status: "blocked",
			Items: []ReleaseAcceptanceGateItem{
				{
					Key:              "gate:accept:permission_policy:areamatrix",
					Category:         "permission",
					Status:           "blocked",
					DecisionStatus:   "not_acceptable",
					AcceptanceType:   "none",
					Owner:            "security_owner",
					RequiredEvidence: []string{"permission policy doctor returns pass"},
				},
			},
		},
		ReleaseExceptionDoctorOptions{},
	)

	if doctor.Status != "fail" {
		t.Fatalf("expected fail status: %+v", doctor)
	}
	assertReleaseExceptionCheck(t, doctor, "exception:gate:accept:permission_policy:areamatrix", "permission", "fail")
}

func TestBuildReleaseExceptionDoctorKeepsSchemaWarningsWhenGatePasses(t *testing.T) {
	doctor := BuildReleaseExceptionDoctor(
		ReleaseAcceptanceGate{
			Status: "pass",
			Items: []ReleaseAcceptanceGateItem{
				{
					Key:              "gate:accept:release_readiness",
					Category:         "release",
					Status:           "pass",
					DecisionStatus:   "ready",
					AcceptanceType:   "none",
					Owner:            "release_owner",
					RequiredEvidence: []string{"release readiness remains ready"},
				},
			},
		},
		ReleaseExceptionDoctorOptions{},
	)

	if doctor.Status != "warn" {
		t.Fatalf("expected warn status while exception writes are disabled: %+v", doctor)
	}
	assertReleaseExceptionCheck(t, doctor, "exception:gate:accept:release_readiness", "release", "pass")
}

func TestBuildReleaseExceptionDoctorPropagatesProjectScope(t *testing.T) {
	doctor := BuildReleaseExceptionDoctor(
		ReleaseAcceptanceGate{
			Status:     "pass",
			Scope:      "project",
			ProjectKey: "areamatrix",
			Preview: ReleaseAcceptancePreview{
				Scope:      "project",
				ProjectKey: "areamatrix",
			},
		},
		ReleaseExceptionDoctorOptions{ProjectKey: "areamatrix"},
	)

	if doctor.Scope != "project" || doctor.ProjectKey != "areamatrix" {
		t.Fatalf("expected project-scoped exception doctor: %+v", doctor)
	}
	if doctor.Gate.ProjectKey != "areamatrix" {
		t.Fatalf("expected nested gate to keep project key: %+v", doctor.Gate)
	}
}

func assertReleaseExceptionCheck(t *testing.T, doctor ReleaseExceptionDoctor, key string, category string, status string) {
	t.Helper()
	for _, check := range doctor.Checks {
		if check.Key == key {
			if check.Category != category || check.Status != status {
				t.Fatalf("check %s = %+v, want category=%s status=%s", key, check, category, status)
			}
			if check.Message == "" {
				t.Fatalf("check %s missing message: %+v", key, check)
			}
			return
		}
	}
	t.Fatalf("check %s not found: %+v", key, doctor.Checks)
}
