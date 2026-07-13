package project

import (
	"testing"
	"time"
)

func TestBuildPermissionPolicyDoctorPassesAreaMatrixBaseline(t *testing.T) {
	created := time.Date(2026, 6, 30, 12, 0, 0, 0, time.UTC)
	doctor := BuildPermissionPolicyDoctor(
		Record{ID: 1, Key: "areamatrix"},
		testPermissionProjectConfig(created),
		true,
		testPermissionRows(),
		PermissionPolicyDoctorOptions{GeneratedAt: created},
	)

	if doctor.Status != "pass" || doctor.Mode != "read_only_permission_policy_doctor" {
		t.Fatalf("unexpected permission doctor: %+v", doctor)
	}
	assertPermissionCheck(t, doctor, "project_config", "pass")
	assertPermissionCheck(t, doctor, "default_read_only", "pass")
	assertPermissionCheck(t, doctor, "status_export_write", "pass")
	assertPermissionCheck(t, doctor, "dangerous_write_denies", "pass")
	assertPermissionCheck(t, doctor, "command_policy", "pass")
	assertPermissionCheck(t, doctor, "secret_policy", "pass")
	assertPermissionCheck(t, doctor, "worker_capability_policy", "pass")
	assertPermissionCheckMetadataBool(t, doctor, "worker_capability_policy", "manage_workers", false)
	if !doctor.GeneratedAt.Equal(created) {
		t.Fatalf("generated_at = %s, want %s", doctor.GeneratedAt, created)
	}
}

func TestBuildPermissionPolicyDoctorFailsMissingDangerousDeny(t *testing.T) {
	config := testPermissionProjectConfig(time.Now().UTC())
	config.Permissions["forbidden_paths"] = []any{".areamatrix/**"}
	doctor := BuildPermissionPolicyDoctor(
		Record{ID: 1, Key: "areamatrix"},
		config,
		true,
		[]permissionRow{
			{Effect: "allow", Capability: "write_status", ResourceType: "capability", Pattern: "write_status"},
			{Effect: "allow", Capability: "write_status", ResourceType: "path", Pattern: ".areaflow/status.json"},
			{Effect: "deny", Capability: "*", ResourceType: "path", Pattern: ".areamatrix/**"},
		},
		PermissionPolicyDoctorOptions{},
	)

	if doctor.Status != "fail" {
		t.Fatalf("expected fail status: %+v", doctor)
	}
	assertPermissionCheck(t, doctor, "dangerous_write_denies", "fail")
}

func TestBuildPermissionPolicyDoctorWarnsRunCommandsEnabled(t *testing.T) {
	config := testPermissionProjectConfig(time.Now().UTC())
	capabilities := config.Permissions["capabilities"].(map[string]any)
	capabilities["run_commands"] = true
	doctor := BuildPermissionPolicyDoctor(
		Record{ID: 1, Key: "areamatrix"},
		config,
		true,
		testPermissionRows(),
		PermissionPolicyDoctorOptions{},
	)

	if doctor.Status != "warn" {
		t.Fatalf("expected warn status: %+v", doctor)
	}
	assertPermissionCheck(t, doctor, "default_read_only", "warn")
	assertPermissionCheck(t, doctor, "command_policy", "warn")
}

func TestBuildPermissionPolicyDoctorExposesManageWorkersCapability(t *testing.T) {
	config := testPermissionProjectConfig(time.Now().UTC())
	capabilities := config.Permissions["capabilities"].(map[string]any)
	capabilities["manage_workers"] = true
	doctor := BuildPermissionPolicyDoctor(
		Record{ID: 1, Key: "areamatrix"},
		config,
		true,
		testPermissionRows(),
		PermissionPolicyDoctorOptions{},
	)

	assertPermissionCheckMetadataBool(t, doctor, "worker_capability_policy", "manage_workers", true)
}

func testPermissionProjectConfig(loadedAt time.Time) ProjectConfigRecord {
	return ProjectConfigRecord{
		ID:         1,
		ProjectID:  1,
		ConfigPath: "examples/areamatrix/areaflow.yaml",
		ConfigHash: "hash-config",
		Permissions: map[string]any{
			"capabilities": map[string]any{
				"read_project":    true,
				"write_status":    true,
				"write_workflow":  false,
				"write_generated": false,
				"write_code":      false,
				"run_commands":    false,
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
		},
		Engines: map[string]any{
			"profiles": []any{
				map[string]any{"id": "codex-cli", "secret_ref": "none"},
				map[string]any{"id": "openai-main", "secret_ref": "openai/default"},
			},
		},
		StatusExport: map[string]any{
			"path": ".areaflow/status.json",
		},
		Metadata: map[string]any{
			"commands": map[string]any{
				"allowed": []any{"./dev workflow doctor"},
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

func testPermissionRows() []permissionRow {
	return []permissionRow{
		{Effect: "allow", Capability: "write_status", ResourceType: "capability", Pattern: "write_status"},
		{Effect: "deny", Capability: "write_workflow", ResourceType: "capability", Pattern: "write_workflow"},
		{Effect: "deny", Capability: "write_generated", ResourceType: "capability", Pattern: "write_generated"},
		{Effect: "deny", Capability: "write_code", ResourceType: "capability", Pattern: "write_code"},
		{Effect: "deny", Capability: "run_commands", ResourceType: "capability", Pattern: "run_commands"},
		{Effect: "deny", Capability: "manage_git", ResourceType: "capability", Pattern: "manage_git"},
		{Effect: "deny", Capability: "network", ResourceType: "capability", Pattern: "network"},
		{Effect: "deny", Capability: "use_secrets", ResourceType: "capability", Pattern: "use_secrets"},
		{Effect: "deny", Capability: "execute_agents", ResourceType: "capability", Pattern: "execute_agents"},
		{Effect: "allow", Capability: "write_status", ResourceType: "path", Pattern: ".areaflow/status.json"},
		{Effect: "deny", Capability: "*", ResourceType: "path", Pattern: "workflow/versions/*/execution/**"},
		{Effect: "deny", Capability: "*", ResourceType: "path", Pattern: "workflow/versions/*/execution/_shared/progress.json"},
		{Effect: "deny", Capability: "*", ResourceType: "path", Pattern: ".areamatrix/**"},
		{Effect: "deny", Capability: "*", ResourceType: "path", Pattern: "**/*.sqlite"},
		{Effect: "deny", Capability: "*", ResourceType: "path", Pattern: "**/*.db"},
		{Effect: "allow", Capability: "run_commands", ResourceType: "command", Pattern: "./dev workflow doctor"},
		{Effect: "deny", Capability: "run_commands", ResourceType: "command", Pattern: "./task-loop run"},
		{Effect: "deny", Capability: "run_commands", ResourceType: "command", Pattern: "git reset --hard"},
		{Effect: "deny", Capability: "run_commands", ResourceType: "command", Pattern: "git checkout --"},
		{Effect: "deny", Capability: "run_commands", ResourceType: "command", Pattern: "rm -rf"},
	}
}

func assertPermissionCheck(t *testing.T, doctor PermissionPolicyDoctor, key string, status string) {
	t.Helper()
	for _, check := range doctor.Checks {
		if check.Key == key {
			if check.Status != status {
				t.Fatalf("check %s status = %q, want %q: %+v", key, check.Status, status, check)
			}
			return
		}
	}
	t.Fatalf("check %s not found: %+v", key, doctor.Checks)
}

func assertPermissionCheckMetadataBool(t *testing.T, doctor PermissionPolicyDoctor, key string, metadataKey string, want bool) {
	t.Helper()
	for _, check := range doctor.Checks {
		if check.Key != key {
			continue
		}
		got, ok := check.Metadata[metadataKey].(bool)
		if !ok || got != want {
			t.Fatalf("check %s metadata %s = %v/%t, want %t: %+v", key, metadataKey, check.Metadata[metadataKey], ok, want, check)
		}
		return
	}
	t.Fatalf("check %s not found: %+v", key, doctor.Checks)
}
