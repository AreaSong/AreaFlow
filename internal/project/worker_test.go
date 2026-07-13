package project

import (
	"encoding/json"
	"testing"
)

func TestNormalizeCapabilityList(t *testing.T) {
	got := normalizeCapabilityList([]string{" read_project ", "", "write_artifacts", "read_project"})
	want := []string{"read_project", "write_artifacts"}
	if len(got) != len(want) {
		t.Fatalf("capability count = %d, want %d: %+v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("capability[%d] = %q, want %q: %+v", i, got[i], want[i], got)
		}
	}
}

func TestMissingWorkerCapabilities(t *testing.T) {
	missing := missingWorkerCapabilities(
		[]string{"read_project"},
		[]string{"read_project", "write_artifacts", "write_artifacts"},
	)
	if len(missing) != 1 || missing[0] != "write_artifacts" {
		t.Fatalf("missing = %+v, want write_artifacts", missing)
	}
	if missing := missingWorkerCapabilities([]string{"read_project", "write_artifacts"}, []string{"write_artifacts"}); len(missing) != 0 {
		t.Fatalf("missing = %+v, want none", missing)
	}
}

func TestNormalizeWorkerRunOnceOptionsRunID(t *testing.T) {
	options := normalizeWorkerRunOnceOptions(WorkerRunOnceOptions{
		WorkerKey: "local-1",
		RunID:     42,
	})
	if options.RunID != 42 {
		t.Fatalf("run id = %d, want 42", options.RunID)
	}
	if options.LeaseTimeoutSeconds != 300 {
		t.Fatalf("lease timeout = %d, want default 300", options.LeaseTimeoutSeconds)
	}

	options = normalizeWorkerRunOnceOptions(WorkerRunOnceOptions{
		WorkerKey: "local-1",
		RunID:     -1,
	})
	if options.RunID != 0 {
		t.Fatalf("negative run id should normalize to project-level scope, got %d", options.RunID)
	}
}

func TestWorkerLifecycleCommandRequestHashAndIdempotencyKey(t *testing.T) {
	record := Record{Key: "areamatrix"}
	register := normalizeRegisterWorkerOptions(record, RegisterWorkerOptions{
		WorkerKey:                " local-1 ",
		WorkerType:               " local_host ",
		Hostname:                 "dev-host",
		PID:                      123,
		Capabilities:             []string{"write_artifacts", "read_project"},
		Metadata:                 map[string]any{"scope": "fixture"},
		HeartbeatIntervalSeconds: 15,
		LeaseTimeoutSeconds:      120,
		Actor:                    "local-user",
		Reason:                   "register",
	})
	first, err := workerRegisterRequestHash(record, register)
	if err != nil {
		t.Fatalf("worker register hash failed: %v", err)
	}
	second, err := workerRegisterRequestHash(record, register)
	if err != nil {
		t.Fatalf("second worker register hash failed: %v", err)
	}
	if first != second {
		t.Fatalf("worker register hash differed: %s != %s", first, second)
	}
	firstKey := workerRegisterIdempotencyKey(record, register, first)
	secondKey := workerRegisterIdempotencyKey(record, register, first)
	if firstKey == secondKey {
		t.Fatalf("default worker register keys should be unique: %s", firstKey)
	}
	if want := "worker.register:areamatrix:local-1:"; len(firstKey) <= len(want) || firstKey[:len(want)] != want {
		t.Fatalf("unexpected worker register idempotency key: %s", firstKey)
	}

	heartbeat := normalizeWorkerHeartbeatOptions(WorkerHeartbeatOptions{
		Status:   " online ",
		Metadata: map[string]any{"tick": true},
		Actor:    "local-user",
		Reason:   "heartbeat",
	})
	heartbeatHash, err := workerHeartbeatRequestHash(record, " local-1 ", heartbeat)
	if err != nil {
		t.Fatalf("worker heartbeat hash failed: %v", err)
	}
	changed := heartbeat
	changed.Status = "draining"
	changedHash, err := workerHeartbeatRequestHash(record, "local-1", changed)
	if err != nil {
		t.Fatalf("changed worker heartbeat hash failed: %v", err)
	}
	if heartbeatHash == changedHash {
		t.Fatal("worker heartbeat hash should include status")
	}
	heartbeatKey := workerHeartbeatIdempotencyKey(record, " local-1 ", heartbeat, heartbeatHash)
	if want := "worker.heartbeat:areamatrix:local-1:"; len(heartbeatKey) <= len(want) || heartbeatKey[:len(want)] != want {
		t.Fatalf("unexpected worker heartbeat idempotency key: %s", heartbeatKey)
	}
}

func TestLeaseCommandRequestHashAndIdempotencyKey(t *testing.T) {
	record := Record{Key: "areamatrix"}
	acquire := normalizeAcquireLeaseOptions(AcquireLeaseOptions{
		WorkerKey:            " local-1 ",
		RunTaskID:            4,
		LeaseKind:            " run_task ",
		AllowedCapabilities:  []string{"read_project"},
		Scope:                map[string]any{"task": "copy-ready"},
		Metadata:             map[string]any{"dry_run": true},
		LeaseTimeoutSeconds:  120,
		RecoverExpiredBefore: true,
		Actor:                "local-user",
		Reason:               "claim",
	})
	first, err := leaseAcquireRequestHash(record, acquire)
	if err != nil {
		t.Fatalf("lease acquire hash failed: %v", err)
	}
	second, err := leaseAcquireRequestHash(record, acquire)
	if err != nil {
		t.Fatalf("second lease acquire hash failed: %v", err)
	}
	if first != second {
		t.Fatalf("lease acquire hash differed: %s != %s", first, second)
	}
	key := leaseAcquireIdempotencyKey(record, acquire, first)
	if want := "lease.acquire:areamatrix:local-1:4:"; len(key) <= len(want) || key[:len(want)] != want {
		t.Fatalf("unexpected lease acquire idempotency key: %s", key)
	}

	changed := acquire
	changed.Reason = "different audit reason"
	changedHash, err := leaseAcquireRequestHash(record, changed)
	if err != nil {
		t.Fatalf("changed lease acquire hash failed: %v", err)
	}
	if first == changedHash {
		t.Fatalf("lease acquire hash should include reason")
	}
}

func TestLeaseReleaseAndRecoverIdempotencyKeys(t *testing.T) {
	record := Record{Key: "areamatrix"}
	release := normalizeReleaseLeaseOptions(ReleaseLeaseOptions{
		WorkerKey: "local-1",
		LeaseID:   9,
		Status:    "completed",
		Metadata:  map[string]any{"attempt_id": float64(3)},
		Actor:     "local-user",
		Reason:    "done",
	})
	releaseHash, err := leaseReleaseRequestHash(record, release)
	if err != nil {
		t.Fatalf("lease release hash failed: %v", err)
	}
	releaseKey := leaseReleaseIdempotencyKey(record, release, releaseHash)
	if want := "lease.release:areamatrix:local-1:9:completed:"; len(releaseKey) <= len(want) || releaseKey[:len(want)] != want {
		t.Fatalf("unexpected lease release idempotency key: %s", releaseKey)
	}

	recover := normalizeRecoverLeasesOptions(RecoverLeasesOptions{
		Limit:    3,
		Metadata: map[string]any{"trigger": "sweep"},
		Actor:    "local-user",
		Reason:   "recover",
	})
	recoverHash, err := leaseRecoverRequestHash(record, recover)
	if err != nil {
		t.Fatalf("lease recover hash failed: %v", err)
	}
	firstKey := leaseRecoverIdempotencyKey(record, recover, recoverHash)
	secondKey := leaseRecoverIdempotencyKey(record, recover, recoverHash)
	if firstKey == secondKey {
		t.Fatalf("default lease recover keys should be unique for periodic sweeps: %s", firstKey)
	}
	if want := "lease.recover:areamatrix:3:"; len(firstKey) <= len(want) || firstKey[:len(want)] != want {
		t.Fatalf("unexpected lease recover idempotency key: %s", firstKey)
	}
}

func TestCommandResponseValueHelpers(t *testing.T) {
	ids := int64SliceFromAny([]any{float64(1), json.Number("2"), int64(3), "skip"})
	if len(ids) != 3 || ids[0] != 1 || ids[1] != 2 || ids[2] != 3 {
		t.Fatalf("ids = %+v, want [1 2 3]", ids)
	}
	values := stringSliceFromAny([]any{" read_project ", "", "write_artifacts"})
	if len(values) != 2 || values[0] != "read_project" || values[1] != "write_artifacts" {
		t.Fatalf("values = %+v, want normalized strings", values)
	}
	if got := metadataInt64(map[string]any{"lease_id": float64(9)}, "lease_id"); got != 9 {
		t.Fatalf("metadata int64 = %d, want 9", got)
	}
}

func TestLeaseCommandResponseSafetyFacts(t *testing.T) {
	response := leaseCommandResponse(map[string]any{
		"decision":      "allowed",
		"lease_id":      int64(7),
		"lease_created": true,
	})
	if response["lease_id"] != int64(7) || response["lease_created"] != true {
		t.Fatalf("lease response should preserve lease facts: %+v", response)
	}
	if response["project_write_attempted"] != false ||
		response["execution_write_attempted"] != false ||
		response["engine_call_attempted"] != false ||
		response["commands_run"] != false ||
		response["secrets_resolved"] != false ||
		response["network_used"] != false ||
		response["attempt_created"] != false ||
		response["artifact_created"] != false ||
		response["worker_run_once"] != false {
		t.Fatalf("lease response should record dry-run boundary facts: %+v", response)
	}

	denied := leaseCommandResponse(map[string]any{"decision": "denied"})
	if denied["lease_created"] != false {
		t.Fatalf("denied lease response should default lease_created=false: %+v", denied)
	}
}

func TestWorkerLifecycleCommandResponseSafetyFacts(t *testing.T) {
	response := workerLifecycleCommandResponse(map[string]any{
		"decision":           "allowed",
		"worker_id":          int64(7),
		"heartbeat_recorded": true,
	})
	if response["worker_id"] != int64(7) || response["heartbeat_recorded"] != true {
		t.Fatalf("worker lifecycle response should preserve worker facts: %+v", response)
	}
	if response["project_write_attempted"] != false ||
		response["execution_write_attempted"] != false ||
		response["engine_call_attempted"] != false ||
		response["commands_run"] != false ||
		response["secrets_resolved"] != false ||
		response["network_used"] != false ||
		response["lease_created"] != false ||
		response["attempt_created"] != false ||
		response["artifact_created"] != false ||
		response["worker_run_once"] != false {
		t.Fatalf("worker lifecycle response should record dry-run boundary facts: %+v", response)
	}
}

func TestLeaseAcquireRequestHashStableAfterScopeCopy(t *testing.T) {
	record := Record{Key: "areamatrix"}
	options := normalizeAcquireLeaseOptions(AcquireLeaseOptions{
		WorkerKey: "local-1",
		RunTaskID: 4,
		Scope:     map[string]any{"custom": "value"},
	})
	before, err := leaseAcquireRequestHash(record, options)
	if err != nil {
		t.Fatalf("hash before scope merge failed: %v", err)
	}
	scopePayload := map[string]any{}
	for key, value := range options.Scope {
		scopePayload[key] = value
	}
	scopePayload["run_task_id"] = int64(4)
	scopePayload["task_key"] = "task-1"
	scopePayload["task_kind"] = "copy_ready"
	if _, exists := options.Scope["task_key"]; exists {
		t.Fatalf("scope should not be mutated by lease scope payload construction: %+v", options.Scope)
	}
	after, err := leaseAcquireRequestHash(record, options)
	if err != nil {
		t.Fatalf("hash after scope merge simulation failed: %v", err)
	}
	if before != after {
		t.Fatalf("hash should stay stable when internal scope payload is copied: %s != %s", before, after)
	}
}

func TestBuildWorkerPoolSchedulePreview(t *testing.T) {
	preview := BuildWorkerPoolSchedulePreview(WorkerPoolSummary{
		Projects: []WorkerPoolProjectSummary{
			{
				Project:       Record{Key: "blocked"},
				QueuedTasks:   2,
				OnlineWorkers: 0,
				Capabilities:  []string{"read_project"},
				Scheduling: SchedulingPolicy{
					Priority:             50,
					MaxParallelTasks:     2,
					AgentRole:            "remote_worker",
					RequiredCapabilities: []string{"read_project"},
					EngineProfile:        "codex-cli",
				},
			},
			{
				Project:       Record{Key: "ready"},
				QueuedTasks:   1,
				OnlineWorkers: 2,
				ActiveLeases:  1,
				Capabilities:  []string{"read_project", "write_artifacts"},
				WorkerTypes:   []string{"local_host"},
				Scheduling: SchedulingPolicy{
					Priority:             150,
					MaxParallelTasks:     3,
					AgentRole:            "local_worker",
					RequiredCapabilities: []string{"read_project", "write_artifacts"},
					EngineProfile:        "codex-cli",
				},
			},
		},
	})
	if preview.Recommended != 1 || preview.Blocked != 1 || preview.QueuedTasks != 3 || preview.AvailableSlot != 1 {
		t.Fatalf("unexpected preview totals: %+v", preview)
	}
	if len(preview.Projects) != 2 || preview.Projects[0].Project.Key != "ready" {
		t.Fatalf("recommended project should be sorted first: %+v", preview.Projects)
	}
	if !preview.Projects[0].Recommended || preview.Projects[0].NextAction != "worker_run_once_preview" {
		t.Fatalf("ready project should be recommended: %+v", preview.Projects[0])
	}
	if preview.Projects[0].Priority != 150 || preview.Projects[0].MaxParallel != 3 {
		t.Fatalf("ready project should use scheduling policy: %+v", preview.Projects[0])
	}
	if preview.Projects[0].Role.Status != "ready" || !preview.Projects[0].Role.Matched {
		t.Fatalf("ready project should match local worker role: %+v", preview.Projects[0].Role)
	}
	if len(preview.Projects[0].RequiredCaps) != 2 || preview.Projects[0].RequiredCaps[1] != "write_artifacts" {
		t.Fatalf("ready project required capabilities = %+v", preview.Projects[0].RequiredCaps)
	}
	if preview.Projects[1].Recommended || len(preview.Projects[1].BlockedReasons) == 0 {
		t.Fatalf("blocked project should include blockers: %+v", preview.Projects[1])
	}
}

func TestBuildWorkerPoolSchedulePreviewMissingRequiredCapability(t *testing.T) {
	preview := BuildWorkerPoolSchedulePreview(WorkerPoolSummary{
		Projects: []WorkerPoolProjectSummary{
			{
				Project:       Record{Key: "missing-cap"},
				QueuedTasks:   1,
				OnlineWorkers: 1,
				Capabilities:  []string{"read_project"},
				WorkerTypes:   []string{"local_host"},
				Scheduling: SchedulingPolicy{
					Priority:             100,
					MaxParallelTasks:     1,
					AgentRole:            "local_worker",
					RequiredCapabilities: []string{"read_project", "write_artifacts"},
				},
			},
		},
	})
	if preview.Recommended != 0 || preview.Blocked != 1 {
		t.Fatalf("unexpected preview totals: %+v", preview)
	}
	if len(preview.Projects) != 1 {
		t.Fatalf("project count = %d", len(preview.Projects))
	}
	got := preview.Projects[0].BlockedReasons
	if len(got) != 1 || got[0] != "missing_required_capability:write_artifacts" {
		t.Fatalf("blocked reasons = %+v", got)
	}
}

func TestBuildWorkerPoolSchedulePreviewBlockedByEngine(t *testing.T) {
	preview := BuildWorkerPoolSchedulePreview(WorkerPoolSummary{
		Projects: []WorkerPoolProjectSummary{
			{
				Project:       Record{Key: "engine-blocked"},
				QueuedTasks:   1,
				OnlineWorkers: 1,
				Capabilities:  []string{"read_project"},
				WorkerTypes:   []string{"local_host"},
				Scheduling: SchedulingPolicy{
					Priority:             100,
					MaxParallelTasks:     1,
					AgentRole:            "local_worker",
					RequiredCapabilities: []string{"read_project"},
					EngineProfile:        "codex-cli",
				},
				Engine: EngineReadiness{
					ProfileID:      "codex-cli",
					Provider:       "codex-cli",
					SecretRef:      "none",
					SecretReady:    true,
					ResourceLimits: map[string]any{},
					Status:         "blocked",
					BlockedReasons: []string{"engine_profile_disabled"},
				},
			},
		},
	})
	if preview.Recommended != 0 || preview.Blocked != 1 {
		t.Fatalf("unexpected preview totals: %+v", preview)
	}
	got := preview.Projects[0].BlockedReasons
	if len(got) != 1 || got[0] != "engine_profile_disabled" {
		t.Fatalf("blocked reasons = %+v", got)
	}
}

func TestBuildWorkerPoolSchedulePreviewBlockedByResourceLimit(t *testing.T) {
	preview := BuildWorkerPoolSchedulePreview(WorkerPoolSummary{
		Projects: []WorkerPoolProjectSummary{
			{
				Project:       Record{Key: "resource-blocked"},
				QueuedTasks:   1,
				OnlineWorkers: 2,
				ActiveLeases:  1,
				Capabilities:  []string{"read_project"},
				WorkerTypes:   []string{"local_host"},
				Scheduling: SchedulingPolicy{
					Priority:             100,
					MaxParallelTasks:     2,
					AgentRole:            "local_worker",
					RequiredCapabilities: []string{"read_project"},
					EngineProfile:        "codex-cli",
				},
				Engine: EngineReadiness{
					ProfileID:      "codex-cli",
					Provider:       "codex-cli",
					Enabled:        true,
					SecretRef:      "none",
					SecretReady:    true,
					ResourceLimits: map[string]any{"max_active_leases": float64(1)},
					Status:         "ready",
				},
			},
		},
	})
	if preview.Recommended != 0 || preview.Blocked != 1 {
		t.Fatalf("unexpected preview totals: %+v", preview)
	}
	got := preview.Projects[0].BlockedReasons
	if len(got) != 1 || got[0] != "resource_limit:max_active_leases" {
		t.Fatalf("blocked reasons = %+v", got)
	}
	if preview.Projects[0].Resources.Status != "blocked" {
		t.Fatalf("resource status = %+v", preview.Projects[0].Resources)
	}
}

func TestBuildWorkerPoolSchedulePreviewBlockedByAgentRole(t *testing.T) {
	preview := BuildWorkerPoolSchedulePreview(WorkerPoolSummary{
		Projects: []WorkerPoolProjectSummary{
			{
				Project:       Record{Key: "role-blocked"},
				QueuedTasks:   1,
				OnlineWorkers: 1,
				Capabilities:  []string{"read_project"},
				WorkerTypes:   []string{"local_host"},
				Scheduling: SchedulingPolicy{
					Priority:             100,
					MaxParallelTasks:     1,
					AgentRole:            "remote_worker",
					RequiredCapabilities: []string{"read_project"},
				},
			},
		},
	})
	if preview.Recommended != 0 || preview.Blocked != 1 {
		t.Fatalf("unexpected preview totals: %+v", preview)
	}
	got := preview.Projects[0].BlockedReasons
	if len(got) != 1 || got[0] != "missing_agent_role:remote_worker" {
		t.Fatalf("blocked reasons = %+v", got)
	}
}
