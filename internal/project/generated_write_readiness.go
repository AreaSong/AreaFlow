package project

import (
	"context"
	"fmt"
	"time"
)

type GeneratedWriteReadinessOptions struct {
	GeneratedAt time.Time
}

type GeneratedWriteReadiness struct {
	Project                       Record
	Status                        string
	Mode                          string
	Items                         []ReadinessItem
	RequiredCapabilities          []string
	AllowedGeneratedPrefixes      []string
	RequiredWritePaths            []string
	ConfiguredWritePaths          []string
	ConfiguredForbiddenPaths      []string
	Blockers                      []string
	ReviewBlockers                []string
	ForbiddenActions              []string
	ReadyForReview                bool
	ApplyOpen                     bool
	RealAreaMatrixWriteOpened     bool
	GeneratedOnly                 bool
	ProjectConfigRead             bool
	ProjectReadAttempted          bool
	ProjectWriteAttempted         bool
	ExecutionWriteAttempted       bool
	AreaFlowArtifactWritten       bool
	AreaFlowExecutionStateWritten bool
	EngineCallAttempted           bool
	CommandsRun                   bool
	SecretsResolved               bool
	NetworkUsed                   bool
	TaskClaimed                   bool
	WorkerStarted                 bool
	LeaseCreated                  bool
	AttemptCreated                bool
	ArtifactCreated               bool
	GeneratedAt                   time.Time
}

func (s Store) GeneratedWriteReadiness(ctx context.Context, record Record, options GeneratedWriteReadinessOptions) (GeneratedWriteReadiness, error) {
	options = normalizeGeneratedWriteReadinessOptions(options)
	config, hasConfig, err := s.ActiveProjectConfig(ctx, record.ID)
	if err != nil {
		return GeneratedWriteReadiness{}, err
	}
	permissions, err := s.projectPermissionRows(ctx, record.ID)
	if err != nil {
		return GeneratedWriteReadiness{}, err
	}
	return BuildGeneratedWriteReadiness(record, config, hasConfig, permissions, options), nil
}

func normalizeGeneratedWriteReadinessOptions(options GeneratedWriteReadinessOptions) GeneratedWriteReadinessOptions {
	if options.GeneratedAt.IsZero() {
		options.GeneratedAt = time.Now().UTC()
	}
	return options
}

func BuildGeneratedWriteReadiness(record Record, config ProjectConfigRecord, hasConfig bool, permissions []permissionRow, options GeneratedWriteReadinessOptions) GeneratedWriteReadiness {
	options = normalizeGeneratedWriteReadinessOptions(options)
	requiredCapabilities := []string{"read_project", "write_artifacts", "write_generated"}
	requiredWritePaths := []string{".areaflow/generated/**", ".areamatrix/generated/**"}
	forbiddenActions := []string{
		"queue_run",
		"claim_task",
		"start_worker",
		"create_lease",
		"create_attempt",
		"create_artifact",
		"write_project_file",
		"write_source_file",
		"write_workflow_execution",
		"write_progress_json",
		"git_checkpoint",
		"execute_engine",
		"run_commands",
		"resolve_secrets",
		"use_network",
	}
	result := GeneratedWriteReadiness{
		Project:                   record,
		Status:                    "blocked",
		Mode:                      "read_only_generated_write_readiness",
		RequiredCapabilities:      requiredCapabilities,
		AllowedGeneratedPrefixes:  managedGeneratedWritePrefixes,
		RequiredWritePaths:        requiredWritePaths,
		ForbiddenActions:          forbiddenActions,
		ReadyForReview:            true,
		ApplyOpen:                 false,
		RealAreaMatrixWriteOpened: false,
		GeneratedOnly:             true,
		ProjectConfigRead:         hasConfig,
		ProjectReadAttempted:      false,
		ProjectWriteAttempted:     false,
		ExecutionWriteAttempted:   false,
		EngineCallAttempted:       false,
		CommandsRun:               false,
		SecretsResolved:           false,
		NetworkUsed:               false,
		TaskClaimed:               false,
		WorkerStarted:             false,
		LeaseCreated:              false,
		AttemptCreated:            false,
		ArtifactCreated:           false,
		GeneratedAt:               options.GeneratedAt,
	}
	if hasConfig {
		result.ConfiguredWritePaths = stringSliceFromConfigPart(config.Permissions, "write_paths")
		result.ConfiguredForbiddenPaths = stringSliceFromConfigPart(config.Permissions, "forbidden_paths")
	}

	if !hasConfig {
		addGeneratedWriteReadinessItem(&result, "project_config", "blocked", "active project config is required before generated-only dogfood review", nil, true)
	} else {
		addGeneratedWriteReadinessItem(&result, "project_config", "pass", "active project config is available", map[string]any{
			"config_path": config.ConfigPath,
			"config_hash": config.ConfigHash,
			"loaded_at":   config.LoadedAt,
		}, false)
	}
	if hasConfig {
		addGeneratedCapabilityItem(&result, config, permissions, requiredCapabilities)
		addGeneratedPrefixPathItem(&result, config, permissions, requiredWritePaths)
		addGeneratedDangerousDenyItem(&result, config)
		addGeneratedHighRiskClosedItem(&result, config, permissions)
		addGeneratedRollbackContractItem(&result)
	}
	if len(result.ReviewBlockers) > 0 {
		result.ReadyForReview = false
	}
	addGeneratedWriteReadinessItem(&result, "real_areamatrix_apply_open", "blocked", "real AreaMatrix generated-only apply remains closed until explicit approval opens it", map[string]any{
		"apply_open":                   result.ApplyOpen,
		"real_areamatrix_write_opened": result.RealAreaMatrixWriteOpened,
		"ready_for_review":             result.ReadyForReview,
	}, false)
	addGeneratedWriteReadinessItem(&result, "read_only_readiness", "pass", "readiness did not queue runs, claim tasks, create leases, create attempts, create artifacts or write project files", map[string]any{
		"project_config_read":               result.ProjectConfigRead,
		"project_read_attempted":            result.ProjectReadAttempted,
		"project_write_attempted":           result.ProjectWriteAttempted,
		"execution_write_attempted":         result.ExecutionWriteAttempted,
		"area_flow_artifact_written":        result.AreaFlowArtifactWritten,
		"area_flow_execution_state_written": result.AreaFlowExecutionStateWritten,
		"engine_call_attempted":             result.EngineCallAttempted,
		"commands_run":                      result.CommandsRun,
		"secrets_resolved":                  result.SecretsResolved,
		"network_used":                      result.NetworkUsed,
		"task_claimed":                      result.TaskClaimed,
		"worker_started":                    result.WorkerStarted,
		"lease_created":                     result.LeaseCreated,
		"attempt_created":                   result.AttemptCreated,
		"artifact_created":                  result.ArtifactCreated,
	}, false)
	return result
}

func addGeneratedCapabilityItem(result *GeneratedWriteReadiness, config ProjectConfigRecord, permissions []permissionRow, required []string) {
	capabilities := mapFromConfigPart(config.Permissions, "capabilities")
	index := permissionPolicyIndex(permissions)
	missing := []string{}
	for _, capability := range required {
		if !generatedCapabilityAllowed(capabilities, index, capability) {
			missing = append(missing, capability)
		}
	}
	if len(missing) == 0 {
		addGeneratedWriteReadinessItem(result, "required_capabilities", "pass", "project config and permission rows allow generated-only planning capabilities", map[string]any{
			"required_capabilities": required,
		}, false)
		return
	}
	addGeneratedWriteReadinessItem(result, "required_capabilities", "blocked", "project is missing required generated-only planning capabilities", map[string]any{
		"required_capabilities": required,
		"missing":               missing,
	}, true)
}

func generatedCapabilityAllowed(capabilities map[string]bool, index map[string]permissionRow, capability string) bool {
	return capabilities[capability] &&
		hasPermission(index, "allow", capability, "capability", capability) &&
		!hasPermission(index, "deny", capability, "capability", capability)
}

func addGeneratedPrefixPathItem(result *GeneratedWriteReadiness, config ProjectConfigRecord, permissions []permissionRow, requiredWritePaths []string) {
	writePaths := stringSliceFromConfigPart(config.Permissions, "write_paths")
	forbiddenPaths := stringSliceFromConfigPart(config.Permissions, "forbidden_paths")
	index := permissionPolicyIndex(permissions)
	missing := []string{}
	denied := []string{}
	for _, path := range requiredWritePaths {
		if !stringSliceContains(writePaths, path) || !hasPermission(index, "allow", "write_generated", "path", path) {
			missing = append(missing, path)
			continue
		}
		if pathMatchesAny(path, forbiddenPaths) || hasPermission(index, "deny", "*", "path", path) || hasPermission(index, "deny", "write_generated", "path", path) {
			denied = append(denied, path)
		}
	}
	if len(missing) == 0 && len(denied) == 0 {
		addGeneratedWriteReadinessItem(result, "generated_prefix_path_policy", "pass", "generated-only write paths are explicitly allowlisted and not denied", map[string]any{
			"required_write_paths":       requiredWritePaths,
			"configured_write_paths":     writePaths,
			"configured_forbidden_paths": forbiddenPaths,
		}, false)
		return
	}
	addGeneratedWriteReadinessItem(result, "generated_prefix_path_policy", "blocked", "generated-only write paths are not safely allowlisted", map[string]any{
		"required_write_paths":       requiredWritePaths,
		"configured_write_paths":     writePaths,
		"configured_forbidden_paths": forbiddenPaths,
		"missing":                    missing,
		"denied":                     denied,
	}, true)
}

func addGeneratedDangerousDenyItem(result *GeneratedWriteReadiness, config ProjectConfigRecord) {
	forbiddenPaths := stringSliceFromConfigPart(config.Permissions, "forbidden_paths")
	requiredDenies := []string{
		"workflow/versions/*/execution/**",
		"workflow/versions/*/execution/_shared/progress.json",
		"**/*.sqlite",
		"**/*.db",
	}
	missing := []string{}
	for _, path := range requiredDenies {
		if !stringSliceContains(forbiddenPaths, path) {
			missing = append(missing, path)
		}
	}
	if len(missing) == 0 {
		addGeneratedWriteReadinessItem(result, "dangerous_path_denies", "pass", "execution, progress and local database paths remain denied", map[string]any{
			"required_denies": requiredDenies,
		}, false)
		return
	}
	addGeneratedWriteReadinessItem(result, "dangerous_path_denies", "blocked", "generated-only readiness requires dangerous write denies", map[string]any{
		"required_denies": requiredDenies,
		"missing":         missing,
	}, true)
}

func addGeneratedHighRiskClosedItem(result *GeneratedWriteReadiness, config ProjectConfigRecord, permissions []permissionRow) {
	capabilities := mapFromConfigPart(config.Permissions, "capabilities")
	index := permissionPolicyIndex(permissions)
	closed := []string{"write_workflow", "write_code", "run_commands", "manage_git", "network", "use_secrets", "execute_agents"}
	open := []string{}
	for _, capability := range closed {
		if capabilities[capability] || hasPermission(index, "allow", capability, "capability", capability) {
			open = append(open, capability)
		}
	}
	if len(open) == 0 {
		addGeneratedWriteReadinessItem(result, "high_risk_capabilities_closed", "pass", "source write, command, git, network, secret and engine capabilities remain closed", map[string]any{
			"closed_capabilities": closed,
		}, false)
		return
	}
	addGeneratedWriteReadinessItem(result, "high_risk_capabilities_closed", "blocked", "generated-only dogfood review requires unrelated high-risk capabilities to remain closed", map[string]any{
		"required_closed": closed,
		"open":            open,
	}, true)
}

func addGeneratedRollbackContractItem(result *GeneratedWriteReadiness) {
	addGeneratedWriteReadinessItem(result, "rollback_contract", "pass", "future generated-only apply must use expected-before hash/size, write-set manifest, preimage artifact, verify attempt and rollback verification", map[string]any{
		"required_fields": []string{
			"target_path",
			"expected_before_sha256",
			"expected_before_size",
			"after_sha256",
			"after_size",
			"write_set_artifact_id",
			"preimage_artifact_id",
			"rollback_verified",
		},
	}, false)
}

func addGeneratedWriteReadinessItem(result *GeneratedWriteReadiness, key string, status string, message string, metadata map[string]any, reviewBlocker bool) {
	if metadata == nil {
		metadata = map[string]any{}
	}
	result.Items = append(result.Items, ReadinessItem{
		Key:      key,
		Status:   status,
		Message:  message,
		Metadata: metadata,
	})
	if status == "blocked" || status == "fail" {
		blocker := fmt.Sprintf("%s: %s", key, message)
		result.Blockers = append(result.Blockers, blocker)
		if reviewBlocker {
			result.ReviewBlockers = append(result.ReviewBlockers, blocker)
		}
	}
}
