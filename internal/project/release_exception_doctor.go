package project

import (
	"context"
	"fmt"
	"time"
)

type ReleaseExceptionDoctorOptions struct {
	GeneratedAt time.Time
	ProjectID   int64
	ProjectKey  string
}

type ReleaseExceptionDoctorCheck struct {
	Key      string
	Category string
	Status   string
	Message  string
	Metadata map[string]any
}

type ReleaseExceptionDoctor struct {
	Real100Guardrail
	Status           string
	Mode             string
	Scope            string
	ProjectKey       string
	Gate             ReleaseAcceptanceGate
	Checks           []ReleaseExceptionDoctorCheck
	Capabilities     []string
	ForbiddenActions []string
	GeneratedAt      time.Time
}

func (s Store) ReleaseExceptionDoctor(ctx context.Context, options ReleaseExceptionDoctorOptions) (ReleaseExceptionDoctor, error) {
	options = normalizeReleaseExceptionDoctorOptions(options)
	gate, err := s.ReleaseAcceptanceGate(ctx, ReleaseAcceptanceGateOptions{GeneratedAt: options.GeneratedAt, ProjectID: options.ProjectID, ProjectKey: options.ProjectKey})
	if err != nil {
		return ReleaseExceptionDoctor{}, err
	}
	return BuildReleaseExceptionDoctor(gate, options), nil
}

func normalizeReleaseExceptionDoctorOptions(options ReleaseExceptionDoctorOptions) ReleaseExceptionDoctorOptions {
	options.GeneratedAt, options.ProjectKey = normalizeReleaseScopeFields(options.GeneratedAt, options.ProjectKey)
	return options
}

func BuildReleaseExceptionDoctor(gate ReleaseAcceptanceGate, options ReleaseExceptionDoctorOptions) ReleaseExceptionDoctor {
	options = normalizeReleaseExceptionDoctorOptions(options)
	scope, projectKey := releaseScopeAndProjectKey(options.ProjectID, options.ProjectKey, gate.ProjectKey, gate.Preview.ProjectKey)
	doctor := ReleaseExceptionDoctor{
		Real100Guardrail: ReleasePreviewReal100Guardrail(),
		Status:           "pass",
		Mode:             "read_only_release_exception_doctor",
		Scope:            scope,
		ProjectKey:       projectKey,
		Gate:             gate,
		Checks:           []ReleaseExceptionDoctorCheck{},
		Capabilities: []string{
			"read_release_acceptance_gate",
			"check_exception_record_requirements",
			"report_exception_write_guardrails",
		},
		ForbiddenActions: []string{
			"write_database",
			"write_project_files",
			"write_artifact_store",
			"mark_gap_accepted",
			"create_approval",
			"execute_commands",
			"start_worker",
			"apply_release",
		},
		GeneratedAt: options.GeneratedAt,
	}
	doctor.addCheck(releaseExceptionSchemaCheck())
	doctor.addCheck(releaseExceptionAuditCheck())
	doctor.addCheck(releaseExceptionWriteGuardrailCheck())
	if len(gate.Items) == 0 {
		doctor.addCheck(ReleaseExceptionDoctorCheck{
			Key:      "exception_scope",
			Category: "scope",
			Status:   "fail",
			Message:  "release acceptance gate has no items to diagnose",
			Metadata: map[string]any{"gate_status": gate.Status},
		})
		return doctor
	}
	for _, item := range gate.Items {
		doctor.addCheck(releaseExceptionCheckForGateItem(item))
	}
	return doctor
}

func (d *ReleaseExceptionDoctor) addCheck(check ReleaseExceptionDoctorCheck) {
	if check.Metadata == nil {
		check.Metadata = map[string]any{}
	}
	d.Checks = append(d.Checks, check)
	if worseReleaseExceptionDoctorStatus(check.Status, d.Status) {
		d.Status = check.Status
	}
}

func releaseExceptionSchemaCheck() ReleaseExceptionDoctorCheck {
	return ReleaseExceptionDoctorCheck{
		Key:      "exception_record_schema",
		Category: "schema",
		Status:   "warn",
		Message:  "release exception record schema is designed but not enabled for writes",
		Metadata: map[string]any{
			"required_fields": []string{
				"exception_key",
				"source_decision",
				"acceptance_type",
				"owner",
				"reason",
				"required_evidence",
				"expires_or_review_at",
				"rollback_plan",
				"audit_event_id",
			},
			"writes_enabled": false,
		},
	}
}

func releaseExceptionAuditCheck() ReleaseExceptionDoctorCheck {
	return ReleaseExceptionDoctorCheck{
		Key:      "exception_audit_contract",
		Category: "audit",
		Status:   "warn",
		Message:  "release exception acceptance requires explicit audit events before writes are enabled",
		Metadata: map[string]any{
			"required_actions": []string{
				"release.exception.request",
				"release.exception.approve",
				"release.exception.revoke",
			},
			"writes_enabled": false,
		},
	}
}

func releaseExceptionWriteGuardrailCheck() ReleaseExceptionDoctorCheck {
	return ReleaseExceptionDoctorCheck{
		Key:      "exception_write_guardrails",
		Category: "safety",
		Status:   "pass",
		Message:  "release exception doctor is read-only and forbids accepting gaps",
		Metadata: map[string]any{
			"doctor_writes_database":     false,
			"doctor_marks_gap_accepted":  false,
			"doctor_creates_approval":    false,
			"doctor_applies_release":     false,
			"doctor_executes_commands":   false,
			"doctor_starts_worker":       false,
			"exception_writes_confirmed": false,
		},
	}
}

func releaseExceptionCheckForGateItem(item ReleaseAcceptanceGateItem) ReleaseExceptionDoctorCheck {
	metadata := map[string]any{
		"gate_item_key":      item.Key,
		"gate_item_status":   item.Status,
		"decision_status":    item.DecisionStatus,
		"acceptance_type":    item.AcceptanceType,
		"owner":              item.Owner,
		"required_evidence":  item.RequiredEvidence,
		"next_command":       item.NextCommand,
		"exception_writable": false,
	}
	switch item.DecisionStatus {
	case "ready":
		return ReleaseExceptionDoctorCheck{
			Key:      "exception:" + item.Key,
			Category: item.Category,
			Status:   "pass",
			Message:  "release acceptance gate item is ready and does not require a new exception record",
			Metadata: metadata,
		}
	case "needs_decision":
		return ReleaseExceptionDoctorCheck{
			Key:      "exception:" + item.Key,
			Category: item.Category,
			Status:   "warn",
			Message:  "release exception record is required before this gate item can pass",
			Metadata: metadata,
		}
	case "not_acceptable":
		return ReleaseExceptionDoctorCheck{
			Key:      "exception:" + item.Key,
			Category: item.Category,
			Status:   "fail",
			Message:  "release blocker is not acceptable and must be remediated instead of recorded as an exception",
			Metadata: metadata,
		}
	default:
		return ReleaseExceptionDoctorCheck{
			Key:      "exception:" + item.Key,
			Category: item.Category,
			Status:   "fail",
			Message:  fmt.Sprintf("release gate item has unsupported decision status %q", item.DecisionStatus),
			Metadata: metadata,
		}
	}
}

func worseReleaseExceptionDoctorStatus(candidate string, current string) bool {
	return releaseExceptionDoctorStatusRank(candidate) > releaseExceptionDoctorStatusRank(current)
}

func releaseExceptionDoctorStatusRank(status string) int {
	switch status {
	case "fail", "blocked":
		return 3
	case "warn", "needs_attention":
		return 2
	case "pass", "ready":
		return 1
	default:
		return 0
	}
}
