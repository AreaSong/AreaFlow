package project

import (
	"testing"
	"time"
)

func TestBuildCodexCLIAdapterPreviewBlocksDefaultAreaMatrixConfig(t *testing.T) {
	created := time.Date(2026, 7, 1, 23, 10, 0, 0, time.UTC)
	record := Record{ID: 1, Key: "areamatrix", Name: "AreaMatrix"}
	config := codexPreviewTestConfig(false, map[string]bool{
		"read_project":    true,
		"write_artifacts": true,
		"run_commands":    false,
		"execute_agents":  false,
	})
	preview := BuildCodexCLIAdapterPreview(record, config, true, codexPreviewTestPermissions(config), CommandPermission{}, CodexCLIAdapterPreviewOptions{
		GeneratedAt: created,
	})

	if preview.Status != "blocked" || preview.Mode != "read_only_codex_cli_adapter_preview" {
		t.Fatalf("unexpected preview status: %+v", preview)
	}
	if preview.Engine.Status != "blocked" || preview.Engine.ProfileID != "codex-cli" {
		t.Fatalf("unexpected engine readiness: %+v", preview.Engine)
	}
	assertStringPresent(t, preview.Blockers, "engine_profile_disabled")
	assertStringPresent(t, preview.Blockers, "missing_capability:execute_agents")
	assertStringPresent(t, preview.Blockers, "missing_capability:run_commands")
	assertStringPresent(t, preview.Blockers, "command_not_allowed")
	if preview.Command.Allowed || preview.Command.Reason != "run_commands capability not allowed" {
		t.Fatalf("unexpected command preview: %+v", preview.Command)
	}
	if preview.EngineCallAttempted || preview.CommandsRun || preview.SecretsResolved || preview.NetworkUsed ||
		preview.ProjectWriteAttempted || preview.ExecutionWriteAttempted || preview.ExecutionAllowed {
		t.Fatalf("preview must not attempt execution side effects: %+v", preview)
	}
	if preview.ArtifactRedaction.Status != "ready" || len(preview.ArtifactRedaction.RedactedFields) == 0 {
		t.Fatalf("unexpected artifact redaction plan: %+v", preview.ArtifactRedaction)
	}
}

func TestBuildCodexCLIAdapterPreviewNeedsApprovalWhenPreflightsPass(t *testing.T) {
	record := Record{ID: 1, Key: "areamatrix", Name: "AreaMatrix"}
	config := codexPreviewTestConfig(true, map[string]bool{
		"read_project":    true,
		"write_artifacts": true,
		"run_commands":    true,
		"execute_agents":  true,
	})
	preview := BuildCodexCLIAdapterPreview(record, config, true, codexPreviewTestPermissions(config), CommandPermission{
		CapabilityAllowed: true,
		CommandAllowed:    true,
	}, CodexCLIAdapterPreviewOptions{})

	if preview.Status != "needs_approval" {
		t.Fatalf("status = %q, want needs_approval: %+v", preview.Status, preview)
	}
	assertStringPresent(t, preview.Blockers, "execution_approval_required")
	if !preview.Command.Allowed || preview.Command.Reason != "allowed" {
		t.Fatalf("unexpected command preview: %+v", preview.Command)
	}
	for _, capability := range preview.Capabilities {
		if !capability.Allowed {
			t.Fatalf("capability should be allowed: %+v", capability)
		}
	}
	for _, path := range preview.Paths {
		if !path.Allowed {
			t.Fatalf("path should be allowed or explicitly denied safely: %+v", path)
		}
	}
	if preview.ExecutionAllowed {
		t.Fatal("preview should still require explicit execution approval")
	}
}

func codexPreviewTestConfig(engineEnabled bool, capabilities map[string]bool) ProjectConfigRecord {
	capabilityMap := map[string]any{}
	for key, value := range capabilities {
		capabilityMap[key] = value
	}
	return ProjectConfigRecord{
		Permissions: map[string]any{
			"capabilities": capabilityMap,
			"read_paths": []any{
				"AGENTS.md",
				"docs/**",
				"workflow/**",
			},
			"forbidden_paths": []any{
				"workflow/versions/*/execution/**",
				"**/*.db",
			},
		},
		Scheduling: map[string]any{
			"required_capabilities": []any{"read_project", "write_artifacts"},
			"engine_profile":        "codex-cli",
		},
		Engines: map[string]any{
			"default": "codex-cli",
			"profiles": []any{
				map[string]any{
					"id":         "codex-cli",
					"provider":   "codex-cli",
					"secret_ref": "none",
					"enabled":    engineEnabled,
					"resource_limits": map[string]any{
						"max_active_leases": float64(1),
					},
				},
			},
		},
	}
}

func codexPreviewTestPermissions(config ProjectConfigRecord) []permissionRow {
	capabilities := mapFromConfigPart(config.Permissions, "capabilities")
	rows := []permissionRow{}
	for capability, allowed := range capabilities {
		effect := "deny"
		if allowed {
			effect = "allow"
		}
		rows = append(rows, permissionRow{Effect: effect, Capability: capability, ResourceType: "capability", Pattern: capability})
	}
	for _, path := range stringSliceFromConfigPart(config.Permissions, "read_paths") {
		rows = append(rows, permissionRow{Effect: "allow", Capability: "read_project", ResourceType: "path", Pattern: path})
	}
	for _, path := range stringSliceFromConfigPart(config.Permissions, "forbidden_paths") {
		rows = append(rows, permissionRow{Effect: "deny", Capability: "*", ResourceType: "path", Pattern: path})
	}
	rows = append(rows, permissionRow{Effect: "allow", Capability: "run_commands", ResourceType: "command", Pattern: "codex exec"})
	return rows
}

func assertStringPresent(t *testing.T, values []string, want string) {
	t.Helper()
	for _, value := range values {
		if value == want {
			return
		}
	}
	t.Fatalf("%q not found in %+v", want, values)
}
