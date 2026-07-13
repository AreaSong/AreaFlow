package project

import (
	"context"
	"fmt"
	"time"
)

type AreaMatrixExecutionCutoverReadinessOptions struct{}

type AreaMatrixExecutionCutoverReadiness struct {
	Project          Record
	Status           string
	Mode             string
	Items            []AreaMatrixExecutionCutoverReadinessItem
	MigrationPath    []string
	CommandEvidence  map[string]int
	Capabilities     []string
	ForbiddenActions []string
	SafetyFacts      map[string]bool
	NextSteps        []AreaMatrixExecutionCutoverNextStep
	GeneratedAt      time.Time
}

type AreaMatrixExecutionCutoverReadinessItem struct {
	Key              string
	Category         string
	Status           string
	Message          string
	RequiredEvidence []string
	NextCommand      string
	Metadata         map[string]any
}

type AreaMatrixExecutionCutoverNextStep struct {
	Key         string
	Owner       string
	Action      string
	RiskLevel   string
	BlockedBy   []string
	NextCommand string
	Metadata    map[string]any
}

var executionCutoverMigrationPath = []string{
	"Import",
	"Mirror",
	"Shadow",
	"Authoring Cutover",
	"Execution Beta",
	"Execution Cutover",
	"Archive",
	"Shim Retirement",
}

var executionCutoverCommandEvidenceTypes = []string{
	"runner.preview",
	"run.start",
	"run.drain",
	"run.cancel",
	"worker.register",
	"worker.heartbeat",
	"lease.acquire",
	"lease.release",
	"lease.recover",
	"run.fixture_queue",
	"worker.fixture_execute",
	"run.read_only_verify_queue",
	"worker.read_only_verify",
	"run.approved_artifact_write_queue",
	"worker.approved_artifact_write",
	"run.fixture_project_write_queue",
	"worker.fixture_project_write",
	"run.managed_generated_write_queue",
	"worker.managed_generated_write",
}

func (s Store) AreaMatrixExecutionCutoverReadiness(ctx context.Context, record Record, _ AreaMatrixExecutionCutoverReadinessOptions) (AreaMatrixExecutionCutoverReadiness, error) {
	verification, err := s.ProjectVerificationBundle(ctx, record, 10)
	if err != nil {
		return AreaMatrixExecutionCutoverReadiness{}, err
	}
	shim, err := s.ShimReadiness(ctx, record)
	if err != nil {
		return AreaMatrixExecutionCutoverReadiness{}, err
	}
	versions, err := s.ListWorkflowVersions(ctx, record)
	if err != nil {
		return AreaMatrixExecutionCutoverReadiness{}, err
	}
	commandEvidence, err := s.completedCommandEvidenceCounts(ctx, record.ID)
	if err != nil {
		return AreaMatrixExecutionCutoverReadiness{}, err
	}
	return AreaMatrixExecutionCutoverReadinessFromParts(record, verification, shim, versions, commandEvidence), nil
}

func AreaMatrixExecutionCutoverReadinessFromParts(record Record, verification ProjectVerificationBundle, shim ShimReadiness, versions []WorkflowVersion, commandEvidence map[string]int) AreaMatrixExecutionCutoverReadiness {
	readiness := AreaMatrixExecutionCutoverReadiness{
		Project:       record,
		Status:        "pass",
		Mode:          "read_only_areamatrix_execution_cutover_readiness",
		MigrationPath: append([]string{}, executionCutoverMigrationPath...),
		CommandEvidence: copyCommandEvidence(
			commandEvidence,
			executionCutoverCommandEvidenceTypes,
		),
		Capabilities: []string{
			"read_project",
			"write_artifacts",
			"write_generated",
			"run_commands",
			"manage_workers",
			"execute_agents",
			"manage_git",
		},
		ForbiddenActions: []string{
			"write_areamatrix_files",
			"forward_task_loop_run",
			"write_execution_directory",
			"modify_progress_json",
			"run_codex_cli",
			"resolve_secrets",
			"create_checkpoint",
			"apply_execution_cutover",
		},
		SafetyFacts: map[string]bool{
			"read_only":                        true,
			"project_write_attempted":          false,
			"execution_write_attempted":        false,
			"task_loop_run_forwarded":          false,
			"engine_call_attempted":            false,
			"commands_run":                     false,
			"secrets_resolved":                 false,
			"network_used":                     false,
			"worker_scheduled":                 false,
			"approval_created":                 false,
			"execution_cutover_apply_open":     false,
			"areamatrix_shim_files_written":    false,
			"workflow_readme_controlled_write": false,
			"retained_generated_apply_open":    false,
			"source_write_open":                false,
			"checkpoint_apply_open":            false,
			"repair_apply_open":                false,
		},
		GeneratedAt: time.Now().UTC(),
	}

	readiness.add("import_mirror_shadow", "migration", statusFromPhaseGate(verification.PhaseGate.Status), "v0.2 verification bundle must pass before execution cutover readiness", []string{"project verify-bundle"}, "areaflow project verify-bundle "+record.Key+" --json", map[string]any{
		"phase_gate":        verification.PhaseGate.Name,
		"phase_gate_status": verification.PhaseGate.Status,
		"blockers":          verification.PhaseGate.Blockers,
	})
	readiness.add("authoring_cutover", "migration", statusBool(hasAuthoringCutoverEvidence(versions)), "at least one AreaFlow-authored workflow version must have authoring cutover evidence", []string{"project cutover-apply", "cutover_readiness_gate"}, "areaflow project cutover-readiness "+record.Key+" --version <label> --json", map[string]any{
		"authoring_cutover_versions": authoringCutoverVersionLabels(versions),
	})
	readiness.add("compatibility_shim", "compatibility", statusForExecutionCutoverShim(shim), "AreaMatrix compatibility shim readiness must pass before task-loop execution cutover", []string{"shim readiness evidence", "explicit edit approval"}, "areaflow project shim-readiness "+record.Key+" --json", map[string]any{
		"shim_status": shim.Status,
	})
	readiness.add("task_loop_run_policy", "compatibility", statusBool(shimCommandBlocked(shim.Preview, "./task-loop run")), "./task-loop run must remain blocked before explicit execution cutover", []string{"compatibility contract"}, "areaflow project compatibility "+record.Key+" --json", map[string]any{
		"task_loop_run_forwarded": false,
	})

	readiness.addCommandEvidence("worker_lease_lifecycle", "execution_beta", commandEvidence, []string{"worker.register", "worker.heartbeat", "lease.acquire", "lease.release", "lease.recover"}, "worker lifecycle, lease acquire/release and recovery must have command/audit evidence", "areaflow worker pool-summary --json")
	readiness.addCommandEvidence("run_control", "execution_beta", commandEvidence, []string{"run.start", "run.drain", "run.cancel"}, "start, drain and cancel must have command/audit evidence", "areaflow run <start|drain|cancel> <run-id> --json")
	readiness.addCommandEvidence("fixture_execution", "execution_beta", commandEvidence, []string{"runner.preview", "run.fixture_queue", "worker.fixture_execute"}, "fixture execution must prove run/task/attempt/artifact/lease evidence before managed execution", "areaflow run fixture-queue "+record.Key+" <version> --json")
	readiness.addCommandEvidence("read_only_verify", "execution_beta", commandEvidence, []string{"run.read_only_verify_queue", "worker.read_only_verify"}, "read-only verify must prove allowlisted project read evidence without writes", "areaflow run read-only-verify-queue "+record.Key+" <version> --json")
	readiness.addCommandEvidence("approved_artifact_write", "execution_beta", commandEvidence, []string{"run.approved_artifact_write_queue", "worker.approved_artifact_write"}, "approved artifact write must prove AreaFlow-owned artifact evidence", "areaflow run approved-artifact-write-queue "+record.Key+" <version> --json")
	readiness.addCommandEvidence("fixture_project_write", "execution_beta", commandEvidence, []string{"run.fixture_project_write_queue", "worker.fixture_project_write"}, "fixture project write rollback drill must prove expected-before, preimage, verify and rollback evidence", "areaflow run fixture-project-write-queue "+record.Key+" <version> --json")
	readiness.addCommandEvidence("managed_generated_write_apply", "execution_beta", commandEvidence, []string{"run.managed_generated_write_queue", "worker.managed_generated_write"}, "generated-only apply must have fixture/temp rollback drill evidence before real AreaMatrix apply", "areaflow run managed-generated-write-queue "+record.Key+" <version> --json")

	readiness.add("real_areamatrix_generated_apply", "execution_cutover", "blocked", "real AreaMatrix retained generated-only apply is not open", []string{"R3 approval", "expected-before hash", "rollback verification", "non-target fingerprints"}, "areaflow project generated-write-apply-beta-gate "+record.Key+" --json", map[string]any{
		"apply_open": false,
	})
	readiness.add("copy_repair_checkpoint", "execution_cutover", "blocked", "copy, repair and checkpoint apply are not open for AreaMatrix execution cutover", []string{"copy attempt evidence", "repair attempt evidence", "checkpoint gate evidence"}, "areaflow run execution-plan <run-id> --json", map[string]any{
		"copy_apply_open":       false,
		"repair_apply_open":     false,
		"checkpoint_apply_open": false,
	})
	readiness.add("explicit_execution_cutover_approval", "approval", "blocked", "explicit execution cutover approval is required before task-loop can forward or apply", []string{"R3/R4 approval", "rollback plan", "audit evidence"}, "areaflow project execution-cutover-readiness "+record.Key+" --json", map[string]any{
		"approval_required": true,
		"approval_created":  false,
	})

	readiness.NextSteps = []AreaMatrixExecutionCutoverNextStep{
		{
			Key:         "land_areamatrix_shim",
			Owner:       "project_owner",
			Action:      "land AreaMatrix compatibility shim after explicit edit approval",
			RiskLevel:   "R2 managed_write",
			BlockedBy:   []string{"explicit_edit_approval"},
			NextCommand: "areaflow project shim-readiness " + record.Key + " --json",
			Metadata:    map[string]any{"writes_areamatrix": true},
		},
		{
			Key:         "real_generated_apply_beta",
			Owner:       "execution_owner",
			Action:      "open the first real AreaMatrix generated-only retained apply beta",
			RiskLevel:   "R3 execution",
			BlockedBy:   []string{"generated_write_apply_beta_gate", "explicit R3 approval"},
			NextCommand: "areaflow project generated-write-apply-beta-gate " + record.Key + " --json",
			Metadata:    map[string]any{"source_write_open": false},
		},
		{
			Key:         "copy_repair_checkpoint_design",
			Owner:       "execution_owner",
			Action:      "prove copy, verify, repair and checkpoint separation before task-loop replacement",
			RiskLevel:   "R3 execution",
			BlockedBy:   []string{"copy_apply_not_open", "repair_apply_not_open", "checkpoint_apply_not_open"},
			NextCommand: "areaflow run execution-plan <run-id> --json",
			Metadata:    map[string]any{"task_loop_run_forwarded": false},
		},
	}

	return readiness
}

func (r *AreaMatrixExecutionCutoverReadiness) add(key string, category string, status string, message string, requiredEvidence []string, nextCommand string, metadata map[string]any) {
	r.Items = append(r.Items, AreaMatrixExecutionCutoverReadinessItem{
		Key:              key,
		Category:         category,
		Status:           status,
		Message:          message,
		RequiredEvidence: append([]string{}, requiredEvidence...),
		NextCommand:      nextCommand,
		Metadata:         metadata,
	})
	r.Status = combineExecutionCutoverStatus(r.Status, status)
}

func (r *AreaMatrixExecutionCutoverReadiness) addCommandEvidence(key string, category string, counts map[string]int, required []string, message string, nextCommand string) {
	missing := missingCommandEvidence(counts, required)
	status := "pass"
	if len(missing) > 0 {
		status = "blocked"
	}
	r.add(key, category, status, message, required, nextCommand, map[string]any{
		"required_command_types": required,
		"missing_command_types":  missing,
		"command_counts":         commandEvidenceSubset(counts, required),
	})
}

func (s Store) completedCommandEvidenceCounts(ctx context.Context, projectID int64) (map[string]int, error) {
	rows, err := s.pool.Query(ctx, `
SELECT command_type, COUNT(*)
FROM command_requests
WHERE project_id = $1 AND completed_at IS NOT NULL
GROUP BY command_type`,
		projectID,
	)
	if err != nil {
		return nil, fmt.Errorf("list command evidence counts: %w", err)
	}
	defer rows.Close()
	counts := map[string]int{}
	for rows.Next() {
		var commandType string
		var count int
		if err := rows.Scan(&commandType, &count); err != nil {
			return nil, fmt.Errorf("scan command evidence count: %w", err)
		}
		counts[commandType] = count
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate command evidence counts: %w", err)
	}
	return counts, nil
}

func copyCommandEvidence(counts map[string]int, known []string) map[string]int {
	copied := map[string]int{}
	for _, key := range known {
		copied[key] = counts[key]
	}
	for key, value := range counts {
		if _, exists := copied[key]; !exists {
			copied[key] = value
		}
	}
	return copied
}

func commandEvidenceSubset(counts map[string]int, required []string) map[string]int {
	subset := map[string]int{}
	for _, key := range required {
		subset[key] = counts[key]
	}
	return subset
}

func missingCommandEvidence(counts map[string]int, required []string) []string {
	missing := []string{}
	for _, key := range required {
		if counts[key] == 0 {
			missing = append(missing, key)
		}
	}
	return missing
}

func hasAuthoringCutoverEvidence(versions []WorkflowVersion) bool {
	return len(authoringCutoverVersionLabels(versions)) > 0
}

func authoringCutoverVersionLabels(versions []WorkflowVersion) []string {
	labels := []string{}
	for _, version := range versions {
		if workflowVersionAuthoringCutoverApplied(version) {
			labels = append(labels, version.DisplayLabel)
		}
	}
	return labels
}

func statusFromPhaseGate(status string) string {
	if status == "pass" {
		return "pass"
	}
	return "blocked"
}

func statusForExecutionCutoverShim(shim ShimReadiness) string {
	if shim.Status == "pass" {
		return "pass"
	}
	return "blocked"
}

func combineExecutionCutoverStatus(current string, next string) string {
	if current == "blocked" || next == "blocked" {
		return "blocked"
	}
	if current == "fail" || next == "fail" {
		return "fail"
	}
	if current == "warn" || next == "warn" {
		return "warn"
	}
	return "pass"
}
