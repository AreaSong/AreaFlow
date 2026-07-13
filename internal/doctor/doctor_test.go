package doctor

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/jackc/pgx/v5"

	"github.com/areasong/areaflow/internal/adapter/areamatrix"
	"github.com/areasong/areaflow/internal/project"
)

type fakeStore struct {
	snapshot          project.Snapshot
	inventory         project.ImportInventory
	config            project.ProjectConfigRecord
	hasConfig         bool
	configErr         error
	snapshotErr       error
	inventoryErr      error
	commandOK         bool
	commandReason     string
	commandPermission project.CommandPermission
}

func (s fakeStore) LatestImportSnapshot(context.Context, int64) (project.Snapshot, error) {
	return s.snapshot, s.snapshotErr
}

func (s fakeStore) ImportInventory(context.Context, int64) (project.ImportInventory, error) {
	return s.inventory, s.inventoryErr
}

func (s fakeStore) ActiveProjectConfig(context.Context, int64) (project.ProjectConfigRecord, bool, error) {
	return s.config, s.hasConfig, s.configErr
}

func (s fakeStore) CanRunCommand(context.Context, int64, string) (bool, string, error) {
	if s.commandReason != "" {
		return s.commandOK, s.commandReason, nil
	}
	return s.commandOK, "allowed", nil
}

func (s fakeStore) CommandPermission(context.Context, int64, string) (project.CommandPermission, error) {
	if s.commandPermission != (project.CommandPermission{}) {
		return s.commandPermission, nil
	}
	return project.CommandPermission{
		CapabilityAllowed: s.commandOK,
		CommandAllowed:    s.commandOK,
		Reason:            s.commandReason,
	}, nil
}

func denyIfRunnerCalled(t *testing.T) CommandRunner {
	t.Helper()
	return func(context.Context, string, string, ...string) CommandResult {
		t.Fatal("native command runner should not be called")
		return CommandResult{}
	}
}

func TestAreaMatrixDoctorPassesForMatchingImport(t *testing.T) {
	root := t.TempDir()
	createProfileFiles(t, root)
	config := createProjectConfig(t, root, "areamatrix")

	current := sampleSnapshot("hash-a")
	store := fakeStore{
		snapshot: project.Snapshot{SourceHash: "hash-a", Summary: map[string]any{}},
		inventory: project.ImportInventory{
			Versions:        1,
			Residuals:       1,
			Artifacts:       1,
			ImportSnapshots: 1,
		},
		commandOK: true,
		config:    config,
		hasConfig: true,
	}

	report, err := AreaMatrixWithLoader(context.Background(), sampleRecord(root), store, fixedLoader(current), fixedRunner(0, "workflow doctor: OK", ""), false)
	if err != nil {
		t.Fatalf("doctor failed: %v", err)
	}
	if report.OverallStatus() != StatusPass {
		t.Fatalf("status = %s, checks = %+v", report.OverallStatus(), report.Checks)
	}
	if !hasCheck(report, "native_workflow_doctor", StatusPass) {
		t.Fatalf("missing native workflow doctor pass: %+v", report.Checks)
	}
	if !hasCheck(report, "project_config_drift", StatusPass) {
		t.Fatalf("missing project config drift pass: %+v", report.Checks)
	}
}

func TestAreaMatrixDoctorFailsOnHashDrift(t *testing.T) {
	root := t.TempDir()
	createProfileFiles(t, root)
	config := createProjectConfig(t, root, "areamatrix")

	current := sampleSnapshot("hash-current")
	store := fakeStore{
		snapshot: project.Snapshot{SourceHash: "hash-imported", Summary: map[string]any{}},
		inventory: project.ImportInventory{
			Versions:        1,
			Residuals:       1,
			Artifacts:       1,
			ImportSnapshots: 1,
		},
		commandOK: true,
		config:    config,
		hasConfig: true,
	}

	report, err := AreaMatrixWithLoader(context.Background(), sampleRecord(root), store, fixedLoader(current), fixedRunner(0, "workflow doctor: OK", ""), false)
	if err != nil {
		t.Fatalf("doctor failed: %v", err)
	}
	if report.OverallStatus() != StatusFail {
		t.Fatalf("status = %s, want fail", report.OverallStatus())
	}
	if !hasCheck(report, "hash_drift", StatusFail) {
		t.Fatalf("missing hash_drift failure: %+v", report.Checks)
	}
}

func TestAreaMatrixDoctorWarnsOnProjectConfigDrift(t *testing.T) {
	root := t.TempDir()
	createProfileFiles(t, root)
	config := createProjectConfig(t, root, "areamatrix")
	if err := os.WriteFile(filepath.Join(root, "areaflow.yaml"), []byte(sampleConfigYAML("changed")), 0o644); err != nil {
		t.Fatalf("rewrite config: %v", err)
	}

	current := sampleSnapshot("hash-a")
	store := fakeStore{
		snapshot: project.Snapshot{SourceHash: "hash-a", Summary: map[string]any{}},
		inventory: project.ImportInventory{
			Versions:        1,
			Residuals:       1,
			Artifacts:       1,
			ImportSnapshots: 1,
		},
		commandOK: true,
		config:    config,
		hasConfig: true,
	}

	report, err := AreaMatrixWithLoader(context.Background(), sampleRecord(root), store, fixedLoader(current), fixedRunner(0, "workflow doctor: OK", ""), false)
	if err != nil {
		t.Fatalf("doctor failed: %v", err)
	}
	if report.OverallStatus() != StatusWarn {
		t.Fatalf("status = %s, want warn", report.OverallStatus())
	}
	if !hasCheck(report, "project_config_drift", StatusWarn) {
		t.Fatalf("missing project config drift warn: %+v", report.Checks)
	}
}

func TestAreaMatrixDoctorSkipsNativeDoctorWhenCommandDenied(t *testing.T) {
	root := t.TempDir()
	createProfileFiles(t, root)
	config := createProjectConfig(t, root, "areamatrix")

	current := sampleSnapshot("hash-a")
	store := fakeStore{
		snapshot: project.Snapshot{SourceHash: "hash-a", Summary: map[string]any{}},
		inventory: project.ImportInventory{
			Versions:        1,
			Residuals:       1,
			Artifacts:       1,
			ImportSnapshots: 1,
		},
		commandPermission: project.CommandPermission{CommandAllowed: true},
		config:            config,
		hasConfig:         true,
	}

	report, err := AreaMatrixWithLoader(context.Background(), sampleRecord(root), store, fixedLoader(current), fixedRunner(0, "", ""), false)
	if err != nil {
		t.Fatalf("doctor failed: %v", err)
	}
	if report.OverallStatus() != StatusWarn {
		t.Fatalf("status = %s, want warn", report.OverallStatus())
	}
	if !hasCheck(report, "native_workflow_doctor", StatusWarn) {
		t.Fatalf("missing native workflow doctor warn: %+v", report.Checks)
	}
}

func TestAreaMatrixDoctorRunsNativeDoctorWithExplicitAllowNative(t *testing.T) {
	root := t.TempDir()
	createProfileFiles(t, root)
	config := createProjectConfig(t, root, "areamatrix")

	current := sampleSnapshot("hash-a")
	store := fakeStore{
		snapshot: project.Snapshot{SourceHash: "hash-a", Summary: map[string]any{}},
		inventory: project.ImportInventory{
			Versions:        1,
			Residuals:       1,
			Artifacts:       1,
			ImportSnapshots: 1,
		},
		commandPermission: project.CommandPermission{CommandAllowed: true},
		config:            config,
		hasConfig:         true,
	}

	report, err := AreaMatrixWithLoader(context.Background(), sampleRecord(root), store, fixedLoader(current), fixedRunner(0, "workflow doctor: OK", ""), true)
	if err != nil {
		t.Fatalf("doctor failed: %v", err)
	}
	if report.OverallStatus() != StatusPass {
		t.Fatalf("status = %s, checks = %+v", report.OverallStatus(), report.Checks)
	}
	if !hasCheck(report, "native_workflow_doctor", StatusPass) {
		t.Fatalf("missing native workflow doctor pass: %+v", report.Checks)
	}
}

func TestAreaMatrixDoctorAllowNativeDoesNotBypassCommandAllowlist(t *testing.T) {
	root := t.TempDir()
	createProfileFiles(t, root)
	config := createProjectConfig(t, root, "areamatrix")

	current := sampleSnapshot("hash-a")
	store := fakeStore{
		snapshot: project.Snapshot{SourceHash: "hash-a", Summary: map[string]any{}},
		inventory: project.ImportInventory{
			Versions:        1,
			Residuals:       1,
			Artifacts:       1,
			ImportSnapshots: 1,
		},
		commandPermission: project.CommandPermission{CapabilityAllowed: false, CommandAllowed: false},
		config:            config,
		hasConfig:         true,
	}

	report, err := AreaMatrixWithLoader(context.Background(), sampleRecord(root), store, fixedLoader(current), denyIfRunnerCalled(t), true)
	if err != nil {
		t.Fatalf("doctor failed: %v", err)
	}
	if report.OverallStatus() != StatusWarn {
		t.Fatalf("status = %s, want warn", report.OverallStatus())
	}
	if !hasCheck(report, "native_workflow_doctor", StatusWarn) {
		t.Fatalf("missing native workflow doctor warn: %+v", report.Checks)
	}
}

func TestAreaMatrixDoctorAllowNativeDoesNotBypassForbiddenCommand(t *testing.T) {
	root := t.TempDir()
	createProfileFiles(t, root)
	config := createProjectConfig(t, root, "areamatrix")

	current := sampleSnapshot("hash-a")
	store := fakeStore{
		snapshot: project.Snapshot{SourceHash: "hash-a", Summary: map[string]any{}},
		inventory: project.ImportInventory{
			Versions:        1,
			Residuals:       1,
			Artifacts:       1,
			ImportSnapshots: 1,
		},
		commandPermission: project.CommandPermission{
			CapabilityAllowed: true,
			CommandAllowed:    true,
			Denied:            true,
			Reason:            "command denied by forbidden command",
		},
		config:    config,
		hasConfig: true,
	}

	report, err := AreaMatrixWithLoader(context.Background(), sampleRecord(root), store, fixedLoader(current), denyIfRunnerCalled(t), true)
	if err != nil {
		t.Fatalf("doctor failed: %v", err)
	}
	if report.OverallStatus() != StatusWarn {
		t.Fatalf("status = %s, want warn", report.OverallStatus())
	}
	if !hasCheck(report, "native_workflow_doctor", StatusWarn) {
		t.Fatalf("missing native workflow doctor warn: %+v", report.Checks)
	}
}

func TestAreaMatrixDoctorReportsMissingImport(t *testing.T) {
	root := t.TempDir()
	createProfileFiles(t, root)
	config := createProjectConfig(t, root, "areamatrix")

	store := fakeStore{
		snapshotErr: pgx.ErrNoRows,
		inventory: project.ImportInventory{
			Versions:  0,
			Residuals: 0,
			Artifacts: 0,
		},
		config:    config,
		hasConfig: true,
	}

	report, err := AreaMatrixWithLoader(context.Background(), sampleRecord(root), store, fixedLoader(sampleSnapshot("hash-a")), fixedRunner(0, "", ""), false)
	if err != nil {
		t.Fatalf("doctor failed: %v", err)
	}
	if report.OverallStatus() != StatusFail {
		t.Fatalf("status = %s, want fail", report.OverallStatus())
	}
	if !hasCheck(report, "import_snapshot", StatusFail) {
		t.Fatalf("missing import_snapshot failure: %+v", report.Checks)
	}
}

func TestAreaMatrixDoctorReturnsInventoryError(t *testing.T) {
	store := fakeStore{inventoryErr: errors.New("db offline")}

	_, err := AreaMatrixWithLoader(context.Background(), sampleRecord(t.TempDir()), store, fixedLoader(sampleSnapshot("hash-a")), fixedRunner(0, "", ""), false)
	if err == nil {
		t.Fatal("expected inventory error")
	}
}

func TestReportSummaryIncludesCountsAndChecks(t *testing.T) {
	report := Report{
		Project: "areamatrix",
		Profile: "areamatrix/v0",
		Checks: []Check{
			{Name: "one", Status: StatusPass, Message: "ok", Details: []Detail{{Key: "a", Value: "b"}}},
			{Name: "two", Status: StatusWarn, Message: "watch"},
			{Name: "three", Status: StatusFail, Message: "bad"},
		},
	}

	summary := report.Summary()
	if summary["overall_status"] != string(StatusFail) {
		t.Fatalf("overall_status = %v", summary["overall_status"])
	}
	counts, ok := summary["counts"].(map[string]int)
	if !ok {
		t.Fatalf("counts has unexpected type: %T", summary["counts"])
	}
	if counts["pass"] != 1 || counts["warn"] != 1 || counts["fail"] != 1 {
		t.Fatalf("unexpected counts: %+v", counts)
	}
	checks, ok := summary["checks"].([]map[string]any)
	if !ok {
		t.Fatalf("checks has unexpected type: %T", summary["checks"])
	}
	if len(checks) != 3 || checks[0]["name"] != "one" {
		t.Fatalf("unexpected checks: %+v", checks)
	}
}

func sampleRecord(root string) project.Record {
	return project.Record{
		ID:              1,
		Key:             "areamatrix",
		Adapter:         "areamatrix",
		WorkflowProfile: "areamatrix/v0",
		RootPath:        root,
	}
}

func sampleSnapshot(hash string) areamatrix.Snapshot {
	return areamatrix.Snapshot{
		StatusSourceHash: hash,
		Versions: []areamatrix.Version{
			{
				Label: "v-template",
				ArtifactCounts: map[string]int{
					"discussion":   1,
					"middle-layer": 1,
					"changes":      1,
					"plans":        1,
					"drafts":       1,
					"queue":        1,
					"promotion":    1,
					"execution":    1,
					"projection":   1,
					"closeout":     1,
				},
			},
		},
		Residuals: []areamatrix.Residual{{Key: "r1"}},
		Artifacts: []areamatrix.Artifact{{SourcePath: "a"}},
		TaskSummary: areamatrix.TaskSummary{
			V1ExecutionTotal: 1,
			V1ExecutionDone:  1,
		},
	}
}

func fixedLoader(snapshot areamatrix.Snapshot) SnapshotLoader {
	return func(string) (areamatrix.Snapshot, error) {
		return snapshot, nil
	}
}

func fixedRunner(exitCode int, output string, stderr string) CommandRunner {
	return func(context.Context, string, string, ...string) CommandResult {
		return CommandResult{
			ExitCode: exitCode,
			Output:   output,
			Error:    stderr,
		}
	}
}

func createProfileFiles(t *testing.T, root string) {
	t.Helper()
	for _, rel := range []string{
		"docs",
		"workflow",
		"workflow/templates",
	} {
		if err := os.MkdirAll(filepath.Join(root, filepath.FromSlash(rel)), 0o755); err != nil {
			t.Fatalf("create %s: %v", rel, err)
		}
	}
	for _, rel := range []string{
		"workflow/intake.md",
		"workflow/templates/README.md",
	} {
		if err := os.WriteFile(filepath.Join(root, filepath.FromSlash(rel)), []byte("ok\n"), 0o644); err != nil {
			t.Fatalf("write %s: %v", rel, err)
		}
	}
}

func createProjectConfig(t *testing.T, root string, projectID string) project.ProjectConfigRecord {
	t.Helper()
	path := filepath.Join(root, "areaflow.yaml")
	if err := os.WriteFile(path, []byte(sampleConfigYAML(projectID)), 0o644); err != nil {
		t.Fatalf("write areaflow.yaml: %v", err)
	}
	cfg, err := project.LoadConfig(path)
	if err != nil {
		t.Fatalf("load areaflow.yaml: %v", err)
	}
	return project.ProjectConfigRecord{
		ID:              1,
		ProjectID:       1,
		ProtocolVersion: cfg.Version,
		ConfigPath:      "areaflow.yaml",
		ConfigHash:      cfg.SourceHash,
		Ownership:       map[string]any{"mode": cfg.Ownership.Mode},
		StatusExport:    map[string]any{"path": cfg.StatusExport.Path},
		Migration:       map[string]any{"phase": cfg.Migration.Phase},
		Active:          true,
	}
}

func sampleConfigYAML(projectID string) string {
	return `version: 1
project:
  id: ` + projectID + `
  name: AreaMatrix
  root: /tmp/AreaMatrix
  kind: product-repo
  adapter: areamatrix
  workflow_profile: areamatrix
artifact_store:
  backend: local
  root: ~/.areaflow/artifacts
status_export:
  enabled: true
`
}

func hasCheck(report Report, name string, status Status) bool {
	for _, check := range report.Checks {
		if check.Name == name && check.Status == status {
			return true
		}
	}
	return false
}
