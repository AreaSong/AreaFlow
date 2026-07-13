package app

import (
	"bytes"
	"context"
	"go/ast"
	"go/parser"
	"go/token"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/areasong/areaflow/internal/project"
)

func TestHelp(t *testing.T) {
	var stdout bytes.Buffer
	cmd := command{stdout: &stdout, stderr: &bytes.Buffer{}}

	if err := cmd.run(context.Background(), []string{"help"}); err != nil {
		t.Fatalf("help failed: %v", err)
	}

	if !strings.Contains(stdout.String(), "areaflow server") {
		t.Fatalf("help output did not include server command: %s", stdout.String())
	}
	if !strings.Contains(stdout.String(), "areaflow service status") {
		t.Fatalf("help output did not include service status command: %s", stdout.String())
	}
	if !strings.Contains(stdout.String(), "areaflow desktop service-control-gate") {
		t.Fatalf("help output did not include desktop service control gate command: %s", stdout.String())
	}
	if !strings.Contains(stdout.String(), "areaflow desktop notification-gate") {
		t.Fatalf("help output did not include desktop notification gate command: %s", stdout.String())
	}
	if !strings.Contains(stdout.String(), "areaflow desktop tray-menu-gate") {
		t.Fatalf("help output did not include desktop tray menu gate command: %s", stdout.String())
	}
	if !strings.Contains(stdout.String(), "areaflow security boundary-readiness") {
		t.Fatalf("help output did not include security boundary readiness command: %s", stdout.String())
	}
	if !strings.Contains(stdout.String(), "areaflow completion audit") {
		t.Fatalf("help output did not include completion audit command: %s", stdout.String())
	}
	if !strings.Contains(stdout.String(), "areaflow completion audit-snapshot record") {
		t.Fatalf("help output did not include completion audit snapshot command: %s", stdout.String())
	}
	if !strings.Contains(stdout.String(), "areaflow completion audit-snapshot readiness") {
		t.Fatalf("help output did not include completion audit snapshot readiness command: %s", stdout.String())
	}
	if !strings.Contains(stdout.String(), "areaflow completion archive-proof record") {
		t.Fatalf("help output did not include archive proof command: %s", stdout.String())
	}
	if !strings.Contains(stdout.String(), "areaflow completion shim-retirement-proof record") {
		t.Fatalf("help output did not include shim retirement proof command: %s", stdout.String())
	}
	if !strings.Contains(stdout.String(), "areaflow completion execution-cutover-proof record") {
		t.Fatalf("help output did not include execution cutover proof command: %s", stdout.String())
	}
	if !strings.Contains(stdout.String(), "areaflow completion validation-proof record") {
		t.Fatalf("help output did not include validation proof command: %s", stdout.String())
	}
	if !strings.Contains(stdout.String(), "areaflow completion source-alignment-proof record") {
		t.Fatalf("help output did not include source alignment proof command: %s", stdout.String())
	}
	if !strings.Contains(stdout.String(), "areaflow completion task-matrix-proof record") {
		t.Fatalf("help output did not include task matrix proof command: %s", stdout.String())
	}
	if !strings.Contains(stdout.String(), "areaflow completion security-closure-proof record") {
		t.Fatalf("help output did not include security closure proof command: %s", stdout.String())
	}
	if !strings.Contains(stdout.String(), "areaflow completion backup-restore-proof record") {
		t.Fatalf("help output did not include backup restore proof command: %s", stdout.String())
	}
	if !strings.Contains(stdout.String(), "areaflow completion release-packaging-proof record") {
		t.Fatalf("help output did not include release packaging proof command: %s", stdout.String())
	}
	if !strings.Contains(stdout.String(), "areaflow completion protected-path-proof record") {
		t.Fatalf("help output did not include protected path proof command: %s", stdout.String())
	}
	if !strings.Contains(stdout.String(), "areaflow ops readiness") {
		t.Fatalf("help output did not include operations readiness command: %s", stdout.String())
	}
	if !strings.Contains(stdout.String(), "areaflow ops migration-ledger-readiness") {
		t.Fatalf("help output did not include migration ledger readiness command: %s", stdout.String())
	}
	if !strings.Contains(stdout.String(), "areaflow ops smoke-proof record") {
		t.Fatalf("help output did not include operations smoke proof command: %s", stdout.String())
	}
	if !strings.Contains(stdout.String(), "areaflow support bundle-preview") {
		t.Fatalf("help output did not include support bundle preview command: %s", stdout.String())
	}
	if !strings.Contains(stdout.String(), "areaflow backup manifest") {
		t.Fatalf("help output did not include backup manifest command: %s", stdout.String())
	}
	if !strings.Contains(stdout.String(), "areaflow backup restore-plan") {
		t.Fatalf("help output did not include backup restore-plan command: %s", stdout.String())
	}
	if !strings.Contains(stdout.String(), "areaflow release readiness") {
		t.Fatalf("help output did not include release readiness command: %s", stdout.String())
	}
	if !strings.Contains(stdout.String(), "areaflow release remediation-plan") {
		t.Fatalf("help output did not include release remediation plan command: %s", stdout.String())
	}
	if !strings.Contains(stdout.String(), "areaflow release acceptance-preview") {
		t.Fatalf("help output did not include release acceptance preview command: %s", stdout.String())
	}
	if !strings.Contains(stdout.String(), "areaflow release acceptance-gate") {
		t.Fatalf("help output did not include release acceptance gate command: %s", stdout.String())
	}
	if !strings.Contains(stdout.String(), "areaflow release exception-doctor") {
		t.Fatalf("help output did not include release exception doctor command: %s", stdout.String())
	}
	if !strings.Contains(stdout.String(), "areaflow release exception-record-preview") {
		t.Fatalf("help output did not include release exception record preview command: %s", stdout.String())
	}
	if !strings.Contains(stdout.String(), "areaflow release exception-schema-preview") {
		t.Fatalf("help output did not include release exception schema preview command: %s", stdout.String())
	}
	if !strings.Contains(stdout.String(), "areaflow release exception-migration-approval-gate") {
		t.Fatalf("help output did not include release exception migration approval gate command: %s", stdout.String())
	}
	if !strings.Contains(stdout.String(), "areaflow release exception-apply-preview") {
		t.Fatalf("help output did not include release exception apply preview command: %s", stdout.String())
	}
	if !strings.Contains(stdout.String(), "areaflow release final-gate") {
		t.Fatalf("help output did not include release final gate command: %s", stdout.String())
	}
	if !strings.Contains(stdout.String(), "areaflow release evidence-bundle") {
		t.Fatalf("help output did not include release evidence bundle command: %s", stdout.String())
	}
	if !strings.Contains(stdout.String(), "areaflow release package-preview") {
		t.Fatalf("help output did not include release package preview command: %s", stdout.String())
	}
	if !strings.Contains(stdout.String(), "areaflow release distribution-preview") {
		t.Fatalf("help output did not include release distribution preview command: %s", stdout.String())
	}
	if !strings.Contains(stdout.String(), "areaflow release publish-gate") {
		t.Fatalf("help output did not include release publish gate command: %s", stdout.String())
	}
	if !strings.Contains(stdout.String(), "areaflow release publish-approval-preview") {
		t.Fatalf("help output did not include release publish approval preview command: %s", stdout.String())
	}
	if !strings.Contains(stdout.String(), "areaflow release rollout-plan-preview") {
		t.Fatalf("help output did not include release rollout plan preview command: %s", stdout.String())
	}
	if !strings.Contains(stdout.String(), "areaflow audit coverage") {
		t.Fatalf("help output did not include audit coverage command: %s", stdout.String())
	}
	if !strings.Contains(stdout.String(), "areaflow permissions doctor") {
		t.Fatalf("help output did not include permissions doctor command: %s", stdout.String())
	}
	if !strings.Contains(stdout.String(), "areaflow artifact integrity") {
		t.Fatalf("help output did not include artifact integrity command: %s", stdout.String())
	}
	if !strings.Contains(stdout.String(), "areaflow conformance check") {
		t.Fatalf("help output did not include conformance check command: %s", stdout.String())
	}
	if !strings.Contains(stdout.String(), "areaflow migrate up") {
		t.Fatalf("help output did not include migrate command: %s", stdout.String())
	}
	if !strings.Contains(stdout.String(), "areaflow project add") {
		t.Fatalf("help output did not include project command: %s", stdout.String())
	}
	if !strings.Contains(stdout.String(), "areaflow project summary") {
		t.Fatalf("help output did not include project summary command: %s", stdout.String())
	}
	if !strings.Contains(stdout.String(), "areaflow project readiness") {
		t.Fatalf("help output did not include project readiness command: %s", stdout.String())
	}
	if !strings.Contains(stdout.String(), "areaflow project generated-write-readiness") {
		t.Fatalf("help output did not include project generated write readiness command: %s", stdout.String())
	}
	if !strings.Contains(stdout.String(), "areaflow project generated-write-apply-beta-gate") {
		t.Fatalf("help output did not include project generated write apply beta gate command: %s", stdout.String())
	}
	if !strings.Contains(stdout.String(), "areaflow project import-diff") {
		t.Fatalf("help output did not include project import-diff command: %s", stdout.String())
	}
	if !strings.Contains(stdout.String(), "areaflow project compatibility") {
		t.Fatalf("help output did not include project compatibility command: %s", stdout.String())
	}
	if !strings.Contains(stdout.String(), "areaflow project shim-preview") {
		t.Fatalf("help output did not include project shim preview command: %s", stdout.String())
	}
	if !strings.Contains(stdout.String(), "areaflow project shim-readiness") {
		t.Fatalf("help output did not include project shim readiness command: %s", stdout.String())
	}
	if !strings.Contains(stdout.String(), "areaflow project shim-authorization") {
		t.Fatalf("help output did not include project shim authorization command: %s", stdout.String())
	}
	if !strings.Contains(stdout.String(), "areaflow project shim-apply areamatrix") {
		t.Fatalf("help output did not include project shim apply command: %s", stdout.String())
	}
	if !strings.Contains(stdout.String(), "areaflow project cutover-apply") {
		t.Fatalf("help output did not include project cutover apply command: %s", stdout.String())
	}
	if !strings.Contains(stdout.String(), "areaflow project import") {
		t.Fatalf("help output did not include project import command: %s", stdout.String())
	}
	if !strings.Contains(stdout.String(), "areaflow project export-status") {
		t.Fatalf("help output did not include project export-status command: %s", stdout.String())
	}
	if !strings.Contains(stdout.String(), "areaflow project status-projections") {
		t.Fatalf("help output did not include project status-projections command: %s", stdout.String())
	}
	if !strings.Contains(stdout.String(), "areaflow project status-projection-authorization") {
		t.Fatalf("help output did not include project status projection authorization command: %s", stdout.String())
	}
	if !strings.Contains(stdout.String(), "areaflow project status-projection-apply-packet") {
		t.Fatalf("help output did not include project status projection apply packet command: %s", stdout.String())
	}
	if !strings.Contains(stdout.String(), "areaflow project status-projection-apply-gate") {
		t.Fatalf("help output did not include project status projection apply gate command: %s", stdout.String())
	}
	if !strings.Contains(stdout.String(), "areaflow project status-projection-apply") {
		t.Fatalf("help output did not include project status projection apply command: %s", stdout.String())
	}
	if !strings.Contains(stdout.String(), "areaflow project doctor") {
		t.Fatalf("help output did not include project doctor command: %s", stdout.String())
	}
	if !strings.Contains(stdout.String(), "areaflow project events") {
		t.Fatalf("help output did not include project events command: %s", stdout.String())
	}
	if !strings.Contains(stdout.String(), "areaflow workflow version create") {
		t.Fatalf("help output did not include workflow version create command: %s", stdout.String())
	}
	if !strings.Contains(stdout.String(), "areaflow workflow profile list") {
		t.Fatalf("help output did not include workflow profile list command: %s", stdout.String())
	}
	if !strings.Contains(stdout.String(), "areaflow workflow profile check") {
		t.Fatalf("help output did not include workflow profile command: %s", stdout.String())
	}
	if !strings.Contains(stdout.String(), "areaflow run execution-gate") {
		t.Fatalf("help output did not include run execution-gate command: %s", stdout.String())
	}
	if !strings.Contains(stdout.String(), "areaflow run execution-plan") {
		t.Fatalf("help output did not include run execution-plan command: %s", stdout.String())
	}
	if !strings.Contains(stdout.String(), "areaflow run project-write-design-gate") {
		t.Fatalf("help output did not include run project-write-design-gate command: %s", stdout.String())
	}
	if !strings.Contains(stdout.String(), "areaflow run managed-generated-write-gate") {
		t.Fatalf("help output did not include run managed-generated-write-gate command: %s", stdout.String())
	}
	if !strings.Contains(stdout.String(), "areaflow run fixture-queue") {
		t.Fatalf("help output did not include run fixture-queue command: %s", stdout.String())
	}
	if !strings.Contains(stdout.String(), "areaflow run read-only-verify-queue") {
		t.Fatalf("help output did not include run read-only-verify-queue command: %s", stdout.String())
	}
	if !strings.Contains(stdout.String(), "areaflow run approved-artifact-write-queue") {
		t.Fatalf("help output did not include run approved-artifact-write-queue command: %s", stdout.String())
	}
	if !strings.Contains(stdout.String(), "areaflow run fixture-project-write-queue") {
		t.Fatalf("help output did not include run fixture-project-write-queue command: %s", stdout.String())
	}
	if !strings.Contains(stdout.String(), "areaflow run managed-generated-write-queue") {
		t.Fatalf("help output did not include run managed-generated-write-queue command: %s", stdout.String())
	}
	if !strings.Contains(stdout.String(), "areaflow worker fixture-execute") {
		t.Fatalf("help output did not include worker fixture-execute command: %s", stdout.String())
	}
	if !strings.Contains(stdout.String(), "areaflow worker read-only-verify") {
		t.Fatalf("help output did not include worker read-only-verify command: %s", stdout.String())
	}
	if !strings.Contains(stdout.String(), "areaflow worker approved-artifact-write") {
		t.Fatalf("help output did not include worker approved-artifact-write command: %s", stdout.String())
	}
	if !strings.Contains(stdout.String(), "areaflow worker fixture-project-write") {
		t.Fatalf("help output did not include worker fixture-project-write command: %s", stdout.String())
	}
	if !strings.Contains(stdout.String(), "areaflow worker managed-generated-write") {
		t.Fatalf("help output did not include worker managed-generated-write command: %s", stdout.String())
	}
	if !strings.Contains(stdout.String(), "areaflow worker register") {
		t.Fatalf("help output did not include worker command: %s", stdout.String())
	}
}

func TestWorkflowProfileListJSON(t *testing.T) {
	var stdout bytes.Buffer
	cmd := command{stdout: &stdout, stderr: &bytes.Buffer{}}

	if err := cmd.run(context.Background(), []string{"workflow", "profile", "list", "--json"}); err != nil {
		t.Fatalf("workflow profile list failed: %v", err)
	}
	out := stdout.String()
	for _, expected := range []string{
		`"profiles": [`,
		`"profile_id": "areamatrix"`,
		`"profile_version": 0`,
		`"stage_count": 16`,
		`"gate_count": 17`,
		`"transition_count": 15`,
	} {
		if !strings.Contains(out, expected) {
			t.Fatalf("profile list output missing %s: %s", expected, out)
		}
	}
}

func TestWorkflowProfileCheckJSON(t *testing.T) {
	var stdout bytes.Buffer
	cmd := command{stdout: &stdout, stderr: &bytes.Buffer{}}

	if err := cmd.run(context.Background(), []string{"workflow", "profile", "check", "areamatrix", "--json"}); err != nil {
		t.Fatalf("workflow profile check failed: %v", err)
	}
	out := stdout.String()
	for _, expected := range []string{
		`"profile_id": "areamatrix"`,
		`"status": "pass"`,
		`"stage_count": 16`,
		`"gate_count": 17`,
		`"transition_count": 15`,
	} {
		if !strings.Contains(out, expected) {
			t.Fatalf("profile check output missing %s: %s", expected, out)
		}
	}
	if !strings.Contains(out, `"sha256": "`) {
		t.Fatalf("profile check output missing sha256: %s", out)
	}
}

func TestUnknownCommand(t *testing.T) {
	cmd := command{stdout: &bytes.Buffer{}, stderr: &bytes.Buffer{}}

	err := cmd.run(context.Background(), []string{"nope"})
	if err == nil {
		t.Fatal("expected unknown command error")
	}
	if !strings.Contains(err.Error(), "unknown command") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestProjectAddRequiresConfig(t *testing.T) {
	cmd := command{stdout: &bytes.Buffer{}, stderr: &bytes.Buffer{}}

	err := cmd.run(context.Background(), []string{"project", "add"})
	if err == nil {
		t.Fatal("expected project add usage error")
	}
	if !strings.Contains(err.Error(), "--config") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestProjectStatusProjectionSubcommandHelpDoesNotRequireDatabase(t *testing.T) {
	cases := []struct {
		name     string
		args     []string
		expected string
	}{
		{
			name:     "authorization",
			args:     []string{"project", "status-projection-authorization", "--help"},
			expected: "usage: areaflow project status-projection-authorization <id>",
		},
		{
			name:     "apply packet",
			args:     []string{"project", "status-projection-apply-packet", "--help"},
			expected: "usage: areaflow project status-projection-apply-packet <id>",
		},
		{
			name:     "apply gate",
			args:     []string{"project", "status-projection-apply-gate", "--help"},
			expected: "usage: areaflow project status-projection-apply-gate <id>",
		},
		{
			name:     "apply",
			args:     []string{"project", "status-projection-apply", "--help"},
			expected: "usage: areaflow project status-projection-apply <id>",
		},
		{
			name:     "shim apply packet",
			args:     []string{"project", "shim-apply-packet", "--help"},
			expected: "usage: areaflow project shim-apply-packet <id>",
		},
		{
			name:     "shim apply gate",
			args:     []string{"project", "shim-apply-gate", "--help"},
			expected: "usage: areaflow project shim-apply-gate <id>",
		},
		{
			name:     "shim apply",
			args:     []string{"project", "shim-apply", "--help"},
			expected: "usage: areaflow project shim-apply <id>",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var stdout bytes.Buffer
			cmd := command{stdout: &stdout, stderr: &bytes.Buffer{}}

			if err := cmd.run(context.Background(), tc.args); err != nil {
				t.Fatalf("project subcommand help failed: %v", err)
			}
			if !strings.Contains(stdout.String(), tc.expected) {
				t.Fatalf("help output missing %q: %s", tc.expected, stdout.String())
			}
		})
	}
}

func TestEventLimit(t *testing.T) {
	limit, err := eventLimit(nil)
	if err != nil {
		t.Fatalf("default limit failed: %v", err)
	}
	if limit != 10 {
		t.Fatalf("limit = %d, want 10", limit)
	}

	limit, err = eventLimit([]string{"--limit", "3"})
	if err != nil {
		t.Fatalf("explicit limit failed: %v", err)
	}
	if limit != 3 {
		t.Fatalf("limit = %d, want 3", limit)
	}

	if _, err := eventLimit([]string{"--limit", "0"}); err == nil {
		t.Fatal("expected non-positive limit error")
	}
	if _, err := eventLimit([]string{"--bad", "3"}); err == nil {
		t.Fatal("expected usage error")
	}
}

func TestOutputJSON(t *testing.T) {
	enabled, err := outputJSON([]string{"--json"})
	if err != nil {
		t.Fatalf("json flag failed: %v", err)
	}
	if !enabled {
		t.Fatal("json flag should be enabled")
	}
	enabled, err = outputJSON(nil)
	if err != nil {
		t.Fatalf("default json flag failed: %v", err)
	}
	if enabled {
		t.Fatal("json flag should be disabled by default")
	}
	if _, err := outputJSON([]string{"--bad"}); err == nil {
		t.Fatal("expected unsupported flag error")
	}
}

func TestExecutionPlanFlags(t *testing.T) {
	flags, err := executionPlanFlags([]string{"--json"})
	if err != nil {
		t.Fatalf("execution plan flags failed: %v", err)
	}
	if !flags.json {
		t.Fatal("json flag should be enabled")
	}
	flags, err = executionPlanFlags(nil)
	if err != nil {
		t.Fatalf("default execution plan flags failed: %v", err)
	}
	if flags.json {
		t.Fatal("json flag should be disabled by default")
	}
	if _, err := executionPlanFlags([]string{"--capability", "write_code"}); err == nil {
		t.Fatal("expected unsupported execution plan flag error")
	}
}

func TestProjectWriteDesignGateFlags(t *testing.T) {
	flags, err := projectWriteDesignGateFlags([]string{"--json"})
	if err != nil {
		t.Fatalf("project write design gate flags failed: %v", err)
	}
	if !flags.json {
		t.Fatal("json flag should be enabled")
	}
	flags, err = projectWriteDesignGateFlags(nil)
	if err != nil {
		t.Fatalf("default project write design gate flags failed: %v", err)
	}
	if flags.json {
		t.Fatal("json flag should be disabled by default")
	}
	if _, err := projectWriteDesignGateFlags([]string{"--capability", "write_code"}); err == nil {
		t.Fatal("expected unsupported project write design gate flag error")
	}
}

func TestManagedGeneratedWriteGateFlags(t *testing.T) {
	flags, err := managedGeneratedWriteGateFlags([]string{"--json"})
	if err != nil {
		t.Fatalf("managed generated write gate flags failed: %v", err)
	}
	if !flags.json {
		t.Fatal("json flag should be enabled")
	}
	flags, err = managedGeneratedWriteGateFlags(nil)
	if err != nil {
		t.Fatalf("default managed generated write gate flags failed: %v", err)
	}
	if flags.json {
		t.Fatal("json flag should be disabled by default")
	}
	if _, err := managedGeneratedWriteGateFlags([]string{"--capability", "write_code"}); err == nil {
		t.Fatal("expected unsupported managed generated write gate flag error")
	}
}

func TestExecutionPlanPreviewToJSON(t *testing.T) {
	generated := time.Date(2026, 7, 2, 6, 0, 0, 0, time.UTC)
	run := project.RunRecord{
		ID:                3,
		ProjectID:         1,
		WorkflowVersionID: 2,
		RunType:           "approved_artifact_write",
		RunKind:           "execution",
		Status:            "queued",
		DryRun:            false,
		StartedAt:         generated,
	}
	preview := project.ExecutionPlanPreview{
		Project: project.Record{
			ID:              1,
			Key:             "areamatrix",
			Name:            "AreaMatrix",
			Kind:            "product-repo",
			Adapter:         "areamatrix",
			WorkflowProfile: "areamatrix",
		},
		Version: project.WorkflowVersion{
			ID:              2,
			DisplayLabel:    "v2",
			VersionKind:     "workflow_version",
			LifecycleStatus: "authored",
			ImportMode:      "authored",
			CreatedAt:       generated,
			UpdatedAt:       generated,
		},
		Run:    run,
		Status: "blocked",
		Mode:   "read_only_execution_plan_preview",
		Gate: project.ExecutionApprovalGate{
			Run:                  run,
			Status:               "pass",
			Mode:                 "read_only_execution_approval_gate",
			RequiredCapabilities: []string{"write_artifacts"},
		},
		Steps: []project.ExecutionPlanStep{
			{
				Key:            "copy",
				AttemptKind:    "copy",
				Status:         "blocked",
				Message:        "copy remains closed",
				Blockers:       []string{"managed_project_write_not_open"},
				ReadsProject:   true,
				WritesProject:  true,
				WritesAreaFlow: true,
				UsesEngine:     true,
				RunsCommands:   true,
			},
			{
				Key:             "approved_artifact_write",
				AttemptKind:     "approved_artifact_write",
				Status:          "ready",
				Message:         "artifact-store-only step is open",
				WritesAreaFlow:  true,
				CreatesAttempt:  true,
				CreatesArtifact: true,
			},
		},
		Blockers:                      []string{"copy: managed_project_write_not_open"},
		ForbiddenActions:              []string{"write_managed_project"},
		ProjectReadAttempted:          false,
		ProjectWriteAttempted:         false,
		ExecutionWriteAttempted:       false,
		AreaFlowArtifactWritten:       false,
		AreaFlowExecutionStateWritten: false,
		EngineCallAttempted:           false,
		CommandsRun:                   false,
		SecretsResolved:               false,
		NetworkUsed:                   false,
		TaskClaimed:                   false,
		WorkerStarted:                 false,
		AttemptCreated:                false,
		ArtifactCreated:               false,
		GeneratedAt:                   generated,
	}
	out := executionPlanPreviewToJSON(preview)
	if out.Status != "blocked" || out.Mode != "read_only_execution_plan_preview" {
		t.Fatalf("unexpected execution plan JSON status: %+v", out)
	}
	if out.ProjectWriteAttempted || out.ExecutionWriteAttempted || out.EngineCallAttempted ||
		out.CommandsRun || out.SecretsResolved || out.NetworkUsed || out.TaskClaimed ||
		out.WorkerStarted || out.AttemptCreated || out.ArtifactCreated {
		t.Fatalf("execution plan JSON should preserve read-only safety facts: %+v", out)
	}
	if len(out.Steps) != 2 || !out.Steps[0].WritesProject || !out.Steps[0].UsesEngine || !out.Steps[0].RunsCommands {
		t.Fatalf("copy step should expose risky unopened work: %+v", out.Steps)
	}
	if out.Steps[1].Status != "ready" || out.Steps[1].WritesProject || !out.Steps[1].WritesAreaFlow || !out.Steps[1].CreatesArtifact {
		t.Fatalf("approved artifact write step should be artifact-only: %+v", out.Steps[1])
	}
}

func TestProjectWriteDesignGateToJSON(t *testing.T) {
	generated := time.Date(2026, 7, 2, 8, 30, 0, 0, time.UTC)
	run := project.RunRecord{
		ID:                3,
		ProjectID:         1,
		WorkflowVersionID: 2,
		RunType:           "approved_artifact_write",
		RunKind:           "execution",
		Status:            "queued",
		DryRun:            false,
		StartedAt:         generated,
	}
	gate := project.ProjectWriteDesignGate{
		Project: project.Record{
			ID:              1,
			Key:             "areamatrix",
			Name:            "AreaMatrix",
			Kind:            "product-repo",
			Adapter:         "areamatrix",
			WorkflowProfile: "areamatrix",
		},
		Version: project.WorkflowVersion{
			ID:              2,
			DisplayLabel:    "v2",
			VersionKind:     "workflow_version",
			LifecycleStatus: "authored",
			ImportMode:      "authored",
			CreatedAt:       generated,
			UpdatedAt:       generated,
		},
		Run: run,
		Gate: project.ExecutionApprovalGate{
			Run:                  run,
			Status:               "pass",
			Mode:                 "read_only_project_write_design_gate_execution_approval",
			RequiredCapabilities: []string{"read_project", "write_artifacts", "write_code"},
		},
		Status:                        "ready",
		Mode:                          "read_only_project_write_design_gate",
		RequiredCapabilities:          []string{"read_project", "write_artifacts", "write_code"},
		WriteSetFields:                []string{"operation", "target_path", "expected_before_sha256", "rollback_plan_artifact_id"},
		UnsupportedOperations:         []string{"delete", "project_root_escape"},
		ApplySequence:                 []string{"project_write_design_gate", "fixture_rollback_drill", "managed_project_generated_only_write"},
		ForbiddenActions:              []string{"write_managed_project", "execute_engine"},
		ProjectWriteApplyOpen:         false,
		ProjectReadAttempted:          false,
		ProjectWriteAttempted:         false,
		ExecutionWriteAttempted:       false,
		AreaFlowArtifactWritten:       false,
		AreaFlowExecutionStateWritten: false,
		EngineCallAttempted:           false,
		CommandsRun:                   false,
		SecretsResolved:               false,
		NetworkUsed:                   false,
		TaskClaimed:                   false,
		WorkerStarted:                 false,
		AttemptCreated:                false,
		ArtifactCreated:               false,
		GeneratedAt:                   generated,
	}
	out := projectWriteDesignGateToJSON(gate)
	if out.Status != "ready" || out.Mode != "read_only_project_write_design_gate" || out.ProjectWriteApplyOpen {
		t.Fatalf("unexpected project write design gate JSON status: %+v", out)
	}
	if out.ProjectReadAttempted || out.ProjectWriteAttempted || out.ExecutionWriteAttempted ||
		out.AreaFlowArtifactWritten || out.AreaFlowExecutionStateWritten || out.EngineCallAttempted ||
		out.CommandsRun || out.SecretsResolved || out.NetworkUsed || out.TaskClaimed || out.WorkerStarted ||
		out.AttemptCreated || out.ArtifactCreated {
		t.Fatalf("project write design gate JSON should preserve read-only safety facts: %+v", out)
	}
	if !stringSliceContains(out.WriteSetFields, "expected_before_sha256") || !stringSliceContains(out.WriteSetFields, "rollback_plan_artifact_id") {
		t.Fatalf("write-set JSON should preserve required safety fields: %+v", out.WriteSetFields)
	}
	if !stringSliceContains(out.UnsupportedOperations, "delete") || !stringSliceContains(out.ApplySequence, "fixture_rollback_drill") {
		t.Fatalf("design gate JSON should preserve destructive-op denial and fixture rollback sequence: %+v", out)
	}
}

func TestManagedGeneratedWriteGateToJSON(t *testing.T) {
	generated := time.Date(2026, 7, 2, 9, 30, 0, 0, time.UTC)
	run := project.RunRecord{
		ID:                3,
		ProjectID:         1,
		WorkflowVersionID: 2,
		RunType:           "fixture_project_write",
		RunKind:           "execution",
		Status:            "rollback_verified",
		DryRun:            false,
		StartedAt:         generated,
	}
	gate := project.ManagedGeneratedWriteGate{
		Project: project.Record{
			ID:              1,
			Key:             "areamatrix",
			Name:            "AreaMatrix",
			Kind:            "product-repo",
			Adapter:         "areamatrix",
			WorkflowProfile: "areamatrix",
		},
		Version: project.WorkflowVersion{
			ID:              2,
			DisplayLabel:    "v2",
			VersionKind:     "workflow_version",
			LifecycleStatus: "authored",
			ImportMode:      "authored",
			CreatedAt:       generated,
			UpdatedAt:       generated,
		},
		Run: run,
		Gate: project.ExecutionApprovalGate{
			Run:                  run,
			Status:               "pass",
			Mode:                 "read_only_managed_generated_write_gate_execution_approval",
			RequiredCapabilities: []string{"read_project", "write_artifacts", "write_generated"},
		},
		Status:                        "ready",
		Mode:                          "read_only_managed_generated_write_gate",
		RequiredCapabilities:          []string{"read_project", "write_artifacts", "write_generated"},
		AllowedGeneratedPrefixes:      []string{".areaflow/generated/", ".areamatrix/generated/"},
		RequiredWriteSetFields:        []string{"operation", "target_path", "generated_only", "rollback_plan_artifact_id"},
		UnsupportedOperations:         []string{"source_write", "workflow_execution_write"},
		ApplySequence:                 []string{"fixture_rollback_drill", "managed_generated_write_gate", "managed_project_generated_only_write"},
		ForbiddenActions:              []string{"write_managed_project", "execute_engine"},
		GeneratedOnlyWriteReady:       true,
		GeneratedOnlyApplyOpen:        false,
		ProjectReadAttempted:          false,
		ProjectWriteAttempted:         false,
		ExecutionWriteAttempted:       false,
		AreaFlowArtifactWritten:       false,
		AreaFlowExecutionStateWritten: false,
		EngineCallAttempted:           false,
		CommandsRun:                   false,
		SecretsResolved:               false,
		NetworkUsed:                   false,
		TaskClaimed:                   false,
		WorkerStarted:                 false,
		LeaseCreated:                  false,
		AttemptCreated:                false,
		ArtifactCreated:               false,
		GeneratedAt:                   generated,
	}
	out := managedGeneratedWriteGateToJSON(gate)
	if out.Status != "ready" || out.Mode != "read_only_managed_generated_write_gate" || !out.GeneratedOnlyWriteReady || out.GeneratedOnlyApplyOpen {
		t.Fatalf("unexpected managed generated write gate JSON status: %+v", out)
	}
	if out.ProjectReadAttempted || out.ProjectWriteAttempted || out.ExecutionWriteAttempted ||
		out.AreaFlowArtifactWritten || out.AreaFlowExecutionStateWritten || out.EngineCallAttempted ||
		out.CommandsRun || out.SecretsResolved || out.NetworkUsed || out.TaskClaimed || out.WorkerStarted ||
		out.LeaseCreated || out.AttemptCreated || out.ArtifactCreated {
		t.Fatalf("managed generated write gate JSON should preserve read-only safety facts: %+v", out)
	}
	if !stringSliceContains(out.AllowedGeneratedPrefixes, ".areaflow/generated/") || !stringSliceContains(out.RequiredWriteSetFields, "generated_only") {
		t.Fatalf("generated-only JSON should preserve required safety fields: %+v", out)
	}
	if !stringSliceContains(out.UnsupportedOperations, "source_write") || !stringSliceContains(out.ApplySequence, "managed_project_generated_only_write") {
		t.Fatalf("managed generated write gate JSON should preserve source-write denial and apply sequence: %+v", out)
	}
}

func TestBundleFlags(t *testing.T) {
	flags, err := bundleFlags([]string{"--json", "--limit", "3"})
	if err != nil {
		t.Fatalf("bundle flags failed: %v", err)
	}
	if !flags.json || flags.limit != 3 {
		t.Fatalf("unexpected bundle flags: %+v", flags)
	}

	flags, err = bundleFlags(nil)
	if err != nil {
		t.Fatalf("default bundle flags failed: %v", err)
	}
	if flags.json || flags.limit != 10 {
		t.Fatalf("unexpected default bundle flags: %+v", flags)
	}

	if _, err := bundleFlags([]string{"--limit", "0"}); err == nil {
		t.Fatal("expected invalid bundle limit error")
	}
	if _, err := bundleFlags([]string{"--bad"}); err == nil {
		t.Fatal("expected unsupported bundle flag error")
	}
}

func TestCutoverReadinessFlags(t *testing.T) {
	flags, err := cutoverReadinessFlags([]string{"--version", "v2", "--json", "--limit", "3"})
	if err != nil {
		t.Fatalf("cutover readiness flags failed: %v", err)
	}
	if flags.version != "v2" || !flags.json || flags.limit != 3 {
		t.Fatalf("unexpected cutover readiness flags: %+v", flags)
	}
	if _, err := cutoverReadinessFlags([]string{"--json"}); err == nil {
		t.Fatal("expected missing version error")
	}
	if _, err := cutoverReadinessFlags([]string{"--version", "v2", "--limit", "0"}); err == nil {
		t.Fatal("expected invalid cutover readiness limit error")
	}
}

func TestCutoverApplyFlags(t *testing.T) {
	flags, err := cutoverApplyFlags([]string{
		"--version", "v2",
		"--json",
		"--idempotency-key", "cutover-key",
		"--actor", "local-user",
		"--reason", "apply cutover",
		"--mode", "authoring_cutover",
	})
	if err != nil {
		t.Fatalf("cutover apply flags failed: %v", err)
	}
	if flags.version != "v2" || !flags.json || flags.idempotencyKey != "cutover-key" || flags.actor != "local-user" || flags.reason != "apply cutover" || flags.mode != "authoring_cutover" {
		t.Fatalf("unexpected cutover apply flags: %+v", flags)
	}
	if _, err := cutoverApplyFlags([]string{"--json"}); err == nil {
		t.Fatal("expected missing version error")
	}
	if _, err := cutoverApplyFlags([]string{"--version"}); err == nil {
		t.Fatal("expected missing version value error")
	}
	if _, err := cutoverApplyFlags([]string{"--bad"}); err == nil {
		t.Fatal("expected unsupported cutover apply flag error")
	}
}

func TestDoctorFlags(t *testing.T) {
	flags, err := doctorFlags([]string{"--allow-native", "--json", "--idempotency-key", "doctor-key"})
	if err != nil {
		t.Fatalf("doctor flags failed: %v", err)
	}
	if !flags.allowNative || !flags.json || flags.idempotencyKey != "doctor-key" {
		t.Fatalf("unexpected doctor flags: %+v", flags)
	}
	flags, err = doctorFlags(nil)
	if err != nil {
		t.Fatalf("default doctor flags failed: %v", err)
	}
	if flags.allowNative || flags.json || flags.idempotencyKey != "" {
		t.Fatalf("unexpected default doctor flags: %+v", flags)
	}
	if _, err := doctorFlags([]string{"--idempotency-key"}); err == nil {
		t.Fatal("expected missing idempotency key error")
	}
	if _, err := doctorFlags([]string{"--bad"}); err == nil {
		t.Fatal("expected unsupported doctor flag error")
	}
}

func TestImportFlags(t *testing.T) {
	flags, err := importFlags([]string{
		"--json",
		"--idempotency-key", "import-key",
		"--actor", "local-user",
		"--reason", "fixture import",
	})
	if err != nil {
		t.Fatalf("import flags failed: %v", err)
	}
	if !flags.json || flags.idempotencyKey != "import-key" || flags.actor != "local-user" || flags.reason != "fixture import" {
		t.Fatalf("unexpected import flags: %+v", flags)
	}
	flags, err = importFlags(nil)
	if err != nil {
		t.Fatalf("default import flags failed: %v", err)
	}
	if flags.json || flags.idempotencyKey != "" || flags.actor != "" || flags.reason != "" {
		t.Fatalf("unexpected default import flags: %+v", flags)
	}
	if _, err := importFlags([]string{"--idempotency-key"}); err == nil {
		t.Fatal("expected missing idempotency key error")
	}
	if _, err := importFlags([]string{"--bad"}); err == nil {
		t.Fatal("expected unsupported import flag error")
	}
}

func TestWorkflowVersionCreateFlags(t *testing.T) {
	flags, err := workflowVersionCreateFlags([]string{
		"--json",
		"--idempotency-key", "key-1",
		"--actor", "local-user",
		"--reason", "create v2",
	})
	if err != nil {
		t.Fatalf("workflow version create flags failed: %v", err)
	}
	if !flags.json {
		t.Fatal("json flag should be enabled")
	}
	if flags.idempotencyKey != "key-1" || flags.actor != "local-user" || flags.reason != "create v2" {
		t.Fatalf("unexpected flags: %+v", flags)
	}

	if _, err := workflowVersionCreateFlags([]string{"--idempotency-key"}); err == nil {
		t.Fatal("expected missing idempotency key error")
	}
	if _, err := workflowVersionCreateFlags([]string{"--bad"}); err == nil {
		t.Fatal("expected unsupported flag error")
	}
}

func TestWorkflowItemReadyFlags(t *testing.T) {
	flags, err := workflowItemReadyFlags([]string{
		"--json",
		"--stage", "promotion_preview",
		"--item-type", "promotion_preview",
		"--actor", "tester",
		"--reason", "ready path",
	})
	if err != nil {
		t.Fatalf("workflow item ready flags failed: %v", err)
	}
	if !flags.json || flags.stage != "promotion_preview" || flags.itemType != "promotion_preview" {
		t.Fatalf("unexpected ready flags: %+v", flags)
	}
	if flags.actor != "tester" || flags.reason != "ready path" {
		t.Fatalf("unexpected ready metadata flags: %+v", flags)
	}
	if _, err := workflowItemReadyFlags([]string{"--stage", "queue"}); err == nil {
		t.Fatal("expected missing item type error")
	}
	if _, err := workflowItemReadyFlags([]string{"--bad"}); err == nil {
		t.Fatal("expected bad flag error")
	}
}

func TestStageSkeletonFlags(t *testing.T) {
	flags, err := stageSkeletonFlags([]string{"--json", "--actor", "local-user", "--reason", "ensure"})
	if err != nil {
		t.Fatalf("stage skeleton flags failed: %v", err)
	}
	if !flags.json || flags.actor != "local-user" || flags.reason != "ensure" {
		t.Fatalf("unexpected flags: %+v", flags)
	}
	if _, err := stageSkeletonFlags([]string{"--actor"}); err == nil {
		t.Fatal("expected missing actor error")
	}
	if _, err := stageSkeletonFlags([]string{"--bad"}); err == nil {
		t.Fatal("expected unsupported flag error")
	}
}

func TestGateFlags(t *testing.T) {
	flags, err := gateFlags([]string{"--json", "--actor", "local-user", "--reason", "check"})
	if err != nil {
		t.Fatalf("gate flags failed: %v", err)
	}
	if !flags.json || flags.actor != "local-user" || flags.reason != "check" {
		t.Fatalf("unexpected flags: %+v", flags)
	}
	if _, err := gateFlags([]string{"--reason"}); err == nil {
		t.Fatal("expected missing reason error")
	}
	if _, err := gateFlags([]string{"--bad"}); err == nil {
		t.Fatal("expected unsupported flag error")
	}
}

func TestGateListFlags(t *testing.T) {
	flags, err := gateListFlags([]string{"--json", "--limit", "3"})
	if err != nil {
		t.Fatalf("gate list flags failed: %v", err)
	}
	if !flags.json || flags.limit != 3 {
		t.Fatalf("unexpected flags: %+v", flags)
	}
	if _, err := gateListFlags([]string{"--limit", "0"}); err == nil {
		t.Fatal("expected invalid limit error")
	}
	if _, err := gateListFlags([]string{"--bad"}); err == nil {
		t.Fatal("expected unsupported flag error")
	}
}

func TestTransitionPreviewFlags(t *testing.T) {
	flags, err := transitionPreviewFlags([]string{
		"--json",
		"--from", "promotion_preview",
		"--to", "approval",
		"--actor", "local-user",
		"--reason", "preview",
	})
	if err != nil {
		t.Fatalf("transition preview flags failed: %v", err)
	}
	if !flags.json || flags.fromStage != "promotion_preview" || flags.toStage != "approval" {
		t.Fatalf("unexpected transition flags: %+v", flags)
	}
	if flags.actor != "local-user" || flags.reason != "preview" {
		t.Fatalf("unexpected transition actor/reason: %+v", flags)
	}
	if _, err := transitionPreviewFlags([]string{"--from"}); err == nil {
		t.Fatal("expected missing from stage error")
	}
	if _, err := transitionPreviewFlags([]string{"--bad"}); err == nil {
		t.Fatal("expected unsupported flag error")
	}
}

func TestApprovalRecordFlags(t *testing.T) {
	flags, err := approvalRecordFlags([]string{
		"--json",
		"--decision", "rejected",
		"--transition-preview-id", "4",
		"--kind", "workflow_transition",
		"--risk-level", "high",
		"--idempotency-key", "approval-4",
		"--actor", "local-user",
		"--reason", "blocked",
	})
	if err != nil {
		t.Fatalf("approval flags failed: %v", err)
	}
	if !flags.json || flags.decision != "rejected" || flags.transitionPreviewID != 4 {
		t.Fatalf("unexpected approval flags: %+v", flags)
	}
	if flags.kind != "workflow_transition" || flags.riskLevel != "high" || flags.actor != "local-user" || flags.reason != "blocked" {
		t.Fatalf("unexpected approval metadata flags: %+v", flags)
	}
	if flags.idempotencyKey != "approval-4" {
		t.Fatalf("idempotency key = %q, want approval-4", flags.idempotencyKey)
	}
	if _, err := approvalRecordFlags([]string{"--transition-preview-id", "0"}); err == nil {
		t.Fatal("expected invalid transition preview id error")
	}
	if _, err := approvalRecordFlags([]string{"--bad"}); err == nil {
		t.Fatal("expected unsupported flag error")
	}
}

func TestRunnerPreviewFlags(t *testing.T) {
	flags, err := runnerPreviewFlags([]string{
		"--json",
		"--actor", "local-user",
		"--reason", "preview",
		"--risk-level", "medium",
		"--risk-policy", "pause",
		"--idempotency-key", "key-1",
	})
	if err != nil {
		t.Fatalf("runner preview flags failed: %v", err)
	}
	if !flags.json || flags.actor != "local-user" || flags.reason != "preview" {
		t.Fatalf("unexpected runner preview flags: %+v", flags)
	}
	if flags.riskLevel != "medium" || flags.riskPolicy != "pause" || flags.idempotencyKey != "key-1" {
		t.Fatalf("unexpected runner preview metadata flags: %+v", flags)
	}
	if _, err := runnerPreviewFlags([]string{"--actor"}); err == nil {
		t.Fatal("expected missing actor value error")
	}
	if _, err := runnerPreviewFlags([]string{"--bad"}); err == nil {
		t.Fatal("expected unsupported runner preview flag error")
	}
}

func TestFixtureExecutionQueueFlags(t *testing.T) {
	flags, err := fixtureExecutionQueueFlags([]string{"--json", "--actor", "local-user", "--reason", "queue", "--idempotency-key", "fixture-queue-key"})
	if err != nil {
		t.Fatalf("fixture execution queue flags failed: %v", err)
	}
	if !flags.json || flags.actor != "local-user" || flags.reason != "queue" || flags.idempotencyKey != "fixture-queue-key" {
		t.Fatalf("unexpected fixture execution queue flags: %+v", flags)
	}
	if _, err := fixtureExecutionQueueFlags([]string{"--actor"}); err == nil {
		t.Fatal("expected missing actor value error")
	}
	if _, err := fixtureExecutionQueueFlags([]string{"--bad"}); err == nil {
		t.Fatal("expected unsupported fixture execution queue flag error")
	}
}

func TestReadOnlyVerifyQueueFlags(t *testing.T) {
	flags, err := readOnlyVerifyQueueFlags([]string{
		"--json",
		"--target-path", "docs/README.md",
		"--actor", "local-user",
		"--reason", "queue verify",
		"--idempotency-key", "read-only-queue-key",
	})
	if err != nil {
		t.Fatalf("read-only verify queue flags failed: %v", err)
	}
	if !flags.json || flags.targetPath != "docs/README.md" || flags.actor != "local-user" || flags.reason != "queue verify" || flags.idempotencyKey != "read-only-queue-key" {
		t.Fatalf("unexpected read-only verify queue flags: %+v", flags)
	}
	if _, err := readOnlyVerifyQueueFlags([]string{"--target-path"}); err == nil {
		t.Fatal("expected missing target path value error")
	}
	if _, err := readOnlyVerifyQueueFlags([]string{"--json"}); err == nil {
		t.Fatal("expected missing target path error")
	}
	if _, err := readOnlyVerifyQueueFlags([]string{"--bad"}); err == nil {
		t.Fatal("expected unsupported read-only verify queue flag error")
	}
}

func TestApprovedArtifactWriteQueueFlags(t *testing.T) {
	flags, err := approvedArtifactWriteQueueFlags([]string{
		"--json",
		"--artifact-label", "approval-note",
		"--actor", "local-user",
		"--reason", "write artifact",
		"--idempotency-key", "approved-artifact-queue-key",
	})
	if err != nil {
		t.Fatalf("approved artifact write queue flags failed: %v", err)
	}
	if !flags.json || flags.artifactLabel != "approval-note" || flags.actor != "local-user" || flags.reason != "write artifact" || flags.idempotencyKey != "approved-artifact-queue-key" {
		t.Fatalf("unexpected approved artifact write queue flags: %+v", flags)
	}
	if _, err := approvedArtifactWriteQueueFlags([]string{"--artifact-label"}); err == nil {
		t.Fatal("expected missing artifact label value error")
	}
	if _, err := approvedArtifactWriteQueueFlags([]string{"--bad"}); err == nil {
		t.Fatal("expected unsupported approved artifact write queue flag error")
	}
}

func TestFixtureProjectWriteQueueFlags(t *testing.T) {
	flags, err := fixtureProjectWriteQueueFlags([]string{
		"--json",
		"--target-path", "fixtures/input.txt",
		"--content", "after content",
		"--expected-before-sha256", "before123",
		"--expected-before-size", "12",
		"--actor", "local-user",
		"--reason", "queue fixture write",
		"--idempotency-key", "fixture-project-write-queue-key",
	})
	if err != nil {
		t.Fatalf("fixture project write queue flags failed: %v", err)
	}
	if !flags.json ||
		flags.targetPath != "fixtures/input.txt" ||
		flags.content != "after content" ||
		flags.expectedBeforeSHA256 != "before123" ||
		flags.expectedBeforeSize != 12 ||
		flags.actor != "local-user" ||
		flags.reason != "queue fixture write" ||
		flags.idempotencyKey != "fixture-project-write-queue-key" {
		t.Fatalf("unexpected fixture project write queue flags: %+v", flags)
	}
	if _, err := fixtureProjectWriteQueueFlags([]string{"--target-path"}); err == nil {
		t.Fatal("expected missing target path value error")
	}
	if _, err := fixtureProjectWriteQueueFlags([]string{"--target-path", "fixtures/input.txt", "--expected-before-size", "bad", "--expected-before-sha256", "before123"}); err == nil {
		t.Fatal("expected invalid expected before size error")
	}
	if _, err := fixtureProjectWriteQueueFlags([]string{"--json"}); err == nil {
		t.Fatal("expected missing required fixture project write queue fields error")
	}
	if _, err := fixtureProjectWriteQueueFlags([]string{"--bad"}); err == nil {
		t.Fatal("expected unsupported fixture project write queue flag error")
	}
}

func TestManagedGeneratedWriteQueueFlags(t *testing.T) {
	flags, err := managedGeneratedWriteQueueFlags([]string{
		"--json",
		"--target-path", ".areaflow/generated/status.json",
		"--content", "after content",
		"--expected-before-sha256", "before123",
		"--expected-before-size", "12",
		"--actor", "local-user",
		"--reason", "queue managed generated write",
		"--idempotency-key", "managed-generated-write-queue-key",
	})
	if err != nil {
		t.Fatalf("managed generated write queue flags failed: %v", err)
	}
	if !flags.json ||
		flags.targetPath != ".areaflow/generated/status.json" ||
		flags.content != "after content" ||
		flags.expectedBeforeSHA256 != "before123" ||
		flags.expectedBeforeSize != 12 ||
		flags.actor != "local-user" ||
		flags.reason != "queue managed generated write" ||
		flags.idempotencyKey != "managed-generated-write-queue-key" {
		t.Fatalf("unexpected managed generated write queue flags: %+v", flags)
	}
	if _, err := managedGeneratedWriteQueueFlags([]string{"--target-path"}); err == nil {
		t.Fatal("expected missing target path value error")
	}
	if _, err := managedGeneratedWriteQueueFlags([]string{"--target-path", ".areaflow/generated/status.json", "--expected-before-size", "bad", "--expected-before-sha256", "before123"}); err == nil {
		t.Fatal("expected invalid expected before size error")
	}
	if _, err := managedGeneratedWriteQueueFlags([]string{"--json"}); err == nil {
		t.Fatal("expected missing required managed generated write queue fields error")
	}
	if _, err := managedGeneratedWriteQueueFlags([]string{"--bad"}); err == nil {
		t.Fatal("expected unsupported managed generated write queue flag error")
	}
}

func TestExecutionGateFlags(t *testing.T) {
	flags, err := executionGateFlags([]string{"--json", "--capability", "read_project", "--capability", "run_commands"})
	if err != nil {
		t.Fatalf("execution gate flags failed: %v", err)
	}
	if !flags.json || len(flags.capabilities) != 2 || flags.capabilities[0] != "read_project" || flags.capabilities[1] != "run_commands" {
		t.Fatalf("unexpected execution gate flags: %+v", flags)
	}
	if _, err := executionGateFlags([]string{"--capability"}); err == nil {
		t.Fatal("expected missing capability value error")
	}
	if _, err := executionGateFlags([]string{"--bad"}); err == nil {
		t.Fatal("expected unsupported execution gate flag error")
	}
}

func TestWorkerFlags(t *testing.T) {
	register, err := workerRegisterFlags([]string{
		"--json",
		"--worker-key", "local-1",
		"--worker-type", "local_host",
		"--hostname", "dev-host",
		"--pid", "123",
		"--capability", "read_project",
		"--capability", "write_artifacts",
		"--heartbeat-interval", "15",
		"--lease-timeout", "120",
		"--idempotency-key", "register-key",
		"--actor", "local-user",
		"--reason", "register",
	})
	if err != nil {
		t.Fatalf("worker register flags failed: %v", err)
	}
	if !register.json || register.workerKey != "local-1" || register.workerType != "local_host" {
		t.Fatalf("unexpected worker register flags: %+v", register)
	}
	if register.pid != 123 || register.heartbeatIntervalSeconds != 15 || register.leaseTimeoutSeconds != 120 {
		t.Fatalf("unexpected worker register numeric flags: %+v", register)
	}
	if len(register.capabilities) != 2 || register.capabilities[1] != "write_artifacts" {
		t.Fatalf("unexpected worker capabilities: %+v", register)
	}
	if register.idempotencyKey != "register-key" {
		t.Fatalf("unexpected worker register idempotency key: %+v", register)
	}
	if _, err := workerRegisterFlags([]string{"--pid", "nope"}); err == nil {
		t.Fatal("expected invalid worker pid error")
	}

	heartbeat, err := workerHeartbeatFlags([]string{"--json", "--status", "draining", "--idempotency-key", "heartbeat-key", "--actor", "local-user", "--reason", "drain"})
	if err != nil {
		t.Fatalf("worker heartbeat flags failed: %v", err)
	}
	if !heartbeat.json || heartbeat.status != "draining" || heartbeat.reason != "drain" || heartbeat.idempotencyKey != "heartbeat-key" {
		t.Fatalf("unexpected worker heartbeat flags: %+v", heartbeat)
	}
	if _, err := workerHeartbeatFlags([]string{"--status"}); err == nil {
		t.Fatal("expected missing worker heartbeat status error")
	}

	list, err := workerListFlags([]string{"--json", "--limit", "3"})
	if err != nil {
		t.Fatalf("worker list flags failed: %v", err)
	}
	if !list.json || list.limit != 3 {
		t.Fatalf("unexpected worker list flags: %+v", list)
	}
	if _, err := workerListFlags([]string{"--limit", "0"}); err == nil {
		t.Fatal("expected invalid worker list limit error")
	}
}

func TestLeaseFlags(t *testing.T) {
	acquire, err := leaseAcquireFlags([]string{
		"--json",
		"--run-task-id", "4",
		"--lease-kind", "run_task",
		"--capability", "read_project",
		"--lease-timeout", "120",
		"--recover-expired",
		"--idempotency-key", "lease-acquire-4",
		"--actor", "local-user",
		"--reason", "claim",
	})
	if err != nil {
		t.Fatalf("lease acquire flags failed: %v", err)
	}
	if !acquire.json || acquire.runTaskID != 4 || acquire.leaseKind != "run_task" || !acquire.recoverExpired || acquire.idempotencyKey != "lease-acquire-4" {
		t.Fatalf("unexpected lease acquire flags: %+v", acquire)
	}
	if len(acquire.capabilities) != 1 || acquire.capabilities[0] != "read_project" {
		t.Fatalf("unexpected lease acquire capabilities: %+v", acquire)
	}
	if _, err := leaseAcquireFlags([]string{"--json"}); err == nil {
		t.Fatal("expected missing run task id error")
	}

	release, err := leaseReleaseFlags([]string{"--json", "--lease-id", "9", "--status", "completed", "--idempotency-key", "lease-release-9"})
	if err != nil {
		t.Fatalf("lease release flags failed: %v", err)
	}
	if !release.json || release.leaseID != 9 || release.status != "completed" || release.idempotencyKey != "lease-release-9" {
		t.Fatalf("unexpected lease release flags: %+v", release)
	}
	if _, err := leaseReleaseFlags([]string{"--lease-id", "0"}); err == nil {
		t.Fatal("expected invalid lease id error")
	}

	recover, err := leaseRecoverFlags([]string{"--json", "--limit", "3", "--idempotency-key", "lease-recover-3", "--reason", "sweep"})
	if err != nil {
		t.Fatalf("lease recover flags failed: %v", err)
	}
	if !recover.json || recover.limit != 3 || recover.idempotencyKey != "lease-recover-3" || recover.reason != "sweep" {
		t.Fatalf("unexpected lease recover flags: %+v", recover)
	}
	if _, err := leaseRecoverFlags([]string{"--limit", "0"}); err == nil {
		t.Fatal("expected invalid lease recover limit error")
	}
}

func TestWorkerRunOnceFlags(t *testing.T) {
	flags, err := workerRunOnceFlags([]string{
		"--json",
		"--run-id", "42",
		"--capability", "read_project",
		"--lease-timeout", "120",
		"--actor", "local-user",
		"--reason", "tick",
	})
	if err != nil {
		t.Fatalf("worker run-once flags failed: %v", err)
	}
	if !flags.json || flags.leaseTimeoutSeconds != 120 || flags.reason != "tick" {
		t.Fatalf("unexpected worker run-once flags: %+v", flags)
	}
	if flags.runID != 42 {
		t.Fatalf("run id = %d, want 42", flags.runID)
	}
	if len(flags.capabilities) != 1 || flags.capabilities[0] != "read_project" {
		t.Fatalf("unexpected worker run-once capabilities: %+v", flags)
	}
	if _, err := workerRunOnceFlags([]string{"--lease-timeout", "0"}); err == nil {
		t.Fatal("expected invalid worker run-once lease timeout error")
	}
	if _, err := workerRunOnceFlags([]string{"--run-id", "0"}); err == nil {
		t.Fatal("expected invalid worker run-once run id error")
	}
}

func TestFixtureExecutionFlags(t *testing.T) {
	flags, err := fixtureExecutionFlags([]string{
		"--json",
		"--run-id", "42",
		"--capability", "read_project",
		"--lease-timeout", "120",
		"--idempotency-key", "fixture-exec-key",
		"--actor", "local-user",
		"--reason", "fixture",
	})
	if err != nil {
		t.Fatalf("fixture execution flags failed: %v", err)
	}
	if !flags.json || flags.runID != 42 || flags.leaseTimeoutSeconds != 120 || flags.idempotencyKey != "fixture-exec-key" {
		t.Fatalf("unexpected fixture execution flags: %+v", flags)
	}
	if len(flags.capabilities) != 1 || flags.capabilities[0] != "read_project" {
		t.Fatalf("unexpected fixture execution capabilities: %+v", flags)
	}
	if flags.actor != "local-user" || flags.reason != "fixture" {
		t.Fatalf("unexpected fixture execution actor/reason: %+v", flags)
	}
	if _, err := fixtureExecutionFlags([]string{"--json"}); err == nil {
		t.Fatal("expected missing run id error")
	}
	if _, err := fixtureExecutionFlags([]string{"--run-id", "0"}); err == nil {
		t.Fatal("expected invalid run id error")
	}
	if _, err := fixtureExecutionFlags([]string{"--bad"}); err == nil {
		t.Fatal("expected unsupported fixture execution flag error")
	}
}

func TestReadOnlyVerifyFlags(t *testing.T) {
	flags, err := readOnlyVerifyFlags([]string{
		"--json",
		"--run-id", "42",
		"--capability", "read_project",
		"--lease-timeout", "120",
		"--idempotency-key", "read-only-exec-key",
		"--actor", "local-user",
		"--reason", "verify",
	})
	if err != nil {
		t.Fatalf("read-only verify flags failed: %v", err)
	}
	if !flags.json || flags.runID != 42 || flags.leaseTimeoutSeconds != 120 || flags.idempotencyKey != "read-only-exec-key" {
		t.Fatalf("unexpected read-only verify flags: %+v", flags)
	}
	if len(flags.capabilities) != 1 || flags.capabilities[0] != "read_project" {
		t.Fatalf("unexpected read-only verify capabilities: %+v", flags)
	}
	if flags.actor != "local-user" || flags.reason != "verify" {
		t.Fatalf("unexpected read-only verify actor/reason: %+v", flags)
	}
	if _, err := readOnlyVerifyFlags([]string{"--json"}); err == nil {
		t.Fatal("expected missing run id error")
	}
	if _, err := readOnlyVerifyFlags([]string{"--run-id", "0"}); err == nil {
		t.Fatal("expected invalid run id error")
	}
	if _, err := readOnlyVerifyFlags([]string{"--bad"}); err == nil {
		t.Fatal("expected unsupported read-only verify flag error")
	}
}

func TestApprovedArtifactWriteFlags(t *testing.T) {
	flags, err := approvedArtifactWriteFlags([]string{
		"--json",
		"--run-id", "42",
		"--capability", "write_artifacts",
		"--lease-timeout", "120",
		"--idempotency-key", "approved-artifact-write-key",
		"--actor", "local-user",
		"--reason", "write artifact",
	})
	if err != nil {
		t.Fatalf("approved artifact write flags failed: %v", err)
	}
	if !flags.json || flags.runID != 42 || flags.leaseTimeoutSeconds != 120 || flags.idempotencyKey != "approved-artifact-write-key" {
		t.Fatalf("unexpected approved artifact write flags: %+v", flags)
	}
	if len(flags.capabilities) != 1 || flags.capabilities[0] != "write_artifacts" {
		t.Fatalf("unexpected approved artifact write capabilities: %+v", flags)
	}
	if flags.actor != "local-user" || flags.reason != "write artifact" {
		t.Fatalf("unexpected approved artifact write actor/reason: %+v", flags)
	}
	if _, err := approvedArtifactWriteFlags([]string{"--json"}); err == nil {
		t.Fatal("expected missing run id error")
	}
	if _, err := approvedArtifactWriteFlags([]string{"--run-id", "0"}); err == nil {
		t.Fatal("expected invalid run id error")
	}
	if _, err := approvedArtifactWriteFlags([]string{"--bad"}); err == nil {
		t.Fatal("expected unsupported approved artifact write flag error")
	}
}

func TestFixtureProjectWriteFlags(t *testing.T) {
	flags, err := fixtureProjectWriteFlags([]string{
		"--json",
		"--run-id", "42",
		"--capability", "read_project",
		"--capability", "write_artifacts",
		"--capability", "write_code",
		"--lease-timeout", "120",
		"--idempotency-key", "fixture-project-write-key",
		"--actor", "local-user",
		"--reason", "write fixture project",
	})
	if err != nil {
		t.Fatalf("fixture project write flags failed: %v", err)
	}
	if !flags.json || flags.runID != 42 || flags.leaseTimeoutSeconds != 120 || flags.idempotencyKey != "fixture-project-write-key" {
		t.Fatalf("unexpected fixture project write flags: %+v", flags)
	}
	if len(flags.capabilities) != 3 || flags.capabilities[0] != "read_project" || flags.capabilities[1] != "write_artifacts" || flags.capabilities[2] != "write_code" {
		t.Fatalf("unexpected fixture project write capabilities: %+v", flags)
	}
	if flags.actor != "local-user" || flags.reason != "write fixture project" {
		t.Fatalf("unexpected fixture project write actor/reason: %+v", flags)
	}
	if _, err := fixtureProjectWriteFlags([]string{"--json"}); err == nil {
		t.Fatal("expected missing run id error")
	}
	if _, err := fixtureProjectWriteFlags([]string{"--run-id", "0"}); err == nil {
		t.Fatal("expected invalid run id error")
	}
	if _, err := fixtureProjectWriteFlags([]string{"--bad"}); err == nil {
		t.Fatal("expected unsupported fixture project write flag error")
	}
}

func TestManagedGeneratedWriteFlags(t *testing.T) {
	flags, err := managedGeneratedWriteFlags([]string{
		"--json",
		"--run-id", "42",
		"--capability", "read_project",
		"--capability", "write_artifacts",
		"--capability", "write_generated",
		"--lease-timeout", "120",
		"--idempotency-key", "managed-generated-write-key",
		"--actor", "local-user",
		"--reason", "write managed generated",
	})
	if err != nil {
		t.Fatalf("managed generated write flags failed: %v", err)
	}
	if !flags.json || flags.runID != 42 || flags.leaseTimeoutSeconds != 120 || flags.idempotencyKey != "managed-generated-write-key" {
		t.Fatalf("unexpected managed generated write flags: %+v", flags)
	}
	if len(flags.capabilities) != 3 || flags.capabilities[0] != "read_project" || flags.capabilities[1] != "write_artifacts" || flags.capabilities[2] != "write_generated" {
		t.Fatalf("unexpected managed generated write capabilities: %+v", flags)
	}
	if flags.actor != "local-user" || flags.reason != "write managed generated" {
		t.Fatalf("unexpected managed generated write actor/reason: %+v", flags)
	}
	if _, err := managedGeneratedWriteFlags([]string{"--json"}); err == nil {
		t.Fatal("expected missing run id error")
	}
	if _, err := managedGeneratedWriteFlags([]string{"--run-id", "0"}); err == nil {
		t.Fatal("expected invalid run id error")
	}
	if _, err := managedGeneratedWriteFlags([]string{"--bad"}); err == nil {
		t.Fatal("expected unsupported managed generated write flag error")
	}
}

func TestServiceStatusFlags(t *testing.T) {
	flags, err := serviceStatusFlags([]string{"--json", "--web-url", "http://127.0.0.1:5175"})
	if err != nil {
		t.Fatalf("service status flags failed: %v", err)
	}
	if !flags.json || flags.webURL != "http://127.0.0.1:5175" {
		t.Fatalf("unexpected service status flags: %+v", flags)
	}
	if _, err := serviceStatusFlags([]string{"--web-url"}); err == nil {
		t.Fatal("expected missing web url error")
	}
}

func TestAuditCoverageFlags(t *testing.T) {
	flags, err := auditCoverageFlags([]string{"--json", "--project", "areamatrix"})
	if err != nil {
		t.Fatalf("audit coverage flags failed: %v", err)
	}
	if !flags.json || flags.projectKey != "areamatrix" {
		t.Fatalf("unexpected audit coverage flags: %+v", flags)
	}
	if _, err := auditCoverageFlags([]string{"--project"}); err == nil {
		t.Fatal("expected missing project key error")
	}
	if _, err := auditCoverageFlags([]string{"--project", " "}); err == nil {
		t.Fatal("expected blank project key error")
	}
}

func TestLocalServiceStatusToJSON(t *testing.T) {
	created := time.Date(2026, 6, 30, 8, 0, 0, 0, time.UTC)
	out := localServiceStatusToJSON(project.LocalServiceStatus{
		Status: "ready",
		Mode:   "local_service",
		API: project.LocalServiceComponentStatus{
			Status:  "ready",
			Message: "AreaFlow API is available",
		},
		Database: project.LocalServiceComponentStatus{
			Status:  "ready",
			Message: "PostgreSQL connection is healthy",
		},
		WorkerPool: project.LocalServiceWorkerPoolStatus{
			Status:             "warn",
			Message:            "worker pool has recovery items",
			TotalProjects:      1,
			TotalWorkers:       2,
			TotalOnlineWorkers: 1,
			TotalQueuedTasks:   3,
			TotalNeedsRecovery: 1,
		},
		Dashboard: project.LocalServiceDashboardStatus{
			URL:     "http://127.0.0.1:5174",
			APIURL:  "http://127.0.0.1:3847/api/v1",
			Status:  "ready",
			Message: "dashboard should use AreaFlow API as source of truth",
		},
		Capabilities:     []string{"observe_api", "open_web_dashboard"},
		ForbiddenActions: []string{"maintain_second_database"},
		GeneratedAt:      created,
	})

	if out.Status != "ready" || out.Mode != "local_service" {
		t.Fatalf("unexpected service status json: %+v", out)
	}
	if out.WorkerPool.Status != "warn" || out.WorkerPool.TotalQueuedTasks != 3 {
		t.Fatalf("unexpected worker pool json: %+v", out.WorkerPool)
	}
	if out.Dashboard.APIURL != "http://127.0.0.1:3847/api/v1" || out.GeneratedAt == "" {
		t.Fatalf("unexpected dashboard json: %+v", out)
	}
	if len(out.Capabilities) != 2 || len(out.ForbiddenActions) != 1 {
		t.Fatalf("unexpected service guardrails json: %+v", out)
	}
}

func TestDesktopServiceControlGateToJSON(t *testing.T) {
	created := time.Date(2026, 7, 4, 14, 0, 0, 0, time.UTC)
	out := desktopServiceControlGateToJSON(project.BuildDesktopServiceControlGate(project.DesktopServiceControlGateOptions{
		GeneratedAt: created,
	}))

	if out.Status != "blocked" || out.Mode != "read_only_desktop_service_control_gate" {
		t.Fatalf("unexpected desktop service control gate json: %+v", out)
	}
	if len(out.Actions) < 5 || out.GeneratedAt == "" {
		t.Fatalf("desktop service control gate missing action matrix: %+v", out)
	}
	if out.ProcessControlAttempted || out.CommandCreated || out.AuditEventWritten || out.WorkflowExecutionStarted {
		t.Fatalf("desktop service control gate opened forbidden capability: %+v", out)
	}
	seenStart := false
	for _, action := range out.Actions {
		if action.Key == "start_service" {
			seenStart = true
			if action.Status != "blocked" || action.DefaultUIState != "disabled" {
				t.Fatalf("start_service should remain blocked/disabled: %+v", action)
			}
			if !containsString(action.Blockers, "desktop_service_control_not_open") {
				t.Fatalf("start_service missing blocker: %+v", action)
			}
		}
	}
	if !seenStart {
		t.Fatalf("desktop service control gate missing start_service action: %+v", out.Actions)
	}
}

func TestDesktopNotificationGateToJSON(t *testing.T) {
	created := time.Date(2026, 7, 4, 15, 0, 0, 0, time.UTC)
	out := desktopNotificationGateToJSON(project.BuildDesktopNotificationGate(project.DesktopNotificationGateOptions{
		GeneratedAt: created,
	}))

	if out.Status != "blocked" || out.Mode != "read_only_desktop_notification_gate" {
		t.Fatalf("unexpected desktop notification gate json: %+v", out)
	}
	if len(out.Actions) < 4 || out.GeneratedAt == "" {
		t.Fatalf("desktop notification gate missing action matrix: %+v", out)
	}
	if out.EventStreamOpened || out.NotificationRequested || out.CommandCreated || out.AuditEventWritten || out.WorkflowExecutionStarted {
		t.Fatalf("desktop notification gate opened forbidden capability: %+v", out)
	}
	seenEnable := false
	for _, action := range out.Actions {
		if action.Key == "enable_system_notifications" {
			seenEnable = true
			if action.Status != "blocked" || action.DefaultUIState != "disabled" {
				t.Fatalf("enable_system_notifications should remain blocked/disabled: %+v", action)
			}
			if !containsString(action.Blockers, "notification_permission_flow_not_implemented") {
				t.Fatalf("enable_system_notifications missing blocker: %+v", action)
			}
		}
	}
	if !seenEnable {
		t.Fatalf("desktop notification gate missing enable_system_notifications action: %+v", out.Actions)
	}
}

func TestDesktopTrayMenuGateToJSON(t *testing.T) {
	created := time.Date(2026, 7, 4, 15, 30, 0, 0, time.UTC)
	out := desktopTrayMenuGateToJSON(project.BuildDesktopTrayMenuGate(project.DesktopTrayMenuGateOptions{
		GeneratedAt: created,
	}))

	if out.Status != "blocked" || out.Mode != "read_only_desktop_tray_menu_gate" {
		t.Fatalf("unexpected desktop tray menu gate json: %+v", out)
	}
	if len(out.Actions) < 6 || out.GeneratedAt == "" {
		t.Fatalf("desktop tray menu gate missing action matrix: %+v", out)
	}
	if out.TrayMenuCreated || out.OSIntegrationRequested || out.ServiceControlAttempted || out.NotificationRequested || out.CommandCreated || out.WorkflowExecutionStarted {
		t.Fatalf("desktop tray menu gate opened forbidden capability: %+v", out)
	}
	seenStart := false
	for _, action := range out.Actions {
		if action.Key == "start_service" {
			seenStart = true
			if action.Status != "blocked" || action.DefaultUIState != "disabled" {
				t.Fatalf("tray start_service should remain blocked/disabled: %+v", action)
			}
			if !containsString(action.Blockers, "service_control_gate_blocked") {
				t.Fatalf("tray start_service missing blocker: %+v", action)
			}
		}
	}
	if !seenStart {
		t.Fatalf("desktop tray menu gate missing start_service action: %+v", out.Actions)
	}
}

func TestSecurityBoundaryReadinessToJSON(t *testing.T) {
	created := time.Date(2026, 7, 3, 10, 0, 0, 0, time.UTC)
	out := securityBoundaryReadinessToJSON(project.BuildSecurityBoundaryReadiness(project.SecurityBoundaryReadinessOptions{GeneratedAt: created}))

	if out.Status != "ready" || out.Mode != "read_only_security_boundary_readiness" {
		t.Fatalf("unexpected security boundary readiness json: %+v", out)
	}
	if out.SecretResolveOpen || out.RemoteWorkerCredentialsOpen || out.AuthorizationChanged ||
		out.APITokenIssuanceOpen || out.TeamPermissionEnforcementOpen || out.ExternalAPICallOpen {
		t.Fatalf("security boundary readiness opened forbidden capability: %+v", out)
	}
	if out.GeneratedAt == "" || len(out.Items) < 8 {
		t.Fatalf("security boundary readiness missing evidence shape: %+v", out)
	}

	seenSecretResolve := false
	seenTokenLifecycle := false
	seenResolveForbidden := false
	for _, item := range out.Items {
		if item.Key == "secret_resolve" {
			seenSecretResolve = true
		}
		if item.Key == "api_token_lifecycle" {
			seenTokenLifecycle = true
		}
	}
	for _, action := range out.ForbiddenActions {
		if action == "resolve_secret_plaintext" {
			seenResolveForbidden = true
		}
	}
	if !seenSecretResolve || !seenTokenLifecycle || !seenResolveForbidden {
		t.Fatalf("security boundary readiness missing closed-boundary evidence: %+v", out)
	}
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func assertReal100Guardrail(t *testing.T, status, scope string, blockers []string, wantScope string, wantBlockers []string) {
	t.Helper()
	if status != project.Real100StatusBlocked || scope != wantScope || !reflect.DeepEqual(blockers, wantBlockers) {
		t.Fatalf("unexpected real 100 guardrail: status=%q scope=%q blockers=%v", status, scope, blockers)
	}
}

func assertReal100BreakdownHasKey(t *testing.T, items []project.Real100BreakdownItem, key string) {
	t.Helper()
	for _, item := range items {
		if item.Key == key {
			return
		}
	}
	t.Fatalf("missing real 100 breakdown key %q in %+v", key, items)
}

func TestCompletionAuditToJSON(t *testing.T) {
	created := time.Date(2026, 7, 3, 12, 0, 0, 0, time.UTC)
	out := completionAuditToJSON(project.BuildCompletionAudit(project.CompletionAuditOptions{GeneratedAt: created}, project.CompletionAuditParts{
		ReleaseFinalGate:          &project.ReleaseFinalGate{Status: "blocked", Mode: "read_only_release_final_gate"},
		SecurityBoundaryReadiness: ptrSecurityReadiness(project.BuildSecurityBoundaryReadiness(project.SecurityBoundaryReadinessOptions{GeneratedAt: created})),
		LocalServiceStatus:        &project.LocalServiceStatus{Status: "ready", Mode: "local_service"},
	}))

	if out.Status != "blocked" || out.Mode != "read_only_completion_audit" || out.Scope != "v1.0" {
		t.Fatalf("unexpected completion audit json: %+v", out)
	}
	if out.ReleaseFinalGateStatus != "incomplete" || out.AreaMatrixDogfoodStatus != "incomplete" ||
		out.ProtectedPathProofStatus != "blocked" {
		t.Fatalf("unexpected completion aggregate statuses: %+v", out)
	}
	assertReal100Guardrail(
		t,
		out.Real100Status,
		out.ReadinessScope,
		out.Real100Blockers,
		project.CompletionAuditReadinessScope,
		[]string{
			"real_areamatrix_read_only_shim_not_landed",
			"real_areamatrix_execution_cutover_not_proven",
			"real_areamatrix_archive_not_proven",
			"real_areamatrix_shim_retirement_not_proven",
			"release_candidate_snapshot_not_ready",
			"package_a_status_projection_not_applied",
		},
	)
	assertReal100BreakdownHasKey(t, out.Real100Breakdown.NeedsExactAuthorization, "package_a_exact_authorization")
	assertReal100BreakdownHasKey(t, out.Real100Breakdown.NeedsRealAreaMatrixWrite, "package_a_status_projection_apply")
	assertReal100BreakdownHasKey(t, out.Real100Breakdown.AreaFlowOnlyCanContinue, "E1_design_source_alignment")
	if !out.SafetyFacts["read_only"] ||
		out.SafetyFacts["release_package_created"] ||
		out.SafetyFacts["publish_attempted"] ||
		out.SafetyFacts["restore_apply_attempted"] ||
		out.SafetyFacts["secret_resolved"] ||
		out.SafetyFacts["remote_worker_credentials_issued"] ||
		out.SafetyFacts["area_matrix_protected_paths_touched"] {
		t.Fatalf("unexpected completion safety facts: %+v", out.SafetyFacts)
	}
	seenDogfood := false
	seenProtectedPaths := false
	for _, item := range out.Items {
		if item.Key == "E4_areamatrix_dogfood_completion" && item.Status == "blocked" {
			seenDogfood = true
		}
		if item.Key == "E9_areamatrix_protected_path_proof" && item.Status == "blocked" {
			seenProtectedPaths = true
		}
	}
	if !seenDogfood || !seenProtectedPaths || out.GeneratedAt == "" {
		t.Fatalf("completion audit json missing expected items: %+v", out)
	}
}

func TestCompletionAuditSnapshotToJSON(t *testing.T) {
	out := completionAuditSnapshotToJSON(project.CompletionAuditSnapshot{
		Project:                         project.Record{ID: 7, Key: "areamatrix"},
		Status:                          "recorded",
		Decision:                        "allowed",
		Message:                         "completion audit snapshot recorded",
		AuditStatus:                     "complete",
		AuditScope:                      "v1.0",
		AuditHash:                       "audit-hash",
		ReleaseCandidateLabel:           "v1.0-rc1",
		EvidenceClass:                   "fixture",
		EvidenceURI:                     "local:completion-audit",
		ProofEventIDs:                   map[string]int64{"E1.latest_source_alignment_proof_event_id": 10},
		EventID:                         11,
		AuditEventID:                    12,
		IdempotencyKey:                  "snapshot-key",
		Created:                         true,
		ProjectWriteAttempted:           false,
		ExecutionWriteAttempted:         false,
		ReleasePackageCreated:           false,
		PublishAttempted:                false,
		RestoreApplyAttempted:           false,
		SecretResolved:                  false,
		RemoteWorkerCredentialsIssued:   false,
		AreaMatrixProtectedPathsTouched: false,
		CommandsRun:                     false,
		SmokeRunAttempted:               false,
		WorkerStarted:                   false,
		Metadata:                        map[string]any{"summary": "release candidate evidence"},
	})

	if out.Project.Key != "areamatrix" || out.AuditStatus != "complete" ||
		out.ReleaseCandidateLabel != "v1.0-rc1" || out.EvidenceClass != "fixture" || out.AuditHash != "audit-hash" {
		t.Fatalf("unexpected completion audit snapshot json: %+v", out)
	}
	assertReal100Guardrail(
		t,
		out.Real100Status,
		out.ReadinessScope,
		out.Real100Blockers,
		project.CompletionAuditReadinessScope,
		project.Real100CompletionAuditBlockers(),
	)
	if len(out.ProofEventIDs) != 1 || out.ProofEventIDs["E1.latest_source_alignment_proof_event_id"] != 10 {
		t.Fatalf("completion audit snapshot missing proof ids: %+v", out)
	}
	if out.ProjectWriteAttempted || out.ExecutionWriteAttempted || out.ReleasePackageCreated ||
		out.PublishAttempted || out.RestoreApplyAttempted || out.SecretResolved ||
		out.RemoteWorkerCredentialsIssued || out.AreaMatrixProtectedPathsTouched ||
		out.CommandsRun || out.SmokeRunAttempted || out.WorkerStarted {
		t.Fatalf("completion audit snapshot opened unsafe facts: %+v", out)
	}
}

func TestCompletionAuditSnapshotReadinessToJSON(t *testing.T) {
	readiness := project.CompletionAuditSnapshotReadiness{
		Project:       project.Record{ID: 7, Key: "areamatrix"},
		Status:        "blocked",
		Message:       "release candidate completion audit snapshot is not ready",
		HasSnapshot:   true,
		RequiredClass: "release_candidate",
		BundleHash:    "bundle-hash-1",
		Latest: project.CompletionAuditSnapshot{
			Project:               project.Record{ID: 7, Key: "areamatrix"},
			Status:                "recorded",
			AuditStatus:           "complete",
			AuditScope:            "v1.0",
			AuditHash:             "fixture-hash",
			ReleaseCandidateLabel: "v1.0-fixture",
			EvidenceClass:         "fixture",
			EvidenceURI:           "scripts/smoke-completion-audit-full-proof.sh",
		},
		Items: []project.ReadinessItem{
			{
				Key:     "completion_audit_snapshot_fixture_only",
				Status:  "blocked",
				Message: "Latest completion audit snapshot is fixture evidence, not release_candidate evidence",
				Metadata: map[string]any{
					"fixture_snapshot":           true,
					"release_candidate_snapshot": false,
				},
			},
		},
		SafetyFacts: map[string]bool{"read_only": true, "project_write_attempted": false},
	}

	out := completionAuditSnapshotReadinessToJSON(readiness)
	if out.Project.Key != "areamatrix" || out.Status != "blocked" || !out.HasSnapshot ||
		out.RequiredClass != "release_candidate" || out.BundleHash != "bundle-hash-1" ||
		out.Latest.EvidenceClass != "fixture" {
		t.Fatalf("unexpected snapshot readiness json: %+v", out)
	}
	assertReal100Guardrail(
		t,
		out.Real100Status,
		out.ReadinessScope,
		out.Real100Blockers,
		project.CompletionAuditReadinessScope,
		project.Real100CompletionAuditBlockers(),
	)
	assertReal100Guardrail(
		t,
		out.Latest.Real100Status,
		out.Latest.ReadinessScope,
		out.Latest.Real100Blockers,
		project.CompletionAuditReadinessScope,
		project.Real100CompletionAuditBlockers(),
	)
	if len(out.Items) != 1 || out.Items[0].Key != "completion_audit_snapshot_fixture_only" ||
		out.Items[0].Metadata["release_candidate_snapshot"] != false {
		t.Fatalf("snapshot readiness item missing fixture blocker: %+v", out.Items)
	}
	if len(out.Gaps) != 1 || out.Gaps[0].Key != "completion_audit_snapshot_fixture_only" ||
		out.Gaps[0].Category != "snapshot" ||
		!containsString(out.Gaps[0].Blockers, "completion_audit_snapshot_fixture_only") {
		t.Fatalf("snapshot readiness gaps missing fixture blocker: %+v", out.Gaps)
	}
	if out.Closure.Ready ||
		out.Closure.ReadyForReleaseCandidateClosure ||
		out.Closure.SnapshotStatus != "fixture_only" ||
		out.Closure.Snapshot.Status != "fixture_only" ||
		out.Closure.Snapshot.Ready ||
		out.Closure.RequiredClass != "release_candidate" ||
		out.Closure.RequiredEvidenceClass != "release_candidate" ||
		out.Closure.GapCount != 1 ||
		!containsString(out.Closure.Blockers, "completion_audit_snapshot_fixture_only") {
		t.Fatalf("snapshot readiness closure missing fixture blocker: %+v", out.Closure)
	}
	if !out.SafetyFacts["read_only"] || out.SafetyFacts["project_write_attempted"] {
		t.Fatalf("snapshot readiness safety facts changed: %+v", out.SafetyFacts)
	}
}

func TestSupportBundlePreviewToJSON(t *testing.T) {
	created := time.Date(2026, 7, 3, 13, 0, 0, 0, time.UTC)
	out := supportBundlePreviewToJSON(project.BuildSupportBundlePreview(project.BackupManifest{
		Status:       "ready",
		ManifestHash: "backup-hash",
		Projects: []project.BackupProjectManifest{
			{Project: project.Record{Key: "areamatrix", Name: "AreaMatrix", RootPath: "/tmp/areamatrix"}},
		},
	}, project.AuditCoverage{Status: "warn", TotalAuditEvents: 3}, project.SupportBundlePreviewOptions{GeneratedAt: created}))

	if out.Status != "ready" || out.Mode != "metadata_only_support_bundle_preview" || out.Scope == "" {
		t.Fatalf("unexpected support bundle preview json: %+v", out)
	}
	if len(out.Projects) != 1 || out.Projects[0].Key != "areamatrix" || len(out.PathReferences) != 1 {
		t.Fatalf("unexpected support bundle project refs: %+v", out)
	}
	if !out.SafetyFacts["read_only"] || !out.SafetyFacts["metadata_only"] ||
		out.SafetyFacts["export_open"] ||
		out.SafetyFacts["secret_values_included"] ||
		out.SafetyFacts["raw_artifact_contents_included"] ||
		out.SafetyFacts["area_matrix_protected_paths_touched"] {
		t.Fatalf("support bundle preview opened unsafe fact: %+v", out.SafetyFacts)
	}
	if !stringSliceContains(out.ForbiddenActions, "export_support_bundle") ||
		!stringSliceContains(out.ForbiddenActions, "read_secret_values") ||
		out.GeneratedAt == "" {
		t.Fatalf("support bundle preview missing guardrails: %+v", out)
	}
}

func TestMigrationLedgerReadinessToJSON(t *testing.T) {
	created := time.Date(2026, 7, 3, 13, 30, 0, 0, time.UTC)
	out := migrationLedgerReadinessToJSON(project.MigrationLedgerReadiness{
		Status:                               "needs_attention",
		Mode:                                 "read_only_migration_ledger_readiness",
		Entries:                              []project.MigrationLedgerEntry{{Name: "000001_v0_1_core.sql", Applied: true, Status: "ready", RequiredEvidence: []string{"embedded migration exists"}}},
		AppliedCount:                         1,
		SchemaMigrationsTablePresent:         true,
		FullLedgerTablePresent:               false,
		PreflightApplyVerifyRemediationReady: false,
		ForbiddenActions:                     []string{"apply_migration", "write_migration_ledger"},
		SafetyFacts: map[string]bool{
			"read_only":                           true,
			"migration_apply_attempted":           false,
			"database_write_attempted":            false,
			"area_matrix_protected_paths_touched": false,
		},
		GeneratedAt: created,
	})

	if out.Status != "needs_attention" || out.Mode != "read_only_migration_ledger_readiness" {
		t.Fatalf("unexpected migration ledger readiness json: %+v", out)
	}
	if out.FullLedgerTablePresent || out.PreflightApplyVerifyRemediationReady || len(out.Entries) != 1 {
		t.Fatalf("unexpected migration ledger proof state: %+v", out)
	}
	if !out.SafetyFacts["read_only"] || out.SafetyFacts["migration_apply_attempted"] ||
		out.SafetyFacts["database_write_attempted"] || out.SafetyFacts["area_matrix_protected_paths_touched"] {
		t.Fatalf("migration ledger readiness opened unsafe fact: %+v", out.SafetyFacts)
	}
	if !stringSliceContains(out.ForbiddenActions, "apply_migration") || out.GeneratedAt == "" {
		t.Fatalf("migration ledger readiness missing guardrails: %+v", out)
	}
}

func TestOperationsReadinessToJSON(t *testing.T) {
	created := time.Date(2026, 7, 3, 14, 0, 0, 0, time.UTC)
	support := project.BuildSupportBundlePreview(project.BackupManifest{Status: "ready", ManifestHash: "backup-hash"}, project.AuditCoverage{Status: "warn"}, project.SupportBundlePreviewOptions{GeneratedAt: created})
	ledger := project.MigrationLedgerReadiness{
		Status:                               "needs_attention",
		Mode:                                 "read_only_migration_ledger_readiness",
		AppliedCount:                         1,
		SchemaMigrationsTablePresent:         true,
		FullLedgerTablePresent:               false,
		PreflightApplyVerifyRemediationReady: false,
		SafetyFacts:                          map[string]bool{"read_only": true, "database_write_attempted": false},
		GeneratedAt:                          created,
	}
	out := operationsReadinessToJSON(project.BuildOperationsReadiness(
		project.LocalServiceStatus{Status: "ready", Mode: "local_service", GeneratedAt: created},
		support,
		ledger,
		project.OperationsReadinessOptions{GeneratedAt: created},
	))

	if out.Status != "needs_attention" || out.Mode != "read_only_operations_readiness" ||
		out.TelemetryDefault != "local_only" || out.ManagedOpsStatus != "deferred_v1x" {
		t.Fatalf("unexpected operations readiness json: %+v", out)
	}
	if !out.SafetyFacts["read_only"] ||
		out.SafetyFacts["support_bundle_exported"] ||
		out.SafetyFacts["remote_telemetry_enabled"] ||
		out.SafetyFacts["managed_upgrade_attempted"] ||
		out.SafetyFacts["area_matrix_protected_paths_touched"] {
		t.Fatalf("operations readiness opened unsafe fact: %+v", out.SafetyFacts)
	}
	seenBootstrap := false
	seenDeferred := false
	for _, item := range out.Items {
		if item.Key == "install_migrate_start_register_smoke" && item.Status == "needs_attention" {
			seenBootstrap = true
		}
		if item.Key == "managed_ops_deferred" && item.Status == "deferred" {
			seenDeferred = true
		}
	}
	if !seenBootstrap || !seenDeferred || out.GeneratedAt == "" {
		t.Fatalf("operations readiness missing expected items: %+v", out)
	}
}

func TestOperationsSmokeProofToJSON(t *testing.T) {
	out := operationsSmokeProofToJSON(project.OperationsSmokeProof{
		Project:                         project.Record{ID: 7, Key: "areamatrix-fixture"},
		ProofKey:                        "v1_stable_fixture_smoke",
		Status:                          "recorded",
		EvidenceStatus:                  "pass",
		Decision:                        "allowed",
		Message:                         "operations smoke proof recorded",
		EventID:                         11,
		AuditEventID:                    12,
		IdempotencyKey:                  "ops-proof-key",
		Created:                         true,
		ProjectWriteAttempted:           false,
		ExecutionWriteAttempted:         false,
		EngineCallAttempted:             false,
		ServiceProcessControlAttempted:  false,
		SupportBundleExported:           false,
		MigrationApplyAttempted:         false,
		RemoteTelemetryEnabled:          false,
		AreaMatrixProtectedPathsTouched: false,
		RecordCommandRunsSmoke:          false,
		Metadata:                        map[string]any{"summary": "smoke passed"},
	})

	if out.ProofKey != "v1_stable_fixture_smoke" || out.EvidenceStatus != "pass" || out.Project.Key != "areamatrix-fixture" {
		t.Fatalf("unexpected operations smoke proof json: %+v", out)
	}
	if out.ProjectWriteAttempted || out.ServiceProcessControlAttempted || out.SupportBundleExported ||
		out.MigrationApplyAttempted || out.RemoteTelemetryEnabled || out.AreaMatrixProtectedPathsTouched ||
		out.RecordCommandRunsSmoke {
		t.Fatalf("operations smoke proof opened unsafe facts: %+v", out)
	}
}

func TestProtectedPathProofToJSON(t *testing.T) {
	emptyGitStatusHash := "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
	out := protectedPathProofToJSON(project.ProtectedPathProof{
		Project:                         project.Record{ID: 7, Key: "areamatrix"},
		Status:                          "recorded",
		ProofStatus:                     "clean",
		Decision:                        "allowed",
		Message:                         "AreaMatrix protected path proof recorded",
		EventID:                         21,
		AuditEventID:                    22,
		IdempotencyKey:                  "protected-proof-key",
		Created:                         true,
		ProjectWriteAttempted:           false,
		ExecutionWriteAttempted:         false,
		EngineCallAttempted:             false,
		CommandsRun:                     false,
		GitStatusRunByCommand:           false,
		AreaMatrixProtectedPathsTouched: false,
		GitStatusOutputHash:             emptyGitStatusHash,
		GitStatusOutputLines:            0,
		Metadata: map[string]any{
			"summary":                               "protected paths clean",
			"git_status_output_empty":               true,
			"protected_path_set_hash":               "set-hash",
			"protected_path_set_count":              int64(7),
			"protected_path_proof_binding_status":   "pass",
			"protected_path_proof_binding_blockers": []string{},
		},
	})

	if out.Project.Key != "areamatrix" || out.ProofStatus != "clean" || out.Decision != "allowed" {
		t.Fatalf("unexpected protected path proof json: %+v", out)
	}
	if out.ProjectWriteAttempted || out.ExecutionWriteAttempted || out.EngineCallAttempted ||
		out.CommandsRun || out.GitStatusRunByCommand || out.AreaMatrixProtectedPathsTouched ||
		out.GitStatusOutputHash != emptyGitStatusHash || out.GitStatusOutputLines != 0 {
		t.Fatalf("protected path proof opened unsafe facts: %+v", out)
	}
	if !out.GitStatusOutputEmpty ||
		out.ProtectedPathSetHash != "set-hash" ||
		out.ProtectedPathSetCount != 7 ||
		out.ProtectedPathProofBindingStatus != "pass" ||
		len(out.ProtectedPathProofBindingBlockers) != 0 {
		t.Fatalf("protected path proof json missing binding fields: %+v", out)
	}

	authorizedHash := strings.Repeat("a", 64)
	authorized := protectedPathProofToJSON(project.ProtectedPathProof{
		Project:                         project.Record{ID: 7, Key: "areamatrix"},
		Status:                          "recorded",
		ProofStatus:                     "authorized",
		Decision:                        "allowed",
		Message:                         "AreaMatrix protected path proof recorded",
		GitStatusOutputHash:             authorizedHash,
		GitStatusOutputLines:            1,
		AreaMatrixProtectedPathsTouched: true,
		Metadata: map[string]any{
			"summary":                               "protected paths approved",
			"evidence_uri":                          "local:protected-path-authorization",
			"authorized_approval_id":                "approval-123",
			"authorized_allowed_paths":              []string{"workflow/README.md"},
			"authorized_dirty_output_hash":          authorizedHash,
			"authorized_reviewer":                   "release-owner",
			"authorized_rollback_evidence_uri":      "local:rollback-proof",
			"authorized_touched_paths":              []string{"workflow/README.md"},
			"git_status_output_empty":               false,
			"protected_path_set_hash":               "set-hash",
			"protected_path_set_count":              int64(7),
			"protected_path_proof_binding_status":   "pass",
			"protected_path_proof_binding_blockers": []string{},
		},
	})
	if authorized.Summary != "protected paths approved" ||
		authorized.EvidenceURI != "local:protected-path-authorization" ||
		authorized.AuthorizedApprovalID != "approval-123" ||
		authorized.AuthorizedDirtyOutputHash != authorizedHash ||
		authorized.AuthorizedReviewer != "release-owner" ||
		authorized.AuthorizedRollbackEvidenceURI != "local:rollback-proof" ||
		authorized.AuthorizedProofComplete == nil ||
		*authorized.AuthorizedProofComplete != true {
		t.Fatalf("authorized protected path proof json missing top-level fields: %+v", authorized)
	}
	if !stringSliceContains(authorized.AuthorizedAllowedPaths, "workflow/README.md") ||
		!stringSliceContains(authorized.AuthorizedTouchedPaths, "workflow/README.md") {
		t.Fatalf("authorized protected path proof json missing paths: %+v", authorized)
	}
	if authorized.GitStatusOutputHash != authorizedHash || authorized.GitStatusOutputLines != 1 {
		t.Fatalf("authorized protected path proof json missing git status output facts: %+v", authorized)
	}
	if authorized.GitStatusOutputEmpty ||
		authorized.ProtectedPathSetHash != "set-hash" ||
		authorized.ProtectedPathSetCount != 7 ||
		authorized.ProtectedPathProofBindingStatus != "pass" ||
		len(authorized.ProtectedPathProofBindingBlockers) != 0 {
		t.Fatalf("authorized protected path proof json missing binding fields: %+v", authorized)
	}
}

func TestProtectedPathProofFlagsParseAuthorizedFields(t *testing.T) {
	flags, err := protectedPathProofFlagsFromArgs([]string{
		"--status", "authorized",
		"--summary", "protected paths approved",
		"--evidence-uri", "local:authorization",
		"--git-status-output", " M workflow/README.md",
		"--approval-id", "approval-123",
		"--allowed-path", "workflow/README.md",
		"--dirty-output-hash", strings.Repeat("a", 64),
		"--reviewer", "release-owner",
		"--rollback-evidence-uri", "local:rollback-proof",
		"--json",
	})
	if err != nil {
		t.Fatalf("parse protected path proof flags: %v", err)
	}
	if !flags.json ||
		flags.status != "authorized" ||
		flags.gitStatusOutput != " M workflow/README.md" ||
		flags.approvalID != "approval-123" ||
		len(flags.allowedPaths) != 1 ||
		flags.allowedPaths[0] != "workflow/README.md" ||
		flags.reviewer != "release-owner" ||
		flags.rollbackEvidenceURI != "local:rollback-proof" {
		t.Fatalf("unexpected protected path proof flags: %+v", flags)
	}
}

func TestCompletionAuditSnapshotFlagsParseReviewMetadata(t *testing.T) {
	flags, err := completionAuditSnapshotFlagsFromArgs([]string{
		"--release-candidate", "v1.0-rc1",
		"--evidence-class", "release_candidate",
		"--evidence-uri", "docs/development/real-release-candidate-evidence.md",
		"--summary", "real release candidate evidence reviewed",
		"--review-decision", "approved",
		"--reviewed-by", "release-owner",
		"--reviewed-at", "2026-07-04T12:00:00Z",
		"--json",
	})
	if err != nil {
		t.Fatalf("parse completion audit snapshot flags: %v", err)
	}
	if !flags.json ||
		flags.releaseCandidateLabel != "v1.0-rc1" ||
		flags.evidenceClass != "release_candidate" ||
		flags.reviewDecision != "approved" ||
		flags.reviewedBy != "release-owner" ||
		!flags.reviewedAt.Equal(time.Date(2026, 7, 4, 12, 0, 0, 0, time.UTC)) {
		t.Fatalf("completion audit snapshot flags missing review metadata: %+v", flags)
	}
}

func TestArchiveProofToJSON(t *testing.T) {
	out := archiveProofToJSON(project.ArchiveProof{
		Project:                         project.Record{ID: 7, Key: "areamatrix"},
		Status:                          "recorded",
		ProofStatus:                     "incomplete",
		Decision:                        "needs_attention",
		Message:                         "AreaMatrix archive proof is incomplete",
		Facts:                           []string{"historical_workflow_versions_marked_immutable"},
		MissingFacts:                    []string{"archive_does_not_rewrite_progress_json"},
		EventID:                         31,
		AuditEventID:                    32,
		IdempotencyKey:                  "archive-proof-key",
		Created:                         true,
		ProjectWriteAttempted:           false,
		ExecutionWriteAttempted:         false,
		ArtifactBytesCopied:             false,
		ArtifactBytesDeleted:            false,
		HistoricalFilesDeleted:          false,
		HistoricalFilesMoved:            false,
		ProgressJSONRewritten:           false,
		AreaMatrixProtectedPathsTouched: false,
		CommandsRun:                     false,
		ArchiveScope:                    "areamatrix_historical_execution_reference_only",
		ArchiveReferenceMode:            "metadata_indexed_reference_only",
		ArchiveSourcePaths: []string{
			".areaflow/status.json",
			"workflow/README.md",
			"workflow/versions/**/execution/**",
			"workflow/versions/**/execution/_shared/progress.json",
		},
		ArchiveForbiddenActions: []string{
			"copy_artifact_bytes",
			"delete_artifact_bytes",
			"delete_historical_files",
			"move_historical_files",
			"rewrite_progress_json",
			"run_commands",
			"write_areamatrix_protected_paths",
		},
		ArchiveRollbackTarget: "execution_forwarding_read_only_shim",
		ArchiveFailClosed:     true,
		Metadata: map[string]any{
			"summary":                         "archive proof partial",
			"archive_scope_binding_status":    "pass",
			"archive_scope_binding_blockers":  []any{},
			"archive_binding_contract":        "archive_scope_binding_v1",
			"archive_source_paths_hash":       "archive-source-paths-hash",
			"archive_forbidden_actions_hash":  "archive-forbidden-actions-hash",
			"archive_scope_binding_hash":      "archive-scope-binding-hash",
			"archive_scope":                   "areamatrix_historical_execution_reference_only",
			"archive_reference_mode":          "metadata_indexed_reference_only",
			"archive_rollback_target":         "execution_forwarding_read_only_shim",
			"archive_fail_closed":             true,
			"archive_forbidden_actions_count": float64(7),
			"archive_source_paths_count":      float64(4),
		},
	})

	if out.Project.Key != "areamatrix" || out.ProofStatus != "incomplete" || out.Decision != "needs_attention" {
		t.Fatalf("unexpected archive proof json: %+v", out)
	}
	if len(out.Facts) != 1 || len(out.MissingFacts) != 1 {
		t.Fatalf("archive proof json missing fact lists: %+v", out)
	}
	if out.ProjectWriteAttempted || out.ExecutionWriteAttempted || out.ArtifactBytesCopied ||
		out.ArtifactBytesDeleted || out.HistoricalFilesDeleted || out.HistoricalFilesMoved ||
		out.ProgressJSONRewritten || out.AreaMatrixProtectedPathsTouched || out.CommandsRun {
		t.Fatalf("archive proof opened unsafe facts: %+v", out)
	}
	if out.ArchiveScopeBindingStatus != "pass" || len(out.ArchiveScopeBindingBlockers) != 0 {
		t.Fatalf("archive proof json missing binding status: %+v", out)
	}
	if out.ArchiveBindingContract != "archive_scope_binding_v1" ||
		out.ArchiveSourcePathsHash != "archive-source-paths-hash" ||
		out.ArchiveForbiddenActionsHash != "archive-forbidden-actions-hash" ||
		out.ArchiveScopeBindingHash != "archive-scope-binding-hash" {
		t.Fatalf("archive proof json missing binding hashes: %+v", out)
	}
	if out.ArchiveScope != "areamatrix_historical_execution_reference_only" ||
		out.ArchiveReferenceMode != "metadata_indexed_reference_only" ||
		out.ArchiveRollbackTarget != "execution_forwarding_read_only_shim" ||
		!out.ArchiveFailClosed {
		t.Fatalf("archive proof json missing binding fields: %+v", out)
	}
	if !reflect.DeepEqual(out.ArchiveSourcePaths, []string{
		".areaflow/status.json",
		"workflow/README.md",
		"workflow/versions/**/execution/**",
		"workflow/versions/**/execution/_shared/progress.json",
	}) {
		t.Fatalf("archive proof json missing source paths: %+v", out.ArchiveSourcePaths)
	}
	if !reflect.DeepEqual(out.ArchiveForbiddenActions, []string{
		"copy_artifact_bytes",
		"delete_artifact_bytes",
		"delete_historical_files",
		"move_historical_files",
		"rewrite_progress_json",
		"run_commands",
		"write_areamatrix_protected_paths",
	}) {
		t.Fatalf("archive proof json missing forbidden actions: %+v", out.ArchiveForbiddenActions)
	}
}

func TestShimRetirementProofToJSON(t *testing.T) {
	out := shimRetirementProofToJSON(project.ShimRetirementProof{
		Project:                         project.Record{ID: 7, Key: "areamatrix"},
		Status:                          "recorded",
		ProofStatus:                     "incomplete",
		Decision:                        "needs_attention",
		Message:                         "AreaMatrix shim retirement proof is incomplete",
		Facts:                           []string{"archive_gate_passed"},
		MissingFacts:                    []string{"user_facing_retirement_notice_present"},
		EventID:                         41,
		AuditEventID:                    42,
		IdempotencyKey:                  "shim-proof-key",
		Created:                         true,
		ProjectWriteAttempted:           false,
		ExecutionWriteAttempted:         false,
		CommandsRun:                     false,
		LegacyRunnerStarted:             false,
		LegacyProgressWritten:           false,
		LegacyLogsWritten:               false,
		LegacyCheckpointWritten:         false,
		HistoricalFilesDeleted:          false,
		ProgressJSONRewritten:           false,
		AreaMatrixProtectedPathsTouched: false,
		ShimRetirementScope:             "read_only_shim_retirement_after_execution_forwarding_v1",
		ShimRetirementPrerequisites: []string{
			"archive_gate_passed",
			"execution_cutover_gate_passed",
			"protected_path_proof_recorded",
		},
		ShimRetiredSurfaces: []string{
			"legacy_task_loop_runner",
			"legacy_progress_json_writes",
			"legacy_logs_writes",
			"legacy_checkpoint_writes",
		},
		ShimRollbackTarget:         "read_only_shim",
		ShimFailClosed:             true,
		ShimReopenRequiresApproval: true,
		Metadata: map[string]any{
			"summary":                                "shim retirement proof partial",
			"shim_retirement_scope_binding_status":   "pass",
			"shim_retirement_scope_binding_blockers": []any{},
			"shim_retirement_binding_contract":       "shim_retirement_scope_binding_v1",
			"shim_retirement_prerequisites_hash":     "shim-prerequisites-hash",
			"shim_retired_surfaces_hash":             "shim-retired-surfaces-hash",
			"shim_retirement_scope_binding_hash":     "shim-scope-binding-hash",
			"shim_retirement_scope":                  "read_only_shim_retirement_after_execution_forwarding_v1",
			"shim_rollback_target":                   "read_only_shim",
			"shim_fail_closed":                       true,
			"shim_reopen_requires_approval":          true,
			"shim_retirement_prerequisites_count":    float64(3),
			"shim_retired_surfaces_count":            float64(4),
		},
	})

	if out.Project.Key != "areamatrix" || out.ProofStatus != "incomplete" || out.Decision != "needs_attention" {
		t.Fatalf("unexpected shim retirement proof json: %+v", out)
	}
	if len(out.Facts) != 1 || len(out.MissingFacts) != 1 {
		t.Fatalf("shim retirement proof json missing fact lists: %+v", out)
	}
	if out.ProjectWriteAttempted || out.ExecutionWriteAttempted || out.CommandsRun ||
		out.LegacyRunnerStarted || out.LegacyProgressWritten || out.LegacyLogsWritten ||
		out.LegacyCheckpointWritten || out.HistoricalFilesDeleted || out.ProgressJSONRewritten ||
		out.AreaMatrixProtectedPathsTouched {
		t.Fatalf("shim retirement proof opened unsafe facts: %+v", out)
	}
	if out.ShimRetirementScopeBindingStatus != "pass" || len(out.ShimRetirementScopeBindingBlockers) != 0 {
		t.Fatalf("shim retirement proof json missing binding status: %+v", out)
	}
	if out.ShimRetirementBindingContract != "shim_retirement_scope_binding_v1" ||
		out.ShimRetirementPrerequisitesHash != "shim-prerequisites-hash" ||
		out.ShimRetiredSurfacesHash != "shim-retired-surfaces-hash" ||
		out.ShimRetirementScopeBindingHash != "shim-scope-binding-hash" {
		t.Fatalf("shim retirement proof json missing binding hashes: %+v", out)
	}
	if out.ShimRetirementScope != "read_only_shim_retirement_after_execution_forwarding_v1" ||
		out.ShimRollbackTarget != "read_only_shim" ||
		!out.ShimFailClosed ||
		!out.ShimReopenRequiresApproval {
		t.Fatalf("shim retirement proof json missing binding fields: %+v", out)
	}
	if !reflect.DeepEqual(out.ShimRetirementPrerequisites, []string{
		"archive_gate_passed",
		"execution_cutover_gate_passed",
		"protected_path_proof_recorded",
	}) {
		t.Fatalf("shim retirement proof json missing prerequisites: %+v", out.ShimRetirementPrerequisites)
	}
	if !reflect.DeepEqual(out.ShimRetiredSurfaces, []string{
		"legacy_task_loop_runner",
		"legacy_progress_json_writes",
		"legacy_logs_writes",
		"legacy_checkpoint_writes",
	}) {
		t.Fatalf("shim retirement proof json missing retired surfaces: %+v", out.ShimRetiredSurfaces)
	}
}

func TestExecutionCutoverProofToJSON(t *testing.T) {
	out := executionCutoverProofToJSON(project.ExecutionCutoverProof{
		Project:                         project.Record{ID: 7, Key: "areamatrix"},
		Status:                          "recorded",
		ProofStatus:                     "incomplete",
		Decision:                        "needs_attention",
		Message:                         "execution cutover proof is incomplete",
		Facts:                           []string{"explicit_execution_cutover_approval_recorded"},
		MissingFacts:                    []string{"execution_cutover_command_response_recorded"},
		EventID:                         45,
		AuditEventID:                    46,
		IdempotencyKey:                  "execution-cutover-proof-key",
		Created:                         true,
		ExecutionCutoverScope:           "execution_forwarding_v1_read_only_evidence_only",
		AllowedTaskTypes:                []string{"read_only_verify"},
		ForbiddenActions:                []string{"engine_execution"},
		RollbackTarget:                  "read_only_shim",
		RollbackMode:                    "fail_closed_to_read_only_shim",
		FailClosed:                      true,
		ReopenRequiresApproval:          true,
		ProjectWriteAttempted:           false,
		ExecutionWriteAttempted:         false,
		TaskLoopRunForwardedByCommand:   false,
		EngineCallAttempted:             false,
		CommandsRun:                     false,
		LegacyProgressWritten:           false,
		LegacyLogsWritten:               false,
		LegacyCheckpointWritten:         false,
		AreaMatrixProtectedPathsTouched: false,
		Metadata: map[string]any{
			"summary":                                  "execution cutover proof partial",
			"execution_cutover_scope_binding_status":   "pass",
			"execution_cutover_scope_binding_blockers": []any{},
			"execution_cutover_binding_contract":       "execution_cutover_scope_binding_v1",
			"allowed_task_types_hash":                  "allowed-task-types-hash",
			"forbidden_actions_hash":                   "forbidden-actions-hash",
			"execution_cutover_binding_hash":           "execution-cutover-binding-hash",
			"execution_cutover_scope_binding_hash":     "execution-cutover-scope-binding-hash",
		},
	})

	if out.Project.Key != "areamatrix" || out.ProofStatus != "incomplete" || out.Decision != "needs_attention" {
		t.Fatalf("unexpected execution cutover proof json: %+v", out)
	}
	if len(out.Facts) != 1 || len(out.MissingFacts) != 1 {
		t.Fatalf("execution cutover proof json missing fact lists: %+v", out)
	}
	if out.ExecutionCutoverScope != "execution_forwarding_v1_read_only_evidence_only" ||
		len(out.AllowedTaskTypes) != 1 ||
		out.RollbackTarget != "read_only_shim" ||
		!out.FailClosed ||
		!out.ReopenRequiresApproval {
		t.Fatalf("execution cutover proof json missing scope binding: %+v", out)
	}
	if out.ExecutionCutoverScopeBindingStatus != "pass" || len(out.ExecutionCutoverScopeBindingBlockers) != 0 {
		t.Fatalf("execution cutover proof json missing binding status: %+v", out)
	}
	if out.ExecutionCutoverBindingContract != "execution_cutover_scope_binding_v1" ||
		out.AllowedTaskTypesHash != "allowed-task-types-hash" ||
		out.ForbiddenActionsHash != "forbidden-actions-hash" ||
		out.ExecutionCutoverBindingHash != "execution-cutover-binding-hash" ||
		out.ExecutionCutoverScopeBindingHash != "execution-cutover-scope-binding-hash" {
		t.Fatalf("execution cutover proof json missing binding hashes: %+v", out)
	}
	if out.ProjectWriteAttempted || out.ExecutionWriteAttempted || out.TaskLoopRunForwardedByCommand ||
		out.EngineCallAttempted || out.CommandsRun || out.LegacyProgressWritten || out.LegacyLogsWritten ||
		out.LegacyCheckpointWritten || out.AreaMatrixProtectedPathsTouched || out.SourceWriteOpen ||
		out.GeneratedRetainedWriteOpen || out.RepairApplyOpen || out.CheckpointApplyOpen ||
		out.EngineExecutionOpen || out.SecretResolveOpen || out.NetworkAPIIntegrationOpen ||
		out.PublishApplyOpen || out.RestoreApplyOpen {
		t.Fatalf("execution cutover proof opened unsafe facts: %+v", out)
	}
}

func TestValidationProofToJSON(t *testing.T) {
	out := validationProofToJSON(project.ValidationProof{
		Project:                         project.Record{ID: 7, Key: "areamatrix"},
		Status:                          "recorded",
		ProofStatus:                     "incomplete",
		Decision:                        "needs_attention",
		Message:                         "validation proof is incomplete",
		Facts:                           []string{"go_test_passed"},
		MissingFacts:                    []string{"web_smoke_passed"},
		EventID:                         51,
		AuditEventID:                    52,
		IdempotencyKey:                  "validation-proof-key",
		Created:                         true,
		ProjectWriteAttempted:           false,
		ExecutionWriteAttempted:         false,
		EngineCallAttempted:             false,
		CommandsRun:                     false,
		SmokeRunAttempted:               false,
		WebBuildRunByCommand:            false,
		AreaMatrixProtectedPathsTouched: false,
		Metadata:                        map[string]any{"summary": "validation proof partial"},
	})

	if out.Project.Key != "areamatrix" || out.ProofStatus != "incomplete" || out.Decision != "needs_attention" {
		t.Fatalf("unexpected validation proof json: %+v", out)
	}
	if len(out.Facts) != 1 || len(out.MissingFacts) != 1 {
		t.Fatalf("validation proof json missing fact lists: %+v", out)
	}
	if out.ProjectWriteAttempted || out.ExecutionWriteAttempted || out.EngineCallAttempted ||
		out.CommandsRun || out.SmokeRunAttempted || out.WebBuildRunByCommand ||
		out.AreaMatrixProtectedPathsTouched {
		t.Fatalf("validation proof opened unsafe facts: %+v", out)
	}
}

func TestSourceAlignmentProofToJSON(t *testing.T) {
	hash := strings.Repeat("a", 64)
	out := sourceAlignmentProofToJSON(project.SourceAlignmentProof{
		Project:                         project.Record{ID: 7, Key: "areamatrix"},
		Status:                          "recorded",
		ProofStatus:                     "incomplete",
		Decision:                        "needs_attention",
		Message:                         "source alignment proof is incomplete",
		Facts:                           []string{"zero_to_hundred_phases_aligned"},
		MissingFacts:                    []string{"preview_only_not_claimed_as_apply"},
		EventID:                         61,
		AuditEventID:                    62,
		IdempotencyKey:                  "source-alignment-proof-key",
		Created:                         true,
		ProjectWriteAttempted:           false,
		ExecutionWriteAttempted:         false,
		CommandsRun:                     false,
		DocsWritten:                     false,
		AreaMatrixProtectedPathsTouched: false,
		Metadata: map[string]any{
			"summary":                                  "source alignment proof partial",
			"source_alignment_binding_status":          "pass",
			"source_alignment_binding_blockers":        []string{},
			"source_alignment_source_paths":            []string{"docs/product/master-plan.md"},
			"source_alignment_source_hashes":           map[string]any{"docs/product/master-plan.md": hash},
			"source_alignment_source_set_hash":         hash,
			"source_alignment_source_file_count":       int64(1),
			"source_alignment_missing_source_count":    int64(0),
			"source_alignment_unreadable_source_count": int64(0),
		},
	})

	if out.Project.Key != "areamatrix" || out.ProofStatus != "incomplete" || out.Decision != "needs_attention" {
		t.Fatalf("unexpected source alignment proof json: %+v", out)
	}
	if len(out.Facts) != 1 || len(out.MissingFacts) != 1 {
		t.Fatalf("source alignment proof json missing fact lists: %+v", out)
	}
	if out.ProjectWriteAttempted || out.ExecutionWriteAttempted || out.CommandsRun ||
		out.DocsWritten || out.AreaMatrixProtectedPathsTouched {
		t.Fatalf("source alignment proof opened unsafe facts: %+v", out)
	}
	if out.SourceAlignmentBindingStatus != "pass" ||
		len(out.SourceAlignmentBindingBlockers) != 0 ||
		out.SourceAlignmentSourceSetHash != hash ||
		out.SourceAlignmentSourceFileCount != 1 ||
		out.MissingSourceCount != 0 ||
		out.UnreadableSourceCount != 0 ||
		out.SourceAlignmentSourceHashes["docs/product/master-plan.md"] != hash {
		t.Fatalf("source alignment binding fields missing from json: %+v", out)
	}
}

func TestTaskMatrixProofToJSON(t *testing.T) {
	backlogHash := strings.Repeat("a", 64)
	statusAuditHash := strings.Repeat("b", 64)
	sourceSetHash := "source-set-hash"
	out := taskMatrixProofToJSON(project.TaskMatrixProof{
		Project:                         project.Record{ID: 7, Key: "areamatrix"},
		Status:                          "recorded",
		ProofStatus:                     "incomplete",
		Decision:                        "needs_attention",
		Message:                         "task matrix proof is incomplete",
		Facts:                           []string{"all_v0_v1_tasks_have_status_evidence_and_boundary"},
		MissingFacts:                    []string{"v1x_deferred_tasks_have_contracts"},
		EventID:                         71,
		AuditEventID:                    72,
		IdempotencyKey:                  "task-matrix-proof-key",
		Created:                         true,
		ProjectWriteAttempted:           false,
		ExecutionWriteAttempted:         false,
		CommandsRun:                     false,
		DocsWritten:                     false,
		TasksWritten:                    false,
		AreaMatrixProtectedPathsTouched: false,
		Metadata: map[string]any{
			"summary":                                 "task matrix proof partial",
			"task_matrix_binding_status":              "pass",
			"task_matrix_binding_blockers":            []string{},
			"task_matrix_source_paths":                []string{"tasks/backlog/0-100-platform-backlog.md", "docs/development/task-backlog-status-audit.md"},
			"task_matrix_source_set_hash":             sourceSetHash,
			"task_backlog_hash":                       backlogHash,
			"task_status_audit_hash":                  statusAuditHash,
			"planned_v1_required_task_count":          int64(0),
			"missing_evidence_v1_required_task_count": int64(0),
			"blocked_v1_required_task_count":          int64(0),
		},
	})

	if out.Project.Key != "areamatrix" || out.ProofStatus != "incomplete" || out.Decision != "needs_attention" {
		t.Fatalf("unexpected task matrix proof json: %+v", out)
	}
	if len(out.Facts) != 1 || len(out.MissingFacts) != 1 {
		t.Fatalf("task matrix proof json missing fact lists: %+v", out)
	}
	if out.ProjectWriteAttempted || out.ExecutionWriteAttempted || out.CommandsRun ||
		out.DocsWritten || out.TasksWritten || out.AreaMatrixProtectedPathsTouched {
		t.Fatalf("task matrix proof opened unsafe facts: %+v", out)
	}
	if out.TaskMatrixBindingStatus != "pass" ||
		out.TaskMatrixSourceSetHash != sourceSetHash ||
		out.TaskBacklogHash != backlogHash ||
		out.TaskStatusAuditHash != statusAuditHash ||
		out.PlannedV1RequiredTaskCount != 0 ||
		out.MissingEvidenceV1RequiredTaskCount != 0 ||
		out.BlockedV1RequiredTaskCount != 0 ||
		len(out.TaskMatrixBindingBlockers) != 0 ||
		len(out.TaskMatrixSourcePaths) != 2 {
		t.Fatalf("task matrix proof json missing binding fields: %+v", out)
	}
}

func TestTaskMatrixProofFlagsParseBindingFields(t *testing.T) {
	backlogHash := strings.Repeat("a", 64)
	statusAuditHash := strings.Repeat("b", 64)
	sourceSetHash := strings.Repeat("c", 64)

	flags, err := taskMatrixProofFlagsFromArgs([]string{
		"--status", "complete",
		"--fact", "all_v0_v1_tasks_have_status_evidence_and_boundary",
		"--summary", "task matrix reviewed",
		"--evidence-uri", "local:task-matrix",
		"--source-set-hash", sourceSetHash,
		"--backlog-hash", backlogHash,
		"--task-status-audit-hash", statusAuditHash,
		"--planned-v1-required-task-count", "0",
		"--missing-evidence-v1-required-task-count", "0",
		"--blocked-v1-required-task-count", "0",
	})
	if err != nil {
		t.Fatalf("parse task matrix proof flags: %v", err)
	}
	if flags.taskMatrixSourceSetHash != sourceSetHash ||
		flags.taskBacklogHash != backlogHash ||
		flags.taskStatusAuditHash != statusAuditHash ||
		!flags.plannedV1RequiredTaskCountSet ||
		!flags.missingEvidenceV1RequiredTaskCountSet ||
		!flags.blockedV1RequiredTaskCountSet {
		t.Fatalf("task matrix binding flags missing: %+v", flags)
	}
}

func TestSecurityClosureProofToJSON(t *testing.T) {
	out := securityClosureProofToJSON(project.SecurityClosureProof{
		Project:                         project.Record{ID: 7, Key: "areamatrix"},
		Status:                          "recorded",
		ProofStatus:                     "incomplete",
		Decision:                        "needs_attention",
		Message:                         "security closure proof is incomplete",
		Facts:                           []string{"project_key_isolation_covers_workflow_run_lease_artifact_secret_audit"},
		MissingFacts:                    []string{"no_forbidden_v1_security_capability_opened"},
		EventID:                         81,
		AuditEventID:                    82,
		IdempotencyKey:                  "security-closure-proof-key",
		Created:                         true,
		ProjectWriteAttempted:           false,
		ExecutionWriteAttempted:         false,
		AuthorizationChanged:            false,
		SecretPlaintextRead:             false,
		RemoteWorkerCredentialsIssued:   false,
		CommandsRun:                     false,
		AreaMatrixProtectedPathsTouched: false,
		Metadata: map[string]any{
			"summary":                         "security closure proof partial",
			"security_closure_binding_status": "pass",
			"security_closure_binding_hash":   "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
			"security_boundary_status":        "ready",
			"permission_doctor_status":        "pass",
			"audit_coverage_status":           "pass",
		},
	})

	if out.Project.Key != "areamatrix" || out.ProofStatus != "incomplete" || out.Decision != "needs_attention" {
		t.Fatalf("unexpected security closure proof json: %+v", out)
	}
	if len(out.Facts) != 1 || len(out.MissingFacts) != 1 {
		t.Fatalf("security closure proof json missing fact lists: %+v", out)
	}
	if out.ProjectWriteAttempted || out.ExecutionWriteAttempted || out.AuthorizationChanged ||
		out.SecretPlaintextRead || out.RemoteWorkerCredentialsIssued || out.CommandsRun ||
		out.AreaMatrixProtectedPathsTouched {
		t.Fatalf("security closure proof opened unsafe facts: %+v", out)
	}
	if out.SecurityClosureBindingStatus != "pass" ||
		out.SecurityClosureBindingHash != "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef" ||
		out.SecurityBoundaryStatus != "ready" ||
		out.PermissionDoctorStatus != "pass" ||
		out.AuditCoverageStatus != "pass" {
		t.Fatalf("security closure proof json missing binding fields: %+v", out)
	}
}

func TestBackupRestoreProofToJSON(t *testing.T) {
	out := backupRestoreProofToJSON(project.BackupRestoreProof{
		Project:                         project.Record{ID: 7, Key: "areamatrix"},
		Status:                          "recorded",
		ProofStatus:                     "incomplete",
		Decision:                        "needs_attention",
		Message:                         "backup restore proof is incomplete",
		Facts:                           []string{"backup_manifest_covers_pg_metadata_and_areaflow_artifact_metadata"},
		MissingFacts:                    []string{"no_restore_apply_or_artifact_mutation_opened"},
		EventID:                         91,
		AuditEventID:                    92,
		IdempotencyKey:                  "backup-restore-proof-key",
		Created:                         true,
		ProjectWriteAttempted:           false,
		ExecutionWriteAttempted:         false,
		DatabaseRestoreAttempted:        false,
		ArtifactBytesCopied:             false,
		ArtifactBytesDeleted:            false,
		ArtifactBytesUploaded:           false,
		ArtifactGCAttempted:             false,
		CommandsRun:                     false,
		AreaMatrixProtectedPathsTouched: false,
		Metadata:                        map[string]any{"summary": "backup restore proof partial"},
	})

	if out.Project.Key != "areamatrix" || out.ProofStatus != "incomplete" || out.Decision != "needs_attention" {
		t.Fatalf("unexpected backup restore proof json: %+v", out)
	}
	if len(out.Facts) != 1 || len(out.MissingFacts) != 1 {
		t.Fatalf("backup restore proof json missing fact lists: %+v", out)
	}
	if out.ProjectWriteAttempted || out.ExecutionWriteAttempted || out.DatabaseRestoreAttempted ||
		out.ArtifactBytesCopied || out.ArtifactBytesDeleted || out.ArtifactBytesUploaded ||
		out.ArtifactGCAttempted || out.CommandsRun || out.AreaMatrixProtectedPathsTouched {
		t.Fatalf("backup restore proof opened unsafe facts: %+v", out)
	}
}

func TestReleasePackagingProofToJSON(t *testing.T) {
	out := releasePackagingProofToJSON(project.ReleasePackagingProof{
		Project:                         project.Record{ID: 7, Key: "areamatrix"},
		Status:                          "recorded",
		ProofStatus:                     "incomplete",
		Decision:                        "needs_attention",
		Message:                         "release packaging proof is incomplete",
		Facts:                           []string{"release_final_gate_passed"},
		MissingFacts:                    []string{"no_release_package_publish_rollout_apply_opened"},
		EventID:                         101,
		AuditEventID:                    102,
		IdempotencyKey:                  "release-packaging-proof-key",
		Created:                         true,
		ProjectWriteAttempted:           false,
		ExecutionWriteAttempted:         false,
		ReleasePackageCreated:           false,
		ReleaseStateWritten:             false,
		ReleaseApprovalCreated:          false,
		RolloutStateCreated:             false,
		MigrationApplyAttempted:         false,
		TagCreated:                      false,
		PackageSigned:                   false,
		ArtifactUploaded:                false,
		GitPushAttempted:                false,
		PublishAttempted:                false,
		CommandsRun:                     false,
		AreaMatrixProtectedPathsTouched: false,
		Metadata:                        map[string]any{"summary": "release packaging proof partial"},
	})

	if out.Project.Key != "areamatrix" || out.ProofStatus != "incomplete" || out.Decision != "needs_attention" {
		t.Fatalf("unexpected release packaging proof json: %+v", out)
	}
	if len(out.Facts) != 1 || len(out.MissingFacts) != 1 {
		t.Fatalf("release packaging proof json missing fact lists: %+v", out)
	}
	if out.ProjectWriteAttempted || out.ExecutionWriteAttempted || out.ReleasePackageCreated ||
		out.ReleaseStateWritten || out.ReleaseApprovalCreated || out.RolloutStateCreated ||
		out.MigrationApplyAttempted || out.TagCreated || out.PackageSigned || out.ArtifactUploaded ||
		out.GitPushAttempted || out.PublishAttempted || out.CommandsRun || out.AreaMatrixProtectedPathsTouched {
		t.Fatalf("release packaging proof opened unsafe facts: %+v", out)
	}
}

func TestBackupManifestToJSON(t *testing.T) {
	created := time.Date(2026, 6, 30, 10, 0, 0, 0, time.UTC)
	out := backupManifestToJSON(project.BackupManifest{
		Status:        "ready",
		Mode:          "read_only_manifest",
		Scope:         "project",
		ProjectKey:    "areamatrix",
		SchemaVersion: 1,
		GeneratedAt:   created,
		ManifestHash:  "abc123",
		TableCounts: []project.BackupTableCount{
			{Table: "projects", Rows: 1},
			{Table: "artifacts", Rows: 2},
		},
		Projects: []project.BackupProjectManifest{
			{
				Project: project.Record{
					Key:             "areamatrix",
					Name:            "AreaMatrix",
					Kind:            "product-repo",
					Adapter:         "areamatrix",
					WorkflowProfile: "areamatrix",
				},
				Inventory:     project.ImportInventory{Versions: 2, Residuals: 10, Artifacts: 6},
				ArtifactCount: 1,
				Artifacts: []project.BackupArtifactSummary{
					{
						ID:                7,
						ProjectID:         1,
						WorkflowVersionID: 2,
						ArtifactType:      "runner_preview_report",
						StorageBackend:    "local",
						URI:               "/tmp/areaflow/artifacts/areamatrix/report.json",
						SourcePath:        "report.json",
						SHA256:            "def456",
						SizeBytes:         128,
						ContentType:       "application/json",
						CreatedAt:         created,
					},
				},
			},
		},
		Capabilities:     []string{"export_postgres_metadata", "export_artifact_metadata"},
		ForbiddenActions: []string{"restore_database", "read_artifact_contents"},
	})

	if out.Status != "ready" || out.Mode != "read_only_manifest" || out.Scope != "project" || out.ProjectKey != "areamatrix" || out.ManifestHash != "abc123" {
		t.Fatalf("unexpected backup manifest json: %+v", out)
	}
	if len(out.TableCounts) != 2 || out.TableCounts[0].Table != "projects" {
		t.Fatalf("unexpected backup table counts: %+v", out.TableCounts)
	}
	if len(out.Projects) != 1 || out.Projects[0].Project.Key != "areamatrix" {
		t.Fatalf("unexpected backup project json: %+v", out.Projects)
	}
	if out.Projects[0].ArtifactCount != 1 || out.Projects[0].Artifacts[0].SHA256 != "def456" {
		t.Fatalf("unexpected backup artifact json: %+v", out.Projects[0].Artifacts)
	}
	if len(out.Capabilities) != 2 || len(out.ForbiddenActions) != 2 || out.GeneratedAt == "" {
		t.Fatalf("unexpected backup guardrails json: %+v", out)
	}
}

func TestRestorePlanToJSON(t *testing.T) {
	created := time.Date(2026, 6, 30, 10, 30, 0, 0, time.UTC)
	out := restorePlanToJSON(project.RestorePlan{
		Status:        "needs_attention",
		Mode:          "read_only_restore_plan",
		Scope:         "project",
		ProjectKey:    "areamatrix",
		SchemaVersion: 1,
		ManifestHash:  "abc123",
		Projects: []project.Record{
			{Key: "areamatrix", Name: "AreaMatrix", Adapter: "areamatrix", WorkflowProfile: "areamatrix"},
		},
		Items: []project.RestorePlanItem{
			{
				Key:      "manifest_shape",
				Category: "manifest",
				Status:   "ready",
				Message:  "backup manifest has schema version and stable hash",
				Metadata: map[string]any{"manifest_hash": "abc123"},
			},
			{
				Key:      "artifact_integrity:areamatrix",
				Category: "artifact",
				Status:   "needs_attention",
				Message:  "artifact integrity has warnings or skipped references",
				Metadata: map[string]any{"skipped_artifacts": float64(1)},
			},
		},
		Capabilities:     []string{"generate_restore_plan"},
		ForbiddenActions: []string{"restore_database", "apply_restore"},
		GeneratedAt:      created,
	})

	if out.Status != "needs_attention" || out.Mode != "read_only_restore_plan" || out.Scope != "project" || out.ProjectKey != "areamatrix" || out.ManifestHash != "abc123" {
		t.Fatalf("unexpected restore plan json: %+v", out)
	}
	if len(out.Projects) != 1 || out.Projects[0].Key != "areamatrix" {
		t.Fatalf("unexpected restore projects json: %+v", out.Projects)
	}
	if len(out.Items) != 2 || out.Items[1].Key != "artifact_integrity:areamatrix" {
		t.Fatalf("unexpected restore items json: %+v", out.Items)
	}
	if len(out.Capabilities) != 1 || len(out.ForbiddenActions) != 2 || out.GeneratedAt == "" {
		t.Fatalf("unexpected restore guardrails json: %+v", out)
	}
}

func TestReleaseReadinessToJSON(t *testing.T) {
	created := time.Date(2026, 6, 30, 12, 0, 0, 0, time.UTC)
	record := project.Record{Key: "areamatrix", Name: "AreaMatrix", Adapter: "areamatrix", WorkflowProfile: "areamatrix"}
	out := releaseReadinessToJSON(project.ReleaseReadiness{
		Status:     "needs_attention",
		Mode:       "read_only_release_readiness",
		Scope:      "project",
		ProjectKey: "areamatrix",
		Backup: project.BackupManifest{
			Status:        "ready",
			Mode:          "read_only_manifest",
			SchemaVersion: 1,
			ManifestHash:  "hash-a",
			Projects:      []project.BackupProjectManifest{{Project: record}},
			GeneratedAt:   created,
		},
		RestorePlan: project.RestorePlan{
			Status:        "needs_attention",
			Mode:          "read_only_restore_plan",
			SchemaVersion: 1,
			ManifestHash:  "hash-a",
			Projects:      []project.Record{record},
			Items:         []project.RestorePlanItem{{Key: "artifact_inventory", Status: "needs_attention"}},
			GeneratedAt:   created,
		},
		AuditCoverage: project.AuditCoverage{
			Status:              "warn",
			Mode:                "read_only_audit_coverage",
			Scope:               "platform",
			TotalAuditEvents:    10,
			CoveredRequirements: 8,
			GapRequirements:     3,
			GeneratedAt:         created,
		},
		Projects: []project.ReleaseReadinessProject{
			{
				Project:             record,
				Status:              "needs_attention",
				NeedsAttentionItems: 1,
				Permission:          project.PermissionPolicyDoctor{Status: "pass", Mode: "read_only_permission_policy_doctor", Project: record, GeneratedAt: created},
				ArtifactIntegrity:   project.ArtifactIntegrityReport{Status: "warn", Mode: "read_only_artifact_integrity", Project: record, CheckedArtifacts: 2, PassedArtifacts: 1, SkippedArtifacts: 1, GeneratedAt: created},
				Conformance:         project.ConformanceReport{Status: "pass", Mode: "read_only_adapter_profile_conformance", Project: record, ProfileID: "areamatrix", Adapter: "areamatrix", ProfileHash: "hash-profile", StageCount: 16, GateCount: 17, GeneratedAt: created},
			},
		},
		Items: []project.ReleaseReadinessItem{
			{Key: "backup_manifest", Category: "backup", Status: "ready", Message: "backup manifest is ready"},
			{Key: "restore_plan", Category: "restore", Status: "needs_attention", Message: "restore dry-run plan needs attention"},
		},
		Capabilities:     []string{"generate_release_readiness"},
		ForbiddenActions: []string{"restore_database", "start_worker"},
		GeneratedAt:      created,
	})

	if out.Status != "needs_attention" || out.Mode != "read_only_release_readiness" ||
		out.Scope != "project" || out.ProjectKey != "areamatrix" {
		t.Fatalf("unexpected release readiness json: %+v", out)
	}
	if out.Backup.ManifestHash != "hash-a" || out.RestorePlan.Status != "needs_attention" || out.AuditCoverage.GapRequirements != 3 {
		t.Fatalf("unexpected nested release readiness: %+v", out)
	}
	if len(out.Projects) != 1 || out.Projects[0].Project.Key != "areamatrix" || out.Projects[0].ArtifactIntegrity.Status != "warn" {
		t.Fatalf("unexpected project readiness: %+v", out.Projects)
	}
	if len(out.Items) != 2 || out.Items[1].Key != "restore_plan" {
		t.Fatalf("unexpected release items: %+v", out.Items)
	}
	if len(out.ForbiddenActions) != 2 || out.GeneratedAt == "" {
		t.Fatalf("unexpected release guardrails: %+v", out)
	}
	assertReal100Guardrail(
		t,
		out.Real100Status,
		out.ReadinessScope,
		out.Real100Blockers,
		project.ReleasePreviewReadinessScope,
		project.Real100ReleasePreviewBlockers(),
	)
}

func TestReleaseRemediationPlanToJSON(t *testing.T) {
	created := time.Date(2026, 6, 30, 12, 0, 0, 0, time.UTC)
	out := releaseRemediationPlanToJSON(project.ReleaseRemediationPlan{
		Status:     "needs_attention",
		Mode:       "read_only_release_remediation_plan",
		Scope:      "project",
		ProjectKey: "areamatrix",
		Readiness: project.ReleaseReadiness{
			Status:      "needs_attention",
			Mode:        "read_only_release_readiness",
			Scope:       "project",
			ProjectKey:  "areamatrix",
			GeneratedAt: created,
		},
		Actions: []project.ReleaseRemediationAction{
			{
				Key:               "remediate:restore_plan",
				Category:          "restore",
				Status:            "needs_attention",
				SourceItem:        "restore_plan",
				RecommendedAction: "decide artifact archive policy",
				Rationale:         "project references are metadata-only",
				Owner:             "release_owner",
				NextCommand:       "areaflow backup restore-plan --json",
				Acceptance:        "restore plan is ready or accepted",
				Metadata:          map[string]any{"restore_status": "needs_attention"},
			},
			{
				Key:               "remediate:audit_coverage",
				Category:          "audit",
				Status:            "needs_attention",
				SourceItem:        "audit_coverage",
				RecommendedAction: "close enabled audit gaps",
				Rationale:         "audit gaps must be explicit",
				Owner:             "platform_owner",
				NextCommand:       "areaflow audit coverage --json",
				Acceptance:        "audit coverage is pass or accepted",
				Metadata:          map[string]any{"gap_requirements": 3},
			},
		},
		Capabilities:     []string{"generate_remediation_plan"},
		ForbiddenActions: []string{"write_project_files", "mark_gap_accepted"},
		GeneratedAt:      created,
	})

	if out.Status != "needs_attention" || out.Mode != "read_only_release_remediation_plan" ||
		out.Scope != "project" || out.ProjectKey != "areamatrix" {
		t.Fatalf("unexpected remediation plan json: %+v", out)
	}
	if out.Readiness.Status != "needs_attention" || out.Readiness.ProjectKey != "areamatrix" {
		t.Fatalf("unexpected remediation readiness: %+v", out.Readiness)
	}
	if len(out.Actions) != 2 || out.Actions[0].Key != "remediate:restore_plan" || out.Actions[1].NextCommand != "areaflow audit coverage --json" {
		t.Fatalf("unexpected remediation actions: %+v", out.Actions)
	}
	if len(out.ForbiddenActions) != 2 || out.GeneratedAt == "" {
		t.Fatalf("unexpected remediation guardrails: %+v", out)
	}
	assertReal100Guardrail(
		t,
		out.Real100Status,
		out.ReadinessScope,
		out.Real100Blockers,
		project.ReleasePreviewReadinessScope,
		project.Real100ReleasePreviewBlockers(),
	)
}

func TestReleaseAcceptancePreviewToJSON(t *testing.T) {
	created := time.Date(2026, 6, 30, 12, 0, 0, 0, time.UTC)
	out := releaseAcceptancePreviewToJSON(project.ReleaseAcceptancePreview{
		Status:     "needs_decision",
		Mode:       "read_only_release_acceptance_preview",
		Scope:      "project",
		ProjectKey: "areamatrix",
		Remediation: project.ReleaseRemediationPlan{
			Status:     "needs_attention",
			Mode:       "read_only_release_remediation_plan",
			Scope:      "project",
			ProjectKey: "areamatrix",
			Readiness: project.ReleaseReadiness{
				Status:      "needs_attention",
				Mode:        "read_only_release_readiness",
				Scope:       "project",
				ProjectKey:  "areamatrix",
				GeneratedAt: created,
			},
			GeneratedAt: created,
		},
		Decisions: []project.ReleaseAcceptanceDecision{
			{
				Key:              "accept:restore_plan",
				SourceAction:     "remediate:restore_plan",
				Category:         "restore",
				Status:           "needs_decision",
				AcceptanceType:   "metadata_only_history",
				Owner:            "release_owner",
				Reason:           "metadata-only history requires explicit acceptance",
				RequiredEvidence: []string{"release notes state metadata-only artifacts"},
				NextCommand:      "areaflow backup restore-plan --json",
				Metadata:         map[string]any{"restore_status": "needs_attention"},
			},
			{
				Key:              "accept:audit_coverage",
				SourceAction:     "remediate:audit_coverage",
				Category:         "audit",
				Status:           "needs_decision",
				AcceptanceType:   "future_only_gap",
				Owner:            "platform_owner",
				Reason:           "future-only audit gaps require owners",
				RequiredEvidence: []string{"audit coverage lists missing actions"},
				NextCommand:      "areaflow audit coverage --json",
				Metadata:         map[string]any{"gap_requirements": 3},
			},
		},
		Capabilities:     []string{"generate_acceptance_preview"},
		ForbiddenActions: []string{"write_database", "mark_gap_accepted", "create_approval"},
		GeneratedAt:      created,
	})

	if out.Status != "needs_decision" || out.Mode != "read_only_release_acceptance_preview" ||
		out.Scope != "project" || out.ProjectKey != "areamatrix" {
		t.Fatalf("unexpected acceptance preview json: %+v", out)
	}
	if out.Remediation.Status != "needs_attention" || out.Remediation.ProjectKey != "areamatrix" ||
		out.Remediation.Readiness.Status != "needs_attention" {
		t.Fatalf("unexpected acceptance remediation: %+v", out.Remediation)
	}
	if len(out.Decisions) != 2 || out.Decisions[0].AcceptanceType != "metadata_only_history" || out.Decisions[1].AcceptanceType != "future_only_gap" {
		t.Fatalf("unexpected acceptance decisions: %+v", out.Decisions)
	}
	if len(out.Decisions[0].RequiredEvidence) != 1 || out.Decisions[0].NextCommand != "areaflow backup restore-plan --json" {
		t.Fatalf("unexpected acceptance decision guidance: %+v", out.Decisions[0])
	}
	if len(out.ForbiddenActions) != 3 || out.GeneratedAt == "" {
		t.Fatalf("unexpected acceptance guardrails: %+v", out)
	}
	assertReal100Guardrail(
		t,
		out.Real100Status,
		out.ReadinessScope,
		out.Real100Blockers,
		project.ReleasePreviewReadinessScope,
		project.Real100ReleasePreviewBlockers(),
	)
}

func TestReleaseAcceptanceGateToJSON(t *testing.T) {
	created := time.Date(2026, 6, 30, 12, 0, 0, 0, time.UTC)
	out := releaseAcceptanceGateToJSON(project.ReleaseAcceptanceGate{
		Status:     "blocked",
		Mode:       "read_only_release_acceptance_gate",
		Scope:      "project",
		ProjectKey: "areamatrix",
		Preview: project.ReleaseAcceptancePreview{
			Status:      "needs_decision",
			Mode:        "read_only_release_acceptance_preview",
			Scope:       "project",
			ProjectKey:  "areamatrix",
			GeneratedAt: created,
		},
		Items: []project.ReleaseAcceptanceGateItem{
			{
				Key:              "gate:accept:restore_plan",
				Category:         "restore",
				Status:           "blocked",
				DecisionStatus:   "needs_decision",
				AcceptanceType:   "metadata_only_history",
				Message:          "explicit release acceptance evidence is required",
				Owner:            "release_owner",
				RequiredEvidence: []string{"release notes state metadata-only artifacts"},
				NextCommand:      "areaflow backup restore-plan --json",
				Metadata:         map[string]any{"restore_status": "needs_attention"},
			},
		},
		Capabilities:     []string{"evaluate_release_acceptance_gate"},
		ForbiddenActions: []string{"write_database", "mark_gap_accepted", "create_approval"},
		GeneratedAt:      created,
	})

	if out.Status != "blocked" || out.Mode != "read_only_release_acceptance_gate" ||
		out.Scope != "project" || out.ProjectKey != "areamatrix" {
		t.Fatalf("unexpected acceptance gate json: %+v", out)
	}
	if out.Preview.Status != "needs_decision" || out.Preview.ProjectKey != "areamatrix" {
		t.Fatalf("unexpected acceptance gate preview: %+v", out.Preview)
	}
	if len(out.Items) != 1 || out.Items[0].DecisionStatus != "needs_decision" || out.Items[0].AcceptanceType != "metadata_only_history" {
		t.Fatalf("unexpected acceptance gate items: %+v", out.Items)
	}
	if len(out.Items[0].RequiredEvidence) != 1 || out.Items[0].NextCommand != "areaflow backup restore-plan --json" {
		t.Fatalf("unexpected acceptance gate item guidance: %+v", out.Items[0])
	}
	if len(out.ForbiddenActions) != 3 || out.GeneratedAt == "" {
		t.Fatalf("unexpected acceptance gate guardrails: %+v", out)
	}
	assertReal100Guardrail(
		t,
		out.Real100Status,
		out.ReadinessScope,
		out.Real100Blockers,
		project.ReleasePreviewReadinessScope,
		project.Real100ReleasePreviewBlockers(),
	)
}

func TestReleaseExceptionDoctorToJSON(t *testing.T) {
	created := time.Date(2026, 6, 30, 12, 0, 0, 0, time.UTC)
	out := releaseExceptionDoctorToJSON(project.ReleaseExceptionDoctor{
		Status:     "warn",
		Mode:       "read_only_release_exception_doctor",
		Scope:      "project",
		ProjectKey: "areamatrix",
		Gate: project.ReleaseAcceptanceGate{
			Status:      "blocked",
			Mode:        "read_only_release_acceptance_gate",
			Scope:       "project",
			ProjectKey:  "areamatrix",
			GeneratedAt: created,
		},
		Checks: []project.ReleaseExceptionDoctorCheck{
			{
				Key:      "exception_record_schema",
				Category: "schema",
				Status:   "warn",
				Message:  "release exception record schema is designed but not enabled for writes",
				Metadata: map[string]any{"writes_enabled": false},
			},
			{
				Key:      "exception:gate:accept:restore_plan",
				Category: "restore",
				Status:   "warn",
				Message:  "release exception record is required before this gate item can pass",
				Metadata: map[string]any{"exception_writable": false},
			},
		},
		Capabilities:     []string{"check_exception_record_requirements"},
		ForbiddenActions: []string{"write_database", "mark_gap_accepted", "create_approval"},
		GeneratedAt:      created,
	})

	if out.Status != "warn" || out.Mode != "read_only_release_exception_doctor" ||
		out.Scope != "project" || out.ProjectKey != "areamatrix" {
		t.Fatalf("unexpected release exception doctor json: %+v", out)
	}
	if out.Gate.Status != "blocked" || out.Gate.ProjectKey != "areamatrix" {
		t.Fatalf("unexpected release exception doctor gate: %+v", out.Gate)
	}
	if len(out.Checks) != 2 || out.Checks[0].Key != "exception_record_schema" || out.Checks[1].Category != "restore" {
		t.Fatalf("unexpected release exception checks: %+v", out.Checks)
	}
	if len(out.ForbiddenActions) != 3 || out.GeneratedAt == "" {
		t.Fatalf("unexpected release exception guardrails: %+v", out)
	}
	assertReal100Guardrail(
		t,
		out.Real100Status,
		out.ReadinessScope,
		out.Real100Blockers,
		project.ReleasePreviewReadinessScope,
		project.Real100ReleasePreviewBlockers(),
	)
}

func TestReleaseExceptionRecordPreviewToJSON(t *testing.T) {
	created := time.Date(2026, 6, 30, 12, 0, 0, 0, time.UTC)
	out := releaseExceptionRecordPreviewToJSON(project.ReleaseExceptionRecordPreview{
		Status:     "draft",
		Mode:       "read_only_release_exception_record_preview",
		Scope:      "project",
		ProjectKey: "areamatrix",
		Doctor: project.ReleaseExceptionDoctor{
			Status:      "warn",
			Mode:        "read_only_release_exception_doctor",
			Scope:       "project",
			ProjectKey:  "areamatrix",
			GeneratedAt: created,
		},
		Drafts: []project.ReleaseExceptionRecordDraft{
			{
				Key:              "release_exception:restore_plan",
				SourceGateItem:   "gate:accept:restore_plan",
				SourceDecision:   "needs_decision",
				AcceptanceType:   "metadata_only_history",
				Status:           "draft",
				Owner:            "release_owner",
				Reason:           "explicit release acceptance evidence is required",
				RequiredEvidence: []string{"release notes state metadata-only artifacts"},
				AuditActions:     []string{"release.exception.request", "release.exception.approve", "release.exception.revoke"},
				RollbackPlan:     "revoke the exception record and rerun release acceptance gate before release apply",
				ReviewRequired:   true,
				Metadata:         map[string]any{"exception_writable": false},
			},
		},
		Capabilities:     []string{"preview_exception_records"},
		ForbiddenActions: []string{"write_database", "insert_exception_record", "insert_audit_event"},
		GeneratedAt:      created,
	})

	if out.Status != "draft" || out.Mode != "read_only_release_exception_record_preview" ||
		out.Scope != "project" || out.ProjectKey != "areamatrix" {
		t.Fatalf("unexpected release exception record preview json: %+v", out)
	}
	if out.Doctor.Status != "warn" || out.Doctor.ProjectKey != "areamatrix" {
		t.Fatalf("unexpected release exception record doctor: %+v", out.Doctor)
	}
	if len(out.Drafts) != 1 || out.Drafts[0].Key != "release_exception:restore_plan" || out.Drafts[0].Status != "draft" {
		t.Fatalf("unexpected release exception drafts: %+v", out.Drafts)
	}
	if len(out.Drafts[0].AuditActions) != 3 || out.Drafts[0].RollbackPlan == "" || !out.Drafts[0].ReviewRequired {
		t.Fatalf("unexpected release exception draft guidance: %+v", out.Drafts[0])
	}
	if len(out.ForbiddenActions) != 3 || out.GeneratedAt == "" {
		t.Fatalf("unexpected release exception record guardrails: %+v", out)
	}
	assertReal100Guardrail(
		t,
		out.Real100Status,
		out.ReadinessScope,
		out.Real100Blockers,
		project.ReleasePreviewReadinessScope,
		project.Real100ReleasePreviewBlockers(),
	)
}

func TestReleaseExceptionSchemaPreviewToJSON(t *testing.T) {
	created := time.Date(2026, 6, 30, 12, 0, 0, 0, time.UTC)
	out := releaseExceptionSchemaPreviewToJSON(project.ReleaseExceptionSchemaPreview{
		Status:     "needs_approval",
		Mode:       "read_only_release_exception_schema_preview",
		Scope:      "project",
		ProjectKey: "areamatrix",
		RecordPreview: project.ReleaseExceptionRecordPreview{
			Status:      "draft",
			Mode:        "read_only_release_exception_record_preview",
			Scope:       "project",
			ProjectKey:  "areamatrix",
			GeneratedAt: created,
		},
		Tables: []project.ReleaseExceptionSchemaTable{
			{
				Name:    "release_exceptions",
				Purpose: "stores explicit release exception records",
				Columns: []project.ReleaseExceptionSchemaColumn{
					{Name: "exception_key", Type: "TEXT", Nullable: false, Purpose: "stable exception identifier"},
					{Name: "rollback_plan", Type: "TEXT", Nullable: false, Purpose: "rollback plan"},
				},
				Indexes: []project.ReleaseExceptionSchemaIndex{
					{Name: "release_exceptions_key_idx", Columns: []string{"exception_key"}, Unique: true, Purpose: "lookup"},
				},
				ForeignKeys: []project.ReleaseExceptionSchemaForeignKey{
					{Column: "project_id", ReferencesTable: "projects", ReferencesColumn: "id", OnDelete: "CASCADE"},
				},
			},
		},
		ApplySteps: []project.ReleaseExceptionMigrationStep{
			{Order: 1, Action: "create_table", Description: "create release_exceptions table", SQLPreview: "CREATE TABLE IF NOT EXISTS release_exceptions (...)"},
		},
		RollbackSteps: []project.ReleaseExceptionMigrationStep{
			{Order: 1, Action: "drop_table", Description: "drop release_exceptions", SQLPreview: "DROP TABLE IF EXISTS release_exceptions"},
		},
		AuditActions:     []string{"release.exception.request", "release.exception.approve", "release.exception.revoke"},
		Capabilities:     []string{"preview_release_exception_schema"},
		ForbiddenActions: []string{"write_database", "create_migration_file", "run_migration"},
		GeneratedAt:      created,
	})

	if out.Status != "needs_approval" || out.Mode != "read_only_release_exception_schema_preview" ||
		out.Scope != "project" || out.ProjectKey != "areamatrix" {
		t.Fatalf("unexpected release exception schema preview json: %+v", out)
	}
	if out.RecordPreview.Status != "draft" || out.RecordPreview.ProjectKey != "areamatrix" {
		t.Fatalf("unexpected release exception schema record preview: %+v", out.RecordPreview)
	}
	if len(out.Tables) != 1 || out.Tables[0].Name != "release_exceptions" || len(out.Tables[0].Columns) != 2 {
		t.Fatalf("unexpected release exception schema tables: %+v", out.Tables)
	}
	if len(out.ApplySteps) != 1 || out.ApplySteps[0].Action != "create_table" {
		t.Fatalf("unexpected release exception schema apply steps: %+v", out.ApplySteps)
	}
	if len(out.RollbackSteps) != 1 || out.RollbackSteps[0].Action != "drop_table" {
		t.Fatalf("unexpected release exception schema rollback steps: %+v", out.RollbackSteps)
	}
	if len(out.AuditActions) != 3 || len(out.ForbiddenActions) != 3 || out.GeneratedAt == "" {
		t.Fatalf("unexpected release exception schema guardrails: %+v", out)
	}
	assertReal100Guardrail(
		t,
		out.Real100Status,
		out.ReadinessScope,
		out.Real100Blockers,
		project.ReleasePreviewReadinessScope,
		project.Real100ReleasePreviewBlockers(),
	)
}

func TestReleaseExceptionMigrationApprovalGateToJSON(t *testing.T) {
	created := time.Date(2026, 6, 30, 12, 0, 0, 0, time.UTC)
	out := releaseExceptionMigrationApprovalGateToJSON(project.ReleaseExceptionMigrationApprovalGate{
		Status:     "blocked",
		Mode:       "read_only_release_exception_migration_approval_gate",
		Scope:      "project",
		ProjectKey: "areamatrix",
		SchemaPreview: project.ReleaseExceptionSchemaPreview{
			Status:      "needs_approval",
			Mode:        "read_only_release_exception_schema_preview",
			Scope:       "project",
			ProjectKey:  "areamatrix",
			GeneratedAt: created,
		},
		Items: []project.ReleaseExceptionMigrationApprovalGateItem{
			{
				Key:              "migration_approval:release_exception_schema",
				Category:         "migration",
				Status:           "blocked",
				ApprovalStatus:   "needs_approval",
				Message:          "explicit migration approval is required",
				Owner:            "release_owner",
				RequiredEvidence: []string{"approved migration approval record"},
				NextCommand:      "areaflow release exception-schema-preview --json",
				Metadata:         map[string]any{"risk_level": "R4 migration_security", "migration_writable": false},
			},
		},
		Capabilities:     []string{"evaluate_release_exception_migration_approval_gate"},
		ForbiddenActions: []string{"write_database", "create_migration_file", "run_migration", "approve_migration"},
		GeneratedAt:      created,
	})

	if out.Status != "blocked" || out.Mode != "read_only_release_exception_migration_approval_gate" ||
		out.Scope != "project" || out.ProjectKey != "areamatrix" {
		t.Fatalf("unexpected migration approval gate json: %+v", out)
	}
	if out.SchemaPreview.Status != "needs_approval" || out.SchemaPreview.ProjectKey != "areamatrix" {
		t.Fatalf("unexpected nested schema preview: %+v", out.SchemaPreview)
	}
	if len(out.Items) != 1 || out.Items[0].ApprovalStatus != "needs_approval" || out.Items[0].Status != "blocked" {
		t.Fatalf("unexpected migration approval items: %+v", out.Items)
	}
	if out.Items[0].Metadata["migration_writable"] != false {
		t.Fatalf("unexpected migration approval metadata: %+v", out.Items[0].Metadata)
	}
	if len(out.ForbiddenActions) != 4 || out.GeneratedAt == "" {
		t.Fatalf("unexpected migration approval guardrails: %+v", out)
	}
	assertReal100Guardrail(
		t,
		out.Real100Status,
		out.ReadinessScope,
		out.Real100Blockers,
		project.ReleasePreviewReadinessScope,
		project.Real100ReleasePreviewBlockers(),
	)
}

func TestReleaseExceptionApplyPreviewToJSON(t *testing.T) {
	created := time.Date(2026, 6, 30, 12, 0, 0, 0, time.UTC)
	out := releaseExceptionApplyPreviewToJSON(project.ReleaseExceptionApplyPreview{
		Status:     "blocked",
		Mode:       "read_only_release_exception_apply_preview",
		Scope:      "project",
		ProjectKey: "areamatrix",
		MigrationGate: project.ReleaseExceptionMigrationApprovalGate{
			Status:      "blocked",
			Mode:        "read_only_release_exception_migration_approval_gate",
			Scope:       "project",
			ProjectKey:  "areamatrix",
			GeneratedAt: created,
		},
		Items: []project.ReleaseExceptionApplyPreviewItem{
			{
				Key:              "release_exception_apply:migration_approval",
				Category:         "migration",
				Status:           "blocked",
				Action:           "wait_for_migration_approval",
				Message:          "release exception apply is blocked",
				Owner:            "release_owner",
				RequiredEvidence: []string{"release exception migration approval gate returns pass"},
				NextCommand:      "areaflow release exception-migration-approval-gate --json",
				Metadata:         map[string]any{"risk_level": "R4 migration_security", "apply_writable": false},
			},
		},
		ApplySteps: []project.ReleaseExceptionApplyPreviewStep{
			{Order: 1, Action: "verify_migration_approval", Description: "confirm gate passes", BlockedBy: []string{"migration_approval:release_exception_schema"}},
		},
		RollbackSteps: []project.ReleaseExceptionApplyPreviewStep{
			{Order: 1, Action: "disable_exception_writes", Description: "disable writes"},
		},
		Capabilities:     []string{"preview_release_exception_apply_plan"},
		ForbiddenActions: []string{"write_database", "run_migration", "insert_exception_record", "apply_release"},
		GeneratedAt:      created,
	})

	if out.Status != "blocked" || out.Mode != "read_only_release_exception_apply_preview" ||
		out.Scope != "project" || out.ProjectKey != "areamatrix" {
		t.Fatalf("unexpected apply preview json: %+v", out)
	}
	if out.MigrationGate.Status != "blocked" || out.MigrationGate.ProjectKey != "areamatrix" {
		t.Fatalf("unexpected nested migration gate: %+v", out.MigrationGate)
	}
	if len(out.Items) != 1 || out.Items[0].Action != "wait_for_migration_approval" || out.Items[0].Status != "blocked" {
		t.Fatalf("unexpected apply preview items: %+v", out.Items)
	}
	if out.Items[0].Metadata["apply_writable"] != false {
		t.Fatalf("unexpected apply preview metadata: %+v", out.Items[0].Metadata)
	}
	if len(out.ApplySteps) != 1 || out.ApplySteps[0].Action != "verify_migration_approval" || len(out.ApplySteps[0].BlockedBy) != 1 {
		t.Fatalf("unexpected apply steps: %+v", out.ApplySteps)
	}
	if len(out.RollbackSteps) != 1 || out.RollbackSteps[0].Action != "disable_exception_writes" {
		t.Fatalf("unexpected rollback steps: %+v", out.RollbackSteps)
	}
	if len(out.ForbiddenActions) != 4 || out.GeneratedAt == "" {
		t.Fatalf("unexpected apply preview guardrails: %+v", out)
	}
	assertReal100Guardrail(
		t,
		out.Real100Status,
		out.ReadinessScope,
		out.Real100Blockers,
		project.ReleasePreviewReadinessScope,
		project.Real100ReleasePreviewBlockers(),
	)
}

func TestReleaseFinalGateToJSON(t *testing.T) {
	created := time.Date(2026, 6, 30, 12, 0, 0, 0, time.UTC)
	out := releaseFinalGateToJSON(project.ReleaseFinalGate{
		Status:     "blocked",
		Mode:       "read_only_release_final_gate",
		Scope:      "project",
		ProjectKey: "areamatrix",
		Readiness: project.ReleaseReadiness{
			Status:      "needs_attention",
			Mode:        "read_only_release_readiness",
			GeneratedAt: created,
		},
		AcceptanceGate: project.ReleaseAcceptanceGate{
			Status:      "blocked",
			Mode:        "read_only_release_acceptance_gate",
			GeneratedAt: created,
		},
		ExceptionApply: project.ReleaseExceptionApplyPreview{
			Status:      "blocked",
			Mode:        "read_only_release_exception_apply_preview",
			GeneratedAt: created,
		},
		Items: []project.ReleaseFinalGateItem{
			{
				Key:              "final_gate:release_readiness",
				Category:         "readiness",
				Status:           "blocked",
				Message:          "release readiness is not ready",
				Owner:            "release_owner",
				RequiredEvidence: []string{"release readiness status ready"},
				NextCommand:      "areaflow release readiness --json",
				Metadata:         map[string]any{"readiness_status": "needs_attention"},
			},
		},
		Capabilities:     []string{"evaluate_release_final_gate"},
		ForbiddenActions: []string{"write_database", "create_release_package", "apply_release"},
		GeneratedAt:      created,
	})

	if out.Status != "blocked" || out.Mode != "read_only_release_final_gate" ||
		out.Scope != "project" || out.ProjectKey != "areamatrix" {
		t.Fatalf("unexpected final gate json: %+v", out)
	}
	if out.Readiness.Status != "needs_attention" || out.AcceptanceGate.Status != "blocked" || out.ExceptionApply.Status != "blocked" {
		t.Fatalf("unexpected nested final gate sources: %+v", out)
	}
	if len(out.Items) != 1 || out.Items[0].Key != "final_gate:release_readiness" || out.Items[0].Status != "blocked" {
		t.Fatalf("unexpected final gate items: %+v", out.Items)
	}
	if len(out.ForbiddenActions) != 3 || out.GeneratedAt == "" {
		t.Fatalf("unexpected final gate guardrails: %+v", out)
	}
	assertReal100Guardrail(
		t,
		out.Real100Status,
		out.ReadinessScope,
		out.Real100Blockers,
		project.ReleasePreviewReadinessScope,
		project.Real100ReleasePreviewBlockers(),
	)
}

func TestReleaseEvidenceBundleToJSON(t *testing.T) {
	created := time.Date(2026, 6, 30, 12, 0, 0, 0, time.UTC)
	out := releaseEvidenceBundleToJSON(project.ReleaseEvidenceBundle{
		Status:     "blocked",
		Mode:       "read_only_release_evidence_bundle",
		Scope:      "project",
		ProjectKey: "areamatrix",
		BundleHash: "bundle-hash-1",
		FinalGate: project.ReleaseFinalGate{
			Status:      "blocked",
			Mode:        "read_only_release_final_gate",
			GeneratedAt: created,
		},
		Backup: project.BackupManifest{
			Status:        "ready",
			Mode:          "read_only_manifest",
			SchemaVersion: 1,
			ManifestHash:  "abc123",
			GeneratedAt:   created,
		},
		AuditCoverage: project.AuditCoverage{
			Status:      "warn",
			Mode:        "read_only_audit_coverage",
			GeneratedAt: created,
		},
		Items: []project.ReleaseEvidenceBundleItem{
			{
				Key:         "evidence:release_final_gate",
				Category:    "release_gate",
				Status:      "blocked",
				Source:      "release final-gate",
				Description: "release final go/no-go result",
				Metadata:    map[string]any{"final_gate_status": "blocked"},
			},
		},
		Capabilities:     []string{"assemble_release_evidence_index"},
		ForbiddenActions: []string{"create_release_package", "read_artifact_contents", "apply_release"},
		GeneratedAt:      created,
	})

	if out.Status != "blocked" || out.Mode != "read_only_release_evidence_bundle" ||
		out.Scope != "project" || out.ProjectKey != "areamatrix" ||
		out.BundleHash != "bundle-hash-1" {
		t.Fatalf("unexpected evidence bundle json: %+v", out)
	}
	if out.FinalGate.Status != "blocked" || out.Backup.Status != "ready" || out.AuditCoverage.Status != "warn" {
		t.Fatalf("unexpected nested evidence sources: %+v", out)
	}
	if len(out.Items) != 1 || out.Items[0].Key != "evidence:release_final_gate" || out.Items[0].Status != "blocked" {
		t.Fatalf("unexpected evidence items: %+v", out.Items)
	}
	if len(out.ForbiddenActions) != 3 || out.GeneratedAt == "" {
		t.Fatalf("unexpected evidence guardrails: %+v", out)
	}
	assertReal100Guardrail(
		t,
		out.Real100Status,
		out.ReadinessScope,
		out.Real100Blockers,
		project.ReleasePreviewReadinessScope,
		project.Real100ReleasePreviewBlockers(),
	)
}

func TestReleasePackagePreviewToJSON(t *testing.T) {
	created := time.Date(2026, 6, 30, 12, 0, 0, 0, time.UTC)
	out := releasePackagePreviewToJSON(project.ReleasePackagePreview{
		Status:     "blocked",
		Mode:       "read_only_release_package_preview",
		Scope:      "project",
		ProjectKey: "areamatrix",
		EvidenceBundle: project.ReleaseEvidenceBundle{
			Status:      "blocked",
			Mode:        "read_only_release_evidence_bundle",
			BundleHash:  "bundle-hash-1",
			GeneratedAt: created,
		},
		PackageName: "areaflow-v1.0-release-evidence-preview",
		Items: []project.ReleasePackagePreviewItem{
			{
				Key:         "package:manifest",
				Category:    "manifest",
				Status:      "blocked",
				PackagePath: "release/manifest.json",
				Source:      "release evidence-bundle",
				Description: "release package manifest preview",
				Metadata:    map[string]any{"package_writable": false},
			},
		},
		Capabilities:     []string{"preview_release_package_manifest"},
		ForbiddenActions: []string{"create_release_package", "read_artifact_contents", "compress_artifacts"},
		GeneratedAt:      created,
	})

	if out.Status != "blocked" || out.Mode != "read_only_release_package_preview" ||
		out.Scope != "project" || out.ProjectKey != "areamatrix" {
		t.Fatalf("unexpected package preview json: %+v", out)
	}
	if out.EvidenceBundle.Status != "blocked" || out.EvidenceBundle.BundleHash != "bundle-hash-1" ||
		out.PackageName == "" {
		t.Fatalf("unexpected nested package preview state: %+v", out)
	}
	if len(out.Items) != 1 || out.Items[0].Key != "package:manifest" || out.Items[0].PackagePath != "release/manifest.json" {
		t.Fatalf("unexpected package preview items: %+v", out.Items)
	}
	if out.Items[0].Metadata["package_writable"] != false {
		t.Fatalf("unexpected package preview metadata: %+v", out.Items[0].Metadata)
	}
	if len(out.ForbiddenActions) != 3 || out.GeneratedAt == "" {
		t.Fatalf("unexpected package preview guardrails: %+v", out)
	}
	assertReal100Guardrail(
		t,
		out.Real100Status,
		out.ReadinessScope,
		out.Real100Blockers,
		project.ReleasePreviewReadinessScope,
		project.Real100ReleasePreviewBlockers(),
	)
}

func TestReleaseDistributionPreviewToJSON(t *testing.T) {
	created := time.Date(2026, 6, 30, 12, 0, 0, 0, time.UTC)
	out := releaseDistributionPreviewToJSON(project.ReleaseDistributionPreview{
		Status:     "blocked",
		Mode:       "read_only_release_distribution_preview",
		Scope:      "project",
		ProjectKey: "areamatrix",
		PackagePreview: project.ReleasePackagePreview{
			Status:      "blocked",
			Mode:        "read_only_release_package_preview",
			Scope:       "project",
			ProjectKey:  "areamatrix",
			PackageName: "areaflow-v1.0-release-evidence-preview",
			GeneratedAt: created,
		},
		Items: []project.ReleaseDistributionPreviewItem{
			{
				Key:              "distribution:package_preview",
				Category:         "package",
				Status:           "blocked",
				Channel:          "release_package",
				Action:           "wait_for_package_preview",
				Message:          "release distribution is blocked until package preview is ready",
				Owner:            "release-owner",
				RequiredEvidence: []string{"release package preview ready"},
				NextCommand:      "areaflow release package-preview --json",
				Metadata:         map[string]any{"package_writable": false},
			},
		},
		Capabilities:     []string{"preview_release_distribution_channels"},
		ForbiddenActions: []string{"publish_release", "create_git_tag", "sign_release"},
		GeneratedAt:      created,
	})

	if out.Status != "blocked" || out.Mode != "read_only_release_distribution_preview" ||
		out.Scope != "project" || out.ProjectKey != "areamatrix" {
		t.Fatalf("unexpected distribution preview json: %+v", out)
	}
	if out.PackagePreview.Status != "blocked" || out.PackagePreview.PackageName == "" ||
		out.PackagePreview.ProjectKey != "areamatrix" {
		t.Fatalf("unexpected nested distribution preview state: %+v", out)
	}
	if len(out.Items) != 1 || out.Items[0].Key != "distribution:package_preview" || out.Items[0].Channel != "release_package" {
		t.Fatalf("unexpected distribution preview items: %+v", out.Items)
	}
	if out.Items[0].Action != "wait_for_package_preview" || out.Items[0].NextCommand != "areaflow release package-preview --json" {
		t.Fatalf("unexpected distribution preview action: %+v", out.Items[0])
	}
	if out.Items[0].Metadata["package_writable"] != false {
		t.Fatalf("unexpected distribution preview metadata: %+v", out.Items[0].Metadata)
	}
	if len(out.ForbiddenActions) != 3 || out.GeneratedAt == "" {
		t.Fatalf("unexpected distribution preview guardrails: %+v", out)
	}
	assertReal100Guardrail(
		t,
		out.Real100Status,
		out.ReadinessScope,
		out.Real100Blockers,
		project.ReleasePreviewReadinessScope,
		project.Real100ReleasePreviewBlockers(),
	)
}

func TestReleasePublishGateToJSON(t *testing.T) {
	created := time.Date(2026, 6, 30, 12, 0, 0, 0, time.UTC)
	out := releasePublishGateToJSON(project.ReleasePublishGate{
		Status:     "blocked",
		Mode:       "read_only_release_publish_gate",
		Scope:      "project",
		ProjectKey: "areamatrix",
		DistributionPreview: project.ReleaseDistributionPreview{
			Status:      "blocked",
			Mode:        "read_only_release_distribution_preview",
			Scope:       "project",
			ProjectKey:  "areamatrix",
			GeneratedAt: created,
		},
		Items: []project.ReleasePublishGateItem{
			{
				Key:              "publish_gate:distribution_preview",
				Category:         "distribution_preview",
				Status:           "blocked",
				Channel:          "all",
				Message:          "release distribution preview blocks publish",
				Owner:            "release-owner",
				RequiredEvidence: []string{"release distribution preview status ready"},
				NextCommand:      "areaflow release distribution-preview --json",
				Metadata:         map[string]any{"publish_writable": false},
			},
		},
		Capabilities:     []string{"evaluate_release_publish_gate"},
		ForbiddenActions: []string{"publish_release", "create_git_tag", "push_git"},
		GeneratedAt:      created,
	})

	if out.Status != "blocked" || out.Mode != "read_only_release_publish_gate" ||
		out.Scope != "project" || out.ProjectKey != "areamatrix" {
		t.Fatalf("unexpected publish gate json: %+v", out)
	}
	if out.DistributionPreview.Status != "blocked" || out.DistributionPreview.ProjectKey != "areamatrix" {
		t.Fatalf("unexpected nested publish gate state: %+v", out)
	}
	if len(out.Items) != 1 || out.Items[0].Key != "publish_gate:distribution_preview" || out.Items[0].Channel != "all" {
		t.Fatalf("unexpected publish gate items: %+v", out.Items)
	}
	if out.Items[0].Metadata["publish_writable"] != false {
		t.Fatalf("unexpected publish gate metadata: %+v", out.Items[0].Metadata)
	}
	if len(out.ForbiddenActions) != 3 || out.GeneratedAt == "" {
		t.Fatalf("unexpected publish gate guardrails: %+v", out)
	}
	assertReal100Guardrail(
		t,
		out.Real100Status,
		out.ReadinessScope,
		out.Real100Blockers,
		project.ReleasePreviewReadinessScope,
		project.Real100ReleasePreviewBlockers(),
	)
}

func TestReleasePublishApprovalPreviewToJSON(t *testing.T) {
	created := time.Date(2026, 6, 30, 12, 0, 0, 0, time.UTC)
	out := releasePublishApprovalPreviewToJSON(project.ReleasePublishApprovalPreview{
		Status:     "blocked",
		Mode:       "read_only_release_publish_approval_preview",
		Scope:      "project",
		ProjectKey: "areamatrix",
		PublishGate: project.ReleasePublishGate{
			Status:      "blocked",
			Mode:        "read_only_release_publish_gate",
			Scope:       "project",
			ProjectKey:  "areamatrix",
			GeneratedAt: created,
		},
		Items: []project.ReleasePublishApprovalPreviewItem{
			{
				Key:              "publish_approval:publish_gate",
				Category:         "publish_gate",
				Status:           "blocked",
				ApprovalStatus:   "blocked",
				Channel:          "all",
				Message:          "release publish approval cannot be requested until publish gate passes",
				Owner:            "release-owner",
				RequiredEvidence: []string{"publish gate pass"},
				NextCommand:      "areaflow release publish-gate --json",
				Metadata:         map[string]any{"approval_writable": false},
			},
		},
		Capabilities:     []string{"preview_release_publish_approval"},
		ForbiddenActions: []string{"create_approval", "approve_release", "publish_release"},
		GeneratedAt:      created,
	})

	if out.Status != "blocked" || out.Mode != "read_only_release_publish_approval_preview" ||
		out.Scope != "project" || out.ProjectKey != "areamatrix" {
		t.Fatalf("unexpected publish approval preview json: %+v", out)
	}
	if out.PublishGate.Status != "blocked" || out.PublishGate.ProjectKey != "areamatrix" {
		t.Fatalf("unexpected nested publish approval preview state: %+v", out)
	}
	if len(out.Items) != 1 || out.Items[0].Key != "publish_approval:publish_gate" || out.Items[0].ApprovalStatus != "blocked" {
		t.Fatalf("unexpected publish approval preview items: %+v", out.Items)
	}
	if out.Items[0].Metadata["approval_writable"] != false {
		t.Fatalf("unexpected publish approval preview metadata: %+v", out.Items[0].Metadata)
	}
	if len(out.ForbiddenActions) != 3 || out.GeneratedAt == "" {
		t.Fatalf("unexpected publish approval preview guardrails: %+v", out)
	}
	assertReal100Guardrail(
		t,
		out.Real100Status,
		out.ReadinessScope,
		out.Real100Blockers,
		project.ReleasePreviewReadinessScope,
		project.Real100ReleasePreviewBlockers(),
	)
}

func TestReleaseRolloutPlanPreviewToJSON(t *testing.T) {
	created := time.Date(2026, 6, 30, 12, 0, 0, 0, time.UTC)
	out := releaseRolloutPlanPreviewToJSON(project.ReleaseRolloutPlanPreview{
		Status:     "blocked",
		Mode:       "read_only_release_rollout_plan_preview",
		Scope:      "project",
		ProjectKey: "areamatrix",
		PublishApprovalPreview: project.ReleasePublishApprovalPreview{
			Status:      "blocked",
			Mode:        "read_only_release_publish_approval_preview",
			Scope:       "project",
			ProjectKey:  "areamatrix",
			GeneratedAt: created,
		},
		Items: []project.ReleaseRolloutPlanPreviewItem{
			{
				Key:              "rollout_plan:publish_approval",
				Category:         "publish_approval",
				Status:           "blocked",
				Stage:            "preflight",
				Action:           "wait_for_publish_approval_preview",
				Message:          "release rollout plan is blocked until publish approval preview is no longer blocked",
				Owner:            "release-owner",
				RequiredEvidence: []string{"publish approval preview ready"},
				NextCommand:      "areaflow release publish-approval-preview --json",
				Metadata:         map[string]any{"rollout_writable": false},
			},
		},
		RolloutSteps: []project.ReleaseRolloutPlanPreviewStep{
			{Order: 1, Stage: "preflight", Action: "verify_publish_approval", Description: "confirm approval"},
		},
		VerificationCheckpoints: []project.ReleaseRolloutPlanPreviewStep{
			{Order: 1, Stage: "approval", Action: "publish_approval_recorded", Description: "approval recorded"},
		},
		RollbackSteps: []project.ReleaseRolloutPlanPreviewStep{
			{Order: 1, Stage: "pause", Action: "pause_distribution", Description: "pause channels"},
		},
		Capabilities:     []string{"preview_release_rollout_plan"},
		ForbiddenActions: []string{"create_rollout", "write_release_state", "publish_release"},
		GeneratedAt:      created,
	})

	if out.Status != "blocked" || out.Mode != "read_only_release_rollout_plan_preview" ||
		out.Scope != "project" || out.ProjectKey != "areamatrix" {
		t.Fatalf("unexpected rollout plan preview json: %+v", out)
	}
	if out.PublishApprovalPreview.Status != "blocked" || out.PublishApprovalPreview.ProjectKey != "areamatrix" {
		t.Fatalf("unexpected nested rollout plan preview state: %+v", out)
	}
	if len(out.Items) != 1 || out.Items[0].Key != "rollout_plan:publish_approval" || out.Items[0].Action != "wait_for_publish_approval_preview" {
		t.Fatalf("unexpected rollout plan preview items: %+v", out.Items)
	}
	if out.Items[0].Metadata["rollout_writable"] != false {
		t.Fatalf("unexpected rollout plan preview metadata: %+v", out.Items[0].Metadata)
	}
	if len(out.RolloutSteps) != 1 || out.RolloutSteps[0].Action != "verify_publish_approval" {
		t.Fatalf("unexpected rollout steps: %+v", out.RolloutSteps)
	}
	if len(out.VerificationCheckpoints) != 1 || out.VerificationCheckpoints[0].Action != "publish_approval_recorded" {
		t.Fatalf("unexpected rollout verification checkpoints: %+v", out.VerificationCheckpoints)
	}
	if len(out.RollbackSteps) != 1 || out.RollbackSteps[0].Action != "pause_distribution" {
		t.Fatalf("unexpected rollout rollback steps: %+v", out.RollbackSteps)
	}
	if len(out.ForbiddenActions) != 3 || out.GeneratedAt == "" {
		t.Fatalf("unexpected rollout plan preview guardrails: %+v", out)
	}
	assertReal100Guardrail(
		t,
		out.Real100Status,
		out.ReadinessScope,
		out.Real100Blockers,
		project.ReleasePreviewReadinessScope,
		project.Real100ReleasePreviewBlockers(),
	)
}

func TestAuditCoverageToJSON(t *testing.T) {
	created := time.Date(2026, 6, 30, 11, 0, 0, 0, time.UTC)
	out := auditCoverageToJSON(project.AuditCoverage{
		Status:              "warn",
		Mode:                "read_only_audit_coverage",
		Scope:               "project",
		ProjectID:           1,
		ProjectKey:          "areamatrix",
		TotalAuditEvents:    2,
		CoveredRequirements: 1,
		GapRequirements:     1,
		Requirements: []project.AuditCoverageRequirement{
			{
				Key:           "project_registration",
				Category:      "write",
				Description:   "project writes are audited",
				Status:        "pass",
				EvidenceCount: 1,
				RequiredActions: []project.AuditCoverageActionEvidence{
					{Action: "project.upsert", Decision: "allowed", Count: 1, Status: "pass", LastAuditAt: &created},
				},
				LastAuditAt: &created,
			},
			{
				Key:           "secret_resolution",
				Category:      "secret",
				Description:   "secret resolution is audited",
				Status:        "gap",
				EvidenceCount: 0,
				RequiredActions: []project.AuditCoverageActionEvidence{
					{Action: "secret.resolve", Count: 0, Status: "gap"},
				},
				MissingActions: []string{"secret.resolve"},
			},
		},
		GeneratedAt: created,
	})

	if out.Status != "warn" || out.Scope != "project" || out.ProjectKey != "areamatrix" {
		t.Fatalf("unexpected audit coverage json: %+v", out)
	}
	if out.CoveredRequirements != 1 || out.GapRequirements != 1 || len(out.Requirements) != 2 {
		t.Fatalf("unexpected audit coverage counts: %+v", out)
	}
	if out.Requirements[0].RequiredActions[0].LastAuditAt == "" {
		t.Fatalf("expected action evidence timestamp: %+v", out.Requirements[0])
	}
	if len(out.Requirements[1].MissingActions) != 1 || out.Requirements[1].MissingActions[0] != "secret.resolve" {
		t.Fatalf("unexpected missing actions: %+v", out.Requirements[1])
	}
}

func TestPermissionPolicyDoctorToJSON(t *testing.T) {
	created := time.Date(2026, 6, 30, 12, 0, 0, 0, time.UTC)
	out := permissionPolicyDoctorToJSON(project.PermissionPolicyDoctor{
		Status: "pass",
		Mode:   "read_only_permission_policy_doctor",
		Project: project.Record{
			Key:             "areamatrix",
			Name:            "AreaMatrix",
			Adapter:         "areamatrix",
			WorkflowProfile: "areamatrix",
		},
		Checks: []project.PermissionPolicyCheck{
			{
				Key:      "default_read_only",
				Category: "capability",
				Status:   "pass",
				Message:  "high-risk capabilities are disabled by default",
				Metadata: map[string]any{},
			},
			{
				Key:      "status_export_write",
				Category: "path",
				Status:   "pass",
				Message:  "status export path is explicitly allowed and not denied",
				Metadata: map[string]any{"path": ".areaflow/status.json"},
			},
		},
		GeneratedAt: created,
	})

	if out.Status != "pass" || out.Mode != "read_only_permission_policy_doctor" || out.Project.Key != "areamatrix" {
		t.Fatalf("unexpected permission doctor json: %+v", out)
	}
	if len(out.Checks) != 2 || out.Checks[1].Metadata["path"] != ".areaflow/status.json" {
		t.Fatalf("unexpected permission doctor checks: %+v", out.Checks)
	}
	if out.GeneratedAt == "" {
		t.Fatalf("expected generated_at: %+v", out)
	}
}

func TestArtifactIntegrityToJSON(t *testing.T) {
	created := time.Date(2026, 6, 30, 12, 0, 0, 0, time.UTC)
	out := artifactIntegrityToJSON(project.ArtifactIntegrityReport{
		Status:           "warn",
		Mode:             "read_only_artifact_integrity",
		Project:          project.Record{Key: "areamatrix", Name: "AreaMatrix", Adapter: "areamatrix", WorkflowProfile: "areamatrix"},
		CheckedArtifacts: 2,
		PassedArtifacts:  1,
		SkippedArtifacts: 1,
		GeneratedAt:      created,
		Checks: []project.ArtifactIntegrityCheck{
			{
				Artifact: project.ArtifactRecord{ID: 7, ArtifactType: "runner_preview_report", StorageBackend: "local", URI: "/tmp/report.json", SHA256: "abc123", SizeBytes: 12},
				Status:   "pass",
				Message:  "local artifact hash and size match metadata",
				Metadata: map[string]any{"read_contents": true},
			},
			{
				Artifact: project.ArtifactRecord{ID: 8, ArtifactType: "source_ref", StorageBackend: "external_project", URI: "workflow/file.md", SourcePath: "workflow/file.md", SHA256: "def456", SizeBytes: 20},
				Status:   "skipped",
				Message:  "referenced project artifact content remains in managed project",
				Metadata: map[string]any{"read_contents": false},
			},
		},
	})

	if out.Status != "warn" || out.Mode != "read_only_artifact_integrity" || out.Project.Key != "areamatrix" {
		t.Fatalf("unexpected artifact integrity json: %+v", out)
	}
	if out.CheckedArtifacts != 2 || out.PassedArtifacts != 1 || out.SkippedArtifacts != 1 {
		t.Fatalf("unexpected artifact integrity counters: %+v", out)
	}
	if len(out.Checks) != 2 || out.Checks[0].Artifact.ArtifactType != "runner_preview_report" || out.Checks[1].Status != "skipped" {
		t.Fatalf("unexpected artifact integrity checks: %+v", out.Checks)
	}
	if out.GeneratedAt == "" {
		t.Fatalf("expected generated_at: %+v", out)
	}
}

func TestArtifactArchivePreviewToJSON(t *testing.T) {
	created := time.Date(2026, 6, 30, 12, 5, 0, 0, time.UTC)
	out := artifactArchivePreviewToJSON(project.ArtifactArchivePreviewResult{
		Project: project.Record{Key: "areamatrix", Name: "AreaMatrix", Adapter: "areamatrix", WorkflowProfile: "areamatrix"},
		Status:  "needs_attention",
		Mode:    "metadata_only_archive_preview",
		Summary: project.ArtifactArchivePreviewSummary{
			TotalArtifacts:    2,
			ArchiveCandidates: 1,
			ExternalRefs:      1,
		},
		Items: []project.ArtifactArchivePreviewItem{{
			ArtifactID:     7,
			ArtifactType:   "runner_preview_report",
			StorageBackend: "local",
			RetentionClass: "ephemeral",
			ArchiveState:   "archive_candidate",
			Action:         "eligible_for_future_gc_preview",
			Decision:       "preview_only",
		}},
		EventID:                 12,
		AuditEventID:            13,
		IdempotencyKey:          "artifact.archive.preview:test",
		Created:                 true,
		GeneratedAt:             created,
		ProjectWriteAttempted:   false,
		StorageWriteAttempted:   false,
		ArtifactDeleteAttempted: false,
	})

	if out.Status != "needs_attention" || out.Mode != "metadata_only_archive_preview" || out.Project.Key != "areamatrix" {
		t.Fatalf("unexpected archive preview json: %+v", out)
	}
	if out.Summary.ArchiveCandidates != 1 || out.Summary.ExternalRefs != 1 || len(out.Items) != 1 {
		t.Fatalf("unexpected archive preview summary/items: %+v", out)
	}
	if out.ProjectWriteAttempted || out.StorageWriteAttempted || out.ArtifactDeleteAttempted {
		t.Fatalf("archive preview should not attempt writes/deletes: %+v", out)
	}
	if out.GeneratedAt == "" || out.IdempotencyKey == "" {
		t.Fatalf("expected generated_at and idempotency key: %+v", out)
	}
}

func TestConformanceToJSON(t *testing.T) {
	created := time.Date(2026, 6, 30, 12, 0, 0, 0, time.UTC)
	out := conformanceToJSON(project.ConformanceReport{
		Status:      "pass",
		Mode:        "read_only_adapter_profile_conformance",
		Project:     project.Record{Key: "areamatrix", Name: "AreaMatrix", Adapter: "areamatrix", WorkflowProfile: "areamatrix"},
		ProfileID:   "areamatrix",
		Adapter:     "areamatrix",
		ProfileHash: "abc123",
		StageCount:  16,
		GateCount:   17,
		GeneratedAt: created,
		Checks: []project.ConformanceCheck{
			{
				Key:      "project_adapter_profile",
				Category: "binding",
				Status:   "pass",
				Message:  "project adapter/profile binding matches loaded workflow profile defaults",
				Metadata: map[string]any{"profile_id": "areamatrix"},
			},
			{
				Key:      "adapter_snapshot",
				Category: "adapter",
				Status:   "pass",
				Message:  "AreaMatrix adapter can load a read-only project snapshot",
				Metadata: map[string]any{"versions": 2},
			},
		},
	})

	if out.Status != "pass" || out.Mode != "read_only_adapter_profile_conformance" || out.Project.Key != "areamatrix" {
		t.Fatalf("unexpected conformance json: %+v", out)
	}
	if out.ProfileID != "areamatrix" || out.Adapter != "areamatrix" || out.StageCount != 16 || out.GateCount != 17 {
		t.Fatalf("unexpected conformance summary: %+v", out)
	}
	if len(out.Checks) != 2 || out.Checks[0].Key != "project_adapter_profile" || out.Checks[1].Metadata["versions"] != 2 {
		t.Fatalf("unexpected conformance checks: %+v", out.Checks)
	}
	if out.GeneratedAt == "" {
		t.Fatalf("expected generated_at: %+v", out)
	}
}

func TestWorkerToJSON(t *testing.T) {
	created := time.Date(2026, 6, 29, 4, 0, 0, 0, time.UTC)
	heartbeat := created.Add(time.Second)
	out := workerToJSON(project.WorkerRecord{
		ID:                       7,
		ProjectID:                1,
		ActorID:                  8,
		WorkerKey:                "local-1",
		WorkerType:               "local_host",
		Status:                   "online",
		Hostname:                 "dev-host",
		PID:                      123,
		Capabilities:             []string{"read_project"},
		Metadata:                 map[string]any{"mode": "v0.6a"},
		RegisteredAt:             created,
		LastHeartbeatAt:          &heartbeat,
		HeartbeatIntervalSeconds: 30,
		LeaseTimeoutSeconds:      300,
		UpdatedAt:                heartbeat,
	})
	if out.WorkerKey != "local-1" || out.LastHeartbeatAt == "" {
		t.Fatalf("unexpected worker json: %+v", out)
	}
	if len(out.Capabilities) != 1 || out.Capabilities[0] != "read_project" {
		t.Fatalf("unexpected worker capabilities json: %+v", out)
	}
}

func TestLeaseToJSON(t *testing.T) {
	created := time.Date(2026, 6, 29, 4, 10, 0, 0, time.UTC)
	expires := created.Add(5 * time.Minute)
	out := leaseToJSON(project.LeaseRecord{
		ID:                  9,
		ProjectID:           1,
		RunID:               3,
		RunTaskID:           4,
		WorkerID:            7,
		LeaseKind:           "run_task",
		Status:              "active",
		AcquiredAt:          created,
		ExpiresAt:           expires,
		HeartbeatAt:         &created,
		AllowedCapabilities: []string{"read_project"},
		Scope:               map[string]any{"run_task_id": float64(4)},
		Metadata:            map[string]any{"dry_run": true},
	})
	if out.ID != 9 || out.RunTaskID != 4 || out.HeartbeatAt == "" {
		t.Fatalf("unexpected lease json: %+v", out)
	}
	if len(out.AllowedCapabilities) != 1 || out.AllowedCapabilities[0] != "read_project" {
		t.Fatalf("unexpected lease capabilities json: %+v", out)
	}
}

func TestWorkerRunOnceToJSON(t *testing.T) {
	created := time.Date(2026, 6, 29, 4, 20, 0, 0, time.UTC)
	expires := created.Add(5 * time.Minute)
	out := workerRunOnceToJSON(project.WorkerRunOnceResult{
		Project: project.Record{
			Key: "areamatrix",
		},
		Worker: project.WorkerRecord{
			ID:                       7,
			ProjectID:                1,
			WorkerKey:                "local-1",
			WorkerType:               "local_host",
			Status:                   "online",
			Capabilities:             []string{"read_project"},
			Metadata:                 map[string]any{},
			RegisteredAt:             created,
			LastHeartbeatAt:          &created,
			HeartbeatIntervalSeconds: 30,
			LeaseTimeoutSeconds:      300,
			UpdatedAt:                created,
		},
		Lease: project.LeaseRecord{
			ID:                  9,
			ProjectID:           1,
			RunID:               3,
			RunTaskID:           4,
			WorkerID:            7,
			LeaseKind:           "run_task",
			Status:              "completed",
			AcquiredAt:          created,
			ExpiresAt:           expires,
			AllowedCapabilities: []string{"read_project"},
			Scope:               map[string]any{"run_once": true},
			Metadata:            map[string]any{"dry_run": true},
		},
		Task: project.RunTaskRecord{
			ID:                4,
			ProjectID:         1,
			WorkflowVersionID: 2,
			RunID:             3,
			TaskKey:           "v2:runner-preview",
			TaskKind:          "workflow_item_preview",
			Status:            "passed",
			RiskLevel:         "low",
			Sequence:          1,
			Metadata:          map[string]any{"dry_run": true},
			CreatedAt:         created,
			UpdatedAt:         created,
		},
		Attempt: project.RunAttemptRecord{
			ID:                10,
			ProjectID:         1,
			WorkflowVersionID: 2,
			RunID:             3,
			RunTaskID:         4,
			AttemptKind:       "worker_run_once",
			Status:            "passed",
			DryRun:            true,
			Metadata:          map[string]any{"would_execute": false},
			StartedAt:         created,
		},
		Artifact: project.ArtifactRecord{
			ID:                11,
			ProjectID:         1,
			WorkflowVersionID: 2,
			ArtifactType:      "worker_run_once_report",
			StorageBackend:    "local",
			URI:               "/tmp/areaflow/artifacts/areamatrix/workers/local-1/run-once/run-task-4-report.json",
			SourcePath:        "workers/local-1/run-once/run-task-4-report.json",
			SHA256:            "def456",
			SizeBytes:         256,
			ContentType:       "application/json",
			Metadata:          map[string]any{"dry_run": true},
			CreatedAt:         created,
		},
		Claimed: true,
	})
	if !out.Claimed || out.Lease == nil || out.Task == nil || out.Attempt == nil || out.Artifact == nil {
		t.Fatalf("unexpected worker run-once json: %+v", out)
	}
	if out.Lease.Status != "completed" || out.Task.ID != 4 {
		t.Fatalf("unexpected worker run-once lease/task json: %+v", out)
	}
	if out.Attempt.AttemptKind != "worker_run_once" || !out.Attempt.DryRun {
		t.Fatalf("unexpected worker run-once attempt json: %+v", out.Attempt)
	}
	if out.Artifact.ArtifactType != "worker_run_once_report" || out.Artifact.SHA256 != "def456" {
		t.Fatalf("unexpected worker run-once artifact json: %+v", out.Artifact)
	}
}

func TestApprovedArtifactWriteQueueToJSON(t *testing.T) {
	created := time.Date(2026, 6, 29, 4, 25, 0, 0, time.UTC)
	out := approvedArtifactWriteQueueToJSON(project.ApprovedArtifactWriteQueueResult{
		Project:                       project.Record{Key: "areamatrix"},
		Version:                       project.WorkflowVersion{ID: 2, DisplayLabel: "v2", ImportMode: "authored"},
		Run:                           project.RunRecord{ID: 3, WorkflowVersionID: 2, RunType: "approved_artifact_write", RunKind: "execution", Status: "queued", DryRun: false, StartedAt: created},
		Task:                          project.RunTaskRecord{ID: 4, WorkflowVersionID: 2, RunID: 3, TaskKind: "approved_artifact_write_task", Status: "queued", CreatedAt: created, UpdatedAt: created},
		ArtifactLabel:                 "approval-note",
		Created:                       true,
		IdempotencyKey:                "approved-artifact-write-queue:test",
		EventID:                       5,
		AuditEventID:                  6,
		ProjectReadAttempted:          false,
		ProjectWriteAttempted:         false,
		ExecutionWriteAttempted:       false,
		AreaFlowArtifactWritten:       false,
		AreaFlowExecutionStateWritten: true,
		EngineCallAttempted:           false,
		CommandsRun:                   false,
		SecretsResolved:               false,
		NetworkUsed:                   false,
	})
	if out.ArtifactLabel != "approval-note" || out.Run.ID != 3 || out.Task.ID != 4 || !out.Created {
		t.Fatalf("unexpected approved artifact write queue json: %+v", out)
	}
	if out.ProjectReadAttempted || out.ProjectWriteAttempted || out.ExecutionWriteAttempted || out.AreaFlowArtifactWritten || !out.AreaFlowExecutionStateWritten {
		t.Fatalf("unexpected approved artifact write queue safety facts: %+v", out)
	}
}

func TestFixtureProjectWriteQueueToJSON(t *testing.T) {
	created := time.Date(2026, 6, 29, 4, 26, 0, 0, time.UTC)
	out := fixtureProjectWriteQueueToJSON(project.FixtureProjectWriteQueueResult{
		Project:                       project.Record{Key: "areamatrix-fixture"},
		Version:                       project.WorkflowVersion{ID: 2, DisplayLabel: "v2", ImportMode: "authored"},
		Run:                           project.RunRecord{ID: 3, WorkflowVersionID: 2, RunType: "fixture_project_write", RunKind: "execution", Status: "queued", DryRun: false, StartedAt: created},
		Task:                          project.RunTaskRecord{ID: 4, WorkflowVersionID: 2, RunID: 3, TaskKind: "fixture_project_write_task", Status: "queued", CreatedAt: created, UpdatedAt: created},
		WriteSetArtifact:              project.ArtifactRecord{ID: 5, ArtifactType: "fixture_project_write_set", SHA256: "writeset123", CreatedAt: created},
		TargetPath:                    "fixtures/input.txt",
		ExpectedBeforeSHA256:          "before123",
		ExpectedBeforeSize:            12,
		AfterSHA256:                   "after123",
		AfterSize:                     13,
		Created:                       true,
		IdempotencyKey:                "fixture-project-write-queue:test",
		EventID:                       6,
		AuditEventID:                  7,
		ProjectReadAttempted:          false,
		ProjectWriteAttempted:         false,
		ExecutionWriteAttempted:       false,
		AreaFlowArtifactWritten:       true,
		AreaFlowExecutionStateWritten: true,
		EngineCallAttempted:           false,
		CommandsRun:                   false,
		SecretsResolved:               false,
		NetworkUsed:                   false,
	})
	if out.TargetPath != "fixtures/input.txt" || out.Run.ID != 3 || out.Task.ID != 4 || out.WriteSetArtifact.ArtifactType != "fixture_project_write_set" || !out.Created {
		t.Fatalf("unexpected fixture project write queue json: %+v", out)
	}
	if out.ProjectReadAttempted || out.ProjectWriteAttempted || out.ExecutionWriteAttempted || !out.AreaFlowArtifactWritten || !out.AreaFlowExecutionStateWritten {
		t.Fatalf("unexpected fixture project write queue safety facts: %+v", out)
	}
}

func TestManagedGeneratedWriteQueueToJSON(t *testing.T) {
	created := time.Date(2026, 7, 2, 4, 26, 0, 0, time.UTC)
	out := managedGeneratedWriteQueueToJSON(project.ManagedGeneratedWriteQueueResult{
		Project:                       project.Record{Key: "areamatrix-fixture"},
		Version:                       project.WorkflowVersion{ID: 2, DisplayLabel: "v2", ImportMode: "authored"},
		Run:                           project.RunRecord{ID: 3, WorkflowVersionID: 2, RunType: "managed_generated_write", RunKind: "execution", Status: "queued", DryRun: false, StartedAt: created},
		Task:                          project.RunTaskRecord{ID: 4, WorkflowVersionID: 2, RunID: 3, TaskKind: "managed_generated_write_task", Status: "queued", CreatedAt: created, UpdatedAt: created},
		WriteSetArtifact:              project.ArtifactRecord{ID: 5, ArtifactType: "managed_generated_write_set", SHA256: "writeset123", CreatedAt: created},
		TargetPath:                    ".areaflow/generated/status.json",
		ExpectedBeforeSHA256:          "before123",
		ExpectedBeforeSize:            12,
		AfterSHA256:                   "after123",
		AfterSize:                     13,
		Created:                       true,
		IdempotencyKey:                "managed-generated-write-queue:test",
		EventID:                       6,
		AuditEventID:                  7,
		GeneratedOnly:                 true,
		GeneratedOnlyApplyOpen:        true,
		ProjectReadAttempted:          false,
		ProjectWriteAttempted:         false,
		ExecutionWriteAttempted:       false,
		AreaFlowArtifactWritten:       true,
		AreaFlowExecutionStateWritten: true,
		EngineCallAttempted:           false,
		CommandsRun:                   false,
		SecretsResolved:               false,
		NetworkUsed:                   false,
	})
	if out.TargetPath != ".areaflow/generated/status.json" || out.Run.ID != 3 || out.Task.ID != 4 || out.WriteSetArtifact.ArtifactType != "managed_generated_write_set" || !out.Created {
		t.Fatalf("unexpected managed generated write queue json: %+v", out)
	}
	if !out.GeneratedOnly || !out.GeneratedOnlyApplyOpen || out.ProjectReadAttempted || out.ProjectWriteAttempted || out.ExecutionWriteAttempted || !out.AreaFlowArtifactWritten || !out.AreaFlowExecutionStateWritten || out.EngineCallAttempted || out.CommandsRun || out.SecretsResolved || out.NetworkUsed {
		t.Fatalf("unexpected managed generated write queue safety facts: %+v", out)
	}
}

func TestApprovedArtifactWriteToJSON(t *testing.T) {
	created := time.Date(2026, 6, 29, 4, 30, 0, 0, time.UTC)
	expires := created.Add(5 * time.Minute)
	run := project.RunRecord{ID: 3, WorkflowVersionID: 2, RunType: "approved_artifact_write", RunKind: "execution", Status: "artifact_written", DryRun: false, StartedAt: created}
	out := approvedArtifactWriteToJSON(project.ApprovedArtifactWriteResult{
		Project: project.Record{Key: "areamatrix"},
		Version: project.WorkflowVersion{ID: 2, DisplayLabel: "v2", ImportMode: "authored"},
		Run:     run,
		Worker: project.WorkerRecord{
			ID:                       7,
			ProjectID:                1,
			WorkerKey:                "local-1",
			WorkerType:               "local_host",
			Status:                   "online",
			Capabilities:             []string{"write_artifacts"},
			Metadata:                 map[string]any{},
			RegisteredAt:             created,
			LastHeartbeatAt:          &created,
			HeartbeatIntervalSeconds: 30,
			LeaseTimeoutSeconds:      300,
			UpdatedAt:                created,
		},
		Lease: project.LeaseRecord{
			ID:                  8,
			ProjectID:           1,
			RunID:               3,
			RunTaskID:           4,
			WorkerID:            7,
			LeaseKind:           "approved_artifact_write",
			Status:              "completed",
			AcquiredAt:          created,
			ExpiresAt:           expires,
			AllowedCapabilities: []string{"write_artifacts"},
			Scope:               map[string]any{"approved_artifact_write": true},
			Metadata:            map[string]any{},
		},
		Task: project.RunTaskRecord{
			ID:                4,
			ProjectID:         1,
			WorkflowVersionID: 2,
			RunID:             3,
			TaskKind:          "approved_artifact_write_task",
			Status:            "artifact_written",
			CreatedAt:         created,
			UpdatedAt:         created,
		},
		Attempt: project.RunAttemptRecord{
			ID:                9,
			ProjectID:         1,
			WorkflowVersionID: 2,
			RunID:             3,
			RunTaskID:         4,
			AttemptKind:       "approved_artifact_write",
			Status:            "passed",
			DryRun:            false,
			StartedAt:         created,
		},
		Artifact: project.ArtifactRecord{
			ID:                10,
			ProjectID:         1,
			WorkflowVersionID: 2,
			ArtifactType:      "approved_artifact_write_report",
			StorageBackend:    "local",
			URI:               "/tmp/areaflow/artifacts/areamatrix/versions/v2/approved-artifact-write/report.json",
			SourcePath:        "versions/v2/approved-artifact-write/report.json",
			SHA256:            "artifact123",
			SizeBytes:         128,
			ContentType:       "application/json",
			Metadata:          map[string]any{},
			CreatedAt:         created,
		},
		Gate:                          project.ExecutionApprovalGate{Project: project.Record{Key: "areamatrix"}, Version: project.WorkflowVersion{ID: 2, DisplayLabel: "v2"}, Run: run, Status: "pass", Mode: "approved_artifact_write_gate"},
		ArtifactLabel:                 "approval-note",
		Status:                        "artifact_written",
		Decision:                      "allowed",
		Message:                       "approved artifact written",
		Created:                       true,
		IdempotencyKey:                "approved-artifact-write:test",
		AreaFlowArtifactWritten:       true,
		AreaFlowExecutionStateWritten: true,
		TaskClaimed:                   true,
		LeaseCreated:                  true,
		AttemptCreated:                true,
		ArtifactCreated:               true,
		ArtifactWritePassed:           true,
	})
	if out.ArtifactLabel != "approval-note" || out.Status != "artifact_written" || out.Decision != "allowed" {
		t.Fatalf("unexpected approved artifact write json: %+v", out)
	}
	if out.Attempt.AttemptKind != "approved_artifact_write" || out.Attempt.DryRun {
		t.Fatalf("unexpected approved artifact write attempt json: %+v", out.Attempt)
	}
	if out.Artifact.ArtifactType != "approved_artifact_write_report" || out.Artifact.SHA256 != "artifact123" {
		t.Fatalf("unexpected approved artifact write artifact json: %+v", out.Artifact)
	}
	if out.ProjectReadAttempted || out.ProjectWriteAttempted || out.ExecutionWriteAttempted || !out.AreaFlowArtifactWritten || !out.AreaFlowExecutionStateWritten {
		t.Fatalf("unexpected approved artifact write safety facts: %+v", out)
	}
}

func TestFixtureProjectWriteToJSON(t *testing.T) {
	created := time.Date(2026, 6, 29, 4, 31, 0, 0, time.UTC)
	expires := created.Add(5 * time.Minute)
	run := project.RunRecord{ID: 3, WorkflowVersionID: 2, RunType: "fixture_project_write", RunKind: "execution", Status: "rollback_verified", DryRun: false, StartedAt: created}
	out := fixtureProjectWriteToJSON(project.FixtureProjectWriteResult{
		Project: project.Record{Key: "areamatrix-fixture"},
		Version: project.WorkflowVersion{ID: 2, DisplayLabel: "v2", ImportMode: "authored"},
		Run:     run,
		Worker: project.WorkerRecord{
			ID:                       7,
			ProjectID:                1,
			WorkerKey:                "local-1",
			WorkerType:               "local_host",
			Status:                   "online",
			Capabilities:             []string{"read_project", "write_artifacts", "write_code"},
			Metadata:                 map[string]any{},
			RegisteredAt:             created,
			LastHeartbeatAt:          &created,
			HeartbeatIntervalSeconds: 30,
			LeaseTimeoutSeconds:      300,
			UpdatedAt:                created,
		},
		Lease: project.LeaseRecord{
			ID:                  8,
			ProjectID:           1,
			RunID:               3,
			RunTaskID:           4,
			WorkerID:            7,
			LeaseKind:           "fixture_project_write",
			Status:              "completed",
			AcquiredAt:          created,
			ExpiresAt:           expires,
			AllowedCapabilities: []string{"read_project", "write_artifacts", "write_code"},
			Scope:               map[string]any{"fixture_project_write": true},
			Metadata:            map[string]any{},
		},
		Task: project.RunTaskRecord{
			ID:                4,
			ProjectID:         1,
			WorkflowVersionID: 2,
			RunID:             3,
			TaskKind:          "fixture_project_write_task",
			Status:            "rollback_verified",
			CreatedAt:         created,
			UpdatedAt:         created,
		},
		CopyAttempt: project.RunAttemptRecord{
			ID:                9,
			ProjectID:         1,
			WorkflowVersionID: 2,
			RunID:             3,
			RunTaskID:         4,
			AttemptKind:       "copy",
			Status:            "passed",
			DryRun:            false,
			StartedAt:         created,
		},
		VerifyAttempt: project.RunAttemptRecord{
			ID:                10,
			ProjectID:         1,
			WorkflowVersionID: 2,
			RunID:             3,
			RunTaskID:         4,
			AttemptKind:       "verify",
			Status:            "passed",
			DryRun:            false,
			StartedAt:         created,
		},
		RollbackAttempt: project.RunAttemptRecord{
			ID:                11,
			ProjectID:         1,
			WorkflowVersionID: 2,
			RunID:             3,
			RunTaskID:         4,
			AttemptKind:       "rollback",
			Status:            "passed",
			DryRun:            false,
			StartedAt:         created,
		},
		WriteSetArtifact: project.ArtifactRecord{
			ID:                12,
			ProjectID:         1,
			WorkflowVersionID: 2,
			ArtifactType:      "fixture_project_write_set",
			StorageBackend:    "local",
			URI:               "/tmp/areaflow/artifacts/areamatrix-fixture/versions/v2/fixture-project-write/write-set.json",
			SourcePath:        "versions/v2/fixture-project-write/write-set.json",
			SHA256:            "writeset123",
			SizeBytes:         128,
			ContentType:       "application/json",
			Metadata:          map[string]any{},
			CreatedAt:         created,
		},
		PreimageArtifact: project.ArtifactRecord{
			ID:                13,
			ProjectID:         1,
			WorkflowVersionID: 2,
			ArtifactType:      "fixture_project_write_preimage",
			StorageBackend:    "local",
			URI:               "/tmp/areaflow/artifacts/areamatrix-fixture/versions/v2/fixture-project-write/preimage.bin",
			SourcePath:        "versions/v2/fixture-project-write/preimage.bin",
			SHA256:            "before123",
			SizeBytes:         12,
			ContentType:       "application/octet-stream",
			Metadata:          map[string]any{},
			CreatedAt:         created,
		},
		Artifact: project.ArtifactRecord{
			ID:                14,
			ProjectID:         1,
			WorkflowVersionID: 2,
			ArtifactType:      "fixture_project_write_report",
			StorageBackend:    "local",
			URI:               "/tmp/areaflow/artifacts/areamatrix-fixture/versions/v2/fixture-project-write/report.json",
			SourcePath:        "versions/v2/fixture-project-write/report.json",
			SHA256:            "report123",
			SizeBytes:         256,
			ContentType:       "application/json",
			Metadata:          map[string]any{},
			CreatedAt:         created,
		},
		Gate:                          project.ExecutionApprovalGate{Project: project.Record{Key: "areamatrix-fixture"}, Version: project.WorkflowVersion{ID: 2, DisplayLabel: "v2"}, Run: run, Status: "pass", Mode: "fixture_project_write_gate"},
		TargetPath:                    "fixtures/input.txt",
		ExpectedBeforeSHA256:          "before123",
		ExpectedBeforeSize:            12,
		AfterSHA256:                   "after123",
		AfterSize:                     13,
		RestoredSHA256:                "before123",
		RestoredSize:                  12,
		Status:                        "rollback_verified",
		Decision:                      "allowed",
		Message:                       "fixture project write verified and rolled back",
		Created:                       true,
		IdempotencyKey:                "fixture-project-write:test",
		ProjectReadAttempted:          true,
		ProjectReadAllowed:            true,
		ProjectWriteAttempted:         true,
		ProjectWriteAllowed:           true,
		AreaFlowArtifactWritten:       true,
		AreaFlowExecutionStateWritten: true,
		TaskClaimed:                   true,
		LeaseCreated:                  true,
		AttemptCreated:                true,
		ArtifactCreated:               true,
		WriteSetPassed:                true,
		VerificationPassed:            true,
		RollbackAttempted:             true,
		RollbackVerified:              true,
	})
	if out.TargetPath != "fixtures/input.txt" || out.Status != "rollback_verified" || out.Decision != "allowed" {
		t.Fatalf("unexpected fixture project write json: %+v", out)
	}
	if out.CopyAttempt.AttemptKind != "copy" || out.VerifyAttempt.AttemptKind != "verify" || out.RollbackAttempt.AttemptKind != "rollback" {
		t.Fatalf("unexpected fixture project write attempts json: %+v %+v %+v", out.CopyAttempt, out.VerifyAttempt, out.RollbackAttempt)
	}
	if out.WriteSetArtifact.ArtifactType != "fixture_project_write_set" || out.PreimageArtifact.ArtifactType != "fixture_project_write_preimage" || out.Artifact.ArtifactType != "fixture_project_write_report" {
		t.Fatalf("unexpected fixture project write artifacts json: %+v %+v %+v", out.WriteSetArtifact, out.PreimageArtifact, out.Artifact)
	}
	if !out.ProjectReadAttempted || !out.ProjectReadAllowed || !out.ProjectWriteAttempted || !out.ProjectWriteAllowed || out.ExecutionWriteAttempted || !out.AreaFlowArtifactWritten || !out.AreaFlowExecutionStateWritten || !out.WriteSetPassed || !out.VerificationPassed || !out.RollbackAttempted || !out.RollbackVerified {
		t.Fatalf("unexpected fixture project write safety facts: %+v", out)
	}
}

func TestManagedGeneratedWriteToJSON(t *testing.T) {
	created := time.Date(2026, 7, 2, 4, 31, 0, 0, time.UTC)
	expires := created.Add(5 * time.Minute)
	run := project.RunRecord{ID: 3, WorkflowVersionID: 2, RunType: "managed_generated_write", RunKind: "execution", Status: "rollback_verified", DryRun: false, StartedAt: created}
	out := managedGeneratedWriteToJSON(project.ManagedGeneratedWriteResult{
		Project: project.Record{Key: "areamatrix-fixture"},
		Version: project.WorkflowVersion{ID: 2, DisplayLabel: "v2", ImportMode: "authored"},
		Run:     run,
		Worker: project.WorkerRecord{
			ID:                       7,
			ProjectID:                1,
			WorkerKey:                "local-1",
			WorkerType:               "local_host",
			Status:                   "online",
			Capabilities:             []string{"read_project", "write_artifacts", "write_generated"},
			Metadata:                 map[string]any{},
			RegisteredAt:             created,
			LastHeartbeatAt:          &created,
			HeartbeatIntervalSeconds: 30,
			LeaseTimeoutSeconds:      300,
			UpdatedAt:                created,
		},
		Lease: project.LeaseRecord{
			ID:                  8,
			ProjectID:           1,
			RunID:               3,
			RunTaskID:           4,
			WorkerID:            7,
			LeaseKind:           "managed_generated_write",
			Status:              "completed",
			AcquiredAt:          created,
			ExpiresAt:           expires,
			AllowedCapabilities: []string{"read_project", "write_artifacts", "write_generated"},
			Scope:               map[string]any{"managed_generated_write": true},
			Metadata:            map[string]any{},
		},
		Task: project.RunTaskRecord{
			ID:                4,
			ProjectID:         1,
			WorkflowVersionID: 2,
			RunID:             3,
			TaskKind:          "managed_generated_write_task",
			Status:            "rollback_verified",
			CreatedAt:         created,
			UpdatedAt:         created,
		},
		CopyAttempt:     project.RunAttemptRecord{ID: 9, ProjectID: 1, WorkflowVersionID: 2, RunID: 3, RunTaskID: 4, AttemptKind: "copy", Status: "passed", DryRun: false, StartedAt: created},
		VerifyAttempt:   project.RunAttemptRecord{ID: 10, ProjectID: 1, WorkflowVersionID: 2, RunID: 3, RunTaskID: 4, AttemptKind: "verify", Status: "passed", DryRun: false, StartedAt: created},
		RollbackAttempt: project.RunAttemptRecord{ID: 11, ProjectID: 1, WorkflowVersionID: 2, RunID: 3, RunTaskID: 4, AttemptKind: "rollback", Status: "passed", DryRun: false, StartedAt: created},
		WriteSetArtifact: project.ArtifactRecord{
			ID:                12,
			ProjectID:         1,
			WorkflowVersionID: 2,
			ArtifactType:      "managed_generated_write_set",
			StorageBackend:    "local",
			URI:               "/tmp/areaflow/artifacts/areamatrix-fixture/versions/v2/managed-generated-write/write-set.json",
			SourcePath:        "versions/v2/managed-generated-write/write-set.json",
			SHA256:            "writeset123",
			SizeBytes:         128,
			ContentType:       "application/json",
			Metadata:          map[string]any{},
			CreatedAt:         created,
		},
		PreimageArtifact: project.ArtifactRecord{
			ID:                13,
			ProjectID:         1,
			WorkflowVersionID: 2,
			ArtifactType:      "managed_generated_write_preimage",
			StorageBackend:    "local",
			URI:               "/tmp/areaflow/artifacts/areamatrix-fixture/versions/v2/managed-generated-write/preimage.bin",
			SourcePath:        "versions/v2/managed-generated-write/preimage.bin",
			SHA256:            "before123",
			SizeBytes:         12,
			ContentType:       "application/octet-stream",
			Metadata:          map[string]any{},
			CreatedAt:         created,
		},
		Artifact: project.ArtifactRecord{
			ID:                14,
			ProjectID:         1,
			WorkflowVersionID: 2,
			ArtifactType:      "managed_generated_write_report",
			StorageBackend:    "local",
			URI:               "/tmp/areaflow/artifacts/areamatrix-fixture/versions/v2/managed-generated-write/report.json",
			SourcePath:        "versions/v2/managed-generated-write/report.json",
			SHA256:            "report123",
			SizeBytes:         256,
			ContentType:       "application/json",
			Metadata:          map[string]any{},
			CreatedAt:         created,
		},
		Gate:                          project.ExecutionApprovalGate{Project: project.Record{Key: "areamatrix-fixture"}, Version: project.WorkflowVersion{ID: 2, DisplayLabel: "v2"}, Run: run, Status: "pass", Mode: "managed_generated_write_gate"},
		TargetPath:                    ".areaflow/generated/status.json",
		ExpectedBeforeSHA256:          "before123",
		ExpectedBeforeSize:            12,
		AfterSHA256:                   "after123",
		AfterSize:                     13,
		RestoredSHA256:                "before123",
		RestoredSize:                  12,
		Status:                        "rollback_verified",
		Decision:                      "allowed",
		Message:                       "managed generated write verified and rolled back",
		Created:                       true,
		IdempotencyKey:                "managed-generated-write:test",
		GeneratedOnly:                 true,
		GeneratedOnlyApplyOpen:        true,
		ProjectReadAttempted:          true,
		ProjectReadAllowed:            true,
		ProjectWriteAttempted:         true,
		ProjectWriteAllowed:           true,
		AreaFlowArtifactWritten:       true,
		AreaFlowExecutionStateWritten: true,
		TaskClaimed:                   true,
		LeaseCreated:                  true,
		AttemptCreated:                true,
		ArtifactCreated:               true,
		WriteSetPassed:                true,
		VerificationPassed:            true,
		RollbackAttempted:             true,
		RollbackVerified:              true,
	})
	if out.TargetPath != ".areaflow/generated/status.json" || out.Status != "rollback_verified" || out.Decision != "allowed" {
		t.Fatalf("unexpected managed generated write json: %+v", out)
	}
	if out.CopyAttempt.AttemptKind != "copy" || out.VerifyAttempt.AttemptKind != "verify" || out.RollbackAttempt.AttemptKind != "rollback" {
		t.Fatalf("unexpected managed generated write attempts json: %+v %+v %+v", out.CopyAttempt, out.VerifyAttempt, out.RollbackAttempt)
	}
	if out.WriteSetArtifact.ArtifactType != "managed_generated_write_set" || out.PreimageArtifact.ArtifactType != "managed_generated_write_preimage" || out.Artifact.ArtifactType != "managed_generated_write_report" {
		t.Fatalf("unexpected managed generated write artifacts json: %+v %+v %+v", out.WriteSetArtifact, out.PreimageArtifact, out.Artifact)
	}
	if !out.GeneratedOnly || !out.GeneratedOnlyApplyOpen || !out.ProjectReadAttempted || !out.ProjectReadAllowed || !out.ProjectWriteAttempted || !out.ProjectWriteAllowed || out.ExecutionWriteAttempted || !out.AreaFlowArtifactWritten || !out.AreaFlowExecutionStateWritten || out.EngineCallAttempted || out.CommandsRun || out.SecretsResolved || out.NetworkUsed || !out.WriteSetPassed || !out.VerificationPassed || !out.RollbackAttempted || !out.RollbackVerified {
		t.Fatalf("unexpected managed generated write safety facts: %+v", out)
	}
}

func TestRunnerPreviewToJSONIncludesArtifacts(t *testing.T) {
	created := time.Date(2026, 6, 29, 3, 1, 0, 0, time.UTC)
	out := runnerPreviewToJSON(project.RunnerPreviewResult{
		Project: project.Record{
			Key: "areamatrix",
		},
		Version: project.WorkflowVersion{
			ID:           2,
			DisplayLabel: "v2",
			ImportMode:   "authored",
		},
		Run: project.RunRecord{
			ID:                3,
			WorkflowVersionID: 2,
			RunType:           "runner_preview",
			RunKind:           "execution",
			Status:            "passed",
			RiskLevel:         "low",
			RiskPolicy:        "pause",
			DryRun:            true,
			Summary:           map[string]any{"artifact_count": float64(1)},
			Metadata:          map[string]any{"dry_run": true},
			StartedAt:         created,
		},
		Tasks: []project.RunTaskRecord{{
			ID:                4,
			WorkflowVersionID: 2,
			RunID:             3,
			TaskKey:           "v2:runner-preview",
			TaskKind:          "workflow_item_preview",
			Status:            "queued",
			RiskLevel:         "low",
			Sequence:          1,
			Metadata:          map[string]any{"dry_run": true},
			CreatedAt:         created,
			UpdatedAt:         created,
		}},
		Artifacts: []project.ArtifactRecord{{
			ID:                6,
			WorkflowVersionID: 2,
			ArtifactType:      "runner_preview_report",
			StorageBackend:    "local",
			URI:               "/tmp/areaflow/artifacts/areamatrix/v2/runner-preview/run-3-report.json",
			SourcePath:        "v2/runner-preview/run-3-report.json",
			SHA256:            "abc123",
			SizeBytes:         128,
			ContentType:       "application/json",
			Metadata:          map[string]any{"dry_run": true},
			CreatedAt:         created,
		}},
		Preflight: project.RunnerPreflight{
			Status: "pass",
		},
		Created:        true,
		IdempotencyKey: "runner.preview:areamatrix:v2",
	})

	if len(out.Artifacts) != 1 {
		t.Fatalf("artifact count = %d, want 1: %+v", len(out.Artifacts), out)
	}
	if len(out.Tasks) != 1 || out.Tasks[0].Status != "queued" {
		t.Fatalf("unexpected runner preview task json: %+v", out.Tasks)
	}
	if out.Artifacts[0].ArtifactType != "runner_preview_report" || out.Artifacts[0].SHA256 != "abc123" {
		t.Fatalf("unexpected runner preview artifact json: %+v", out.Artifacts[0])
	}
}

func TestRunControlToJSONIncludesAreaMatrixWriteProof(t *testing.T) {
	out := runControlToJSON(project.RunControlResult{
		Project: project.Record{Key: "areamatrix"},
		Run: project.RunRecord{
			ID:     3,
			Status: "running",
			DryRun: true,
		},
		PreviousStatus:           "queued",
		Status:                   "running",
		Decision:                 "allowed",
		Message:                  "run marked running in protected mode",
		ProjectWriteAttempted:    false,
		ExecutionWriteAttempted:  false,
		AreaMatrixWriteAttempted: false,
		EngineCallAttempted:      false,
	})
	if out.Project.Key != "areamatrix" || out.Run.ID != 3 || out.Status != "running" || !out.Run.DryRun {
		t.Fatalf("unexpected run control json: %+v", out)
	}
	if out.ProjectWriteAttempted || out.ExecutionWriteAttempted || out.AreaMatrixWriteAttempted || out.EngineCallAttempted {
		t.Fatalf("run control json should preserve no-write safety facts: %+v", out)
	}
}

func TestSummaryToJSON(t *testing.T) {
	created := time.Date(2026, 6, 29, 3, 0, 30, 0, time.UTC)
	summary := project.ProjectSummary{
		Project: project.Record{
			Key:             "areamatrix",
			Name:            "AreaMatrix",
			Kind:            "product-repo",
			Adapter:         "areamatrix",
			WorkflowProfile: "areamatrix",
			DefaultBranch:   "main",
			RootPath:        "/tmp/AreaMatrix",
		},
		Inventory: project.ImportInventory{
			Versions:        2,
			Residuals:       10,
			Artifacts:       6,
			ImportSnapshots: 1,
			MirrorExports:   1,
		},
		Import: project.Snapshot{
			SourceHash: "hash-a",
			CreatedAt:  created,
			Summary:    map[string]any{"residual_count": float64(10)},
		},
		HasImport: true,
		LatestDoctor: project.EventRecord{
			Severity:  "info",
			CreatedAt: created,
			Metadata:  map[string]any{"overall_status": "pass"},
		},
		HasLatestDoctor:     true,
		DoctorStatus:        "pass",
		DriftStatus:         "pass",
		ConfigDriftStatus:   "pass",
		StageCoverageStatus: "pass",
		NativeDoctorStatus:  "warn",
		Config:              testProjectConfigRecord(created),
		HasConfig:           true,
	}

	out := summaryToJSON(summary)
	if out.Project.Key != "areamatrix" || out.Inventory.Residuals != 10 {
		t.Fatalf("unexpected summary json: %+v", out)
	}
	if out.Import == nil || out.Import.SourceHash != "hash-a" {
		t.Fatalf("unexpected import json: %+v", out.Import)
	}
	if out.Config == nil || out.Config.ConfigHash != "hash-config" {
		t.Fatalf("unexpected config json: %+v", out.Config)
	}
	if out.Doctor == nil || out.Doctor.DriftStatus != "pass" || out.Doctor.StageCoverageStatus != "pass" {
		t.Fatalf("unexpected doctor json: %+v", out.Doctor)
	}
	if out.Doctor.ConfigDriftStatus != "pass" {
		t.Fatalf("unexpected config drift doctor json: %+v", out.Doctor)
	}
	if out.Doctor.NativeDoctorStatus != "warn" {
		t.Fatalf("unexpected native doctor json: %+v", out.Doctor)
	}
}

func TestReadinessToJSON(t *testing.T) {
	readiness := project.ProjectReadinessFromSummary(project.ProjectSummary{
		Project: project.Record{
			Key:             "areamatrix",
			Name:            "AreaMatrix",
			Kind:            "product-repo",
			Adapter:         "areamatrix",
			WorkflowProfile: "areamatrix",
			DefaultBranch:   "main",
			RootPath:        "/tmp/AreaMatrix",
		},
		Inventory: project.ImportInventory{
			Versions:        2,
			Residuals:       10,
			Artifacts:       6,
			ImportSnapshots: 1,
			MirrorExports:   1,
		},
		HasImport:           true,
		HasLatestDoctor:     true,
		DoctorStatus:        "pass",
		DriftStatus:         "pass",
		StageCoverageStatus: "pass",
		NativeDoctorStatus:  "warn",
		LatestEventCount:    2,
	})

	out := readinessToJSON(readiness)

	if out.Project.Key != "areamatrix" || out.Status != "warn" {
		t.Fatalf("unexpected readiness json: %+v", out)
	}
	if len(out.Items) != 11 {
		t.Fatalf("readiness item count = %d, want 11", len(out.Items))
	}
	if out.Summary.Project.Key != "areamatrix" {
		t.Fatalf("unexpected embedded summary: %+v", out.Summary)
	}
}

func TestGeneratedWriteReadinessToJSON(t *testing.T) {
	generatedAt := time.Date(2026, 7, 2, 11, 0, 0, 0, time.UTC)
	readiness := project.GeneratedWriteReadiness{
		Project:                   project.Record{Key: "areamatrix", Name: "AreaMatrix"},
		Status:                    "blocked",
		Mode:                      "read_only_generated_write_readiness",
		RequiredCapabilities:      []string{"read_project", "write_artifacts", "write_generated"},
		AllowedGeneratedPrefixes:  []string{".areaflow/generated/", ".areamatrix/generated/"},
		RequiredWritePaths:        []string{".areaflow/generated/**", ".areamatrix/generated/**"},
		Blockers:                  []string{"real_areamatrix_apply_open: apply closed"},
		ReviewBlockers:            []string{},
		ForbiddenActions:          []string{"queue_run", "write_project_file"},
		ReadyForReview:            true,
		ApplyOpen:                 false,
		RealAreaMatrixWriteOpened: false,
		GeneratedOnly:             true,
		ProjectConfigRead:         true,
		Items: []project.ReadinessItem{
			{Key: "real_areamatrix_apply_open", Status: "blocked", Message: "apply closed"},
		},
		GeneratedAt: generatedAt,
	}

	out := generatedWriteReadinessToJSON(readiness)

	if out.Project.Key != "areamatrix" || out.Status != "blocked" || out.Mode != "read_only_generated_write_readiness" {
		t.Fatalf("unexpected generated write readiness json: %+v", out)
	}
	if !out.ReadyForReview || out.ApplyOpen || out.RealAreaMatrixWriteOpened || !out.GeneratedOnly {
		t.Fatalf("unexpected generated write readiness flags: %+v", out)
	}
	if len(out.Items) != 1 || out.Items[0].Key != "real_areamatrix_apply_open" {
		t.Fatalf("unexpected generated write readiness items: %+v", out.Items)
	}
	if out.GeneratedAt != "2026-07-02T11:00:00Z" {
		t.Fatalf("generated_at = %q", out.GeneratedAt)
	}
}

func TestGeneratedWriteApplyBetaGateToJSON(t *testing.T) {
	generatedAt := time.Date(2026, 7, 2, 12, 45, 0, 0, time.UTC)
	readiness := project.GeneratedWriteReadiness{
		Project:                   project.Record{Key: "areamatrix", Name: "AreaMatrix"},
		Status:                    "blocked",
		Mode:                      "read_only_generated_write_readiness",
		ReadyForReview:            true,
		ApplyOpen:                 false,
		RealAreaMatrixWriteOpened: false,
		AllowedGeneratedPrefixes:  []string{".areaflow/generated/", ".areamatrix/generated/"},
	}
	gate := project.GeneratedWriteApplyBetaGate{
		Project:                   project.Record{Key: "areamatrix", Name: "AreaMatrix"},
		Status:                    "blocked",
		Mode:                      "read_only_generated_write_apply_beta_gate",
		Readiness:                 readiness,
		RequiredCapabilities:      []string{"read_project", "write_artifacts", "write_generated"},
		AllowedGeneratedPrefixes:  []string{".areaflow/generated/", ".areamatrix/generated/"},
		RequiredEvidence:          []string{"explicit R3 approval for real AreaMatrix generated-only apply beta"},
		ForbiddenActions:          []string{"queue_run", "write_project_file"},
		ApprovalRequired:          true,
		ApprovalStatus:            "needs_approval",
		ApplyOpen:                 false,
		RealAreaMatrixWriteOpened: false,
		GeneratedOnly:             true,
		Items: []project.GeneratedWriteApplyBetaGateItem{
			{Key: "generated_apply_beta:explicit_approval", Category: "approval", Status: "blocked", ApprovalStatus: "needs_approval", Message: "approval required"},
		},
		GeneratedAt: generatedAt,
	}

	out := generatedWriteApplyBetaGateToJSON(gate)

	if out.Project.Key != "areamatrix" || out.Status != "blocked" || out.Mode != "read_only_generated_write_apply_beta_gate" {
		t.Fatalf("unexpected generated write apply beta gate json: %+v", out)
	}
	if !out.ApprovalRequired || out.ApprovalStatus != "needs_approval" || out.ApplyOpen || out.RealAreaMatrixWriteOpened {
		t.Fatalf("unexpected generated write apply beta flags: %+v", out)
	}
	if !out.Readiness.ReadyForReview || len(out.Items) != 1 || out.Items[0].ApprovalStatus != "needs_approval" {
		t.Fatalf("unexpected generated write apply beta items/readiness: %+v", out)
	}
	if out.GeneratedAt != "2026-07-02T12:45:00Z" {
		t.Fatalf("generated_at = %q", out.GeneratedAt)
	}
}

func testProjectConfigRecord(loadedAt time.Time) project.ProjectConfigRecord {
	return project.ProjectConfigRecord{
		ID:              1,
		ProjectID:       1,
		ProtocolVersion: 1,
		ConfigPath:      "examples/areamatrix/areaflow.yaml",
		ConfigHash:      "hash-config",
		Ownership:       map[string]any{"mode": "import"},
		StatusExport:    map[string]any{"path": ".areaflow/status.json"},
		Migration:       map[string]any{"phase": "import"},
		Active:          true,
		LoadedAt:        loadedAt,
	}
}

func TestImportDiffToJSON(t *testing.T) {
	created := time.Date(2026, 6, 29, 3, 0, 30, 0, time.UTC)
	diff := project.ProjectImportDiffFromSummary(project.ProjectSummary{
		Project: project.Record{
			Key:             "areamatrix",
			Name:            "AreaMatrix",
			Kind:            "product-repo",
			Adapter:         "areamatrix",
			WorkflowProfile: "areamatrix",
			DefaultBranch:   "main",
			RootPath:        "/tmp/AreaMatrix",
		},
		Import: project.Snapshot{
			SourceHash: "hash-b",
			CreatedAt:  created,
			Summary:    map[string]any{"version_count": float64(2)},
		},
		HasImport: true,
		PreviousImport: project.Snapshot{
			SourceHash: "hash-a",
			CreatedAt:  created.Add(-time.Minute),
			Summary:    map[string]any{"version_count": float64(1)},
		},
		HasPreviousImport: true,
	})

	out := importDiffToJSON(diff)

	if out.Project.Key != "areamatrix" || out.Status != "changed" {
		t.Fatalf("unexpected import diff json: %+v", out)
	}
	if !out.HasPrevious || !out.SourceChanged {
		t.Fatalf("unexpected import diff flags: %+v", out)
	}
	if len(out.Changes) != 9 {
		t.Fatalf("change count = %d, want 9", len(out.Changes))
	}
}

func TestVerificationBundleToJSON(t *testing.T) {
	created := time.Date(2026, 6, 29, 3, 0, 30, 0, time.UTC)
	summary := project.ProjectSummary{
		Project: project.Record{
			Key:             "areamatrix",
			Name:            "AreaMatrix",
			Kind:            "product-repo",
			Adapter:         "areamatrix",
			WorkflowProfile: "areamatrix",
			DefaultBranch:   "main",
			RootPath:        "/tmp/AreaMatrix",
		},
		Import:    project.Snapshot{SourceHash: "hash-a", CreatedAt: created},
		HasImport: true,
	}
	readiness := project.ProjectReadinessFromSummary(summary)
	diff := project.ProjectImportDiffFromSummary(summary)
	bundle := project.ProjectVerificationBundleFromParts(summary, readiness, diff, []project.EventRecord{
		{
			ID:        7,
			Type:      "project.import.completed",
			Severity:  "info",
			Message:   "import completed",
			Metadata:  map[string]any{"overall_status": "pass"},
			CreatedAt: created,
		},
	})

	out := verificationBundleToJSON(bundle)

	if out.Project.Key != "areamatrix" || out.Status == "" {
		t.Fatalf("unexpected verification bundle json: %+v", out)
	}
	if out.PhaseGate.Name != "v0.2-shadow-doctor" || out.PhaseGate.Status == "" {
		t.Fatalf("unexpected phase gate json: %+v", out.PhaseGate)
	}
	if out.Summary.Project.Key != "areamatrix" || out.Readiness.Project.Key != "areamatrix" {
		t.Fatalf("unexpected nested bundle json: %+v", out)
	}
	if len(out.Events) != 1 || out.Events[0].Type != "project.import.completed" {
		t.Fatalf("unexpected bundle events: %+v", out.Events)
	}
}

func TestStatusProjectionsToJSON(t *testing.T) {
	generatedAt := time.Date(2026, 7, 1, 8, 30, 0, 0, time.UTC)
	writtenAt := generatedAt.Add(time.Minute)
	out := statusProjectionsToJSON(
		project.Record{ID: 1, Key: "areamatrix", Name: "AreaMatrix", Adapter: "areamatrix", WorkflowProfile: "areamatrix"},
		[]project.StatusProjectionRecord{
			{
				ID:           10,
				ProjectID:    1,
				TargetKind:   "project_status_json",
				TargetURI:    ".areaflow/status.json",
				SummaryState: "mirroring",
				Payload:      map[string]any{"version_count": float64(2)},
				SourceHash:   "hash-a",
				WriteState:   "written",
				GeneratedAt:  generatedAt,
				WrittenAt:    &writtenAt,
				Metadata:     map[string]any{"legacy_snapshot_kind": "mirror_export"},
			},
		},
	)

	if out.Project.Key != "areamatrix" || len(out.Projections) != 1 {
		t.Fatalf("unexpected status projections json: %+v", out)
	}
	projection := out.Projections[0]
	if projection.TargetKind != "project_status_json" || projection.WriteState != "written" || projection.WrittenAt == "" {
		t.Fatalf("unexpected projection json: %+v", projection)
	}
	if projection.Payload["version_count"] != float64(2) {
		t.Fatalf("unexpected projection payload: %+v", projection.Payload)
	}
}

func TestProjectStatusProjectionAndShimJSONParityWithAPIResponses(t *testing.T) {
	apiTags := apiResponseJSONTags(t, "../api/server.go")
	for _, pair := range []struct {
		cliName string
		cliType any
		apiName string
	}{
		{
			cliName: "statusProjectionAuthorizationPreviewJSON",
			cliType: statusProjectionAuthorizationPreviewJSON{},
			apiName: "statusProjectionAuthorizationPreviewResponse",
		},
		{
			cliName: "statusProjectionApplyGateJSON",
			cliType: statusProjectionApplyGateJSON{},
			apiName: "statusProjectionApplyGateResponse",
		},
		{
			cliName: "statusProjectionApplyPacketPreviewJSON",
			cliType: statusProjectionApplyPacketPreviewJSON{},
			apiName: "statusProjectionApplyPacketPreviewResponse",
		},
		{
			cliName: "statusProjectionApplyPacketJSON",
			cliType: statusProjectionApplyPacketJSON{},
			apiName: "statusProjectionApplyPacketResponse",
		},
		{
			cliName: "statusProjectionApplyAPIRequestJSON",
			cliType: statusProjectionApplyAPIRequestJSON{},
			apiName: "statusProjectionApplyAPIRequestResponse",
		},
		{
			cliName: "shimAuthorizationPacketJSON",
			cliType: shimAuthorizationPacketJSON{},
			apiName: "shimAuthorizationPacketResponse",
		},
		{
			cliName: "shimApplyPacketPreviewJSON",
			cliType: shimApplyPacketPreviewJSON{},
			apiName: "shimApplyPacketPreviewResponse",
		},
		{
			cliName: "shimApplyPacketJSON",
			cliType: shimApplyPacketJSON{},
			apiName: "shimApplyPacketResponse",
		},
		{
			cliName: "shimApplyGateJSON",
			cliType: shimApplyGateJSON{},
			apiName: "shimApplyGateResponse",
		},
	} {
		t.Run(pair.cliName, func(t *testing.T) {
			cliFields := jsonFieldSetFromReflectType(reflect.TypeOf(pair.cliType))
			apiFields, ok := apiTags[pair.apiName]
			if !ok {
				t.Fatalf("API response type %s not found", pair.apiName)
			}
			if missing := missingJSONFields(cliFields, apiFields); len(missing) > 0 {
				t.Fatalf("%s missing API JSON fields from %s: %v", pair.cliName, pair.apiName, missing)
			}
			if extra := missingJSONFields(apiFields, cliFields); len(extra) > 0 {
				t.Fatalf("%s has JSON fields not exposed by %s: %v", pair.cliName, pair.apiName, extra)
			}
		})
	}
}

func TestStatusProjectionApplyToJSON(t *testing.T) {
	generatedAt := time.Date(2026, 7, 1, 9, 0, 0, 0, time.UTC)
	out := statusProjectionApplyToJSON(project.ApplyStatusProjectionResult{
		Project:                 project.Record{ID: 1, Key: "areamatrix", Name: "AreaMatrix"},
		Status:                  "written",
		Decision:                "allowed",
		Message:                 "status projection written",
		EventID:                 11,
		AuditEventID:            12,
		SnapshotID:              13,
		StatusProjectionID:      14,
		TargetKind:              "project_status_json",
		TargetURI:               ".areaflow/status.json",
		WrittenTarget:           "/tmp/areamatrix/.areaflow/status.json",
		WriteHash:               "hash-a",
		WriteSize:               100,
		PreimageCaptured:        true,
		PreimageExists:          true,
		PreimageSHA256:          "before-hash",
		PreimageSize:            42,
		PostWriteVerified:       true,
		PostWriteSHA256:         "hash-a",
		PostWriteSize:           100,
		RootContained:           true,
		StableProjectionValid:   true,
		AtomicReplaceUsed:       true,
		RollbackCompensation:    true,
		SourceHash:              "source-a",
		SummaryState:            "mirroring",
		ApplyGateStatus:         "pass",
		ApplyGateDecision:       "go",
		ApplyGateApprovalStatus: "approved",
		ApplyCommandEligible:    true,
		IdempotencyKey:          "projection-key",
		Created:                 true,
		GeneratedAt:             generatedAt,
		ProjectWriteAttempted:   true,
		ExecutionWriteAttempted: false,
		EngineCallAttempted:     false,
	})

	if out.Project.Key != "areamatrix" || out.StatusProjectionID != 14 || out.TargetURI != ".areaflow/status.json" {
		t.Fatalf("unexpected status projection apply json: %+v", out)
	}
	if !out.ProjectWriteAttempted || out.ExecutionWriteAttempted || out.EngineCallAttempted {
		t.Fatalf("unexpected status projection safety facts: %+v", out)
	}
	if !out.PreimageCaptured || !out.PreimageExists || out.PreimageSHA256 != "before-hash" || out.PreimageSize != 42 {
		t.Fatalf("unexpected status projection preimage facts: %+v", out)
	}
	if !out.PostWriteVerified || out.PostWriteSHA256 != "hash-a" || out.PostWriteSize != 100 {
		t.Fatalf("unexpected status projection post-write facts: %+v", out)
	}
	if !out.RootContained || !out.StableProjectionValid || !out.AtomicReplaceUsed || !out.RollbackCompensation {
		t.Fatalf("unexpected status projection write facts: %+v", out)
	}
	if out.ApplyGateStatus != "pass" || out.ApplyGateDecision != "go" || !out.ApplyCommandEligible {
		t.Fatalf("unexpected apply gate facts: %+v", out)
	}
}

func TestStatusProjectionAuthorizationPreviewToJSON(t *testing.T) {
	generatedAt := time.Date(2026, 7, 4, 9, 0, 0, 0, time.UTC)
	out := statusProjectionAuthorizationPreviewToJSON(project.StatusProjectionAuthorizationPreview{
		Project:                        project.Record{ID: 1, Key: "areamatrix", Name: "AreaMatrix"},
		Status:                         "needs_approval",
		Mode:                           "status_projection_apply_authorization_preview_v1",
		ClaimScope:                     "package_a_status_projection_preflight_only",
		NotReal100:                     true,
		Decision:                       "needs_explicit_approval",
		Message:                        "status projection apply requires an explicit authorization packet before writing the managed project",
		TargetKind:                     "project_status_json",
		TargetURI:                      ".areaflow/status.json",
		TargetPath:                     "/Users/as/Ai-Project/project/AreaMatrix/.areaflow/status.json",
		SchemaURI:                      "schemas/status-projection.schema.json",
		ValidatorPreflight:             "python3 scripts/validate-status-projection-schema.py schemas/status-projection.schema.json /Users/as/Ai-Project/project/AreaMatrix/.areaflow/status.json",
		ProtectedPathFingerprintSHA256: "protected-hash",
		SourceHash:                     "source-a",
		SummaryState:                   "mirroring",
		RequiredAuthorizationPhrase:    project.StatusProjectionApplyRequiredApprovalReason,
		Permission: project.StatusProjectionAuthorizationPermission{
			Capability:        "write_status",
			ResourceType:      "path",
			TargetURI:         ".areaflow/status.json",
			CapabilityAllowed: true,
			PathAllowed:       true,
			Allowed:           true,
			Reason:            "allowed",
		},
		Preimage: project.StatusProjectionPreimage{
			TargetPath:            "/Users/as/Ai-Project/project/AreaMatrix/.areaflow/status.json",
			Exists:                true,
			Readable:              true,
			SizeBytes:             42,
			SHA256:                "before-hash",
			SchemaStatus:          "legacy",
			LegacyShape:           true,
			MissingRequiredFields: []string{"schema_version"},
			Message:               "target uses legacy status projection shape",
		},
		WriteSet: []project.StatusProjectionWriteSetEntry{
			{
				TargetURI:                ".areaflow/status.json",
				TargetPath:               "/Users/as/Ai-Project/project/AreaMatrix/.areaflow/status.json",
				Operation:                "replace_or_create",
				Capability:               "write_status",
				ExpectedBeforeExists:     true,
				ExpectedBeforeSHA256:     "before-hash",
				ExpectedBeforeSizeBytes:  42,
				RequiresPreimageMatch:    true,
				RequiresSchemaValidation: true,
				RollbackAction:           "restore the captured preimage bytes for .areaflow/status.json",
				ProtectedPath:            true,
			},
		},
		RequiredPreflight:                      []string{"areaflow project status-projections areamatrix --json"},
		RequiredPacketFields:                   []string{"expected_before_sha256", "rollback_plan"},
		RequiredCapabilities:                   []string{"write_status"},
		ProtectedPaths:                         []string{".areaflow/status.json"},
		RollbackPlan:                           []string{"restore the captured preimage bytes for .areaflow/status.json"},
		BlockedBy:                              []string{"explicit_status_projection_apply_approval_missing"},
		ForbiddenActions:                       []string{"write_execution"},
		SafetyFacts:                            map[string]bool{"project_write_attempted": false, "execution_write_attempted": false},
		ApprovalRequired:                       true,
		ApprovalStatus:                         "missing",
		WouldWriteProjectFileAfterApproval:     true,
		WouldCreateCommandRequestAfterApproval: true,
		GeneratedAt:                            generatedAt,
	})

	if out.Project.Key != "areamatrix" || out.Status != "needs_approval" || out.ApplyOpen {
		t.Fatalf("unexpected authorization json: %+v", out)
	}
	if out.ClaimScope != "package_a_status_projection_preflight_only" || !out.NotReal100 {
		t.Fatalf("expected authorization JSON guardrail fields: %+v", out)
	}
	if out.SchemaURI != "schemas/status-projection.schema.json" || out.Preimage.SchemaStatus != "legacy" {
		t.Fatalf("unexpected schema/preimage json: %+v", out)
	}
	if out.RequiredAuthorizationPhrase != project.StatusProjectionApplyRequiredApprovalReason {
		t.Fatalf("unexpected required authorization phrase: %q", out.RequiredAuthorizationPhrase)
	}
	if len(out.WriteSet) != 1 || out.WriteSet[0].ExpectedBeforeSHA256 != "before-hash" || !out.WriteSet[0].RequiresPreimageMatch {
		t.Fatalf("unexpected write set json: %+v", out.WriteSet)
	}
	if out.ProjectWriteAttempted || out.ExecutionWriteAttempted || out.EngineCallAttempted {
		t.Fatalf("preview JSON must not report attempted side effects: %+v", out)
	}
}

func TestStatusProjectionApplyGateToJSON(t *testing.T) {
	generatedAt := time.Date(2026, 7, 4, 10, 0, 0, 0, time.UTC)
	authorization := project.StatusProjectionAuthorizationPreview{
		Project:            project.Record{ID: 1, Key: "areamatrix", Name: "AreaMatrix"},
		Status:             "needs_approval",
		Mode:               "status_projection_apply_authorization_preview_v1",
		ClaimScope:         "package_a_status_projection_preflight_only",
		NotReal100:         true,
		Decision:           "needs_explicit_approval",
		Message:            "status projection apply requires an explicit authorization packet before writing the managed project",
		TargetKind:         "project_status_json",
		TargetURI:          ".areaflow/status.json",
		TargetPath:         "/Users/as/Ai-Project/project/AreaMatrix/.areaflow/status.json",
		SchemaURI:          "schemas/status-projection.schema.json",
		ValidatorPreflight: "python3 scripts/validate-status-projection-schema.py schemas/status-projection.schema.json /Users/as/Ai-Project/project/AreaMatrix/.areaflow/status.json",
		SourceHash:         "source-a",
		SummaryState:       "mirroring",
		Preimage: project.StatusProjectionPreimage{
			Exists:       true,
			SizeBytes:    42,
			SHA256:       "before-hash",
			SchemaStatus: "legacy",
			LegacyShape:  true,
		},
		RequiredPacketFields: []string{"expected_before_sha256", "explicit_approval"},
		RequiredCapabilities: []string{"write_status"},
		ProtectedPaths:       []string{".areaflow/status.json"},
		ForbiddenActions:     []string{"write_execution"},
		SafetyFacts:          map[string]bool{"project_write_attempted": false},
		ApprovalRequired:     true,
		ApprovalStatus:       "missing",
		GeneratedAt:          generatedAt,
	}
	out := statusProjectionApplyGateToJSON(project.StatusProjectionApplyGate{
		Project:                        project.Record{ID: 1, Key: "areamatrix", Name: "AreaMatrix"},
		Status:                         "blocked",
		Mode:                           "status_projection_apply_gate_v1",
		ClaimScope:                     "package_a_status_projection_preflight_only",
		NotReal100:                     true,
		Decision:                       "no_go",
		Message:                        "status projection apply packet is blocked",
		TargetURI:                      ".areaflow/status.json",
		TargetPath:                     "/Users/as/Ai-Project/project/AreaMatrix/.areaflow/status.json",
		Authorization:                  authorization,
		Items:                          []project.StatusProjectionApplyGateItem{{Key: "explicit_approval", Category: "approval", Status: "blocked", Expected: "true", Actual: "false", BlockedBy: []string{"explicit_status_projection_apply_approval_missing"}}},
		RequiredPacketFields:           []string{"expected_before_sha256", "explicit_approval"},
		RequiredCapabilities:           []string{"write_status"},
		ProtectedPaths:                 []string{".areaflow/status.json"},
		ForbiddenActions:               []string{"write_execution"},
		SafetyFacts:                    map[string]bool{"command_request_created": false, "status_projection_written": false},
		ApplyCommandEligibleIsNotApply: true,
		RequiresSeparateApplyCommand:   true,
		ApprovalRequired:               true,
		ApprovalStatus:                 "missing_or_incomplete",
		GeneratedAt:                    generatedAt,
	})

	if out.Project.Key != "areamatrix" || out.Status != "blocked" || out.ApplyCommandEligible {
		t.Fatalf("unexpected apply gate json: %+v", out)
	}
	if out.ClaimScope != "package_a_status_projection_preflight_only" || !out.NotReal100 || !out.ApplyCommandEligibleIsNotApply || !out.RequiresSeparateApplyCommand {
		t.Fatalf("expected apply gate JSON non-apply guardrails: %+v", out)
	}
	if out.CommandRequestCreated || out.StatusProjectionWritten || out.ProjectWriteAttempted || out.ExecutionWriteAttempted || out.EngineCallAttempted {
		t.Fatalf("apply gate JSON must be read-only: %+v", out)
	}
	if len(out.Items) != 1 || out.Items[0].BlockedBy[0] != "explicit_status_projection_apply_approval_missing" {
		t.Fatalf("unexpected apply gate items: %+v", out.Items)
	}
	if out.Authorization.Preimage.SchemaStatus != "legacy" {
		t.Fatalf("expected nested authorization preview: %+v", out.Authorization)
	}
}

func TestStatusProjectionApplyPacketPreviewToJSON(t *testing.T) {
	generatedAt := time.Date(2026, 7, 4, 12, 0, 0, 0, time.UTC)
	authorization := project.StatusProjectionAuthorizationPreview{
		Project:            project.Record{ID: 1, Key: "areamatrix", Name: "AreaMatrix"},
		Status:             "needs_approval",
		Mode:               "status_projection_apply_authorization_preview_v1",
		ClaimScope:         "package_a_status_projection_preflight_only",
		NotReal100:         true,
		Decision:           "needs_explicit_approval",
		TargetKind:         "project_status_json",
		TargetURI:          ".areaflow/status.json",
		TargetPath:         "/Users/as/Ai-Project/project/AreaMatrix/.areaflow/status.json",
		SchemaURI:          "schemas/status-projection.schema.json",
		ValidatorPreflight: "python3 scripts/validate-status-projection-schema.py schemas/status-projection.schema.json /Users/as/Ai-Project/project/AreaMatrix/.areaflow/status.json",
		SourceHash:         "source-a",
		Preimage: project.StatusProjectionPreimage{
			Exists:       true,
			SizeBytes:    42,
			SHA256:       "before-hash",
			SchemaStatus: "legacy",
		},
		GeneratedAt: generatedAt,
	}
	gate := project.StatusProjectionApplyGate{
		Project:                        project.Record{ID: 1, Key: "areamatrix", Name: "AreaMatrix"},
		Status:                         "ready",
		Mode:                           "status_projection_apply_gate_v1",
		ClaimScope:                     "package_a_status_projection_preflight_only",
		NotReal100:                     true,
		Decision:                       "go",
		TargetURI:                      ".areaflow/status.json",
		Authorization:                  authorization,
		RequiredAuthorizationPhrase:    project.StatusProjectionApplyRequiredApprovalReason,
		ApplyCommandEligible:           true,
		ApplyCommandEligibleIsNotApply: true,
		RequiresSeparateApplyCommand:   true,
		ApprovalRequired:               true,
		ApprovalStatus:                 "approved",
		GeneratedAt:                    generatedAt,
	}
	out := statusProjectionApplyPacketPreviewToJSON(project.StatusProjectionApplyPacketPreview{
		Project:                     project.Record{ID: 1, Key: "areamatrix", Name: "AreaMatrix"},
		Status:                      "ready",
		Mode:                        "status_projection_apply_packet_preview_v1",
		ClaimScope:                  "package_a_status_projection_preflight_only",
		NotReal100:                  true,
		Decision:                    "ready_for_apply_command",
		Blockers:                    []string{"explicit_status_projection_apply_approval_missing"},
		RequiredAuthorizationPhrase: project.StatusProjectionApplyRequiredApprovalReason,
		Authorization:               authorization,
		Gate:                        gate,
		Packet: project.StatusProjectionApplyPacket{
			TargetURI:                      ".areaflow/status.json",
			ExpectedBeforeExists:           true,
			ExpectedBeforeSHA256:           "before-hash",
			ExpectedBeforeSizeBytes:        42,
			SourceHash:                     "source-a",
			SchemaURI:                      "schemas/status-projection.schema.json",
			ValidatorPreflight:             authorization.ValidatorPreflight,
			ProtectedPathCheck:             "git -C /Users/as/Ai-Project/project/AreaMatrix status --short -- workflow/README.md .areaflow/status.json",
			ProtectedPathFingerprintSHA256: "protected-hash",
			RollbackAction:                 "restore the captured preimage bytes for .areaflow/status.json",
			AcceptedPreimageSchemaStatus:   "legacy",
			ExplicitApproval:               true,
			ApprovalActor:                  "as",
			ApprovalReason:                 "approve status projection apply",
			RequiredAuthorizationPhrase:    project.StatusProjectionApplyRequiredApprovalReason,
		},
		ApplyCommand:        []string{"areaflow", "project", "status-projection-apply", "areamatrix", "--explicit-approval"},
		APIRequest:          project.StatusProjectionApplyAPIRequest{TargetURI: ".areaflow/status.json", SourceHash: "source-a", ProtectedPathFingerprintSHA256: "protected-hash", ExplicitApproval: true, ApprovalActor: "as", RequiredAuthorizationPhrase: project.StatusProjectionApplyRequiredApprovalReason},
		RequiredHumanReview: []string{"review target preimage schema status"},
		ForbiddenActions:    []string{"write_execution"},
		SafetyFacts:         map[string]bool{"project_write_attempted": false, "command_request_created": false},
		WouldCreateCommandRequestAfterApplyCommand: true,
		WouldWriteProjectFileAfterApplyCommand:     true,
		ApplyCommandEligibleIsNotApply:             true,
		RequiresSeparateApplyCommand:               true,
		GeneratedAt:                                generatedAt,
	})

	if out.Project.Key != "areamatrix" || out.Status != "ready" || out.Packet.SourceHash != "source-a" {
		t.Fatalf("unexpected apply packet json: %+v", out)
	}
	if out.ClaimScope != "package_a_status_projection_preflight_only" || !out.NotReal100 || !out.ApplyCommandEligibleIsNotApply || !out.RequiresSeparateApplyCommand {
		t.Fatalf("expected apply packet JSON non-apply guardrails: %+v", out)
	}
	if len(out.Blockers) != 1 || out.Blockers[0] != "explicit_status_projection_apply_approval_missing" {
		t.Fatalf("expected top-level blockers in apply packet json: %+v", out.Blockers)
	}
	if out.RequiredAuthorizationPhrase != project.StatusProjectionApplyRequiredApprovalReason ||
		out.Gate.RequiredAuthorizationPhrase != project.StatusProjectionApplyRequiredApprovalReason ||
		out.Packet.RequiredAuthorizationPhrase != project.StatusProjectionApplyRequiredApprovalReason ||
		out.APIRequest.RequiredAuthorizationPhrase != project.StatusProjectionApplyRequiredApprovalReason {
		t.Fatalf("expected required authorization phrase in apply packet json: %+v", out)
	}
	if !out.Packet.ExplicitApproval || out.Packet.ApprovalActor != "as" || !out.APIRequest.ExplicitApproval {
		t.Fatalf("expected approval facts in packet/API request: %+v", out)
	}
	if out.Packet.ProtectedPathFingerprintSHA256 != "protected-hash" || out.APIRequest.ProtectedPathFingerprintSHA256 != "protected-hash" {
		t.Fatalf("expected protected path fingerprint in packet/API request: %+v", out)
	}
	if !out.WouldCreateCommandRequestAfterApplyCommand || out.CommandRequestCreated || out.ProjectWriteAttempted {
		t.Fatalf("unexpected packet safety facts: %+v", out)
	}
}

func TestCompatibilityContractToJSON(t *testing.T) {
	contract := project.CompatibilityContract{
		Project: project.Record{Key: "areamatrix", Name: "AreaMatrix"},
		Status:  "pass",
		Commands: []project.CompatibilityCommand{
			{
				Command:        "./dev workflow status",
				Mode:           "forward",
				Status:         "pass",
				AreaFlowTarget: "areaflow project summary",
				Fallback:       ".areaflow/status.json",
				Metadata:       map[string]any{"read_only": true},
			},
		},
	}

	out := compatibilityContractToJSON(contract)

	if out.Project.Key != "areamatrix" || out.Status != "pass" {
		t.Fatalf("unexpected compatibility json: %+v", out)
	}
	if len(out.Commands) != 1 || out.Commands[0].Command != "./dev workflow status" {
		t.Fatalf("unexpected compatibility commands: %+v", out.Commands)
	}
}

func TestShimPreviewToJSON(t *testing.T) {
	contract := project.CompatibilityContract{
		Project: project.Record{Key: "areamatrix", Name: "AreaMatrix"},
		Status:  "pass",
		Commands: []project.CompatibilityCommand{
			{
				Command:       "./task-loop run",
				Mode:          "blocked",
				Status:        "pass",
				BlockedReason: "execution and task-loop replacement are out of v0.4 scope",
				Metadata:      map[string]any{"read_only": false},
			},
		},
	}
	out := shimPreviewToJSON(project.ShimPreviewFromCompatibility(contract))

	if out.Project.Key != "areamatrix" || out.Mode != "read_only_planning" {
		t.Fatalf("unexpected shim preview json: %+v", out)
	}
	if len(out.PlannedFiles) == 0 || out.PlannedFiles[0].Path != "scripts/areaflow_shim.py" {
		t.Fatalf("unexpected shim planned files json: %+v", out.PlannedFiles)
	}
	if len(out.CommandMappings) != 1 || out.CommandMappings[0].Mode != "blocked" {
		t.Fatalf("unexpected shim command mappings json: %+v", out.CommandMappings)
	}
}

func TestShimReadinessToJSON(t *testing.T) {
	contract := project.CompatibilityContract{
		Project: project.Record{Key: "areamatrix", Name: "AreaMatrix"},
		Status:  "pass",
		Commands: []project.CompatibilityCommand{
			{
				Command:  "./task-loop run",
				Mode:     "blocked",
				Status:   "pass",
				Metadata: map[string]any{"read_only": false},
			},
		},
	}
	out := shimReadinessToJSON(project.ShimReadinessFromPreview(project.ShimPreviewFromCompatibility(contract)))

	if out.Project.Key != "areamatrix" || out.Status != "blocked" {
		t.Fatalf("unexpected shim readiness json: %+v", out)
	}
	if out.Preview.Mode != "read_only_planning" || len(out.Items) == 0 {
		t.Fatalf("unexpected shim readiness detail json: %+v", out)
	}
	var statusProjection shimReadinessItemJSON
	for _, item := range out.Items {
		if item.Key == "status_projection" {
			statusProjection = item
			break
		}
	}
	if statusProjection.Key == "" {
		t.Fatalf("status_projection readiness item missing: %+v", out.Items)
	}
	if statusProjection.Metadata["schema_contract"] != "stable_fallback_projection_v1" {
		t.Fatalf("unexpected status projection schema contract: %+v", statusProjection.Metadata)
	}
	if statusProjection.Metadata["schema_uri"] != "schemas/status-projection.schema.json" {
		t.Fatalf("unexpected status projection schema uri: %+v", statusProjection.Metadata)
	}
	if statusProjection.Metadata["validator_preflight"] != "python3 scripts/validate-status-projection-schema.py schemas/status-projection.schema.json /Users/as/Ai-Project/project/AreaMatrix/.areaflow/status.json" {
		t.Fatalf("unexpected status projection validator preflight: %+v", statusProjection.Metadata)
	}
	fields, ok := statusProjection.Metadata["required_schema_fields"].([]string)
	if !ok || !stringSliceContains(fields, "compatibility.blocked_commands[]") {
		t.Fatalf("required schema fields missing blocked commands: %+v", statusProjection.Metadata)
	}
	forbidden, ok := statusProjection.Metadata["forbidden_fields"].([]string)
	if !ok || !stringSliceContains(forbidden, "artifact_content") {
		t.Fatalf("forbidden fields missing artifact content: %+v", statusProjection.Metadata)
	}
}

func TestShimAuthorizationPacketToJSON(t *testing.T) {
	contract := project.CompatibilityContract{
		Project: project.Record{Key: "areamatrix", Name: "AreaMatrix"},
		Status:  "pass",
		Commands: []project.CompatibilityCommand{
			{
				Command:  "./task-loop run",
				Mode:     "blocked",
				Status:   "pass",
				Metadata: map[string]any{"read_only": false},
			},
		},
	}
	readiness := project.ShimReadinessFromPreview(project.ShimPreviewFromCompatibility(contract))

	out := shimAuthorizationPacketToJSON(project.ShimAuthorizationPacketFromReadiness(readiness))

	if out.Project.Key != "areamatrix" || out.Status != "blocked" || out.Mode != "read_only_authorization_packet" {
		t.Fatalf("unexpected shim authorization json: %+v", out)
	}
	if len(out.AllowedFiles) == 0 || out.AllowedFiles[0].Path != "scripts/areaflow_shim.py" {
		t.Fatalf("unexpected allowed files json: %+v", out.AllowedFiles)
	}
	if !stringSliceContains(out.ForbiddenPaths, "workflow/versions/**") {
		t.Fatalf("expected workflow versions forbidden path: %+v", out.ForbiddenPaths)
	}
	if !stringSliceContains(out.RequiredPreflight, "areaflow project status-projections areamatrix --json") {
		t.Fatalf("expected status projections preflight: %+v", out.RequiredPreflight)
	}
	if !stringSliceContains(out.RequiredPreflight, "python3 scripts/validate-status-projection-schema.py schemas/status-projection.schema.json /Users/as/Ai-Project/project/AreaMatrix/.areaflow/status.json") {
		t.Fatalf("expected executable status projection schema preflight: %+v", out.RequiredPreflight)
	}
	if !stringSliceContains(out.RequiredPreflight, "verify .areaflow/status.json stable_fallback_projection_v1 includes schema_version/project_id/active_versions/rough_progress/source_snapshot_hash/compatibility.blocked_commands and excludes summary/generated_at/source/source_hash") {
		t.Fatalf("expected stable projection schema preflight: %+v", out.RequiredPreflight)
	}
	if out.SafetyFacts["project_write_attempted"] || out.SafetyFacts["execution_write_attempted"] || out.SafetyFacts["task_loop_run_forwarded"] {
		t.Fatalf("unexpected shim authorization safety facts: %+v", out.SafetyFacts)
	}
}

func TestPrintShimAuthorizationPacketIncludesGateSections(t *testing.T) {
	contract := project.CompatibilityContract{
		Project: project.Record{Key: "areamatrix", Name: "AreaMatrix"},
		Status:  "pass",
		Commands: []project.CompatibilityCommand{
			{
				Command:  "./task-loop run",
				Mode:     "blocked",
				Status:   "pass",
				Metadata: map[string]any{"read_only": false},
			},
		},
	}
	readiness := project.ShimReadinessFromPreview(project.ShimPreviewFromCompatibility(contract))
	packet := project.ShimAuthorizationPacketFromReadiness(readiness)
	var stdout bytes.Buffer
	cmd := command{stdout: &stdout, stderr: &bytes.Buffer{}}

	cmd.printShimAuthorizationPacket(packet)
	text := stdout.String()

	for _, want := range []string{
		"required_preflight.count:",
		"required_preflight: areaflow project shim-authorization areamatrix --json",
		"required_preflight: areaflow project status-projections areamatrix --json",
		"required_preflight: python3 scripts/validate-status-projection-schema.py schemas/status-projection.schema.json /Users/as/Ai-Project/project/AreaMatrix/.areaflow/status.json",
		"required_preflight: verify .areaflow/status.json stable_fallback_projection_v1 includes schema_version/project_id/active_versions/rough_progress/source_snapshot_hash/compatibility.blocked_commands and excludes summary/generated_at/source/source_hash",
		"required_preflight: git -C /Users/as/Ai-Project/project/AreaMatrix status --short -- workflow/README.md .areaflow/status.json",
		"required_preflight: AREAFLOW_DATABASE_URL=... bash scripts/smoke-areamatrix-readonly.sh",
		"post_edit_verification.count:",
		"post_edit_verification: verify ./task-loop run returns blocked",
		"post_edit_verification: git status --short -- workflow/README.md .areaflow/status.json",
		"rollback_scope.count:",
		"rollback_scope: do not write v1 historical execution, progress.json, logs or checkpoints",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("shim authorization output missing %q:\n%s", want, text)
		}
	}
}

func TestShimApplyPacketPreviewToJSON(t *testing.T) {
	preview := project.ShimApplyPacketPreview{
		Project:  project.Record{Key: "areamatrix", Name: "AreaMatrix"},
		Status:   "blocked",
		Mode:     "shim_apply_packet_preview_v1",
		Decision: "readiness_blocked",
		Message:  "shim apply packet is blocked",
		Authorization: project.ShimAuthorizationPacket{
			Project: project.Record{Key: "areamatrix", Name: "AreaMatrix"},
			Status:  "blocked",
			Mode:    "read_only_authorization_packet",
		},
		Gate: project.ShimApplyGate{
			Project: project.Record{Key: "areamatrix", Name: "AreaMatrix"},
			Status:  "blocked",
			Mode:    "shim_apply_gate_v1",
			Items: []project.ShimApplyGateItem{
				{Key: "readiness_blockers", Category: "readiness", Status: "blocked", BlockedBy: []string{"shim_readiness_still_blocked"}},
			},
			SafetyFacts: map[string]bool{"project_write_attempted": false},
		},
		Packet: project.ShimApplyPacket{
			CommandType:                "project.shim.apply",
			ProjectKey:                 "areamatrix",
			AuthorizationSnapshotHash:  "authorization-hash",
			StatusProjectionGateID:     "status-gate-1",
			ProtectedPathFingerprintID: "fingerprint-1",
		},
		ApplyGateCommand: []string{"areaflow", "project", "shim-apply-gate", "areamatrix"},
		SafetyFacts:      map[string]bool{"area_matrix_files_modified": false},
	}

	out := shimApplyPacketPreviewToJSON(preview)

	if out.Project.Key != "areamatrix" || out.Packet.CommandType != "project.shim.apply" || out.Packet.AuthorizationSnapshotHash != "authorization-hash" {
		t.Fatalf("unexpected shim apply packet json: %+v", out)
	}
	if out.Gate.Status != "blocked" || len(out.Gate.Items) != 1 || out.SafetyFacts["area_matrix_files_modified"] {
		t.Fatalf("unexpected shim apply gate json: %+v", out.Gate)
	}
}

func TestPrintShimApplyGateIncludesBlockedItems(t *testing.T) {
	gate := project.ShimApplyGate{
		Project:              project.Record{Key: "areamatrix", Name: "AreaMatrix"},
		Status:               "blocked",
		Mode:                 "shim_apply_gate_v1",
		Decision:             "no_go",
		Message:              "shim apply packet is blocked",
		AllowedFiles:         []string{"scripts/areaflow_shim.py", ".areaflow/status.json"},
		RequiredPacketFields: []string{"allowed_files", "authorization_snapshot_hash", "explicit_approval"},
		RequiredProofFacts:   []string{"status_projection_apply_gate", "protected_path_fingerprint"},
		Items: []project.ShimApplyGateItem{
			{
				Key:       "readiness_blockers",
				Category:  "readiness",
				Status:    "blocked",
				Message:   "readiness blockers must be limited to explicit edit approval before the apply packet can pass",
				Expected:  "none except explicit_edit_approval",
				Actual:    "real_areamatrix_status_projection_schema",
				BlockedBy: []string{"shim_readiness_still_blocked"},
			},
		},
		SafetyFacts: map[string]bool{"project_write_attempted": false},
	}
	var stdout bytes.Buffer
	cmd := command{stdout: &stdout, stderr: &bytes.Buffer{}}

	cmd.printShimApplyGate(gate)
	text := stdout.String()

	for _, want := range []string{
		"shim apply gate: areamatrix status=blocked decision=no_go",
		"required_proof_facts: status_projection_apply_gate, protected_path_fingerprint",
		"blocked_by: shim_readiness_still_blocked",
		"area_matrix_files_modified: false",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("shim apply gate output missing %q:\n%s", want, text)
		}
	}
}

func TestShimApplyCommandToJSON(t *testing.T) {
	result := project.ApplyShimCommandResult{
		Project:  project.Record{Key: "areamatrix", Name: "AreaMatrix"},
		Status:   "recorded",
		Mode:     "shim_apply_command_v1",
		Decision: "allowed",
		Message:  "shim apply command recorded",
		Gate: project.ShimApplyGate{
			Project:  project.Record{Key: "areamatrix", Name: "AreaMatrix"},
			Status:   "pass",
			Decision: "go",
			SafetyFacts: map[string]bool{
				"project_write_attempted": false,
			},
		},
		EventID:                11,
		AuditEventID:           12,
		IdempotencyKey:         "shim-key",
		Created:                true,
		ApplyOpen:              true,
		AreaFlowCommandCreated: true,
		CommandRequestCreated:  true,
		SafetyFacts: map[string]bool{
			"area_matrix_files_modified": false,
		},
	}

	out := shimApplyCommandToJSON(result)

	if out.Project.Key != "areamatrix" || out.Status != "recorded" || out.Decision != "allowed" {
		t.Fatalf("unexpected shim apply command json: %+v", out)
	}
	if !out.ApplyOpen || !out.CommandRequestCreated || !out.AreaFlowCommandCreated || out.ProjectWriteAttempted || out.AreaMatrixFilesModified {
		t.Fatalf("shim apply command json should record only AreaFlow command state: %+v", out)
	}
	if out.EventID != 11 || out.AuditEventID != 12 || out.IdempotencyKey != "shim-key" || !out.Created {
		t.Fatalf("missing command evidence json: %+v", out)
	}
	if out.Gate.Status != "pass" || len(out.Blockers) != 0 {
		t.Fatalf("unexpected gate/blocker json: %+v", out)
	}
}

func TestPrintShimApplyCommand(t *testing.T) {
	result := project.ApplyShimCommandResult{
		Project:  project.Record{Key: "areamatrix", Name: "AreaMatrix"},
		Status:   "recorded",
		Mode:     "shim_apply_command_v1",
		Decision: "allowed",
		Message:  "shim apply command recorded",
		Gate: project.ShimApplyGate{
			Project:              project.Record{Key: "areamatrix", Name: "AreaMatrix"},
			Status:               "pass",
			Decision:             "go",
			ApplyCommandEligible: true,
		},
		ApplyOpen:              true,
		AreaFlowCommandCreated: true,
		CommandRequestCreated:  true,
	}
	var stdout bytes.Buffer
	cmd := command{stdout: &stdout, stderr: &bytes.Buffer{}}

	cmd.printShimApplyCommand(result)
	text := stdout.String()

	for _, want := range []string{
		"shim apply: areamatrix status=recorded decision=allowed",
		"apply_open: true",
		"area_flow_command_created: true",
		"command_request_created: true",
		"gate_status: pass decision=go eligible=true",
		"area_matrix_files_modified: false",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("shim apply command output missing %q:\n%s", want, text)
		}
	}
}

func apiResponseJSONTags(t *testing.T, path string) map[string]map[string]struct{} {
	t.Helper()
	file, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.SkipObjectResolution)
	if err != nil {
		t.Fatalf("parse API response source: %v", err)
	}
	tags := map[string]map[string]struct{}{}
	aliases := map[string]string{}
	for _, decl := range file.Decls {
		gen, ok := decl.(*ast.GenDecl)
		if !ok {
			continue
		}
		for _, spec := range gen.Specs {
			typeSpec, ok := spec.(*ast.TypeSpec)
			if !ok {
				continue
			}
			switch typed := typeSpec.Type.(type) {
			case *ast.StructType:
				tags[typeSpec.Name.Name] = jsonFieldSetFromAST(typed)
			case *ast.Ident:
				aliases[typeSpec.Name.Name] = typed.Name
			}
		}
	}
	for name, target := range aliases {
		if fields, ok := tags[target]; ok {
			tags[name] = fields
		}
	}
	return tags
}

func jsonFieldSetFromAST(structType *ast.StructType) map[string]struct{} {
	fields := map[string]struct{}{}
	for _, field := range structType.Fields.List {
		if field.Tag == nil {
			continue
		}
		tag, err := strconv.Unquote(field.Tag.Value)
		if err != nil {
			continue
		}
		if name := jsonFieldName(reflect.StructTag(tag).Get("json")); name != "" {
			fields[name] = struct{}{}
		}
	}
	return fields
}

func jsonFieldSetFromReflectType(valueType reflect.Type) map[string]struct{} {
	if valueType.Kind() == reflect.Pointer {
		valueType = valueType.Elem()
	}
	fields := map[string]struct{}{}
	if valueType.Kind() != reflect.Struct {
		return fields
	}
	for i := 0; i < valueType.NumField(); i++ {
		if name := jsonFieldName(valueType.Field(i).Tag.Get("json")); name != "" {
			fields[name] = struct{}{}
		}
	}
	return fields
}

func jsonFieldName(tag string) string {
	name, _, _ := strings.Cut(tag, ",")
	if name == "" || name == "-" {
		return ""
	}
	return name
}

func missingJSONFields(expected map[string]struct{}, actual map[string]struct{}) []string {
	missing := []string{}
	for field := range expected {
		if _, ok := actual[field]; !ok {
			missing = append(missing, field)
		}
	}
	sort.Strings(missing)
	return missing
}

func ptrSecurityReadiness(readiness project.SecurityBoundaryReadiness) *project.SecurityBoundaryReadiness {
	return &readiness
}

func stringSliceContains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
