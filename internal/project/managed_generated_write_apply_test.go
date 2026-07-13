package project

import "testing"

func TestManagedGeneratedWriteQueueCommandResponseSafetyFacts(t *testing.T) {
	result := ManagedGeneratedWriteQueueResult{
		Project:                       Record{ID: 1, Key: "areamatrix-fixture"},
		Version:                       WorkflowVersion{ID: 2, DisplayLabel: "v2"},
		Run:                           RunRecord{ID: 3, Status: "queued", DryRun: false},
		Task:                          RunTaskRecord{ID: 4, Status: "queued"},
		WriteSetArtifact:              ArtifactRecord{ID: 5, ArtifactType: "managed_generated_write_set"},
		TargetPath:                    ".areaflow/generated/status.json",
		ExpectedBeforeSHA256:          "before123",
		ExpectedBeforeSize:            12,
		AfterSHA256:                   "after123",
		AfterSize:                     13,
		EventID:                       6,
		AuditEventID:                  7,
		IdempotencyKey:                "managed-generated-write-queue",
		GeneratedOnly:                 true,
		GeneratedOnlyApplyOpen:        true,
		AreaFlowArtifactWritten:       true,
		AreaFlowExecutionStateWritten: true,
	}

	response := managedGeneratedWriteQueueCommandResponse(result)
	if response["project_read_attempted"] != false ||
		response["project_write_attempted"] != false ||
		response["execution_write_attempted"] != false ||
		response["area_flow_artifact_written"] != true ||
		response["area_flow_execution_state_written"] != true ||
		response["engine_call_attempted"] != false ||
		response["commands_run"] != false ||
		response["secrets_resolved"] != false ||
		response["network_used"] != false ||
		response["real_areamatrix_write_opened"] != false {
		t.Fatalf("managed generated write queue response should preserve queue-only safety facts: %+v", response)
	}
	if response["generated_only"] != true ||
		response["generated_only_apply_open"] != true ||
		response["target_path"] != ".areaflow/generated/status.json" ||
		response["write_set_artifact_id"] != int64(5) {
		t.Fatalf("unexpected managed generated write queue facts: %+v", response)
	}
}

func TestManagedGeneratedWriteCommandResponseSafetyFacts(t *testing.T) {
	result := ManagedGeneratedWriteResult{
		Project:                       Record{ID: 1, Key: "areamatrix-fixture"},
		Version:                       WorkflowVersion{ID: 2, DisplayLabel: "v2"},
		Run:                           RunRecord{ID: 3, Status: "rollback_verified", DryRun: false},
		Worker:                        WorkerRecord{ID: 4, WorkerKey: "local-1"},
		Task:                          RunTaskRecord{ID: 5, Status: "rollback_verified"},
		Lease:                         LeaseRecord{ID: 6, Status: "completed"},
		CopyAttempt:                   RunAttemptRecord{ID: 7, AttemptKind: "copy", Status: "passed", DryRun: false},
		VerifyAttempt:                 RunAttemptRecord{ID: 8, AttemptKind: "verify", Status: "passed", DryRun: false},
		RollbackAttempt:               RunAttemptRecord{ID: 9, AttemptKind: "rollback", Status: "passed", DryRun: false},
		WriteSetArtifact:              ArtifactRecord{ID: 10, ArtifactType: "managed_generated_write_set"},
		PreimageArtifact:              ArtifactRecord{ID: 11, ArtifactType: "managed_generated_write_preimage"},
		Artifact:                      ArtifactRecord{ID: 12, ArtifactType: "managed_generated_write_report"},
		Gate:                          ExecutionApprovalGate{Status: "pass"},
		TargetPath:                    ".areamatrix/generated/summary.json",
		ExpectedBeforeSHA256:          "before123",
		ExpectedBeforeSize:            12,
		AfterSHA256:                   "after123",
		AfterSize:                     13,
		RestoredSHA256:                "before123",
		RestoredSize:                  12,
		Status:                        "rollback_verified",
		Decision:                      "allowed",
		Message:                       "managed generated write verified and rolled back in fixture/temp project",
		EventID:                       13,
		AuditEventID:                  14,
		IdempotencyKey:                "managed-generated-write",
		GeneratedOnly:                 true,
		GeneratedOnlyApplyOpen:        true,
		ProjectReadAttempted:          true,
		ProjectReadAllowed:            true,
		ProjectWriteAttempted:         true,
		ProjectWriteAllowed:           true,
		ExecutionWriteAttempted:       false,
		AreaFlowArtifactWritten:       true,
		AreaFlowExecutionStateWritten: true,
		EngineCallAttempted:           false,
		CommandsRun:                   false,
		SecretsResolved:               false,
		NetworkUsed:                   false,
		TaskClaimed:                   true,
		LeaseCreated:                  true,
		AttemptCreated:                true,
		ArtifactCreated:               true,
		WriteSetPassed:                true,
		VerificationPassed:            true,
		RollbackAttempted:             true,
		RollbackVerified:              true,
	}

	response := managedGeneratedWriteCommandResponse(result)
	if response["project_read_attempted"] != true ||
		response["project_read_allowed"] != true ||
		response["project_write_attempted"] != true ||
		response["project_write_allowed"] != true ||
		response["execution_write_attempted"] != false ||
		response["area_flow_artifact_written"] != true ||
		response["area_flow_execution_state_written"] != true ||
		response["engine_call_attempted"] != false ||
		response["commands_run"] != false ||
		response["secrets_resolved"] != false ||
		response["network_used"] != false ||
		response["real_areamatrix_write_opened"] != false {
		t.Fatalf("managed generated write response should preserve generated-only safety facts: %+v", response)
	}
	if response["artifact_type"] != "managed_generated_write_report" ||
		response["target_path"] != ".areamatrix/generated/summary.json" ||
		response["generated_only"] != true ||
		response["write_set_passed"] != true ||
		response["verification_passed"] != true ||
		response["rollback_verified"] != true {
		t.Fatalf("unexpected managed generated write facts: %+v", response)
	}
}

func TestNormalizeManagedGeneratedWriteOptions(t *testing.T) {
	options := normalizeManagedGeneratedWriteOptions(ManagedGeneratedWriteOptions{
		WorkerKey: " local-1 ",
		RunID:     42,
	})
	if options.WorkerKey != "local-1" || options.RunID != 42 {
		t.Fatalf("unexpected normalized managed generated write options: %+v", options)
	}
	if options.LeaseTimeoutSeconds != 300 {
		t.Fatalf("lease timeout = %d, want 300", options.LeaseTimeoutSeconds)
	}
	if len(options.AllowedCapabilities) != 3 ||
		options.AllowedCapabilities[0] != "read_project" ||
		options.AllowedCapabilities[1] != "write_artifacts" ||
		options.AllowedCapabilities[2] != "write_generated" {
		t.Fatalf("required capabilities = %+v, want read_project/write_artifacts/write_generated", options.AllowedCapabilities)
	}
	if options.Actor != "local-user" || options.Reason == "" || options.Metadata == nil {
		t.Fatalf("default actor/reason/metadata not applied: %+v", options)
	}
}

func TestNormalizeManagedGeneratedWriteQueueOptions(t *testing.T) {
	options := normalizeManagedGeneratedWriteQueueOptions(
		Record{ID: 1, Key: "areamatrix-fixture"},
		WorkflowVersion{ID: 2, DisplayLabel: "v2"},
		ManagedGeneratedWriteQueueOptions{
			TargetPath:           " .areaflow/generated/../generated/status.json ",
			Content:              "after",
			ExpectedBeforeSHA256: " ABC123 ",
			ExpectedBeforeSize:   6,
		},
	)
	if options.TargetPath != ".areaflow/generated/status.json" {
		t.Fatalf("target path = %q, want .areaflow/generated/status.json", options.TargetPath)
	}
	if options.ExpectedBeforeSHA256 != "abc123" {
		t.Fatalf("expected before sha256 = %q, want abc123", options.ExpectedBeforeSHA256)
	}
	if options.Actor != "local-user" || options.Reason == "" || options.IdempotencyKey == "" {
		t.Fatalf("default queue fields not applied: %+v", options)
	}
}

func TestManagedGeneratedPathPolicy(t *testing.T) {
	for _, targetPath := range []string{
		".areaflow/generated/status.json",
		".areamatrix/generated/summary.json",
	} {
		if !isManagedGeneratedPath(targetPath) {
			t.Fatalf("expected generated path to be allowed: %s", targetPath)
		}
	}
	for _, targetPath := range []string{
		".areaflow/status.json",
		".areaflow/generated",
		".areamatrix/AREAMATRIX.md",
		"docs/generated/status.json",
		"../.areaflow/generated/status.json",
	} {
		if isManagedGeneratedPath(targetPath) {
			t.Fatalf("expected generated path to be denied: %s", targetPath)
		}
	}
}

func TestFixtureOrTempProjectRecordPolicy(t *testing.T) {
	if !isFixtureOrTempProjectRecord(Record{Key: "areamatrix-fixture"}) {
		t.Fatal("fixture project key should be allowed")
	}
	if !isFixtureOrTempProjectRecord(Record{Kind: "temporary-project"}) {
		t.Fatal("temporary project kind should be allowed")
	}
	if isFixtureOrTempProjectRecord(Record{Key: "areamatrix", Kind: "product-repo"}) {
		t.Fatal("real AreaMatrix project should be denied")
	}
}
