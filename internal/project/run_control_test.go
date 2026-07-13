package project

import (
	"strings"
	"testing"
)

func TestEvaluateRunControlProtectedLifecycle(t *testing.T) {
	record := Record{ID: 1, Key: "areamatrix"}
	run := RunRecord{ID: 3, ProjectID: 1, Status: "queued", DryRun: true}
	start := evaluateRunControl(record, run, normalizeRunControlOptions(run.ID, RunControlOptions{
		Action: "start",
		Actor:  "local-user",
		Reason: "start protected",
	}))
	if start.Decision != "allowed" || start.Status != "running" || start.ProjectWriteAttempted || start.ExecutionWriteAttempted || start.EngineCallAttempted {
		t.Fatalf("unexpected start result: %+v", start)
	}

	run.Status = "running"
	drain := evaluateRunControl(record, run, normalizeRunControlOptions(run.ID, RunControlOptions{Action: "drain"}))
	if drain.Decision != "allowed" || drain.Status != "draining" {
		t.Fatalf("unexpected drain result: %+v", drain)
	}

	cancelQueued := evaluateRunControl(record, RunRecord{ID: 3, ProjectID: 1, Status: "queued", DryRun: true}, normalizeRunControlOptions(run.ID, RunControlOptions{Action: "cancel"}))
	if cancelQueued.Decision != "allowed" || cancelQueued.Status != "cancelled" {
		t.Fatalf("unexpected queued cancel result: %+v", cancelQueued)
	}

	cancelRunning := evaluateRunControl(record, run, normalizeRunControlOptions(run.ID, RunControlOptions{Action: "cancel"}))
	if cancelRunning.Decision != "allowed" || cancelRunning.Status != "cancelling" {
		t.Fatalf("unexpected running cancel result: %+v", cancelRunning)
	}
}

func TestEvaluateRunControlBlocksNonDryRunAndInvalidTransitions(t *testing.T) {
	record := Record{ID: 1, Key: "areamatrix"}
	run := RunRecord{ID: 4, ProjectID: 1, Status: "passed", DryRun: true}
	result := evaluateRunControl(record, run, normalizeRunControlOptions(run.ID, RunControlOptions{Action: "drain"}))
	if result.Decision != "denied" || result.Status != "passed" || len(result.Blockers) == 0 {
		t.Fatalf("expected drain blocked for passed run: %+v", result)
	}

	nonDryRun := RunRecord{ID: 5, ProjectID: 1, Status: "queued", DryRun: false}
	result = evaluateRunControl(record, nonDryRun, normalizeRunControlOptions(nonDryRun.ID, RunControlOptions{Action: "start"}))
	if result.Decision != "denied" || !strings.Contains(strings.Join(result.Blockers, " "), "dry-run") {
		t.Fatalf("expected non-dry-run blocked: %+v", result)
	}
	if result.ProjectWriteAttempted || result.ExecutionWriteAttempted || result.AreaMatrixWriteAttempted || result.EngineCallAttempted {
		t.Fatalf("blocked control should not attempt forbidden actions: %+v", result)
	}
}

func TestRunControlRequestHashAndIdempotencyKey(t *testing.T) {
	options := normalizeRunControlOptions(3, RunControlOptions{
		Action: " start ",
		Actor:  " local-user ",
		Reason: " start protected ",
	})
	if options.Action != "start" || options.Actor != "local-user" || options.Reason != "start protected" {
		t.Fatalf("unexpected normalized options: %+v", options)
	}
	first, err := runControlRequestHash(3, options)
	if err != nil {
		t.Fatalf("first hash failed: %v", err)
	}
	second, err := runControlRequestHash(3, options)
	if err != nil {
		t.Fatalf("second hash failed: %v", err)
	}
	if first != second {
		t.Fatalf("run control hash should be stable: %s != %s", first, second)
	}
	key := runControlIdempotencyKey(3, options, first)
	if want := "run.start:3:"; len(key) <= len(want) || key[:len(want)] != want {
		t.Fatalf("unexpected run control idempotency key: %s", key)
	}

	changed := options
	changed.Reason = "different reason"
	changedHash, err := runControlRequestHash(3, changed)
	if err != nil {
		t.Fatalf("changed hash failed: %v", err)
	}
	if first == changedHash {
		t.Fatalf("run control hash should include reason")
	}
}

func TestRunControlCommandResponseSafetyFacts(t *testing.T) {
	result := RunControlResult{
		Project:        Record{ID: 1, Key: "areamatrix"},
		Run:            RunRecord{ID: 3, WorkflowVersionID: 2, Status: "running", DryRun: true},
		PreviousStatus: "queued",
		Status:         "running",
		Decision:       "allowed",
		Message:        "run marked running in protected mode",
		EventID:        8,
		AuditEventID:   9,
	}
	response := runControlCommandResponse(result)
	if response["project_write_attempted"] != false ||
		response["execution_write_attempted"] != false ||
		response["area_matrix_write_attempted"] != false ||
		response["engine_call_attempted"] != false ||
		response["task_claimed"] != false ||
		response["worker_started"] != false ||
		response["commands_run"] != false ||
		response["secrets_resolved"] != false ||
		response["network_used"] != false {
		t.Fatalf("command response should record no forbidden attempts: %+v", response)
	}
	if response["run_id"] != int64(3) || response["status"] != "running" || response["previous_status"] != "queued" || response["dry_run"] != true {
		t.Fatalf("unexpected command response: %+v", response)
	}
}
