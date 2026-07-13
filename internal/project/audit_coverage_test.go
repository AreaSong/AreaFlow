package project

import (
	"testing"
	"time"
)

func TestBuildAuditCoverage(t *testing.T) {
	created := time.Date(2026, 6, 30, 11, 0, 0, 0, time.UTC)
	coverage := BuildAuditCoverage(AuditCoverageOptions{
		ProjectID:   1,
		ProjectKey:  "areamatrix",
		GeneratedAt: created,
	}, 4, []auditActionCount{
		{Action: "project.upsert", Decision: "allowed", Count: 1, LastAuditAt: created},
		{Action: "status.export", Decision: "allowed", Count: 1, LastAuditAt: created},
		{Action: "workflow.approval.record", Decision: "approved", Count: 1, LastAuditAt: created},
		{Action: "worker.run_once", Decision: "denied", Count: 1, LastAuditAt: created},
	})

	if coverage.Status != "warn" || coverage.Scope != "project" {
		t.Fatalf("unexpected coverage status/scope: %+v", coverage)
	}
	if coverage.ProjectKey != "areamatrix" || coverage.TotalAuditEvents != 4 {
		t.Fatalf("unexpected coverage identity: %+v", coverage)
	}
	if coverage.CoveredRequirements == 0 || coverage.GapRequirements == 0 {
		t.Fatalf("expected mixed coverage: %+v", coverage)
	}
	assertAuditRequirement(t, coverage, "project_registration", "pass")
	assertAuditRequirement(t, coverage, "status_mirror_write", "pass")
	assertAuditRequirement(t, coverage, "approval_decision", "pass")
	assertAuditRequirement(t, coverage, "worker_capability_denial", "pass")
	assertAuditRequirement(t, coverage, "secret_resolution", "gap")
	if !coverage.GeneratedAt.Equal(created) {
		t.Fatalf("generated_at = %s, want %s", coverage.GeneratedAt, created)
	}
}

func TestBuildAuditCoveragePlatformScope(t *testing.T) {
	coverage := BuildAuditCoverage(AuditCoverageOptions{}, 0, nil)

	if coverage.Scope != "platform" || coverage.Status != "warn" {
		t.Fatalf("unexpected platform coverage: %+v", coverage)
	}
	if coverage.CoveredRequirements != 0 || coverage.GapRequirements != len(auditCoverageRequirementSpecs) {
		t.Fatalf("unexpected empty coverage counts: %+v", coverage)
	}
}

func assertAuditRequirement(t *testing.T, coverage AuditCoverage, key string, status string) {
	t.Helper()
	for _, requirement := range coverage.Requirements {
		if requirement.Key == key {
			if requirement.Status != status {
				t.Fatalf("requirement %s status = %q, want %q: %+v", key, requirement.Status, status, requirement)
			}
			return
		}
	}
	t.Fatalf("requirement %s not found: %+v", key, coverage.Requirements)
}
