package project

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	cfg, err := LoadConfig(filepath.Join("..", "..", "examples", "areamatrix", "areaflow.yaml"))
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if cfg.Project.ID != "areamatrix" {
		t.Fatalf("project id = %q", cfg.Project.ID)
	}
	if !filepath.IsAbs(cfg.SourcePath) {
		t.Fatalf("source path = %q, want absolute path", cfg.SourcePath)
	}
	if cfg.SourceHash == "" {
		t.Fatal("source hash should be populated")
	}
	if cfg.ArtifactStore.Backend != "local" {
		t.Fatalf("artifact backend = %q, want local", cfg.ArtifactStore.Backend)
	}
	if cfg.ArtifactStore.Root == "" {
		t.Fatal("artifact store root should be configured")
	}
	if cfg.Ownership.Mode != "import" {
		t.Fatalf("ownership mode = %q, want import", cfg.Ownership.Mode)
	}
	if cfg.Ownership.SourceOfTruth.Workflow != "project" || cfg.Ownership.SourceOfTruth.StatusSummary != "areaflow" {
		t.Fatalf("unexpected source of truth policy: %+v", cfg.Ownership.SourceOfTruth)
	}
	if cfg.Ownership.Cutover.Enabled || cfg.Ownership.Cutover.NewVersionsOwnedBy != "project" || cfg.Ownership.Cutover.ExecutionOwnedBy != "project" {
		t.Fatalf("unexpected cutover policy: %+v", cfg.Ownership.Cutover)
	}
	if !cfg.Permissions.Capabilities["read_project"] {
		t.Fatal("read_project capability should be enabled")
	}
	if cfg.Permissions.Capabilities["write_code"] {
		t.Fatal("write_code capability should be disabled")
	}
	if cfg.Permissions.Capabilities["write_generated"] {
		t.Fatal("write_generated capability should be disabled")
	}
	if len(cfg.Permissions.ForbiddenPath) == 0 {
		t.Fatal("expected forbidden paths")
	}
	if cfg.Scheduling.Priority != 100 || cfg.Scheduling.MaxParallelTasks != 1 {
		t.Fatalf("unexpected scheduling policy: %+v", cfg.Scheduling)
	}
	if cfg.Scheduling.AgentRole != "local_worker" || cfg.Scheduling.EngineProfile != "codex-cli" {
		t.Fatalf("unexpected scheduling routing fields: %+v", cfg.Scheduling)
	}
	if len(cfg.Scheduling.RequiredCapabilities) != 2 || cfg.Scheduling.RequiredCapabilities[0] != "read_project" {
		t.Fatalf("unexpected scheduling capabilities: %+v", cfg.Scheduling.RequiredCapabilities)
	}
	if cfg.Engines.Default != "codex-cli" || len(cfg.Engines.Profiles) != 2 {
		t.Fatalf("unexpected engine profiles: %+v", cfg.Engines)
	}
	profile, ok := engineProfileByID(cfg.Engines, "codex-cli")
	if !ok {
		t.Fatalf("codex-cli engine profile missing: %+v", cfg.Engines)
	}
	if profile.Enabled || profile.SecretRef != "none" {
		t.Fatalf("unexpected codex-cli engine profile: %+v", profile)
	}
	if profile.ResourceLimits["max_active_leases"] == nil || profile.ResourceLimits["max_queued_tasks"] == nil {
		t.Fatalf("expected codex-cli resource limits: %+v", profile.ResourceLimits)
	}
	if !cfg.StatusExport.Enabled || cfg.StatusExport.Path != ".areaflow/status.json" {
		t.Fatalf("unexpected status export policy: %+v", cfg.StatusExport)
	}
	if cfg.StatusExport.HumanSummary.Enabled || cfg.StatusExport.HumanSummary.Path != "workflow/README.md" {
		t.Fatalf("unexpected human summary policy: %+v", cfg.StatusExport.HumanSummary)
	}
	if cfg.Migration.Strategy != "import_mirror_shadow_cutover_archive" || cfg.Migration.Phase != "import" {
		t.Fatalf("unexpected migration policy: %+v", cfg.Migration)
	}
	if len(cfg.Migration.ImportedVersions) != 2 || cfg.Migration.ImmutableImports[0] != "v1-mvp" {
		t.Fatalf("unexpected migration versions: %+v", cfg.Migration)
	}
}

func TestLoadConfigDefaultsProjectProtocolFields(t *testing.T) {
	path := writeConfig(t, `
version: 1
project:
  id: demo
  name: Demo
  root: /tmp/demo
  adapter: generic
  workflow_profile: standard
artifact_store:
  backend: local
  root: ~/.areaflow/artifacts
status_export:
  enabled: true
`)
	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if cfg.Ownership.Mode != "import" {
		t.Fatalf("ownership mode = %q, want import", cfg.Ownership.Mode)
	}
	if cfg.Ownership.SourceOfTruth.ProductDocs != "project" || cfg.Ownership.SourceOfTruth.StatusSummary != "areaflow" {
		t.Fatalf("unexpected ownership defaults: %+v", cfg.Ownership.SourceOfTruth)
	}
	if cfg.StatusExport.Path != ".areaflow/status.json" {
		t.Fatalf("status export path = %q", cfg.StatusExport.Path)
	}
	if cfg.Migration.Strategy != "import_mirror_shadow_cutover_archive" || cfg.Migration.Phase != "import" {
		t.Fatalf("unexpected migration defaults: %+v", cfg.Migration)
	}
	if cfg.Ownership.Cutover.ExecutionOwnedBy != "project" {
		t.Fatalf("execution owner default = %q, want project", cfg.Ownership.Cutover.ExecutionOwnedBy)
	}
}

func TestLoadConfigAcceptsSplitMigrationPhases(t *testing.T) {
	for _, phase := range []string{"authoring_cutover", "execution_beta", "execution_cutover", "shim_retirement", "cutover"} {
		t.Run(phase, func(t *testing.T) {
			path := writeConfig(t, `
version: 1
project:
  id: demo
  name: Demo
  root: /tmp/demo
  adapter: generic
  workflow_profile: standard
migration:
  phase: `+phase+`
`)
			cfg, err := LoadConfig(path)
			if err != nil {
				t.Fatalf("load config: %v", err)
			}
			if cfg.Migration.Phase != phase {
				t.Fatalf("migration phase = %q, want %q", cfg.Migration.Phase, phase)
			}
		})
	}
}

func TestLoadConfigRejectsUnsupportedProtocolValues(t *testing.T) {
	cases := []struct {
		name    string
		content string
		want    string
	}{
		{
			name: "ownership mode",
			content: `
version: 1
project:
  id: demo
  name: Demo
  root: /tmp/demo
  adapter: generic
  workflow_profile: standard
ownership:
  mode: unknown
`,
			want: "ownership.mode",
		},
		{
			name: "source owner",
			content: `
version: 1
project:
  id: demo
  name: Demo
  root: /tmp/demo
  adapter: generic
  workflow_profile: standard
ownership:
  source_of_truth:
    workflow: somebody-else
`,
			want: "ownership.source_of_truth.workflow",
		},
		{
			name: "migration phase",
			content: `
version: 1
project:
  id: demo
  name: Demo
  root: /tmp/demo
  adapter: generic
  workflow_profile: standard
migration:
  phase: execute
`,
			want: "migration.phase",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := LoadConfig(writeConfig(t, tc.content))
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error = %v, want containing %q", err, tc.want)
			}
		})
	}
}

func TestShouldAllowGeneratedWritePath(t *testing.T) {
	cfg := Config{
		Permissions: Permissions{
			Capabilities: map[string]bool{"write_generated": true},
		},
	}
	for _, path := range []string{".areaflow/generated/status.json", ".areamatrix/generated/summary.json", ".areaflow/generated/**"} {
		if !shouldAllowGeneratedWritePath(cfg, path) {
			t.Fatalf("expected generated write path to be allowed: %s", path)
		}
	}
	for _, path := range []string{".areaflow/status.json", "docs/generated/status.json", ".areaflow/generated"} {
		if shouldAllowGeneratedWritePath(cfg, path) {
			t.Fatalf("expected generated write path to be denied: %s", path)
		}
	}
	cfg.Permissions.Capabilities["write_generated"] = false
	if shouldAllowGeneratedWritePath(cfg, ".areaflow/generated/status.json") {
		t.Fatal("generated write path should require write_generated capability")
	}
}

func writeConfig(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "areaflow.yaml")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	return path
}
