package project

import (
	"context"
	"fmt"
	"strings"
	"time"
)

type CodexCLIAdapterPreviewOptions struct {
	Command     string
	GeneratedAt time.Time
}

type CodexCLIAdapterPreview struct {
	Project                 Record
	Status                  string
	Mode                    string
	Engine                  EngineReadiness
	Command                 EngineCommandPreview
	Capabilities            []EngineCapabilityPreflight
	Paths                   []EnginePathPreflight
	ArtifactRedaction       ArtifactRedactionPlan
	ForbiddenActions        []string
	Blockers                []string
	ExecutionAllowed        bool
	ProjectWriteAttempted   bool
	ExecutionWriteAttempted bool
	EngineCallAttempted     bool
	CommandsRun             bool
	SecretsResolved         bool
	NetworkUsed             bool
	GeneratedAt             time.Time
}

type EngineCommandPreview struct {
	Command           string
	Allowed           bool
	Reason            string
	CapabilityAllowed bool
	CommandAllowed    bool
	Denied            bool
}

type EngineCapabilityPreflight struct {
	Capability string
	Required   bool
	Allowed    bool
	Reason     string
}

type EnginePathPreflight struct {
	Path       string
	Capability string
	Effect     string
	Allowed    bool
	Reason     string
}

type ArtifactRedactionPlan struct {
	Status         string
	RetentionClass string
	Rules          []string
	RedactedFields []string
}

func (s Store) CodexCLIAdapterPreview(ctx context.Context, record Record, options CodexCLIAdapterPreviewOptions) (CodexCLIAdapterPreview, error) {
	config, hasConfig, err := s.ActiveProjectConfig(ctx, record.ID)
	if err != nil {
		return CodexCLIAdapterPreview{}, err
	}
	permissions, err := s.projectPermissionRows(ctx, record.ID)
	if err != nil {
		return CodexCLIAdapterPreview{}, err
	}
	permission, err := s.CommandPermission(ctx, record.ID, codexPreviewCommand(options.Command))
	if err != nil {
		return CodexCLIAdapterPreview{}, err
	}
	return BuildCodexCLIAdapterPreview(record, config, hasConfig, permissions, permission, options), nil
}

func BuildCodexCLIAdapterPreview(record Record, config ProjectConfigRecord, hasConfig bool, permissions []permissionRow, commandPermission CommandPermission, options CodexCLIAdapterPreviewOptions) CodexCLIAdapterPreview {
	generatedAt := options.GeneratedAt
	if generatedAt.IsZero() {
		generatedAt = time.Now().UTC()
	}
	command := codexPreviewCommand(options.Command)
	capabilities := map[string]bool{}
	if hasConfig {
		capabilities = mapFromConfigPart(config.Permissions, "capabilities")
	}
	index := permissionPolicyIndex(permissions)
	preview := CodexCLIAdapterPreview{
		Project: record,
		Status:  "ready",
		Mode:    "read_only_codex_cli_adapter_preview",
		Engine:  codexPreviewEngineReadiness(config, hasConfig),
		Command: EngineCommandPreview{
			Command:           command,
			Allowed:           commandPermission.CapabilityAllowed && commandPermission.CommandAllowed && !commandPermission.Denied,
			Reason:            codexCommandReason(commandPermission),
			CapabilityAllowed: commandPermission.CapabilityAllowed,
			CommandAllowed:    commandPermission.CommandAllowed,
			Denied:            commandPermission.Denied,
		},
		ArtifactRedaction: ArtifactRedactionPlan{
			Status:         "ready",
			RetentionClass: "run_evidence",
			Rules: []string{
				"store raw stdout and stderr only as redacted artifacts",
				"redact secret-like environment variables before artifact metadata is persisted",
				"hash prompt and report bodies before they are referenced from audit metadata",
				"do not copy managed project files into artifact store without explicit archive approval",
			},
			RedactedFields: []string{"env", "secret_ref", "stdout", "stderr", "prompt_body"},
		},
		ForbiddenActions: []string{
			"execute_codex_cli",
			"resolve_secrets",
			"write_managed_project",
			"write_workflow_execution",
			"open_network",
		},
		ProjectWriteAttempted:   false,
		ExecutionWriteAttempted: false,
		EngineCallAttempted:     false,
		CommandsRun:             false,
		SecretsResolved:         false,
		NetworkUsed:             false,
		GeneratedAt:             generatedAt,
	}
	if !hasConfig {
		preview.Blockers = append(preview.Blockers, "project_config_missing")
	}
	for _, capability := range codexPreviewRequiredCapabilities(config, hasConfig) {
		allowed := capabilities[capability] || hasPermission(index, "allow", capability, "capability", capability)
		reason := "allowed"
		if !allowed {
			reason = "capability not allowed"
			preview.Blockers = append(preview.Blockers, "missing_capability:"+capability)
		}
		preview.Capabilities = append(preview.Capabilities, EngineCapabilityPreflight{
			Capability: capability,
			Required:   true,
			Allowed:    allowed,
			Reason:     reason,
		})
	}
	if preview.Engine.Provider != "" && preview.Engine.Provider != "codex-cli" {
		preview.Blockers = append(preview.Blockers, "engine_provider_not_codex_cli")
	}
	if preview.Engine.Status == "blocked" {
		preview.Blockers = append(preview.Blockers, preview.Engine.BlockedReasons...)
	}
	if !preview.Command.Allowed {
		preview.Blockers = append(preview.Blockers, "command_not_allowed")
	}
	preview.Paths = codexPreviewPaths(config, hasConfig, capabilities, index)
	for _, path := range preview.Paths {
		if !path.Allowed {
			if path.Effect == "deny" {
				preview.Blockers = append(preview.Blockers, "forbidden_path_deny_missing:"+path.Path)
			} else {
				preview.Blockers = append(preview.Blockers, "path_not_allowed:"+path.Path)
			}
		}
	}
	if len(preview.Blockers) > 0 {
		preview.Status = "blocked"
		preview.ExecutionAllowed = false
		return preview
	}
	preview.Status = "needs_approval"
	preview.Blockers = []string{"execution_approval_required"}
	preview.ExecutionAllowed = false
	return preview
}

func codexPreviewCommand(command string) string {
	command = strings.TrimSpace(command)
	if command == "" {
		return "codex exec"
	}
	return command
}

func codexCommandReason(permission CommandPermission) string {
	if permission.Denied {
		if permission.Reason != "" {
			return permission.Reason
		}
		return "command denied"
	}
	if !permission.CapabilityAllowed {
		return "run_commands capability not allowed"
	}
	if !permission.CommandAllowed {
		return "command not allowed"
	}
	return "allowed"
}

func codexPreviewEngineReadiness(config ProjectConfigRecord, hasConfig bool) EngineReadiness {
	if !hasConfig {
		return EngineReadiness{
			ProfileID:      "codex-cli",
			Provider:       "codex-cli",
			SecretRef:      "none",
			SecretReady:    true,
			ResourceLimits: map[string]any{},
			Status:         "blocked",
			BlockedReasons: []string{"project_config_missing"},
		}
	}
	profileID := stringFromMap(config.Scheduling, "engine_profile")
	if profileID == "" {
		profileID = stringFromMap(config.Engines, "default")
	}
	if profileID == "" {
		profileID = "codex-cli"
	}
	profile, ok := engineProfileFromConfigMap(config.Engines, profileID)
	if !ok {
		return EngineReadiness{
			ProfileID:      profileID,
			Provider:       "codex-cli",
			SecretRef:      "none",
			SecretReady:    true,
			ResourceLimits: map[string]any{},
			Status:         "blocked",
			BlockedReasons: []string{"engine_profile_missing"},
		}
	}
	return engineReadinessFromProfile(profile)
}

func engineProfileFromConfigMap(engines map[string]any, profileID string) (EngineProfileConfig, bool) {
	profiles, ok := engines["profiles"].([]any)
	if !ok {
		return EngineProfileConfig{}, false
	}
	for _, raw := range profiles {
		profileMap, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		if stringFromMap(profileMap, "id") != profileID {
			continue
		}
		return EngineProfileConfig{
			ID:             stringFromMap(profileMap, "id"),
			Provider:       stringFromMap(profileMap, "provider"),
			SecretRef:      stringFromMap(profileMap, "secret_ref"),
			Enabled:        boolFromMap(profileMap, "enabled"),
			ResourceLimits: mapAnyFromMap(profileMap, "resource_limits"),
		}, true
	}
	return EngineProfileConfig{}, false
}

func codexPreviewRequiredCapabilities(config ProjectConfigRecord, hasConfig bool) []string {
	required := []string{"read_project", "write_artifacts", "execute_agents", "run_commands"}
	if hasConfig {
		required = append(required, stringSliceFromConfigPart(config.Scheduling, "required_capabilities")...)
	}
	return normalizeStringList(required)
}

func codexPreviewPaths(config ProjectConfigRecord, hasConfig bool, capabilities map[string]bool, index map[string]permissionRow) []EnginePathPreflight {
	if !hasConfig {
		return []EnginePathPreflight{}
	}
	paths := []EnginePathPreflight{}
	for _, path := range stringSliceFromConfigPart(config.Permissions, "read_paths") {
		paths = append(paths, pathPreflight(path, "read_project", "allow", capabilities, index))
	}
	for _, path := range stringSliceFromConfigPart(config.Permissions, "forbidden_paths") {
		allowed := hasPermission(index, "deny", "*", "path", path)
		reason := "forbidden path denied"
		if !allowed {
			reason = "forbidden path deny missing"
		}
		paths = append(paths, EnginePathPreflight{
			Path:       path,
			Capability: "*",
			Effect:     "deny",
			Allowed:    allowed,
			Reason:     reason,
		})
	}
	paths = append(paths, EnginePathPreflight{
		Path:       "artifact_store",
		Capability: "write_artifacts",
		Effect:     "allow",
		Allowed:    capabilities["write_artifacts"] || hasPermission(index, "allow", "write_artifacts", "capability", "write_artifacts"),
		Reason:     artifactStoreReason(capabilities, index),
	})
	return paths
}

func pathPreflight(path string, capability string, effect string, capabilities map[string]bool, index map[string]permissionRow) EnginePathPreflight {
	allowed := capabilities[capability] && hasPermission(index, effect, capability, "path", path)
	reason := "allowed"
	if !capabilities[capability] {
		reason = fmt.Sprintf("%s capability not allowed", capability)
	} else if !allowed {
		reason = "path not allowed"
	}
	return EnginePathPreflight{
		Path:       path,
		Capability: capability,
		Effect:     effect,
		Allowed:    allowed,
		Reason:     reason,
	}
}

func artifactStoreReason(capabilities map[string]bool, index map[string]permissionRow) string {
	if capabilities["write_artifacts"] || hasPermission(index, "allow", "write_artifacts", "capability", "write_artifacts") {
		return "artifact store metadata write is allowed; managed project writes remain forbidden"
	}
	return "write_artifacts capability not allowed"
}

func boolFromMap(values map[string]any, key string) bool {
	switch value := values[key].(type) {
	case bool:
		return value
	case string:
		return value == "true"
	default:
		return false
	}
}

func mapAnyFromMap(values map[string]any, key string) map[string]any {
	raw, ok := values[key].(map[string]any)
	if !ok {
		return map[string]any{}
	}
	return raw
}
