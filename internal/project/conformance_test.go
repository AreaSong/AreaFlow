package project

import (
	"path/filepath"
	"testing"
	"time"

	areamatrixadapter "github.com/areasong/areaflow/internal/adapter/areamatrix"
	workflowprofile "github.com/areasong/areaflow/internal/workflow"
)

func TestBuildConformanceReportPassesAreaMatrixBaseline(t *testing.T) {
	created := time.Date(2026, 6, 30, 12, 0, 0, 0, time.UTC)
	loaded, err := workflowprofile.LoadBuiltInProfile(filepath.Join("..", ".."), "areamatrix")
	if err != nil {
		t.Fatalf("load profile: %v", err)
	}
	snapshot := &areamatrixadapter.Snapshot{
		Versions:         []areamatrixadapter.Version{{Label: "v1-mvp"}},
		Residuals:        []areamatrixadapter.Residual{{Key: "v1-release"}},
		Artifacts:        []areamatrixadapter.Artifact{{Type: "progress"}},
		StatusSourceHash: "status-hash",
		TaskSummary: areamatrixadapter.TaskSummary{
			V1ExecutionTotal: 637,
			V1ExecutionDone:  637,
		},
	}

	report := BuildConformanceReport(
		Record{ID: 1, Key: "areamatrix", Adapter: "areamatrix", WorkflowProfile: "areamatrix", RootPath: "/tmp/areamatrix"},
		loaded,
		snapshot,
		testConformanceProjectConfig(),
		true,
		ConformanceOptions{GeneratedAt: created},
	)

	if report.Status != "pass" || report.Mode != "read_only_adapter_profile_conformance" {
		t.Fatalf("unexpected conformance report: %+v", report)
	}
	if report.ProfileID != "areamatrix" || report.Adapter != "areamatrix" || report.StageCount != 16 || report.GateCount != 17 {
		t.Fatalf("unexpected profile summary: %+v", report)
	}
	assertConformanceCheck(t, report, "project_adapter_profile", "pass")
	assertConformanceCheck(t, report, "profile_load", "pass")
	assertConformanceCheck(t, report, "profile_validate", "pass")
	assertConformanceCheck(t, report, "profile_item_state_contract", "pass")
	assertConformanceCheck(t, report, "profile_stage_contract", "pass")
	assertConformanceCheck(t, report, "profile_gate_contract", "pass")
	assertConformanceCheck(t, report, "profile_transition_contract", "pass")
	assertConformanceCheck(t, report, "profile_hard_rule_contract", "pass")
	assertConformanceCheck(t, report, "profile_permission_policy_contract", "pass")
	assertConformanceCheck(t, report, "profile_artifact_policy_contract", "pass")
	assertConformanceCheck(t, report, "profile_cutover_policy_contract", "pass")
	assertConformanceCheck(t, report, "adapter_snapshot", "pass")
	assertConformanceCheck(t, report, "adapter_profile_boundary", "pass")
	assertConformanceCheck(t, report, "plugin_seed_catalog_contract", "pass")
	assertConformanceCheck(t, report, "plugin_manifest_draft_contract", "pass")
	assertConformanceCheck(t, report, "plugin_no_execution_boundary", "pass")
	assertConformanceCheck(t, report, "project_config_policy", "pass")
	if !report.GeneratedAt.Equal(created) {
		t.Fatalf("generated_at = %s, want %s", report.GeneratedAt, created)
	}
}

func TestBuildConformanceReportFailsBindingMismatch(t *testing.T) {
	loaded, err := workflowprofile.LoadBuiltInProfile(filepath.Join("..", ".."), "areamatrix")
	if err != nil {
		t.Fatalf("load profile: %v", err)
	}
	report := BuildConformanceReport(
		Record{ID: 1, Key: "other", Adapter: "git-repo", WorkflowProfile: "areamatrix"},
		loaded,
		nil,
		ProjectConfigRecord{},
		false,
		ConformanceOptions{},
	)

	if report.Status != "fail" {
		t.Fatalf("expected fail status: %+v", report)
	}
	assertConformanceCheck(t, report, "project_adapter_profile", "fail")
}

func TestBuildConformanceReportFailsMissingAreaMatrixSnapshot(t *testing.T) {
	loaded, err := workflowprofile.LoadBuiltInProfile(filepath.Join("..", ".."), "areamatrix")
	if err != nil {
		t.Fatalf("load profile: %v", err)
	}
	report := BuildConformanceReport(
		Record{ID: 1, Key: "areamatrix", Adapter: "areamatrix", WorkflowProfile: "areamatrix"},
		loaded,
		nil,
		testConformanceProjectConfig(),
		true,
		ConformanceOptions{},
	)

	if report.Status != "fail" {
		t.Fatalf("expected fail status: %+v", report)
	}
	assertConformanceCheck(t, report, "adapter_snapshot", "fail")
}

func TestBuildConformanceReportFailsSnapshotLoadWithoutQueryError(t *testing.T) {
	loaded, err := workflowprofile.LoadBuiltInProfile(filepath.Join("..", ".."), "areamatrix")
	if err != nil {
		t.Fatalf("load profile: %v", err)
	}
	report := BuildConformanceReport(
		Record{ID: 1, Key: "areamatrix", Adapter: "areamatrix", WorkflowProfile: "areamatrix", RootPath: "/tmp/missing-areamatrix"},
		loaded,
		nil,
		testConformanceProjectConfig(),
		true,
		ConformanceOptions{SnapshotLoadError: "read workflow/residuals/residuals.yaml: file does not exist"},
	)

	if report.Status != "fail" {
		t.Fatalf("expected fail status: %+v", report)
	}
	check := assertConformanceCheck(t, report, "adapter_snapshot", "fail")
	assertConformanceFailure(t, check, "snapshot_load_failed")
	if check.Metadata["load_error"] == "" {
		t.Fatalf("load error metadata missing: %+v", check.Metadata)
	}
}

func TestBuildConformanceReportFailsMissingAreaMatrixConfig(t *testing.T) {
	loaded, err := workflowprofile.LoadBuiltInProfile(filepath.Join("..", ".."), "areamatrix")
	if err != nil {
		t.Fatalf("load profile: %v", err)
	}
	report := BuildConformanceReport(
		Record{ID: 1, Key: "areamatrix", Adapter: "areamatrix", WorkflowProfile: "areamatrix", RootPath: "/tmp/areamatrix"},
		loaded,
		testConformanceSnapshot(),
		ProjectConfigRecord{},
		false,
		ConformanceOptions{},
	)

	if report.Status != "fail" {
		t.Fatalf("expected fail status: %+v", report)
	}
	assertConformanceCheck(t, report, "adapter_snapshot", "pass")
	assertConformanceCheck(t, report, "project_config_policy", "fail")
}

func TestBuildConformanceReportFailsUnsafeProjectConfigPolicy(t *testing.T) {
	loaded, err := workflowprofile.LoadBuiltInProfile(filepath.Join("..", ".."), "areamatrix")
	if err != nil {
		t.Fatalf("load profile: %v", err)
	}
	config := testConformanceProjectConfig()
	capabilities := config.Permissions["capabilities"].(map[string]any)
	capabilities["run_commands"] = true
	commands := config.Metadata["commands"].(map[string]any)
	commands["allowed"] = []any{"./task-loop run"}

	report := BuildConformanceReport(
		Record{ID: 1, Key: "areamatrix", Adapter: "areamatrix", WorkflowProfile: "areamatrix", RootPath: "/tmp/areamatrix"},
		loaded,
		testConformanceSnapshot(),
		config,
		true,
		ConformanceOptions{},
	)

	if report.Status != "fail" {
		t.Fatalf("expected fail status: %+v", report)
	}
	check := assertConformanceCheck(t, report, "project_config_policy", "fail")
	assertConformanceFailure(t, check, "run_commands_unexpectedly_enabled")
	assertConformanceFailure(t, check, "task_loop_run_allowed")
}

func TestBuildConformanceReportFailsTransitionContractDrift(t *testing.T) {
	loaded, err := workflowprofile.LoadBuiltInProfile(filepath.Join("..", ".."), "areamatrix")
	if err != nil {
		t.Fatalf("load profile: %v", err)
	}
	loaded.Profile.Transitions[4].RequiredGate = ""

	report := BuildConformanceReport(
		Record{ID: 1, Key: "areamatrix", Adapter: "areamatrix", WorkflowProfile: "areamatrix", RootPath: "/tmp/areamatrix"},
		loaded,
		testConformanceSnapshot(),
		testConformanceProjectConfig(),
		true,
		ConformanceOptions{},
	)

	if report.Status != "fail" {
		t.Fatalf("expected fail status: %+v", report)
	}
	check := assertConformanceCheck(t, report, "profile_transition_contract", "fail")
	assertConformanceFailure(t, check, "transition_gate:4:discussion_gate")
}

func TestBuildConformanceReportFailsHardRuleContractDrift(t *testing.T) {
	loaded, err := workflowprofile.LoadBuiltInProfile(filepath.Join("..", ".."), "areamatrix")
	if err != nil {
		t.Fatalf("load profile: %v", err)
	}
	loaded.Profile.HardRules = loaded.Profile.HardRules[:len(loaded.Profile.HardRules)-1]

	report := BuildConformanceReport(
		Record{ID: 1, Key: "areamatrix", Adapter: "areamatrix", WorkflowProfile: "areamatrix", RootPath: "/tmp/areamatrix"},
		loaded,
		testConformanceSnapshot(),
		testConformanceProjectConfig(),
		true,
		ConformanceOptions{},
	)

	if report.Status != "fail" {
		t.Fatalf("expected fail status: %+v", report)
	}
	check := assertConformanceCheck(t, report, "profile_hard_rule_contract", "fail")
	assertConformanceFailure(t, check, "closeout_gate must prove evidence before done.")
}

func TestBuildConformanceReportFailsItemStateContractDrift(t *testing.T) {
	loaded, err := workflowprofile.LoadBuiltInProfile(filepath.Join("..", ".."), "areamatrix")
	if err != nil {
		t.Fatalf("load profile: %v", err)
	}
	loaded.Profile.ItemStates[5] = "executing"

	report := BuildConformanceReport(
		Record{ID: 1, Key: "areamatrix", Adapter: "areamatrix", WorkflowProfile: "areamatrix", RootPath: "/tmp/areamatrix"},
		loaded,
		testConformanceSnapshot(),
		testConformanceProjectConfig(),
		true,
		ConformanceOptions{},
	)

	if report.Status != "fail" {
		t.Fatalf("expected fail status: %+v", report)
	}
	check := assertConformanceCheck(t, report, "profile_item_state_contract", "fail")
	assertConformanceFailure(t, check, "item_state_order:5:running")
}

func TestBuildConformanceReportFailsArtifactPolicyContractDrift(t *testing.T) {
	loaded, err := workflowprofile.LoadBuiltInProfile(filepath.Join("..", ".."), "areamatrix")
	if err != nil {
		t.Fatalf("load profile: %v", err)
	}
	loaded.Profile.ArtifactPolicy.ContentSource = "postgres"

	report := BuildConformanceReport(
		Record{ID: 1, Key: "areamatrix", Adapter: "areamatrix", WorkflowProfile: "areamatrix", RootPath: "/tmp/areamatrix"},
		loaded,
		testConformanceSnapshot(),
		testConformanceProjectConfig(),
		true,
		ConformanceOptions{},
	)

	if report.Status != "fail" {
		t.Fatalf("expected fail status: %+v", report)
	}
	check := assertConformanceCheck(t, report, "profile_artifact_policy_contract", "fail")
	assertConformanceFailure(t, check, "content_source_mismatch")
}

func TestBuildConformanceReportFailsPermissionPolicyContractDrift(t *testing.T) {
	loaded, err := workflowprofile.LoadBuiltInProfile(filepath.Join("..", ".."), "areamatrix")
	if err != nil {
		t.Fatalf("load profile: %v", err)
	}
	loaded.Profile.Permissions.WriteRequires[2], loaded.Profile.Permissions.WriteRequires[3] = loaded.Profile.Permissions.WriteRequires[3], loaded.Profile.Permissions.WriteRequires[2]

	report := BuildConformanceReport(
		Record{ID: 1, Key: "areamatrix", Adapter: "areamatrix", WorkflowProfile: "areamatrix", RootPath: "/tmp/areamatrix"},
		loaded,
		testConformanceSnapshot(),
		testConformanceProjectConfig(),
		true,
		ConformanceOptions{},
	)

	if report.Status != "fail" {
		t.Fatalf("expected fail status: %+v", report)
	}
	check := assertConformanceCheck(t, report, "profile_permission_policy_contract", "fail")
	assertConformanceFailure(t, check, "write_requires_order:2:gate_result")
	assertConformanceFailure(t, check, "write_requires_order:3:approval_record")
}

func TestBuildConformanceReportFailsCutoverPolicyContractDrift(t *testing.T) {
	loaded, err := workflowprofile.LoadBuiltInProfile(filepath.Join("..", ".."), "areamatrix")
	if err != nil {
		t.Fatalf("load profile: %v", err)
	}
	loaded.Profile.Cutover.ExecutionCutoverPhase = "v0.4"

	report := BuildConformanceReport(
		Record{ID: 1, Key: "areamatrix", Adapter: "areamatrix", WorkflowProfile: "areamatrix", RootPath: "/tmp/areamatrix"},
		loaded,
		testConformanceSnapshot(),
		testConformanceProjectConfig(),
		true,
		ConformanceOptions{},
	)

	if report.Status != "fail" {
		t.Fatalf("expected fail status: %+v", report)
	}
	check := assertConformanceCheck(t, report, "profile_cutover_policy_contract", "fail")
	assertConformanceFailure(t, check, "execution_cutover_phase_mismatch")
}

func TestPluginSeedCatalogContractFailsUnsafeState(t *testing.T) {
	boundary := defaultPluginMarketplaceBoundary()
	boundary.SeedCatalog[0].RegistryState = "enabled"
	boundary.SeedCatalog[0].ExecuteEnabled = true

	check := checkPluginSeedCatalogContract(boundary)
	if check.Status != "fail" {
		t.Fatalf("expected fail status: %+v", check)
	}
	assertConformanceFailure(t, check, "seed_state_not_allowed:areamatrix-adapter:enabled")
	assertConformanceFailure(t, check, "seed_execute_enabled:areamatrix-adapter")
}

func TestPluginManifestDraftContractFailsMissingField(t *testing.T) {
	boundary := defaultPluginMarketplaceBoundary()
	boundary.ManifestDraftRequiredFields = boundary.ManifestDraftRequiredFields[:len(boundary.ManifestDraftRequiredFields)-1]

	check := checkPluginManifestDraftContract(boundary)
	if check.Status != "fail" {
		t.Fatalf("expected fail status: %+v", check)
	}
	assertConformanceFailure(t, check, "audit_actions")
}

func TestPluginNoExecutionBoundaryFailsOpenedCapability(t *testing.T) {
	boundary := defaultPluginMarketplaceBoundary()
	boundary.NoExecutionFacts["network_access_enabled"] = true
	boundary.UnknownExecutionRung = "v1.0"

	check := checkPluginNoExecutionBoundary(boundary)
	if check.Status != "fail" {
		t.Fatalf("expected fail status: %+v", check)
	}
	assertConformanceFailure(t, check, "network_access_enabled")
	assertConformanceFailure(t, check, "unknown_execution_rung_mismatch")
}

func testConformanceSnapshot() *areamatrixadapter.Snapshot {
	return &areamatrixadapter.Snapshot{
		Versions:         []areamatrixadapter.Version{{Label: "v1-mvp"}},
		Residuals:        []areamatrixadapter.Residual{{Key: "v1-release"}},
		Artifacts:        []areamatrixadapter.Artifact{{Type: "progress"}},
		StatusSourceHash: "status-hash",
		TaskSummary: areamatrixadapter.TaskSummary{
			V1ExecutionTotal: 637,
			V1ExecutionDone:  637,
		},
	}
}

func testConformanceProjectConfig() ProjectConfigRecord {
	loadedAt := time.Date(2026, 6, 30, 12, 0, 0, 0, time.UTC)
	return ProjectConfigRecord{
		ID:              1,
		ProjectID:       1,
		ProtocolVersion: 1,
		ConfigPath:      "examples/areamatrix/areaflow.yaml",
		ConfigHash:      "hash-config",
		Ownership:       map[string]any{"mode": "import"},
		Permissions: map[string]any{
			"capabilities": map[string]any{
				"read_project":    true,
				"write_status":    true,
				"write_artifacts": true,
				"write_workflow":  false,
				"write_generated": false,
				"write_code":      false,
				"run_commands":    false,
				"manage_workers":  false,
				"manage_git":      false,
				"network":         false,
				"use_secrets":     false,
				"execute_agents":  false,
			},
			"write_paths": []any{".areaflow/status.json"},
			"forbidden_paths": []any{
				"workflow/versions/*/execution/**",
				"workflow/versions/*/execution/_shared/progress.json",
				".areamatrix/**",
				"**/*.sqlite",
				"**/*.db",
			},
		},
		Scheduling: map[string]any{
			"required_capabilities": []any{"read_project", "write_artifacts"},
			"max_parallel_tasks":    1,
			"engine_profile":        "codex-cli",
		},
		Engines: map[string]any{
			"profiles": []any{
				map[string]any{"id": "codex-cli", "provider": "codex-cli", "secret_ref": "none", "enabled": false},
				map[string]any{"id": "openai-main", "provider": "openai", "secret_ref": "openai/default", "enabled": false},
			},
		},
		StatusExport: map[string]any{
			"enabled": true,
			"path":    ".areaflow/status.json",
			"human_summary": map[string]any{
				"enabled":      false,
				"path":         "workflow/README.md",
				"block_marker": "AREAFLOW_STATUS",
			},
		},
		Migration: map[string]any{
			"strategy": "import_mirror_shadow_cutover_archive",
			"phase":    "import",
		},
		Metadata: map[string]any{
			"project": map[string]any{
				"adapter":          "areamatrix",
				"workflow_profile": "areamatrix",
			},
			"commands": map[string]any{
				"allowed": []any{"./dev tasks status", "./dev workflow doctor"},
				"forbidden": []any{
					"./task-loop run",
					"git reset --hard",
					"git checkout --",
					"rm -rf",
				},
			},
		},
		Active:   true,
		LoadedAt: loadedAt,
	}
}

func assertConformanceCheck(t *testing.T, report ConformanceReport, key string, status string) ConformanceCheck {
	t.Helper()
	for _, check := range report.Checks {
		if check.Key == key {
			if check.Status != status {
				t.Fatalf("check %s status = %q, want %q: %+v", key, check.Status, status, check)
			}
			return check
		}
	}
	t.Fatalf("check %s not found: %+v", key, report.Checks)
	return ConformanceCheck{}
}

func assertConformanceFailure(t *testing.T, check ConformanceCheck, want string) {
	t.Helper()
	failures, ok := check.Metadata["failures"].([]string)
	if !ok {
		t.Fatalf("check %s failures metadata missing: %+v", check.Key, check.Metadata)
	}
	for _, failure := range failures {
		if failure == want {
			return
		}
	}
	t.Fatalf("check %s failure %q not found: %+v", check.Key, want, failures)
}
