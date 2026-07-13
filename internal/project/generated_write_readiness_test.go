package project

import (
	"testing"
	"time"
)

func TestBuildGeneratedWriteReadinessBlocksAreaMatrixBaseline(t *testing.T) {
	generated := time.Date(2026, 7, 2, 10, 0, 0, 0, time.UTC)
	config := testGeneratedWriteProjectConfig(generated)
	readiness := BuildGeneratedWriteReadiness(
		Record{ID: 1, Key: "areamatrix", Kind: "product-repo"},
		config,
		true,
		testGeneratedWritePermissionRows(false),
		GeneratedWriteReadinessOptions{GeneratedAt: generated},
	)

	if readiness.Status != "blocked" || readiness.Mode != "read_only_generated_write_readiness" {
		t.Fatalf("unexpected generated write readiness: %+v", readiness)
	}
	if readiness.ReadyForReview {
		t.Fatalf("baseline AreaMatrix should not be ready for generated-only write review: %+v", readiness)
	}
	if readiness.ApplyOpen || readiness.RealAreaMatrixWriteOpened {
		t.Fatalf("real AreaMatrix generated apply must remain closed: %+v", readiness)
	}
	if !containsString(readiness.ReviewBlockers, "required_capabilities: project is missing required generated-only planning capabilities") {
		t.Fatalf("expected missing capability review blocker: %+v", readiness.ReviewBlockers)
	}
	assertGeneratedWriteItem(t, readiness, "required_capabilities", "blocked")
	assertGeneratedWriteItem(t, readiness, "generated_prefix_path_policy", "blocked")
	assertGeneratedWriteItem(t, readiness, "real_areamatrix_apply_open", "blocked")
	applyItem := generatedWriteItemByKey(readiness.Items, "real_areamatrix_apply_open")
	if applyItem.Metadata["ready_for_review"] != false {
		t.Fatalf("apply item metadata should reflect final ready_for_review=false: %+v", applyItem.Metadata)
	}
	if readiness.ProjectReadAttempted || readiness.ProjectWriteAttempted || readiness.ExecutionWriteAttempted ||
		readiness.AreaFlowArtifactWritten || readiness.AreaFlowExecutionStateWritten || readiness.EngineCallAttempted ||
		readiness.CommandsRun || readiness.SecretsResolved || readiness.NetworkUsed || readiness.TaskClaimed ||
		readiness.WorkerStarted || readiness.LeaseCreated || readiness.AttemptCreated || readiness.ArtifactCreated {
		t.Fatalf("generated write readiness should be read-only: %+v", readiness)
	}
}

func TestBuildGeneratedWriteReadinessReadyForReviewButApplyClosed(t *testing.T) {
	generated := time.Date(2026, 7, 2, 10, 5, 0, 0, time.UTC)
	config := testGeneratedWriteProjectConfig(generated)
	capabilities := config.Permissions["capabilities"].(map[string]any)
	capabilities["write_generated"] = true
	config.Permissions["write_paths"] = []any{
		".areaflow/status.json",
		".areaflow/generated/**",
		".areamatrix/generated/**",
	}
	config.Permissions["forbidden_paths"] = []any{
		"workflow/versions/*/execution/**",
		"workflow/versions/*/execution/_shared/progress.json",
		"**/*.sqlite",
		"**/*.db",
	}

	readiness := BuildGeneratedWriteReadiness(
		Record{ID: 1, Key: "areamatrix", Kind: "product-repo"},
		config,
		true,
		testGeneratedWritePermissionRows(true),
		GeneratedWriteReadinessOptions{GeneratedAt: generated},
	)

	if readiness.Status != "blocked" {
		t.Fatalf("readiness should remain blocked while apply is closed: %+v", readiness)
	}
	if !readiness.ReadyForReview {
		t.Fatalf("generated-only preconditions should be ready for review: %+v", readiness)
	}
	if readiness.ApplyOpen || readiness.RealAreaMatrixWriteOpened {
		t.Fatalf("generated-only apply should remain closed: %+v", readiness)
	}
	if len(readiness.ReviewBlockers) != 0 {
		t.Fatalf("did not expect review blockers once config is prepared: %+v", readiness.ReviewBlockers)
	}
	assertGeneratedWriteItem(t, readiness, "required_capabilities", "pass")
	assertGeneratedWriteItem(t, readiness, "generated_prefix_path_policy", "pass")
	assertGeneratedWriteItem(t, readiness, "high_risk_capabilities_closed", "pass")
	assertGeneratedWriteItem(t, readiness, "real_areamatrix_apply_open", "blocked")
	applyItem := generatedWriteItemByKey(readiness.Items, "real_areamatrix_apply_open")
	if applyItem.Metadata["ready_for_review"] != true {
		t.Fatalf("apply item metadata should reflect final ready_for_review=true: %+v", applyItem.Metadata)
	}
	if !containsString(readiness.AllowedGeneratedPrefixes, ".areaflow/generated/") || !containsString(readiness.RequiredWritePaths, ".areaflow/generated/**") {
		t.Fatalf("generated path policy missing expected paths: %+v", readiness)
	}
}

func TestBuildGeneratedWriteReadinessBlocksHighRiskCapabilities(t *testing.T) {
	config := testGeneratedWriteProjectConfig(time.Now().UTC())
	capabilities := config.Permissions["capabilities"].(map[string]any)
	capabilities["write_generated"] = true
	capabilities["run_commands"] = true
	config.Permissions["write_paths"] = []any{".areaflow/status.json", ".areaflow/generated/**", ".areamatrix/generated/**"}
	config.Permissions["forbidden_paths"] = []any{
		"workflow/versions/*/execution/**",
		"workflow/versions/*/execution/_shared/progress.json",
		"**/*.sqlite",
		"**/*.db",
	}
	permissions := testGeneratedWritePermissionRows(true)
	permissions = append(permissions, permissionRow{Effect: "allow", Capability: "run_commands", ResourceType: "capability", Pattern: "run_commands"})

	readiness := BuildGeneratedWriteReadiness(
		Record{ID: 1, Key: "areamatrix", Kind: "product-repo"},
		config,
		true,
		permissions,
		GeneratedWriteReadinessOptions{},
	)

	if readiness.ReadyForReview {
		t.Fatalf("run_commands should block generated-only dogfood review: %+v", readiness)
	}
	assertGeneratedWriteItem(t, readiness, "high_risk_capabilities_closed", "blocked")
}

func testGeneratedWriteProjectConfig(loadedAt time.Time) ProjectConfigRecord {
	return ProjectConfigRecord{
		ID:         1,
		ProjectID:  1,
		ConfigPath: "examples/areamatrix/areaflow.yaml",
		ConfigHash: "hash-config",
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
		Engines: map[string]any{
			"profiles": []any{
				map[string]any{"id": "codex-cli", "secret_ref": "none", "enabled": false},
			},
		},
		Active:   true,
		LoadedAt: loadedAt,
	}
}

func testGeneratedWritePermissionRows(generatedAllowed bool) []permissionRow {
	generatedEffect := "deny"
	if generatedAllowed {
		generatedEffect = "allow"
	}
	rows := []permissionRow{
		{Effect: "allow", Capability: "read_project", ResourceType: "capability", Pattern: "read_project"},
		{Effect: "allow", Capability: "write_status", ResourceType: "capability", Pattern: "write_status"},
		{Effect: "allow", Capability: "write_artifacts", ResourceType: "capability", Pattern: "write_artifacts"},
		{Effect: generatedEffect, Capability: "write_generated", ResourceType: "capability", Pattern: "write_generated"},
		{Effect: "deny", Capability: "write_workflow", ResourceType: "capability", Pattern: "write_workflow"},
		{Effect: "deny", Capability: "write_code", ResourceType: "capability", Pattern: "write_code"},
		{Effect: "deny", Capability: "run_commands", ResourceType: "capability", Pattern: "run_commands"},
		{Effect: "deny", Capability: "manage_git", ResourceType: "capability", Pattern: "manage_git"},
		{Effect: "deny", Capability: "network", ResourceType: "capability", Pattern: "network"},
		{Effect: "deny", Capability: "use_secrets", ResourceType: "capability", Pattern: "use_secrets"},
		{Effect: "deny", Capability: "execute_agents", ResourceType: "capability", Pattern: "execute_agents"},
		{Effect: "allow", Capability: "write_status", ResourceType: "path", Pattern: ".areaflow/status.json"},
		{Effect: "deny", Capability: "*", ResourceType: "path", Pattern: "workflow/versions/*/execution/**"},
		{Effect: "deny", Capability: "*", ResourceType: "path", Pattern: "workflow/versions/*/execution/_shared/progress.json"},
		{Effect: "deny", Capability: "*", ResourceType: "path", Pattern: "**/*.sqlite"},
		{Effect: "deny", Capability: "*", ResourceType: "path", Pattern: "**/*.db"},
	}
	if generatedAllowed {
		rows = append(rows,
			permissionRow{Effect: "allow", Capability: "write_generated", ResourceType: "path", Pattern: ".areaflow/generated/**"},
			permissionRow{Effect: "allow", Capability: "write_generated", ResourceType: "path", Pattern: ".areamatrix/generated/**"},
		)
	} else {
		rows = append(rows, permissionRow{Effect: "deny", Capability: "*", ResourceType: "path", Pattern: ".areamatrix/**"})
	}
	return rows
}

func assertGeneratedWriteItem(t *testing.T, readiness GeneratedWriteReadiness, key string, status string) {
	t.Helper()
	item := generatedWriteItemByKey(readiness.Items, key)
	if item.Key == "" {
		t.Fatalf("item %s not found: %+v", key, readiness.Items)
	}
	if item.Status != status {
		t.Fatalf("item %s status = %q, want %q: %+v", key, item.Status, status, item)
	}
}

func generatedWriteItemByKey(items []ReadinessItem, key string) ReadinessItem {
	for _, item := range items {
		if item.Key == key {
			return item
		}
	}
	return ReadinessItem{}
}
