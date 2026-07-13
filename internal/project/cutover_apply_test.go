package project

import (
	"strings"
	"testing"
)

func TestEvaluateCutoverApplyBlocksWithoutCutoverReadinessGate(t *testing.T) {
	record := Record{ID: 1, Key: "areamatrix"}
	version := WorkflowVersion{ID: 2, ProjectID: 1, DisplayLabel: "v2", ImportMode: "authored", LifecycleStatus: "draft", StatusSummary: map[string]any{}}
	readiness := ProjectCutoverReadiness{
		Project: record,
		Version: version,
		PhaseGate: PhaseGate{
			Name:     "v0.4-cutover-readiness",
			Status:   "blocked",
			Blockers: []string{"approval_gate is blocked"},
		},
	}

	result := evaluateCutoverApply(record, version, readiness, ApplyCutoverOptions{Mode: "authoring_cutover"})

	if result.Status != "blocked" || result.Decision != "denied" {
		t.Fatalf("unexpected cutover apply result: %+v", result)
	}
	if len(result.Blockers) != 2 {
		t.Fatalf("expected readiness and gate blockers: %+v", result.Blockers)
	}
	if result.ProjectWriteAttempted || result.ExecutionWriteAttempted {
		t.Fatalf("cutover apply must not write project/execution state: %+v", result)
	}
}

func TestEvaluateCutoverApplyPassesForAuthoringCutoverOnly(t *testing.T) {
	record := Record{ID: 1, Key: "areamatrix"}
	version := WorkflowVersion{ID: 2, ProjectID: 1, DisplayLabel: "v2", ImportMode: "authored", LifecycleStatus: "draft", StatusSummary: map[string]any{}}
	readiness := ProjectCutoverReadiness{
		Project: record,
		Version: version,
		PhaseGate: PhaseGate{
			Name:   "v0.4-cutover-readiness",
			Status: "pass",
		},
		Gates: []GateResult{{ID: 7, GateName: "cutover_readiness_gate", Status: "pass"}},
	}

	result := evaluateCutoverApply(record, version, readiness, ApplyCutoverOptions{Mode: "authoring_cutover"})

	if result.Status != "applied" || result.Decision != "allowed" {
		t.Fatalf("unexpected cutover apply result: %+v", result)
	}
	if result.CutoverReadinessGateID != 7 {
		t.Fatalf("cutover readiness gate id = %d, want 7", result.CutoverReadinessGateID)
	}
	if result.ProjectWriteAttempted || result.ExecutionWriteAttempted {
		t.Fatalf("cutover apply must stay inside AreaFlow state: %+v", result)
	}

	result = evaluateCutoverApply(record, version, readiness, ApplyCutoverOptions{Mode: "execution_cutover"})
	if result.Status != "blocked" || result.Decision != "denied" {
		t.Fatalf("execution cutover should stay blocked: %+v", result)
	}
}

func TestCutoverApplyRequestHashAndDefaultKey(t *testing.T) {
	record := Record{ID: 1, Key: "areamatrix"}
	version := WorkflowVersion{ID: 2, DisplayLabel: "v2"}
	options := normalizeApplyCutoverOptions(record, ApplyCutoverOptions{
		VersionLabel: "v2",
		Actor:        "local-user",
		Reason:       "apply cutover",
	})
	first, err := cutoverApplyRequestHash(record, version, options)
	if err != nil {
		t.Fatalf("first request hash failed: %v", err)
	}
	second, err := cutoverApplyRequestHash(record, version, options)
	if err != nil {
		t.Fatalf("second request hash failed: %v", err)
	}
	if first != second {
		t.Fatalf("cutover hash differed: %s != %s", first, second)
	}
	options.Reason = "different reason"
	changed, err := cutoverApplyRequestHash(record, version, options)
	if err != nil {
		t.Fatalf("changed request hash failed: %v", err)
	}
	if first == changed {
		t.Fatalf("cutover request hash should include audit reason")
	}

	key := cutoverApplyIdempotencyKey(record, version, normalizeApplyCutoverOptions(record, ApplyCutoverOptions{VersionLabel: "v2"}))
	if !strings.HasPrefix(key, "project.cutover.apply:areamatrix:v2:authoring_cutover:") {
		t.Fatalf("unexpected cutover apply idempotency key: %s", key)
	}
}
