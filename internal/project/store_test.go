package project

import (
	"strings"
	"testing"
	"time"
)

func TestCheckStatus(t *testing.T) {
	metadata := map[string]any{
		"checks": []any{
			map[string]any{"name": "hash_drift", "status": "pass"},
			map[string]any{"name": "project_config_drift", "status": "warn"},
			map[string]any{"name": "stage_coverage", "status": "warn"},
			map[string]any{"name": "native_workflow_doctor", "status": "warn"},
		},
	}

	if got := checkStatus(metadata, "hash_drift"); got != "pass" {
		t.Fatalf("hash_drift status = %q, want pass", got)
	}
	if got := checkStatus(metadata, "project_config_drift"); got != "warn" {
		t.Fatalf("project_config_drift status = %q, want warn", got)
	}
	if got := checkStatus(metadata, "stage_coverage"); got != "warn" {
		t.Fatalf("stage_coverage status = %q, want warn", got)
	}
	if got := checkStatus(metadata, "native_workflow_doctor"); got != "warn" {
		t.Fatalf("native_workflow_doctor status = %q, want warn", got)
	}
	if got := checkStatus(metadata, "missing"); got != "" {
		t.Fatalf("missing status = %q, want empty", got)
	}
}

func TestStatusProjectionSummaryState(t *testing.T) {
	if got := statusProjectionSummaryState(nil); got != "mirroring" {
		t.Fatalf("nil summary state = %q, want mirroring", got)
	}
	if got := statusProjectionSummaryState(map[string]any{"version_count": float64(2)}); got != "mirroring" {
		t.Fatalf("default summary state = %q, want mirroring", got)
	}
	if got := statusProjectionSummaryState(map[string]any{"summary_state": " blocked "}); got != "blocked" {
		t.Fatalf("explicit summary state = %q, want blocked", got)
	}
}

func TestStatusProjectionWriteIdempotencyKey(t *testing.T) {
	got := statusProjectionWriteIdempotencyKey(" .areaflow/status.json ", " hash-a ", []byte(`{"version_count":1}`))
	prefix := "project.status_projection.write:.areaflow/status.json:hash-a:"
	if !strings.HasPrefix(got, prefix) {
		t.Fatalf("idempotency key = %q, want prefix %q", got, prefix)
	}

	same := statusProjectionWriteIdempotencyKey(".areaflow/status.json", "hash-a", []byte(`{"version_count":1}`))
	if got != same {
		t.Fatalf("same idempotency key differed: %q != %q", got, same)
	}

	changed := statusProjectionWriteIdempotencyKey(".areaflow/status.json", "hash-a", []byte(`{"version_count":2}`))
	if got == changed {
		t.Fatalf("idempotency key should change with payload hash: %q", got)
	}

	got = statusProjectionWriteIdempotencyKey("", "", nil)
	prefix = "project.status_projection.write:unknown-target:no-source-hash:"
	if !strings.HasPrefix(got, prefix) {
		t.Fatalf("fallback idempotency key = %q, want prefix %q", got, prefix)
	}
}

func TestStatusProjectionWriteRequestHashIncludesPayload(t *testing.T) {
	first, err := statusProjectionWriteRequestHash(".areaflow/status.json", []byte(`{"version_count":1}`), "hash-a")
	if err != nil {
		t.Fatalf("first hash failed: %v", err)
	}
	second, err := statusProjectionWriteRequestHash(".areaflow/status.json", []byte(`{"version_count":1}`), "hash-a")
	if err != nil {
		t.Fatalf("second hash failed: %v", err)
	}
	if first != second {
		t.Fatalf("same projection request hash differed: %s != %s", first, second)
	}

	changed, err := statusProjectionWriteRequestHash(".areaflow/status.json", []byte(`{"version_count":2}`), "hash-a")
	if err != nil {
		t.Fatalf("changed hash failed: %v", err)
	}
	if first == changed {
		t.Fatalf("hash should change when payload changes: %s", first)
	}
}

func TestDoctorReportCommandRequestHashAndDefaultKey(t *testing.T) {
	summaryJSON := []byte(`{"overall_status":"warn","checks":[]}`)
	options := normalizeRecordDoctorReportOptions(RecordDoctorReportOptions{
		Actor:  "local-user",
		Reason: "doctor run",
	})
	first, err := doctorReportRequestHash(1, summaryJSON, options)
	if err != nil {
		t.Fatalf("first doctor hash failed: %v", err)
	}
	second, err := doctorReportRequestHash(1, summaryJSON, options)
	if err != nil {
		t.Fatalf("second doctor hash failed: %v", err)
	}
	if first != second {
		t.Fatalf("doctor hash differed: %s != %s", first, second)
	}
	options.Reason = "different reason"
	changed, err := doctorReportRequestHash(1, summaryJSON, options)
	if err != nil {
		t.Fatalf("changed doctor hash failed: %v", err)
	}
	if first == changed {
		t.Fatalf("doctor hash should include audit reason")
	}

	firstKey := doctorReportIdempotencyKey(1, summaryJSON)
	secondKey := doctorReportIdempotencyKey(1, summaryJSON)
	if firstKey == secondKey {
		t.Fatalf("default doctor report keys should be unique for repeated doctor runs: %s", firstKey)
	}
	if !strings.HasPrefix(firstKey, "project.doctor.record:1:") {
		t.Fatalf("unexpected doctor report key: %s", firstKey)
	}
}

func TestProjectReadinessFromSummary(t *testing.T) {
	summary := ProjectSummary{
		Project: Record{Key: "areamatrix"},
		Inventory: ImportInventory{
			Versions:        2,
			Residuals:       10,
			Artifacts:       6,
			ImportSnapshots: 1,
			MirrorExports:   1,
		},
		HasImport:           true,
		HasLatestDoctor:     true,
		DoctorStatus:        "pass",
		DriftStatus:         "pass",
		ConfigDriftStatus:   "pass",
		StageCoverageStatus: "pass",
		NativeDoctorStatus:  "warn",
		LatestEventCount:    3,
		Config:              testProjectConfigRecord(),
		HasConfig:           true,
	}

	readiness := ProjectReadinessFromSummary(summary)

	if readiness.Project.Key != "areamatrix" {
		t.Fatalf("project key = %q, want areamatrix", readiness.Project.Key)
	}
	if readiness.Status != "warn" {
		t.Fatalf("readiness status = %q, want warn", readiness.Status)
	}
	if len(readiness.Items) != 11 {
		t.Fatalf("readiness item count = %d, want 11", len(readiness.Items))
	}
	assertReadinessItem(t, readiness, "import_snapshot", "pass")
	assertReadinessItem(t, readiness, "import_history", "warn")
	assertReadinessItem(t, readiness, "status_mirror", "pass")
	assertReadinessItem(t, readiness, "events_timeline", "pass")
	assertReadinessItem(t, readiness, "summary_api_ready", "pass")
	assertReadinessItem(t, readiness, "project_config", "pass")
	assertReadinessItem(t, readiness, "doctor_report", "pass")
	assertReadinessItem(t, readiness, "drift_check", "pass")
	assertReadinessItem(t, readiness, "project_config_drift", "pass")
	assertReadinessItem(t, readiness, "stage_coverage", "pass")
	assertReadinessItem(t, readiness, "native_workflow_doctor", "warn")
}

func TestProjectReadinessFromSummaryMissingImport(t *testing.T) {
	readiness := ProjectReadinessFromSummary(ProjectSummary{
		Project: Record{Key: "areamatrix"},
	})

	if readiness.Status != "warn" {
		t.Fatalf("readiness status = %q, want warn", readiness.Status)
	}
	assertReadinessItem(t, readiness, "import_snapshot", "warn")
	assertReadinessItem(t, readiness, "import_history", "warn")
	assertReadinessItem(t, readiness, "project_config", "warn")
	assertReadinessItem(t, readiness, "project_config_drift", "warn")
	assertReadinessItem(t, readiness, "doctor_report", "warn")
}

func TestProjectReadinessFromSummaryWithImportHistory(t *testing.T) {
	readiness := ProjectReadinessFromSummary(ProjectSummary{
		Project:           Record{Key: "areamatrix"},
		HasImport:         true,
		HasPreviousImport: true,
	})

	assertReadinessItem(t, readiness, "import_history", "pass")
}

func testProjectConfigRecord() ProjectConfigRecord {
	loadedAt := time.Date(2026, 6, 29, 3, 2, 0, 0, time.UTC)
	return ProjectConfigRecord{
		ID:              1,
		ProjectID:       1,
		ProtocolVersion: 1,
		ConfigPath:      "examples/areamatrix/areaflow.yaml",
		ConfigHash:      "hash-config",
		Ownership:       map[string]any{"mode": "import"},
		StatusExport:    map[string]any{"path": ".areaflow/status.json"},
		Migration:       map[string]any{"phase": "import"},
		Active:          true,
		LoadedAt:        loadedAt,
	}
}

func TestCompatibilityContractFromSummary(t *testing.T) {
	summary := ProjectSummary{
		Project: Record{Key: "areamatrix"},
		Inventory: ImportInventory{
			MirrorExports: 1,
		},
	}
	permissions := map[string]CommandPermission{
		"./dev workflow doctor": {
			CapabilityAllowed: true,
			CommandAllowed:    true,
		},
	}

	contract := CompatibilityContractFromSummary(summary, permissions)

	if contract.Status != "pass" {
		t.Fatalf("contract status = %q, want pass", contract.Status)
	}
	assertCompatibilityCommand(t, contract, "./dev workflow status", "forward", "pass")
	assertCompatibilityCommand(t, contract, "./dev workflow doctor", "forward", "pass")
	assertCompatibilityCommand(t, contract, "./task-loop run", "blocked", "pass")
}

func TestCompatibilityContractWarnsWhenFallbackMissing(t *testing.T) {
	contract := CompatibilityContractFromSummary(ProjectSummary{
		Project: Record{Key: "areamatrix"},
	}, map[string]CommandPermission{})

	if contract.Status != "warn" {
		t.Fatalf("contract status = %q, want warn", contract.Status)
	}
	assertCompatibilityCommand(t, contract, "./dev workflow status", "fallback_status", "warn")
	assertCompatibilityCommand(t, contract, "./dev workflow doctor", "fallback_status", "warn")
}

func TestShimPreviewFromCompatibility(t *testing.T) {
	contract := CompatibilityContractFromSummary(ProjectSummary{
		Project: Record{Key: "areamatrix"},
		Inventory: ImportInventory{
			MirrorExports: 1,
		},
	}, map[string]CommandPermission{})

	preview := ShimPreviewFromCompatibility(contract)

	if preview.Project.Key != "areamatrix" || preview.Mode != "read_only_planning" {
		t.Fatalf("unexpected shim preview identity: %+v", preview)
	}
	if len(preview.PlannedFiles) == 0 || preview.PlannedFiles[0].Path != "scripts/areaflow_shim.py" {
		t.Fatalf("unexpected planned files: %+v", preview.PlannedFiles)
	}
	if preview.PlannedFiles[0].Required {
		t.Fatalf("areaflow shim helper should be optional in the first read-only shim plan: %+v", preview.PlannedFiles[0])
	}
	if !containsString(preview.ForbiddenCommands, "./task-loop run") {
		t.Fatalf("expected task-loop run to be forbidden: %+v", preview.ForbiddenCommands)
	}
	var taskLoopRun ShimCommandMapping
	for _, mapping := range preview.CommandMappings {
		if mapping.Command == "./task-loop run" {
			taskLoopRun = mapping
		}
	}
	if taskLoopRun.Mode != "blocked" || taskLoopRun.Status != "pass" {
		t.Fatalf("unexpected task-loop run mapping: %+v", taskLoopRun)
	}
}

func TestShimReadinessFromPreviewBlocksWithoutExternalApprovalEvidence(t *testing.T) {
	contract := CompatibilityContractFromSummary(ProjectSummary{
		Project: Record{Key: "areamatrix"},
		Inventory: ImportInventory{
			MirrorExports: 1,
		},
	}, map[string]CommandPermission{})
	readiness := ShimReadinessFromPreview(ShimPreviewFromCompatibility(contract))

	if readiness.Status != "blocked" {
		t.Fatalf("shim readiness status = %q, want blocked", readiness.Status)
	}
	statusProjection := shimReadinessItem(t, readiness, "status_projection")
	if statusProjection.Metadata["schema_contract"] != "stable_fallback_projection_v1" {
		t.Fatalf("unexpected status projection schema contract: %+v", statusProjection.Metadata)
	}
	if statusProjection.Metadata["target_uri"] != ".areaflow/status.json" {
		t.Fatalf("unexpected status projection target uri: %+v", statusProjection.Metadata)
	}
	if statusProjection.Metadata["schema_uri"] != "schemas/status-projection.schema.json" {
		t.Fatalf("unexpected status projection schema uri: %+v", statusProjection.Metadata)
	}
	if statusProjection.Metadata["validator_preflight"] != "python3 scripts/validate-status-projection-schema.py schemas/status-projection.schema.json /Users/as/Ai-Project/project/AreaMatrix/.areaflow/status.json" {
		t.Fatalf("unexpected status projection validator preflight: %+v", statusProjection.Metadata)
	}
	requiredFields, ok := statusProjection.Metadata["required_schema_fields"].([]string)
	if !ok {
		t.Fatalf("required_schema_fields metadata missing or wrong type: %+v", statusProjection.Metadata)
	}
	for _, want := range []string{
		"schema_version",
		"project_id",
		"active_versions[].rough_progress.percent",
		"source_snapshot_hash",
		"compatibility.blocked_commands[]",
	} {
		if !containsString(requiredFields, want) {
			t.Fatalf("required_schema_fields missing %q: %+v", want, requiredFields)
		}
	}
	forbiddenFields, ok := statusProjection.Metadata["forbidden_fields"].([]string)
	if !ok {
		t.Fatalf("forbidden_fields metadata missing or wrong type: %+v", statusProjection.Metadata)
	}
	for _, want := range []string{"summary", "generated_at", "source_hash", "secret", "artifact_content"} {
		if !containsString(forbiddenFields, want) {
			t.Fatalf("forbidden_fields missing %q: %+v", want, forbiddenFields)
		}
	}
	assertShimReadinessItem(t, readiness, "status_projection", "pass")
	assertShimReadinessItem(t, readiness, "task_loop_run_blocked", "pass")
	assertShimReadinessItem(t, readiness, "real_areamatrix_readonly_smoke", "blocked")
	statusSchemaEvidence := shimReadinessItem(t, readiness, "real_areamatrix_status_projection_schema")
	if statusSchemaEvidence.Status != "blocked" ||
		statusSchemaEvidence.Metadata["schema_uri"] != "schemas/status-projection.schema.json" ||
		statusSchemaEvidence.Metadata["validator_preflight"] != "python3 scripts/validate-status-projection-schema.py schemas/status-projection.schema.json /Users/as/Ai-Project/project/AreaMatrix/.areaflow/status.json" {
		t.Fatalf("unexpected status projection schema evidence item: %+v", statusSchemaEvidence)
	}
	assertShimReadinessItem(t, readiness, "explicit_edit_approval", "blocked")
}

func TestShimAuthorizationPacketFromReadiness(t *testing.T) {
	contract := CompatibilityContractFromSummary(ProjectSummary{
		Project: Record{Key: "areamatrix"},
		Inventory: ImportInventory{
			MirrorExports: 1,
		},
	}, map[string]CommandPermission{})
	readiness := ShimReadinessFromPreview(ShimPreviewFromCompatibility(contract))

	packet := ShimAuthorizationPacketFromReadiness(readiness)

	if packet.Project.Key != "areamatrix" || packet.Status != "blocked" || packet.Mode != "read_only_authorization_packet" {
		t.Fatalf("unexpected shim authorization identity: %+v", packet)
	}
	if packet.ReadinessStatus != "blocked" {
		t.Fatalf("readiness status = %q, want blocked", packet.ReadinessStatus)
	}
	assertShimAuthorizationFile(t, packet, "scripts/areaflow_shim.py")
	assertShimAuthorizationFile(t, packet, "scripts/task_loop/console.py")
	assertShimAuthorizationFile(t, packet, "scripts/dev_tools/cli.py")
	assertShimAuthorizationFile(t, packet, "scripts/task_loop/runner.py")
	assertShimAuthorizationFile(t, packet, "workflow/README.md")
	assertShimAuthorizationFileAbsent(t, packet, ".areaflow/status.json")
	if !containsString(packet.ForbiddenPaths, "workflow/versions/**") {
		t.Fatalf("expected workflow versions to be forbidden: %+v", packet.ForbiddenPaths)
	}
	if !containsString(packet.ForbiddenActions, "task-loop run forwarding") {
		t.Fatalf("expected task-loop run forwarding to be forbidden: %+v", packet.ForbiddenActions)
	}
	if !containsString(packet.RequiredPreflight, "areaflow project shim-readiness areamatrix --json") {
		t.Fatalf("expected shim readiness preflight: %+v", packet.RequiredPreflight)
	}
	if !containsString(packet.RequiredPreflight, "areaflow project shim-authorization areamatrix --json") {
		t.Fatalf("expected shim authorization preflight: %+v", packet.RequiredPreflight)
	}
	if !containsString(packet.RequiredPreflight, "areaflow project status-projections areamatrix --json") {
		t.Fatalf("expected status projection metadata preflight: %+v", packet.RequiredPreflight)
	}
	if !containsString(packet.RequiredPreflight, "areaflow project status-projection-authorization areamatrix --json") {
		t.Fatalf("expected status projection authorization preflight: %+v", packet.RequiredPreflight)
	}
	if !containsString(packet.RequiredPreflight, "areaflow project status-projection-apply-packet areamatrix --json") {
		t.Fatalf("expected status projection apply packet preflight: %+v", packet.RequiredPreflight)
	}
	if !containsString(packet.RequiredPreflight, "areaflow project status-projection-apply-gate areamatrix --json") {
		t.Fatalf("expected status projection apply gate preflight: %+v", packet.RequiredPreflight)
	}
	if !containsString(packet.RequiredPreflight, "python3 scripts/validate-status-projection-schema.py schemas/status-projection.schema.json /Users/as/Ai-Project/project/AreaMatrix/.areaflow/status.json") {
		t.Fatalf("expected executable status projection schema preflight: %+v", packet.RequiredPreflight)
	}
	if !containsString(packet.RequiredPreflight, "verify .areaflow/status.json stable_fallback_projection_v1 includes schema_version/project_id/active_versions/rough_progress/source_snapshot_hash/compatibility.blocked_commands and excludes summary/generated_at/source/source_hash") {
		t.Fatalf("expected stable status projection schema preflight: %+v", packet.RequiredPreflight)
	}
	if !containsString(packet.RequiredPreflight, "git -C /Users/as/Ai-Project/project/AreaMatrix status --short -- workflow/README.md .areaflow/status.json scripts/task_loop/console.py scripts/dev_tools/cli.py scripts/task_loop/runner.py scripts/areaflow_shim.py workflow/versions workflow/versions/v1-mvp/execution/_shared/progress.json") {
		t.Fatalf("expected AreaMatrix protected path preflight: %+v", packet.RequiredPreflight)
	}
	if !shimContainsFragment(packet.PostEditVerification, "./task-loop run") ||
		!shimContainsFragment(packet.PostEditVerification, "blocked") {
		t.Fatalf("expected task-loop run post-edit blocked check: %+v", packet.PostEditVerification)
	}
	if !containsString(packet.PostEditVerification, "python3 /Users/as/Ai-Project/project/AreaFlow/scripts/validate-status-projection-schema.py /Users/as/Ai-Project/project/AreaFlow/schemas/status-projection.schema.json .areaflow/status.json") {
		t.Fatalf("expected status projection schema post-edit verification: %+v", packet.PostEditVerification)
	}
	if !containsString(packet.PostEditVerification, "git status --short -- workflow/README.md .areaflow/status.json scripts/task_loop/console.py scripts/dev_tools/cli.py scripts/task_loop/runner.py scripts/areaflow_shim.py workflow/versions workflow/versions/v1-mvp/execution/_shared/progress.json") {
		t.Fatalf("expected AreaMatrix protected path post-edit check: %+v", packet.PostEditVerification)
	}
	if !containsString(packet.RollbackScope, "restore the captured preimage bytes for .areaflow/status.json if projection apply was part of the approved change") {
		t.Fatalf("expected status projection rollback scope: %+v", packet.RollbackScope)
	}
	if packet.SafetyFacts["project_write_attempted"] || packet.SafetyFacts["execution_write_attempted"] ||
		packet.SafetyFacts["status_projection_write_open"] || packet.SafetyFacts["engine_call_attempted"] {
		t.Fatalf("authorization packet should be read-only: %+v", packet.SafetyFacts)
	}
	if packet.NextRequiredApproval == "" {
		t.Fatalf("expected next required approval")
	}
}

func TestShimAuthorizationPacketFiltersStatusProjectionWrites(t *testing.T) {
	readiness := ShimReadiness{
		Preview: ShimPreview{
			PlannedFiles: []ShimFilePlan{
				{Path: "workflow/README.md", Action: "modify"},
				{Path: ".areaflow/status.json", Action: "modify"},
				{Path: "workflow/status-projection-copy.json", Action: "controlled_projection_apply"},
			},
		},
	}

	packet := ShimAuthorizationPacketFromReadiness(readiness)

	assertShimAuthorizationFile(t, packet, "workflow/README.md")
	assertShimAuthorizationFileAbsent(t, packet, ".areaflow/status.json")
	assertShimAuthorizationFileAbsent(t, packet, "workflow/status-projection-copy.json")
}

func TestProjectCutoverReadinessFromPartsBlocksWithoutGates(t *testing.T) {
	summary := ProjectSummary{
		Project: Record{Key: "areamatrix"},
		Inventory: ImportInventory{
			MirrorExports: 1,
		},
		HasImport:           true,
		HasPreviousImport:   true,
		HasLatestDoctor:     true,
		DoctorStatus:        "pass",
		DriftStatus:         "pass",
		ConfigDriftStatus:   "pass",
		StageCoverageStatus: "pass",
		NativeDoctorStatus:  "pass",
		LatestEventCount:    1,
	}
	readiness := ProjectReadinessFromSummary(summary)
	diff := ProjectImportDiffFromSummary(summary)
	bundle := ProjectVerificationBundleFromParts(summary, readiness, diff, []EventRecord{{Type: "project.doctor.completed"}})
	compat := CompatibilityContractFromSummary(summary, map[string]CommandPermission{})

	cutover := ProjectCutoverReadinessFromParts(bundle, compat, WorkflowVersion{DisplayLabel: "v2", ImportMode: "authored"}, nil)

	if cutover.Status != "blocked" {
		t.Fatalf("cutover status = %q, want blocked", cutover.Status)
	}
	assertReadinessItem(t, ProjectReadiness{Items: cutover.Items}, "approval_gate", "blocked")
	assertReadinessItem(t, ProjectReadiness{Items: cutover.Items}, "live_mapping_gate", "blocked")
}

func assertShimReadinessItem(t *testing.T, readiness ShimReadiness, key string, status string) {
	t.Helper()
	item := shimReadinessItem(t, readiness, key)
	if item.Status != status {
		t.Fatalf("shim readiness item %s status = %q, want %q", key, item.Status, status)
	}
}

func shimReadinessItem(t *testing.T, readiness ShimReadiness, key string) ShimReadinessItem {
	t.Helper()
	for _, item := range readiness.Items {
		if item.Key == key {
			return item
		}
	}
	t.Fatalf("shim readiness item %s not found: %+v", key, readiness.Items)
	return ShimReadinessItem{}
}

func assertShimAuthorizationFile(t *testing.T, packet ShimAuthorizationPacket, path string) {
	t.Helper()
	for _, file := range packet.AllowedFiles {
		if file.Path == path {
			return
		}
	}
	t.Fatalf("allowed shim file %s not found: %+v", path, packet.AllowedFiles)
}

func assertShimAuthorizationFileAbsent(t *testing.T, packet ShimAuthorizationPacket, path string) {
	t.Helper()
	for _, file := range packet.AllowedFiles {
		if file.Path == path {
			t.Fatalf("file %s must not be authorized for Package B writes: %+v", path, packet.AllowedFiles)
		}
	}
}

func TestProjectCutoverReadinessFromPartsPassesWithRequiredGates(t *testing.T) {
	summary := ProjectSummary{
		Project: Record{Key: "areamatrix"},
		Inventory: ImportInventory{
			MirrorExports: 1,
		},
		Import:              Snapshot{SourceHash: "hash-a"},
		PreviousImport:      Snapshot{SourceHash: "hash-a"},
		HasImport:           true,
		HasPreviousImport:   true,
		HasLatestDoctor:     true,
		DoctorStatus:        "pass",
		DriftStatus:         "pass",
		ConfigDriftStatus:   "pass",
		StageCoverageStatus: "pass",
		NativeDoctorStatus:  "pass",
		LatestEventCount:    1,
	}
	readiness := ProjectReadinessFromSummary(summary)
	diff := ProjectImportDiffFromSummary(summary)
	bundle := ProjectVerificationBundleFromParts(summary, readiness, diff, []EventRecord{{Type: "project.doctor.completed"}})
	compat := CompatibilityContractFromSummary(summary, map[string]CommandPermission{})

	cutover := ProjectCutoverReadinessFromParts(bundle, compat, WorkflowVersion{DisplayLabel: "v2", ImportMode: "authored"}, []GateResult{
		{ID: 1, GateName: "approval_gate", Status: "pass"},
		{ID: 2, GateName: "live_mapping_gate", Status: "pass"},
	})

	if cutover.Status != "pass" {
		t.Fatalf("cutover status = %q, want pass: %+v", cutover.Status, cutover.PhaseGate)
	}
	if len(cutover.PhaseGate.Blockers) != 0 {
		t.Fatalf("unexpected blockers: %+v", cutover.PhaseGate.Blockers)
	}
}

func TestProjectImportDiffFromSummaryUnchanged(t *testing.T) {
	latest := Snapshot{
		SourceHash: "hash-a",
		CreatedAt:  time.Date(2026, 6, 29, 3, 0, 0, 0, time.UTC),
		Summary: map[string]any{
			"version_count":  float64(2),
			"residual_count": float64(10),
			"tasks": map[string]any{
				"active":         float64(0),
				"done":           float64(1),
				"backlog_open":   float64(0),
				"backlog_closed": float64(5),
			},
			"v1_execution": map[string]any{
				"done":  float64(637),
				"total": float64(637),
			},
		},
	}
	diff := ProjectImportDiffFromSummary(ProjectSummary{
		Project:           Record{Key: "areamatrix"},
		Import:            latest,
		HasImport:         true,
		PreviousImport:    latest,
		HasPreviousImport: true,
	})

	if diff.Status != "unchanged" {
		t.Fatalf("diff status = %q, want unchanged", diff.Status)
	}
	if diff.SourceChanged {
		t.Fatal("source should be unchanged")
	}
	if len(diff.Changes) != 9 {
		t.Fatalf("change count = %d, want 9", len(diff.Changes))
	}
	assertDiffChange(t, diff, "source_hash", "unchanged")
	assertDiffChange(t, diff, "v1_execution.total", "unchanged")
}

func assertCompatibilityCommand(t *testing.T, contract CompatibilityContract, command string, mode string, status string) {
	t.Helper()
	for _, item := range contract.Commands {
		if item.Command == command {
			if item.Mode != mode || item.Status != status {
				t.Fatalf("%s = mode %q status %q, want mode %q status %q", command, item.Mode, item.Status, mode, status)
			}
			return
		}
	}
	t.Fatalf("compatibility command %s not found: %+v", command, contract.Commands)
}

func TestProjectImportDiffFromSummaryChanged(t *testing.T) {
	previous := Snapshot{
		SourceHash: "hash-a",
		Summary: map[string]any{
			"version_count":  float64(1),
			"residual_count": float64(10),
		},
	}
	latest := Snapshot{
		SourceHash: "hash-b",
		Summary: map[string]any{
			"version_count":  float64(2),
			"residual_count": float64(10),
		},
	}
	diff := ProjectImportDiffFromSummary(ProjectSummary{
		Project:           Record{Key: "areamatrix"},
		Import:            latest,
		HasImport:         true,
		PreviousImport:    previous,
		HasPreviousImport: true,
	})

	if diff.Status != "changed" {
		t.Fatalf("diff status = %q, want changed", diff.Status)
	}
	if !diff.SourceChanged {
		t.Fatal("source should be changed")
	}
	assertDiffChange(t, diff, "source_hash", "changed")
	assertDiffChange(t, diff, "version_count", "changed")
	assertDiffChange(t, diff, "residual_count", "unchanged")
}

func TestProjectImportDiffFromSummaryNoPrevious(t *testing.T) {
	diff := ProjectImportDiffFromSummary(ProjectSummary{
		Project:   Record{Key: "areamatrix"},
		Import:    Snapshot{SourceHash: "hash-a"},
		HasImport: true,
	})

	if diff.Status != "no_previous" {
		t.Fatalf("diff status = %q, want no_previous", diff.Status)
	}
	if len(diff.Changes) != 0 {
		t.Fatalf("changes = %+v, want none", diff.Changes)
	}
}

func TestProjectVerificationBundleFromParts(t *testing.T) {
	summary := ProjectSummary{Project: Record{Key: "areamatrix"}}
	readiness := ProjectReadiness{
		Project: summary.Project,
		Status:  "pass",
		Summary: summary,
	}
	diff := ProjectImportDiff{
		Project: summary.Project,
		Status:  "changed",
	}
	events := []EventRecord{{Type: "project.import.completed"}}

	bundle := ProjectVerificationBundleFromParts(summary, readiness, diff, events)

	if bundle.Project.Key != "areamatrix" {
		t.Fatalf("project key = %q, want areamatrix", bundle.Project.Key)
	}
	if bundle.Status != "warn" {
		t.Fatalf("bundle status = %q, want warn for changed import diff", bundle.Status)
	}
	if len(bundle.Events) != 1 {
		t.Fatalf("events = %d, want 1", len(bundle.Events))
	}
}

func TestEvaluateV02PhaseGateAcceptsNativeDoctorWarn(t *testing.T) {
	summary := ProjectSummary{
		Project: Record{Key: "areamatrix"},
		Inventory: ImportInventory{
			MirrorExports: 1,
		},
		HasImport:           true,
		HasPreviousImport:   true,
		DoctorStatus:        "warn",
		DriftStatus:         "pass",
		ConfigDriftStatus:   "warn",
		StageCoverageStatus: "pass",
		NativeDoctorStatus:  "warn",
	}
	gate := EvaluateV02PhaseGate(
		summary,
		ProjectReadiness{Project: summary.Project, Status: "warn", Summary: summary},
		ProjectImportDiff{Project: summary.Project, Status: "unchanged"},
		[]EventRecord{{Type: "project.doctor.completed"}},
	)

	if gate.Status != "pass" {
		t.Fatalf("gate status = %q, want pass: %+v", gate.Status, gate)
	}
	if len(gate.AcceptedWarnings) != 2 {
		t.Fatalf("accepted warnings = %+v, want native doctor and config drift warnings", gate.AcceptedWarnings)
	}
	if len(gate.Blockers) != 0 {
		t.Fatalf("blockers = %+v, want none", gate.Blockers)
	}
}

func TestEvaluateV02PhaseGateBlocksChangedDiff(t *testing.T) {
	summary := ProjectSummary{
		Project: Record{Key: "areamatrix"},
		Inventory: ImportInventory{
			MirrorExports: 1,
		},
		HasImport:           true,
		HasPreviousImport:   true,
		DoctorStatus:        "pass",
		DriftStatus:         "pass",
		ConfigDriftStatus:   "pass",
		StageCoverageStatus: "pass",
		NativeDoctorStatus:  "pass",
	}
	gate := EvaluateV02PhaseGate(
		summary,
		ProjectReadiness{Project: summary.Project, Status: "pass", Summary: summary},
		ProjectImportDiff{Project: summary.Project, Status: "changed"},
		[]EventRecord{{Type: "project.doctor.completed"}},
	)

	if gate.Status != "blocked" {
		t.Fatalf("gate status = %q, want blocked", gate.Status)
	}
	if len(gate.Blockers) == 0 {
		t.Fatal("expected blockers for changed diff")
	}
}

func TestProjectVerificationBundleFromPartsNoImport(t *testing.T) {
	summary := ProjectSummary{Project: Record{Key: "areamatrix"}}
	bundle := ProjectVerificationBundleFromParts(
		summary,
		ProjectReadiness{Project: summary.Project, Status: "warn", Summary: summary},
		ProjectImportDiff{Project: summary.Project, Status: "no_import"},
		nil,
	)

	if bundle.Status != "fail" {
		t.Fatalf("bundle status = %q, want fail for missing import", bundle.Status)
	}
	if bundle.PhaseGate.Status != "blocked" {
		t.Fatalf("phase gate status = %q, want blocked", bundle.PhaseGate.Status)
	}
}

func assertReadinessItem(t *testing.T, readiness ProjectReadiness, key string, status string) {
	t.Helper()
	for _, item := range readiness.Items {
		if item.Key == key {
			if item.Status != status {
				t.Fatalf("readiness item %s status = %q, want %q", key, item.Status, status)
			}
			return
		}
	}
	t.Fatalf("readiness item %s not found in %+v", key, readiness.Items)
}

func assertDiffChange(t *testing.T, diff ProjectImportDiff, key string, status string) {
	t.Helper()
	for _, change := range diff.Changes {
		if change.Key == key {
			if change.Status != status {
				t.Fatalf("diff change %s status = %q, want %q", key, change.Status, status)
			}
			return
		}
	}
	t.Fatalf("diff change %s not found in %+v", key, diff.Changes)
}
