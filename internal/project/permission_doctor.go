package project

import (
	"context"
	"fmt"
	"strings"
	"time"
)

type PermissionPolicyDoctorOptions struct {
	GeneratedAt time.Time
}

type PermissionPolicyCheck struct {
	Key      string
	Category string
	Status   string
	Message  string
	Metadata map[string]any
}

type PermissionPolicyDoctor struct {
	Status      string
	Mode        string
	Project     Record
	Checks      []PermissionPolicyCheck
	GeneratedAt time.Time
}

type permissionRow struct {
	Effect       string
	Capability   string
	ResourceType string
	Pattern      string
}

func (s Store) PermissionPolicyDoctor(ctx context.Context, record Record, options PermissionPolicyDoctorOptions) (PermissionPolicyDoctor, error) {
	options = normalizePermissionPolicyDoctorOptions(options)
	config, hasConfig, err := s.ActiveProjectConfig(ctx, record.ID)
	if err != nil {
		return PermissionPolicyDoctor{}, err
	}
	permissions, err := s.projectPermissionRows(ctx, record.ID)
	if err != nil {
		return PermissionPolicyDoctor{}, err
	}
	doctor := BuildPermissionPolicyDoctor(record, config, hasConfig, permissions, options)
	return doctor, nil
}

func normalizePermissionPolicyDoctorOptions(options PermissionPolicyDoctorOptions) PermissionPolicyDoctorOptions {
	if options.GeneratedAt.IsZero() {
		options.GeneratedAt = time.Now().UTC()
	}
	return options
}

func (s Store) projectPermissionRows(ctx context.Context, projectID int64) ([]permissionRow, error) {
	rows, err := s.pool.Query(ctx, `
SELECT effect, capability, resource_type, pattern
FROM project_permissions
WHERE project_id = $1
ORDER BY resource_type, capability, effect, pattern`,
		projectID,
	)
	if err != nil {
		return nil, fmt.Errorf("list project permissions: %w", err)
	}
	defer rows.Close()

	permissions := []permissionRow{}
	for rows.Next() {
		var permission permissionRow
		if err := rows.Scan(&permission.Effect, &permission.Capability, &permission.ResourceType, &permission.Pattern); err != nil {
			return nil, fmt.Errorf("scan project permission policy: %w", err)
		}
		permissions = append(permissions, permission)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate project permission policy: %w", err)
	}
	return permissions, nil
}

func BuildPermissionPolicyDoctor(record Record, config ProjectConfigRecord, hasConfig bool, permissions []permissionRow, options PermissionPolicyDoctorOptions) PermissionPolicyDoctor {
	options = normalizePermissionPolicyDoctorOptions(options)
	doctor := PermissionPolicyDoctor{
		Status:      "pass",
		Mode:        "read_only_permission_policy_doctor",
		Project:     record,
		Checks:      []PermissionPolicyCheck{},
		GeneratedAt: options.GeneratedAt,
	}
	add := func(check PermissionPolicyCheck) {
		if check.Metadata == nil {
			check.Metadata = map[string]any{}
		}
		doctor.Checks = append(doctor.Checks, check)
		if worsePermissionStatus(check.Status, doctor.Status) {
			doctor.Status = check.Status
		}
	}
	index := permissionPolicyIndex(permissions)
	capabilities := mapFromConfigPart(config.Permissions, "capabilities")
	writePaths := stringSliceFromConfigPart(config.Permissions, "write_paths")
	forbiddenPaths := stringSliceFromConfigPart(config.Permissions, "forbidden_paths")
	allowedCommands := stringSliceFromConfigPart(config.Metadata, "commands", "allowed")
	forbiddenCommands := stringSliceFromConfigPart(config.Metadata, "commands", "forbidden")
	statusExport := config.StatusExport
	scheduling := config.Scheduling
	engines := config.Engines

	add(checkProjectConfigPresent(hasConfig, config))
	add(checkDefaultReadOnly(capabilities))
	add(checkStatusExportPolicy(statusExport, writePaths, forbiddenPaths, index))
	add(checkDangerousWriteDenied(index, forbiddenPaths))
	add(checkCommandPolicy(capabilities, allowedCommands, forbiddenCommands, index))
	add(checkSecretPolicy(capabilities, engines))
	add(checkNetworkPolicy(capabilities))
	add(checkWorkerCapabilityPolicy(capabilities, scheduling))
	add(checkGitPolicy(capabilities))
	add(checkPermissionAuditReadiness())
	return doctor
}

func checkProjectConfigPresent(hasConfig bool, config ProjectConfigRecord) PermissionPolicyCheck {
	if !hasConfig {
		return PermissionPolicyCheck{
			Key:      "project_config",
			Category: "config",
			Status:   "fail",
			Message:  "active project config is missing",
		}
	}
	return PermissionPolicyCheck{
		Key:      "project_config",
		Category: "config",
		Status:   "pass",
		Message:  "active project config is available",
		Metadata: map[string]any{
			"config_path": config.ConfigPath,
			"config_hash": config.ConfigHash,
			"loaded_at":   config.LoadedAt,
		},
	}
}

func checkDefaultReadOnly(capabilities map[string]bool) PermissionPolicyCheck {
	risky := []string{"write_workflow", "write_generated", "write_code", "run_commands", "manage_git", "network", "use_secrets", "execute_agents"}
	enabled := []string{}
	for _, capability := range risky {
		if capabilities[capability] {
			enabled = append(enabled, capability)
		}
	}
	if len(enabled) > 0 {
		return PermissionPolicyCheck{
			Key:      "default_read_only",
			Category: "capability",
			Status:   "warn",
			Message:  "high-risk capabilities are enabled",
			Metadata: map[string]any{"enabled": enabled},
		}
	}
	return PermissionPolicyCheck{
		Key:      "default_read_only",
		Category: "capability",
		Status:   "pass",
		Message:  "high-risk capabilities are disabled by default",
	}
}

func checkStatusExportPolicy(statusExport map[string]any, writePaths []string, forbiddenPaths []string, index map[string]permissionRow) PermissionPolicyCheck {
	path := stringFromMap(statusExport, "path")
	if path == "" {
		path = ".areaflow/status.json"
	}
	allowed := stringSliceContains(writePaths, path) && hasPermission(index, "allow", "write_status", "path", path) && hasPermission(index, "allow", "write_status", "capability", "write_status")
	denied := pathMatchesAny(path, forbiddenPaths)
	if allowed && !denied {
		return PermissionPolicyCheck{
			Key:      "status_export_write",
			Category: "path",
			Status:   "pass",
			Message:  "status export path is explicitly allowed and not denied",
			Metadata: map[string]any{"path": path},
		}
	}
	return PermissionPolicyCheck{
		Key:      "status_export_write",
		Category: "path",
		Status:   "fail",
		Message:  "status export path is not safely allowed",
		Metadata: map[string]any{
			"path":           path,
			"in_write_paths": stringSliceContains(writePaths, path),
			"denied":         denied,
		},
	}
}

func checkDangerousWriteDenied(index map[string]permissionRow, forbiddenPaths []string) PermissionPolicyCheck {
	required := []string{
		"workflow/versions/*/execution/**",
		"workflow/versions/*/execution/_shared/progress.json",
		".areamatrix/**",
		"**/*.sqlite",
		"**/*.db",
	}
	missing := []string{}
	for _, pattern := range required {
		if !stringSliceContains(forbiddenPaths, pattern) && !hasPermission(index, "deny", "*", "path", pattern) {
			missing = append(missing, pattern)
		}
	}
	if len(missing) == 0 {
		return PermissionPolicyCheck{
			Key:      "dangerous_write_denies",
			Category: "path",
			Status:   "pass",
			Message:  "dangerous workflow, app metadata and database paths are denied",
			Metadata: map[string]any{"required": required},
		}
	}
	return PermissionPolicyCheck{
		Key:      "dangerous_write_denies",
		Category: "path",
		Status:   "fail",
		Message:  "required dangerous path denies are missing",
		Metadata: map[string]any{"missing": missing},
	}
}

func checkCommandPolicy(capabilities map[string]bool, allowedCommands []string, forbiddenCommands []string, index map[string]permissionRow) PermissionPolicyCheck {
	requiredForbidden := []string{"./task-loop run", "git reset --hard", "git checkout --", "rm -rf"}
	missingForbidden := []string{}
	for _, command := range requiredForbidden {
		if !stringSliceContains(forbiddenCommands, command) && !hasPermission(index, "deny", "run_commands", "command", command) {
			missingForbidden = append(missingForbidden, command)
		}
	}
	runAllowed := capabilities["run_commands"] || hasPermission(index, "allow", "run_commands", "capability", "run_commands")
	if len(missingForbidden) == 0 && !runAllowed {
		return PermissionPolicyCheck{
			Key:      "command_policy",
			Category: "command",
			Status:   "pass",
			Message:  "dangerous commands are denied and run_commands is disabled",
			Metadata: map[string]any{
				"allowed_commands":   allowedCommands,
				"forbidden_commands": forbiddenCommands,
				"run_commands":       runAllowed,
			},
		}
	}
	status := "warn"
	message := "command policy needs review"
	if len(missingForbidden) > 0 {
		status = "fail"
		message = "required forbidden commands are missing"
	}
	return PermissionPolicyCheck{
		Key:      "command_policy",
		Category: "command",
		Status:   status,
		Message:  message,
		Metadata: map[string]any{
			"allowed_commands":   allowedCommands,
			"forbidden_commands": forbiddenCommands,
			"missing_forbidden":  missingForbidden,
			"run_commands":       runAllowed,
		},
	}
}

func checkSecretPolicy(capabilities map[string]bool, engines map[string]any) PermissionPolicyCheck {
	useSecrets := capabilities["use_secrets"]
	refs := engineSecretRefs(engines)
	nonNoneRefs := []string{}
	for _, ref := range refs {
		if ref != "" && ref != "none" {
			nonNoneRefs = append(nonNoneRefs, ref)
		}
	}
	if !useSecrets {
		return PermissionPolicyCheck{
			Key:      "secret_policy",
			Category: "secret",
			Status:   "pass",
			Message:  "secret usage capability is disabled; secret refs remain unresolved",
			Metadata: map[string]any{
				"secret_refs":     refs,
				"active_refs":     nonNoneRefs,
				"use_secrets":     useSecrets,
				"secrets_resolve": false,
			},
		}
	}
	return PermissionPolicyCheck{
		Key:      "secret_policy",
		Category: "secret",
		Status:   "warn",
		Message:  "secret usage is enabled and needs secret store/audit readiness",
		Metadata: map[string]any{
			"secret_refs": refs,
			"use_secrets": useSecrets,
		},
	}
}

func checkNetworkPolicy(capabilities map[string]bool) PermissionPolicyCheck {
	if !capabilities["network"] {
		return PermissionPolicyCheck{
			Key:      "network_policy",
			Category: "network",
			Status:   "pass",
			Message:  "network capability is disabled",
		}
	}
	return PermissionPolicyCheck{
		Key:      "network_policy",
		Category: "network",
		Status:   "warn",
		Message:  "network capability is enabled and needs allowlist/audit readiness",
	}
}

func checkWorkerCapabilityPolicy(capabilities map[string]bool, scheduling map[string]any) PermissionPolicyCheck {
	required := stringSliceFromConfigPart(scheduling, "required_capabilities")
	manageWorkers := capabilities["manage_workers"]
	missing := []string{}
	for _, capability := range []string{"read_project", "write_artifacts"} {
		if !stringSliceContains(required, capability) {
			missing = append(missing, capability)
		}
	}
	if len(missing) == 0 {
		return PermissionPolicyCheck{
			Key:      "worker_capability_policy",
			Category: "worker",
			Status:   "pass",
			Message:  "worker scheduling declares required capabilities",
			Metadata: map[string]any{
				"required_capabilities": required,
				"manage_workers":        manageWorkers,
			},
		}
	}
	return PermissionPolicyCheck{
		Key:      "worker_capability_policy",
		Category: "worker",
		Status:   "warn",
		Message:  "worker scheduling is missing expected capabilities",
		Metadata: map[string]any{
			"required_capabilities": required,
			"missing":               missing,
			"manage_workers":        manageWorkers,
		},
	}
}

func checkGitPolicy(capabilities map[string]bool) PermissionPolicyCheck {
	if !capabilities["manage_git"] {
		return PermissionPolicyCheck{
			Key:      "git_policy",
			Category: "git",
			Status:   "pass",
			Message:  "git management capability is disabled",
		}
	}
	return PermissionPolicyCheck{
		Key:      "git_policy",
		Category: "git",
		Status:   "warn",
		Message:  "git management capability is enabled and needs allowlist/audit readiness",
	}
}

func checkPermissionAuditReadiness() PermissionPolicyCheck {
	return PermissionPolicyCheck{
		Key:      "permission_audit_readiness",
		Category: "audit",
		Status:   "pass",
		Message:  "permission policy doctor is read-only; mutating permission changes must emit audit_events",
		Metadata: map[string]any{
			"permission_change_action": "permission.change",
			"doctor_writes_audit":      false,
		},
	}
}

func permissionPolicyIndex(rows []permissionRow) map[string]permissionRow {
	index := map[string]permissionRow{}
	for _, row := range rows {
		index[permissionPolicyKey(row.Effect, row.Capability, row.ResourceType, row.Pattern)] = row
	}
	return index
}

func hasPermission(index map[string]permissionRow, effect string, capability string, resourceType string, pattern string) bool {
	_, ok := index[permissionPolicyKey(effect, capability, resourceType, pattern)]
	return ok
}

func permissionPolicyKey(effect string, capability string, resourceType string, pattern string) string {
	return strings.TrimSpace(effect) + "\x00" + strings.TrimSpace(capability) + "\x00" + strings.TrimSpace(resourceType) + "\x00" + strings.TrimSpace(pattern)
}

func mapFromConfigPart(root map[string]any, key string) map[string]bool {
	raw, ok := root[key].(map[string]any)
	if !ok {
		return map[string]bool{}
	}
	out := map[string]bool{}
	for k, v := range raw {
		switch value := v.(type) {
		case bool:
			out[k] = value
		case string:
			out[k] = value == "true"
		}
	}
	return out
}

func stringSliceFromConfigPart(root map[string]any, path ...string) []string {
	var current any = root
	for _, key := range path {
		next, ok := current.(map[string]any)
		if !ok {
			return []string{}
		}
		current = next[key]
	}
	switch value := current.(type) {
	case []any:
		out := []string{}
		for _, item := range value {
			if s, ok := item.(string); ok && strings.TrimSpace(s) != "" {
				out = append(out, strings.TrimSpace(s))
			}
		}
		return out
	case []string:
		return normalizeConfigStringList(value)
	default:
		return []string{}
	}
}

func stringFromMap(values map[string]any, key string) string {
	value, _ := values[key].(string)
	return strings.TrimSpace(value)
}

func engineSecretRefs(engines map[string]any) []string {
	profiles, ok := engines["profiles"].([]any)
	if !ok {
		return []string{}
	}
	refs := []string{}
	for _, raw := range profiles {
		profile, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		ref := stringFromMap(profile, "secret_ref")
		if ref != "" {
			refs = append(refs, ref)
		}
	}
	return refs
}

func pathMatchesAny(path string, patterns []string) bool {
	for _, pattern := range patterns {
		if globMatch(pattern, path) {
			return true
		}
	}
	return false
}

func stringSliceContains(values []string, want string) bool {
	want = strings.TrimSpace(want)
	for _, value := range values {
		if strings.TrimSpace(value) == want {
			return true
		}
	}
	return false
}

func worsePermissionStatus(candidate string, current string) bool {
	return permissionStatusRank(candidate) > permissionStatusRank(current)
}

func permissionStatusRank(status string) int {
	switch status {
	case "fail":
		return 3
	case "warn":
		return 2
	case "pass":
		return 1
	default:
		return 0
	}
}
