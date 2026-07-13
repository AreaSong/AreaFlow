package doctor

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/jackc/pgx/v5"

	"github.com/areasong/areaflow/internal/adapter/areamatrix"
	"github.com/areasong/areaflow/internal/project"
)

type Status string

const (
	StatusPass Status = "pass"
	StatusWarn Status = "warn"
	StatusFail Status = "fail"
)

type Detail struct {
	Key   string
	Value string
}

type Check struct {
	Name    string
	Status  Status
	Message string
	Details []Detail
}

type Report struct {
	Project string
	Profile string
	Checks  []Check
}

type Store interface {
	LatestImportSnapshot(ctx context.Context, projectID int64) (project.Snapshot, error)
	ImportInventory(ctx context.Context, projectID int64) (project.ImportInventory, error)
	ActiveProjectConfig(ctx context.Context, projectID int64) (project.ProjectConfigRecord, bool, error)
	CanRunCommand(ctx context.Context, projectID int64, command string) (bool, string, error)
	CommandPermission(ctx context.Context, projectID int64, command string) (project.CommandPermission, error)
}

type SnapshotLoader func(root string) (areamatrix.Snapshot, error)
type CommandRunner func(ctx context.Context, root string, command string, args ...string) CommandResult

type CommandResult struct {
	ExitCode int
	Output   string
	Error    string
}

func AreaMatrix(ctx context.Context, record project.Record, store Store) (Report, error) {
	return AreaMatrixWithLoader(ctx, record, store, areamatrix.Load, runCommand, false)
}

func AreaMatrixWithNative(ctx context.Context, record project.Record, store Store) (Report, error) {
	return AreaMatrixWithLoader(ctx, record, store, areamatrix.Load, runCommand, true)
}

func AreaMatrixWithLoader(ctx context.Context, record project.Record, store Store, load SnapshotLoader, runner CommandRunner, allowNative bool) (Report, error) {
	report := Report{
		Project: record.Key,
		Profile: record.WorkflowProfile,
	}

	if record.Adapter != "areamatrix" {
		report.add("adapter", StatusFail, fmt.Sprintf("unsupported adapter %q", record.Adapter))
		return report, nil
	}
	if record.RootPath == "" {
		report.add("project_root", StatusFail, "project has no local root path")
		return report, nil
	}

	inventory, err := store.ImportInventory(ctx, record.ID)
	if err != nil {
		return report, err
	}
	report.add("db_inventory", inventoryStatus(inventory), "loaded AreaFlow import inventory",
		detail("versions", inventory.Versions),
		detail("residuals", inventory.Residuals),
		detail("artifacts", inventory.Artifacts),
		detail("import_snapshots", inventory.ImportSnapshots),
		detail("mirror_exports", inventory.MirrorExports),
	)

	imported, importErr := store.LatestImportSnapshot(ctx, record.ID)
	if importErr != nil {
		if errors.Is(importErr, pgx.ErrNoRows) {
			report.add("import_snapshot", StatusFail, "no import snapshot found; run `areaflow project import <id>` first")
		} else {
			return report, importErr
		}
	} else {
		report.add("import_snapshot", StatusPass, "latest import snapshot found",
			detail("source_hash", imported.SourceHash),
		)
	}

	current, err := load(record.RootPath)
	if err != nil {
		report.add("source_snapshot", StatusFail, fmt.Sprintf("failed to load AreaMatrix source snapshot: %v", err))
		return report, nil
	}
	report.add("source_snapshot", StatusPass, "loaded current AreaMatrix source snapshot",
		detail("versions", len(current.Versions)),
		detail("residuals", len(current.Residuals)),
		detail("artifacts", len(current.Artifacts)),
		detail("v1_execution", fmt.Sprintf("%d/%d", current.TaskSummary.V1ExecutionDone, current.TaskSummary.V1ExecutionTotal)),
	)

	verifyAdapterIdempotency(record.RootPath, load, current, &report)
	verifyHashDrift(importErr, imported, current, &report)
	verifyProjectConfigDrift(ctx, record, store, &report)
	verifyInventoryConsistency(inventory, current, &report)
	verifyStageCoverage(record.RootPath, current, &report)
	verifyNativeWorkflowDoctor(ctx, record, store, runner, allowNative, &report)
	return report, nil
}

func (r Report) OverallStatus() Status {
	status := StatusPass
	for _, check := range r.Checks {
		if check.Status == StatusFail {
			return StatusFail
		}
		if check.Status == StatusWarn {
			status = StatusWarn
		}
	}
	return status
}

func (r Report) HasFailures() bool {
	return r.OverallStatus() == StatusFail
}

func (r Report) Summary() map[string]any {
	checks := make([]map[string]any, 0, len(r.Checks))
	counts := map[string]int{
		string(StatusPass): 0,
		string(StatusWarn): 0,
		string(StatusFail): 0,
	}
	for _, check := range r.Checks {
		counts[string(check.Status)]++
		details := map[string]string{}
		for _, detail := range check.Details {
			details[detail.Key] = detail.Value
		}
		checks = append(checks, map[string]any{
			"name":    check.Name,
			"status":  string(check.Status),
			"message": check.Message,
			"details": details,
		})
	}
	return map[string]any{
		"project":        r.Project,
		"profile":        r.Profile,
		"overall_status": string(r.OverallStatus()),
		"counts":         counts,
		"checks":         checks,
	}
}

func (r *Report) add(name string, status Status, message string, details ...Detail) {
	r.Checks = append(r.Checks, Check{
		Name:    name,
		Status:  status,
		Message: message,
		Details: details,
	})
}

func inventoryStatus(inventory project.ImportInventory) Status {
	if inventory.ImportSnapshots == 0 || inventory.Versions == 0 {
		return StatusFail
	}
	return StatusPass
}

func verifyAdapterIdempotency(root string, load SnapshotLoader, current areamatrix.Snapshot, report *Report) {
	second, err := load(root)
	if err != nil {
		report.add("import_idempotency", StatusWarn, fmt.Sprintf("second read failed during idempotency check: %v", err))
		return
	}
	if second.StatusSourceHash != current.StatusSourceHash {
		report.add("import_idempotency", StatusWarn, "AreaMatrix snapshot changed during doctor run",
			detail("first_hash", current.StatusSourceHash),
			detail("second_hash", second.StatusSourceHash),
		)
		return
	}
	report.add("import_idempotency", StatusPass, "adapter snapshot is deterministic for current source",
		detail("source_hash", current.StatusSourceHash),
	)
}

func verifyHashDrift(importErr error, imported project.Snapshot, current areamatrix.Snapshot, report *Report) {
	if importErr != nil {
		report.add("hash_drift", StatusFail, "cannot check drift without an import snapshot")
		return
	}
	if imported.SourceHash != current.StatusSourceHash {
		report.add("hash_drift", StatusFail, "current AreaMatrix source differs from latest AreaFlow import snapshot",
			detail("import_hash", imported.SourceHash),
			detail("current_hash", current.StatusSourceHash),
		)
		return
	}
	report.add("hash_drift", StatusPass, "current AreaMatrix source matches latest import snapshot",
		detail("source_hash", current.StatusSourceHash),
	)
}

func verifyProjectConfigDrift(ctx context.Context, record project.Record, store Store, report *Report) {
	active, ok, err := store.ActiveProjectConfig(ctx, record.ID)
	if err != nil {
		report.add("project_config_drift", StatusWarn, fmt.Sprintf("failed to load active project config snapshot: %v", err))
		return
	}
	if !ok {
		report.add("project_config_drift", StatusWarn, "no active project config snapshot found; run `areaflow project add --config <path>` first")
		return
	}
	configPath := active.ConfigPath
	if !filepath.IsAbs(configPath) {
		configPath = filepath.Join(record.RootPath, filepath.FromSlash(configPath))
	}
	current, err := project.LoadConfig(configPath)
	if err != nil {
		report.add("project_config_drift", StatusWarn, fmt.Sprintf("failed to load current project config: %v", err),
			detail("config_path", configPath),
			detail("stored_hash", active.ConfigHash),
		)
		return
	}
	if current.SourceHash != active.ConfigHash {
		report.add("project_config_drift", StatusWarn, "current areaflow.yaml differs from active AreaFlow config snapshot",
			detail("config_path", configPath),
			detail("stored_hash", active.ConfigHash),
			detail("current_hash", current.SourceHash),
		)
		return
	}
	report.add("project_config_drift", StatusPass, "current areaflow.yaml matches active AreaFlow config snapshot",
		detail("config_path", configPath),
		detail("config_hash", current.SourceHash),
	)
}

func verifyInventoryConsistency(inventory project.ImportInventory, current areamatrix.Snapshot, report *Report) {
	checkCount(report, "version_consistency", inventory.Versions, int64(len(current.Versions)), "workflow version count matches current source")
	checkCount(report, "residual_consistency", inventory.Residuals, int64(len(current.Residuals)), "residual count matches current source")
	checkCount(report, "artifact_consistency", inventory.Artifacts, int64(len(current.Artifacts)), "artifact index count matches current source")
}

func checkCount(report *Report, name string, actual int64, expected int64, passMessage string) {
	if actual != expected {
		report.add(name, StatusFail, "AreaFlow import inventory differs from current source",
			detail("db", actual),
			detail("current", expected),
		)
		return
	}
	report.add(name, StatusPass, passMessage,
		detail("count", actual),
	)
}

func verifyStageCoverage(root string, snapshot areamatrix.Snapshot, report *Report) {
	coverage := map[string]bool{
		"intake":            pathExists(root, "workflow/intake.md"),
		"source_docs":       pathExists(root, "docs"),
		"templates":         pathExists(root, "workflow/templates/README.md"),
		"version_init":      len(snapshot.Versions) > 0,
		"discussion":        hasVersionStage(snapshot, "discussion"),
		"middle_layer":      hasVersionStage(snapshot, "middle-layer"),
		"changes":           hasVersionStage(snapshot, "changes"),
		"plans":             hasVersionStage(snapshot, "plans"),
		"drafts":            hasVersionStage(snapshot, "drafts"),
		"queue":             hasVersionStage(snapshot, "queue"),
		"promotion_preview": hasVersionStage(snapshot, "promotion"),
		"approval":          hasVersionStage(snapshot, "promotion"),
		"execution":         hasVersionStage(snapshot, "execution"),
		"run":               snapshot.TaskSummary.V1ExecutionTotal > 0,
		"projection":        hasVersionStage(snapshot, "projection"),
		"closeout":          hasVersionStage(snapshot, "closeout"),
	}

	missing := []string{}
	covered := []string{}
	for _, stage := range areaMatrixProfileStages() {
		if coverage[stage] {
			covered = append(covered, stage)
		} else {
			missing = append(missing, stage)
		}
	}
	if len(missing) > 0 {
		report.add("stage_coverage", StatusWarn, "AreaMatrix profile has stage coverage gaps",
			detail("covered", strings.Join(covered, ",")),
			detail("missing", strings.Join(missing, ",")),
		)
		return
	}
	report.add("stage_coverage", StatusPass, "AreaMatrix profile stages are covered by current source",
		detail("covered", strings.Join(covered, ",")),
	)
}

func verifyNativeWorkflowDoctor(ctx context.Context, record project.Record, store Store, runner CommandRunner, allowNative bool, report *Report) {
	const command = "./dev workflow doctor"
	permission, err := store.CommandPermission(ctx, record.ID, command)
	if err != nil {
		report.add("native_workflow_doctor", StatusWarn, fmt.Sprintf("native workflow doctor permission check failed: %v", err))
		return
	}
	allowed := permission.CapabilityAllowed && permission.CommandAllowed && !permission.Denied
	reason := nativePermissionReason(permission)
	if allowNative && permission.CommandAllowed && !permission.Denied {
		allowed = true
		reason = "allowed by explicit per-run --allow-native"
	}
	if !allowed {
		report.add("native_workflow_doctor", StatusWarn, "native workflow doctor skipped by command permission gate",
			detail("command", command),
			detail("reason", reason),
			detail("allow_native", allowNative),
		)
		return
	}
	result := runner(ctx, record.RootPath, "./dev", "workflow", "doctor")
	status := StatusPass
	message := "native workflow doctor matched AreaMatrix source"
	if result.ExitCode != 0 {
		status = StatusFail
		message = "native workflow doctor failed"
	}
	report.add("native_workflow_doctor", status, message,
		detail("command", command),
		detail("reason", reason),
		detail("allow_native", allowNative),
		detail("exit_code", result.ExitCode),
		detail("output", trimOutput(result.Output)),
		detail("error", trimOutput(result.Error)),
	)
}

func nativePermissionReason(permission project.CommandPermission) string {
	if permission.Denied {
		return permission.Reason
	}
	if !permission.CapabilityAllowed {
		return "run_commands capability not allowed"
	}
	if !permission.CommandAllowed {
		return "command not allowed"
	}
	return "allowed"
}

func runCommand(ctx context.Context, root string, command string, args ...string) CommandResult {
	cmd := exec.CommandContext(ctx, command, args...)
	cmd.Dir = root
	output, err := cmd.CombinedOutput()
	result := CommandResult{
		Output: string(output),
	}
	if err == nil {
		return result
	}
	if exitErr, ok := err.(*exec.ExitError); ok {
		result.ExitCode = exitErr.ExitCode()
		result.Error = err.Error()
		return result
	}
	result.ExitCode = -1
	result.Error = err.Error()
	return result
}

func trimOutput(value string) string {
	value = strings.TrimSpace(value)
	if len(value) <= 2000 {
		return value
	}
	return value[:2000]
}

func areaMatrixProfileStages() []string {
	return []string{
		"intake",
		"source_docs",
		"templates",
		"version_init",
		"discussion",
		"middle_layer",
		"changes",
		"plans",
		"drafts",
		"queue",
		"promotion_preview",
		"approval",
		"execution",
		"run",
		"projection",
		"closeout",
	}
}

func hasVersionStage(snapshot areamatrix.Snapshot, stage string) bool {
	for _, version := range snapshot.Versions {
		if version.ArtifactCounts[stage] > 0 {
			return true
		}
	}
	return false
}

func pathExists(root string, rel string) bool {
	_, err := os.Stat(filepath.Join(root, filepath.FromSlash(rel)))
	return err == nil
}

func detail(key string, value any) Detail {
	return Detail{Key: key, Value: fmt.Sprint(value)}
}
