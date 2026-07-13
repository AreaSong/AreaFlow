package project

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

type ExecutionApprovalGateOptions struct {
	RequiredCapabilities []string
	SkipEnginePreview    bool
	Mode                 string
	GeneratedAt          time.Time
}

type ExecutionApprovalGate struct {
	Project                 Record
	Run                     RunRecord
	Version                 WorkflowVersion
	Status                  string
	Mode                    string
	Items                   []ReadinessItem
	Blockers                []string
	Warnings                []string
	RequiredCapabilities    []string
	ApprovalFound           bool
	Approval                ApprovalRecord
	ApprovalGateFound       bool
	ApprovalGate            GateResult
	LiveMappingGateFound    bool
	LiveMappingGate         GateResult
	EnginePreview           CodexCLIAdapterPreview
	Workers                 []WorkerRecord
	ForbiddenActions        []string
	ProjectWriteAttempted   bool
	ExecutionWriteAttempted bool
	EngineCallAttempted     bool
	CommandsRun             bool
	SecretsResolved         bool
	NetworkUsed             bool
	TaskClaimed             bool
	WorkerStarted           bool
	AttemptCreated          bool
	ArtifactCreated         bool
	GeneratedAt             time.Time
}

func (s Store) ExecutionApprovalGate(ctx context.Context, runID int64, options ExecutionApprovalGateOptions) (ExecutionApprovalGate, error) {
	detail, err := s.GetRun(ctx, runID)
	if err != nil {
		return ExecutionApprovalGate{}, err
	}
	record, err := s.projectRecordByID(ctx, detail.Run.ProjectID)
	if err != nil {
		return ExecutionApprovalGate{}, err
	}
	version, err := s.workflowVersionByID(ctx, record.ID, detail.Run.WorkflowVersionID)
	if err != nil {
		return ExecutionApprovalGate{}, err
	}
	approval, approvalFound, err := s.latestApprovalRecord(ctx, record.ID, version.ID)
	if err != nil {
		return ExecutionApprovalGate{}, err
	}
	approvalGate, approvalGateFound, err := s.latestGateResult(ctx, record.ID, version.ID, "approval_gate")
	if err != nil {
		return ExecutionApprovalGate{}, err
	}
	liveMappingGate, liveMappingGateFound, err := s.latestGateResult(ctx, record.ID, version.ID, "live_mapping_gate")
	if err != nil {
		return ExecutionApprovalGate{}, err
	}
	enginePreview := CodexCLIAdapterPreview{
		Project:          record,
		Status:           "not_required",
		Mode:             "engine_preview_skipped",
		ForbiddenActions: []string{"execute_codex_cli", "resolve_secrets", "write_managed_project"},
	}
	if !options.SkipEnginePreview {
		var err error
		enginePreview, err = s.CodexCLIAdapterPreview(ctx, record, CodexCLIAdapterPreviewOptions{})
		if err != nil {
			enginePreview = CodexCLIAdapterPreview{
				Project:          record,
				Status:           "blocked",
				Mode:             "read_only_codex_cli_adapter_preview",
				Blockers:         []string{"engine_preview_error:" + err.Error()},
				ForbiddenActions: []string{"execute_codex_cli", "resolve_secrets", "write_managed_project"},
			}
		}
	}
	workers, err := s.ListWorkers(ctx, record, 100)
	if err != nil {
		return ExecutionApprovalGate{}, err
	}
	return BuildExecutionApprovalGate(record, version, detail, approval, approvalFound, approvalGate, approvalGateFound, liveMappingGate, liveMappingGateFound, enginePreview, workers, options), nil
}

func BuildExecutionApprovalGate(
	record Record,
	version WorkflowVersion,
	detail RunDetail,
	approval ApprovalRecord,
	approvalFound bool,
	approvalGate GateResult,
	approvalGateFound bool,
	liveMappingGate GateResult,
	liveMappingGateFound bool,
	enginePreview CodexCLIAdapterPreview,
	workers []WorkerRecord,
	options ExecutionApprovalGateOptions,
) ExecutionApprovalGate {
	generatedAt := options.GeneratedAt
	if generatedAt.IsZero() {
		generatedAt = time.Now().UTC()
	}
	requiredCaps := normalizeCapabilityList(options.RequiredCapabilities)
	if len(requiredCaps) == 0 {
		requiredCaps = []string{"execute_agents", "read_project", "run_commands", "write_artifacts"}
	}
	gate := ExecutionApprovalGate{
		Project:              record,
		Run:                  detail.Run,
		Version:              version,
		Status:               "pass",
		Mode:                 "read_only_execution_approval_gate",
		RequiredCapabilities: requiredCaps,
		ApprovalFound:        approvalFound,
		Approval:             approval,
		ApprovalGateFound:    approvalGateFound,
		ApprovalGate:         approvalGate,
		LiveMappingGateFound: liveMappingGateFound,
		LiveMappingGate:      liveMappingGate,
		EnginePreview:        enginePreview,
		Workers:              workers,
		ForbiddenActions: []string{
			"claim_task",
			"start_worker",
			"create_attempt",
			"create_artifact",
			"execute_engine",
			"run_commands",
			"resolve_secrets",
			"write_managed_project",
			"write_workflow_execution",
		},
		ProjectWriteAttempted:   false,
		ExecutionWriteAttempted: false,
		EngineCallAttempted:     false,
		CommandsRun:             false,
		SecretsResolved:         false,
		NetworkUsed:             false,
		TaskClaimed:             false,
		WorkerStarted:           false,
		AttemptCreated:          false,
		ArtifactCreated:         false,
		GeneratedAt:             generatedAt,
	}
	if strings.TrimSpace(options.Mode) != "" {
		gate.Mode = strings.TrimSpace(options.Mode)
	}
	add := func(key string, status string, message string, metadata map[string]any) {
		gate.Items = append(gate.Items, ReadinessItem{
			Key:      key,
			Status:   status,
			Message:  message,
			Metadata: metadata,
		})
		if status == "blocked" || status == "fail" {
			gate.Status = "blocked"
			gate.Blockers = append(gate.Blockers, key+": "+message)
		}
		if status == "warn" {
			gate.Warnings = append(gate.Warnings, key+": "+message)
		}
	}

	add("run_kind", gateStatusBool(detail.Run.RunKind == "execution"), "run must be an execution run", map[string]any{
		"run_kind": detail.Run.RunKind,
	})
	add("run_status", gateStatusBool(detail.Run.Status == "queued"), "real execution apply starts from queued run state", map[string]any{
		"run_status": detail.Run.Status,
	})
	add("dry_run_boundary", gateStatusBool(!detail.Run.DryRun), "dry-run preview runs cannot enter real execution apply", map[string]any{
		"dry_run": detail.Run.DryRun,
	})
	add("workflow_version_authored", gateStatusBool(version.ImportMode == "authored"), "execution applies only to AreaFlow-authored workflow versions", map[string]any{
		"display_label": version.DisplayLabel,
		"import_mode":   version.ImportMode,
	})
	add("run_tasks_ready", gateStatusBool(hasRunnableRunTask(detail.Tasks)), "run must have at least one queued or pending run_task", map[string]any{
		"task_count":          len(detail.Tasks),
		"runnable_task_count": runnableRunTaskCount(detail.Tasks),
	})

	approvalStatus := "blocked"
	approvalMessage := "missing approved workflow approval record"
	if approvalFound {
		approvalStatus = gateStatusBool(approval.Decision == "approved")
		approvalMessage = "latest workflow approval record must be approved"
	}
	add("workflow_approval", approvalStatus, approvalMessage, map[string]any{
		"approval_found":     approvalFound,
		"approval_record_id": approval.ID,
		"approval_decision":  approval.Decision,
		"approval_kind":      approval.ApprovalKind,
	})
	add("approval_gate", gateStatusBool(approvalGateFound && approvalGate.Status == "pass"), "approval_gate must pass before execution apply", map[string]any{
		"gate_found": approvalGateFound,
		"gate_id":    approvalGate.ID,
		"status":     approvalGate.Status,
	})
	add("live_mapping_gate", gateStatusBool(liveMappingGateFound && liveMappingGate.Status == "pass"), "live_mapping_gate must pass before execution apply", map[string]any{
		"gate_found": liveMappingGateFound,
		"gate_id":    liveMappingGate.ID,
		"status":     liveMappingGate.Status,
	})

	engineStatus := "pass"
	engineMessage := "engine preflight is ready for approval-gated execution"
	if options.SkipEnginePreview {
		engineMessage = "engine preflight is not required for this read-only execution step"
	} else if enginePreview.Status == "blocked" {
		engineStatus = "blocked"
		engineMessage = "engine adapter preview is blocked"
	} else if enginePreview.Status == "needs_approval" {
		engineMessage = "engine adapter preview is ready but still requires this execution approval gate"
	}
	add("engine_adapter_preview", engineStatus, engineMessage, map[string]any{
		"status":            enginePreview.Status,
		"execution_allowed": enginePreview.ExecutionAllowed,
		"blockers":          enginePreview.Blockers,
		"skipped":           options.SkipEnginePreview,
	})

	onlineWorkers := onlineWorkers(workers)
	add("worker_online", gateStatusBool(len(onlineWorkers) > 0), "at least one online worker must be available", map[string]any{
		"online_worker_count": len(onlineWorkers),
		"worker_count":        len(workers),
	})
	add("worker_capabilities", gateStatusBool(workerWithCapabilities(onlineWorkers, requiredCaps)), "an online worker must satisfy required execution capabilities", map[string]any{
		"required_capabilities": requiredCaps,
		"online_worker_count":   len(onlineWorkers),
	})
	add("read_only_boundary", "pass", "execution approval gate did not claim tasks, start workers, run commands, resolve secrets or write project files", map[string]any{
		"project_write_attempted":   false,
		"execution_write_attempted": false,
		"engine_call_attempted":     false,
		"commands_run":              false,
		"secrets_resolved":          false,
		"network_used":              false,
		"task_claimed":              false,
		"worker_started":            false,
		"attempt_created":           false,
		"artifact_created":          false,
	})
	return gate
}

func hasRunnableRunTask(tasks []RunTaskRecord) bool {
	return runnableRunTaskCount(tasks) > 0
}

func runnableRunTaskCount(tasks []RunTaskRecord) int {
	count := 0
	for _, task := range tasks {
		switch task.Status {
		case "queued", "pending":
			count++
		}
	}
	return count
}

func onlineWorkers(workers []WorkerRecord) []WorkerRecord {
	out := []WorkerRecord{}
	for _, worker := range workers {
		if worker.Status == "online" {
			out = append(out, worker)
		}
	}
	return out
}

func workerWithCapabilities(workers []WorkerRecord, required []string) bool {
	for _, worker := range workers {
		if len(missingWorkerCapabilities(worker.Capabilities, required)) == 0 {
			return true
		}
	}
	return false
}

func gateStatusBool(ok bool) string {
	if ok {
		return "pass"
	}
	return "blocked"
}

func (s Store) projectRecordByID(ctx context.Context, projectID int64) (Record, error) {
	var record Record
	err := s.pool.QueryRow(ctx, `
SELECT p.id, p.project_key, p.name, p.kind, p.adapter, p.workflow_profile, p.default_branch,
       COALESCE(c.root_path, ''), COALESCE(a.remote_url, ''), COALESCE(a.root_path, '')
FROM projects p
LEFT JOIN LATERAL (
    SELECT root_path
    FROM project_connections
    WHERE project_id = p.id AND connection_type = 'local_path'
    ORDER BY updated_at DESC, id DESC
    LIMIT 1
) c ON true
LEFT JOIN LATERAL (
    SELECT root_path, remote_url
    FROM project_connections
    WHERE project_id = p.id AND connection_type = 'artifact_store'
    ORDER BY updated_at DESC, id DESC
    LIMIT 1
) a ON true
WHERE p.id = $1`,
		projectID,
	).Scan(
		&record.ID,
		&record.Key,
		&record.Name,
		&record.Kind,
		&record.Adapter,
		&record.WorkflowProfile,
		&record.DefaultBranch,
		&record.RootPath,
		&record.ArtifactBackend,
		&record.ArtifactRoot,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Record{}, fmt.Errorf("project not found: %d", projectID)
		}
		return Record{}, fmt.Errorf("load project by id: %w", err)
	}
	return record, nil
}

func (s Store) workflowVersionByID(ctx context.Context, projectID int64, versionID int64) (WorkflowVersion, error) {
	version, err := scanWorkflowVersion(s.pool.QueryRow(ctx, `
SELECT id, project_id, display_label, version_kind, lifecycle_status,
       COALESCE(source_path, ''), COALESCE(source_hash, ''), import_mode,
       immutable, status_summary, created_at, updated_at, imported_at
FROM workflow_versions
WHERE project_id = $1 AND id = $2`,
		projectID,
		versionID,
	))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return WorkflowVersion{}, fmt.Errorf("%w: id %d", ErrWorkflowVersionNotFound, versionID)
		}
		return WorkflowVersion{}, fmt.Errorf("load workflow version by id: %w", err)
	}
	return version, nil
}
