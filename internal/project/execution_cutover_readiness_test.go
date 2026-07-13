package project

import "testing"

func TestAreaMatrixExecutionCutoverReadinessStaysBlockedWithoutExplicitCutover(t *testing.T) {
	record := Record{ID: 1, Key: "areamatrix"}
	verification := ProjectVerificationBundle{
		Project: record,
		Status:  "warn",
		PhaseGate: PhaseGate{
			Name:   "v0.2-shadow-doctor",
			Status: "pass",
		},
	}
	compat := CompatibilityContract{
		Project: record,
		Status:  "pass",
		Commands: []CompatibilityCommand{
			{
				Command:       "./task-loop run",
				Mode:          "blocked",
				Status:        "pass",
				BlockedReason: "execution and task-loop replacement are out of v0.4 scope",
			},
		},
	}
	shim := ShimReadinessFromPreview(ShimPreviewFromCompatibility(compat))
	versions := []WorkflowVersion{
		{
			DisplayLabel:    "v2",
			ImportMode:      "authored",
			LifecycleStatus: "authoring_cutover",
			StatusSummary: map[string]any{
				"authoring_cutover": map[string]any{"applied": true},
			},
		},
	}
	counts := map[string]int{}
	for _, commandType := range executionCutoverCommandEvidenceTypes {
		counts[commandType] = 1
	}

	readiness := AreaMatrixExecutionCutoverReadinessFromParts(record, verification, shim, versions, counts)

	if readiness.Status != "blocked" {
		t.Fatalf("status = %q, want blocked", readiness.Status)
	}
	assertExecutionCutoverItem(t, readiness, "import_mirror_shadow", "pass")
	assertExecutionCutoverItem(t, readiness, "authoring_cutover", "pass")
	assertExecutionCutoverItem(t, readiness, "worker_lease_lifecycle", "pass")
	assertExecutionCutoverItem(t, readiness, "managed_generated_write_apply", "pass")
	assertExecutionCutoverItem(t, readiness, "compatibility_shim", "blocked")
	assertExecutionCutoverItem(t, readiness, "real_areamatrix_generated_apply", "blocked")
	assertExecutionCutoverItem(t, readiness, "copy_repair_checkpoint", "blocked")
	assertExecutionCutoverItem(t, readiness, "explicit_execution_cutover_approval", "blocked")

	if readiness.SafetyFacts["execution_cutover_apply_open"] || readiness.SafetyFacts["project_write_attempted"] || readiness.SafetyFacts["task_loop_run_forwarded"] {
		t.Fatalf("readiness must stay read-only and closed: %+v", readiness.SafetyFacts)
	}
	if !containsString(readiness.ForbiddenActions, "forward_task_loop_run") {
		t.Fatalf("forbidden actions should include task-loop forwarding: %+v", readiness.ForbiddenActions)
	}
}

func TestAreaMatrixExecutionCutoverReadinessReportsMissingCommandEvidence(t *testing.T) {
	record := Record{ID: 1, Key: "areamatrix"}
	readiness := AreaMatrixExecutionCutoverReadinessFromParts(
		record,
		ProjectVerificationBundle{Project: record, PhaseGate: PhaseGate{Name: "v0.2-shadow-doctor", Status: "blocked"}},
		ShimReadiness{Project: record, Status: "blocked"},
		nil,
		map[string]int{"runner.preview": 1},
	)

	item := executionCutoverItem(readiness, "fixture_execution")
	if item.Status != "blocked" {
		t.Fatalf("fixture_execution status = %q, want blocked", item.Status)
	}
	missing, ok := item.Metadata["missing_command_types"].([]string)
	if !ok || !containsString(missing, "run.fixture_queue") || !containsString(missing, "worker.fixture_execute") {
		t.Fatalf("unexpected missing command metadata: %+v", item.Metadata)
	}
}

func assertExecutionCutoverItem(t *testing.T, readiness AreaMatrixExecutionCutoverReadiness, key string, status string) {
	t.Helper()
	item := executionCutoverItem(readiness, key)
	if item.Key == "" {
		t.Fatalf("missing execution cutover item %q", key)
	}
	if item.Status != status {
		t.Fatalf("item %s status = %q, want %q", key, item.Status, status)
	}
}

func executionCutoverItem(readiness AreaMatrixExecutionCutoverReadiness, key string) AreaMatrixExecutionCutoverReadinessItem {
	for _, item := range readiness.Items {
		if item.Key == key {
			return item
		}
	}
	return AreaMatrixExecutionCutoverReadinessItem{}
}
