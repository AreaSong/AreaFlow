package project

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"
)

const statusProjectionApplyPacketMode = "status_projection_apply_packet_preview_v1"

type StatusProjectionApplyPacketPreviewOptions struct {
	TargetURI        string
	ExplicitApproval bool
	ApprovalActor    string
	ApprovalReason   string
	GeneratedAt      time.Time
}

type StatusProjectionApplyPacketPreview struct {
	Project                                      Record
	Status                                       string
	Mode                                         string
	ClaimScope                                   string
	NotReal100                                   bool
	Decision                                     string
	Message                                      string
	Blockers                                     []string
	RequiredAuthorizationPhrase                  string
	Authorization                                StatusProjectionAuthorizationPreview
	Gate                                         StatusProjectionApplyGate
	Packet                                       StatusProjectionApplyPacket
	ApplyCommand                                 []string
	APIRequest                                   StatusProjectionApplyAPIRequest
	RequiredHumanReview                          []string
	ForbiddenActions                             []string
	SafetyFacts                                  map[string]bool
	WouldCreateCommandRequestAfterApplyCommand   bool
	WouldCreateStatusProjectionAfterApplyCommand bool
	WouldWriteProjectFileAfterApplyCommand       bool
	ApplyCommandEligibleIsNotApply               bool
	RequiresSeparateApplyCommand                 bool
	ProjectWriteAttempted                        bool
	ExecutionWriteAttempted                      bool
	EngineCallAttempted                          bool
	CommandRequestCreated                        bool
	StatusProjectionWritten                      bool
	GeneratedAt                                  time.Time
}

type StatusProjectionApplyPacket struct {
	TargetURI                      string
	ExpectedBeforeExists           bool
	ExpectedBeforeSHA256           string
	ExpectedBeforeSizeBytes        int64
	SourceHash                     string
	SchemaURI                      string
	ValidatorPreflight             string
	ProtectedPathCheck             string
	ProtectedPathFingerprintSHA256 string
	RollbackAction                 string
	AcceptedPreimageSchemaStatus   string
	ExplicitApproval               bool
	ApprovalActor                  string
	ApprovalReason                 string
	RequiredAuthorizationPhrase    string
}

type StatusProjectionApplyAPIRequest struct {
	TargetURI                      string
	ExpectedBeforeExists           bool
	ExpectedBeforeSHA256           string
	ExpectedBeforeSizeBytes        int64
	SourceHash                     string
	SchemaURI                      string
	ValidatorPreflight             string
	ProtectedPathCheck             string
	ProtectedPathFingerprintSHA256 string
	RollbackAction                 string
	AcceptedPreimageSchemaStatus   string
	ExplicitApproval               bool
	ApprovalActor                  string
	ApprovalReason                 string
	RequiredAuthorizationPhrase    string
}

func (s Store) StatusProjectionApplyPacketPreview(ctx context.Context, record Record, options StatusProjectionApplyPacketPreviewOptions) (StatusProjectionApplyPacketPreview, error) {
	options = normalizeStatusProjectionApplyPacketPreviewOptions(options)
	authorization, err := s.StatusProjectionAuthorizationPreview(ctx, record, StatusProjectionAuthorizationPreviewOptions{
		TargetURI:   options.TargetURI,
		GeneratedAt: options.GeneratedAt,
	})
	if err != nil {
		return StatusProjectionApplyPacketPreview{}, err
	}
	return BuildStatusProjectionApplyPacketPreview(authorization, options), nil
}

func normalizeStatusProjectionApplyPacketPreviewOptions(options StatusProjectionApplyPacketPreviewOptions) StatusProjectionApplyPacketPreviewOptions {
	options.TargetURI = stringsTrim(options.TargetURI)
	options.ApprovalActor = stringsTrim(options.ApprovalActor)
	options.ApprovalReason = stringsTrim(options.ApprovalReason)
	if options.TargetURI == "" {
		options.TargetURI = ".areaflow/status.json"
	}
	if options.GeneratedAt.IsZero() {
		options.GeneratedAt = time.Now().UTC()
	}
	return options
}

func BuildStatusProjectionApplyPacketPreview(authorization StatusProjectionAuthorizationPreview, options StatusProjectionApplyPacketPreviewOptions) StatusProjectionApplyPacketPreview {
	options = normalizeStatusProjectionApplyPacketPreviewOptions(options)
	packet := statusProjectionApplyPacketFromAuthorization(authorization, options)
	gateOptions := statusProjectionApplyGateOptionsFromPacket(packet, options.GeneratedAt)
	gate := BuildStatusProjectionApplyGate(authorization, gateOptions)
	preview := StatusProjectionApplyPacketPreview{
		Project:                     authorization.Project,
		Status:                      "needs_approval",
		Mode:                        statusProjectionApplyPacketMode,
		ClaimScope:                  statusProjectionClaimScope,
		NotReal100:                  true,
		Decision:                    "needs_explicit_approval",
		Message:                     "status projection apply packet is generated but still needs explicit approval",
		Blockers:                    statusProjectionApplyPacketBlockers(authorization, gate),
		RequiredAuthorizationPhrase: statusProjectionRequiredAuthorizationPhrase(authorization),
		Authorization:               authorization,
		Gate:                        gate,
		Packet:                      packet,
		ApplyCommand:                statusProjectionApplyPacketCommand(authorization.Project, packet),
		APIRequest:                  statusProjectionApplyPacketAPIRequest(packet),
		RequiredHumanReview: []string{
			"review target preimage schema status",
			"review expected-before hash and size",
			"review rollback action",
			"run validator preflight before apply",
			"run protected path check before and after apply",
		},
		ForbiddenActions:               append([]string{}, authorization.ForbiddenActions...),
		SafetyFacts:                    statusProjectionApplyPacketSafetyFacts(),
		ApplyCommandEligibleIsNotApply: true,
		RequiresSeparateApplyCommand:   true,
		GeneratedAt:                    options.GeneratedAt,
	}
	if gate.ApplyCommandEligible {
		preview.Status = "ready"
		preview.Decision = "ready_for_apply_command"
		preview.Message = "status projection apply packet is ready for the protected apply command"
		preview.Blockers = nil
		preview.WouldCreateCommandRequestAfterApplyCommand = true
		preview.WouldCreateStatusProjectionAfterApplyCommand = true
		preview.WouldWriteProjectFileAfterApplyCommand = true
	} else if gate.Status == "blocked" && authorization.Status == "blocked" {
		preview.Status = "blocked"
		preview.Decision = "blocked"
		preview.Message = "status projection apply packet is blocked by authorization preview"
	}
	return preview
}

func statusProjectionApplyPacketBlockers(authorization StatusProjectionAuthorizationPreview, gate StatusProjectionApplyGate) []string {
	blockers := append([]string{}, authorization.BlockedBy...)
	for _, item := range gate.Items {
		if item.Status == "pass" {
			continue
		}
		blockers = append(blockers, item.BlockedBy...)
		if len(item.BlockedBy) == 0 {
			blockers = append(blockers, item.Key)
		}
	}
	return uniqueStrings(blockers)
}

func statusProjectionApplyPacketFromAuthorization(authorization StatusProjectionAuthorizationPreview, options StatusProjectionApplyPacketPreviewOptions) StatusProjectionApplyPacket {
	return StatusProjectionApplyPacket{
		TargetURI:                      authorization.TargetURI,
		ExpectedBeforeExists:           authorization.Preimage.Exists,
		ExpectedBeforeSHA256:           authorization.Preimage.SHA256,
		ExpectedBeforeSizeBytes:        authorization.Preimage.SizeBytes,
		SourceHash:                     authorization.SourceHash,
		SchemaURI:                      authorization.SchemaURI,
		ValidatorPreflight:             authorization.ValidatorPreflight,
		ProtectedPathCheck:             statusProjectionProtectedPathCheck(authorization.Project),
		ProtectedPathFingerprintSHA256: authorization.ProtectedPathFingerprintSHA256,
		RollbackAction:                 statusProjectionRollbackAction(authorization.Preimage),
		AcceptedPreimageSchemaStatus:   authorization.Preimage.SchemaStatus,
		ExplicitApproval:               options.ExplicitApproval,
		ApprovalActor:                  options.ApprovalActor,
		ApprovalReason:                 options.ApprovalReason,
		RequiredAuthorizationPhrase:    statusProjectionRequiredAuthorizationPhrase(authorization),
	}
}

func statusProjectionApplyGateOptionsFromPacket(packet StatusProjectionApplyPacket, generatedAt time.Time) StatusProjectionApplyGateOptions {
	expectedBeforeExists := packet.ExpectedBeforeExists
	expectedBeforeSize := packet.ExpectedBeforeSizeBytes
	return StatusProjectionApplyGateOptions{
		TargetURI:                      packet.TargetURI,
		ExpectedBeforeExists:           &expectedBeforeExists,
		ExpectedBeforeSHA256:           packet.ExpectedBeforeSHA256,
		ExpectedBeforeSizeBytes:        &expectedBeforeSize,
		SourceHash:                     packet.SourceHash,
		SchemaURI:                      packet.SchemaURI,
		ValidatorPreflight:             packet.ValidatorPreflight,
		ProtectedPathCheck:             packet.ProtectedPathCheck,
		ProtectedPathFingerprintSHA256: packet.ProtectedPathFingerprintSHA256,
		RollbackAction:                 packet.RollbackAction,
		AcceptedPreimageSchemaStatus:   packet.AcceptedPreimageSchemaStatus,
		ExplicitApproval:               packet.ExplicitApproval,
		ApprovalActor:                  packet.ApprovalActor,
		ApprovalReason:                 packet.ApprovalReason,
		GeneratedAt:                    generatedAt,
	}
}

func statusProjectionApplyPacketCommand(record Record, packet StatusProjectionApplyPacket) []string {
	command := []string{
		"areaflow", "project", "status-projection-apply", record.Key,
		"--target", packet.TargetURI,
		"--expected-before-exists", strconv.FormatBool(packet.ExpectedBeforeExists),
		"--expected-before-size", fmt.Sprintf("%d", packet.ExpectedBeforeSizeBytes),
		"--source-hash", packet.SourceHash,
		"--schema-uri", packet.SchemaURI,
		"--validator-preflight", packet.ValidatorPreflight,
		"--protected-path-check", packet.ProtectedPathCheck,
		"--protected-path-fingerprint-sha256", packet.ProtectedPathFingerprintSHA256,
		"--rollback-action", packet.RollbackAction,
		"--accept-preimage-schema", packet.AcceptedPreimageSchemaStatus,
	}
	if packet.ExpectedBeforeSHA256 != "" {
		command = append(command, "--expected-before-sha256", packet.ExpectedBeforeSHA256)
	}
	if packet.ExplicitApproval {
		command = append(command, "--explicit-approval")
	}
	if packet.ApprovalActor != "" {
		command = append(command, "--approval-actor", packet.ApprovalActor)
	}
	if packet.ApprovalReason != "" {
		command = append(command, "--approval-reason", packet.ApprovalReason)
	}
	return command
}

func statusProjectionApplyPacketAPIRequest(packet StatusProjectionApplyPacket) StatusProjectionApplyAPIRequest {
	return StatusProjectionApplyAPIRequest{
		TargetURI:                      packet.TargetURI,
		ExpectedBeforeExists:           packet.ExpectedBeforeExists,
		ExpectedBeforeSHA256:           packet.ExpectedBeforeSHA256,
		ExpectedBeforeSizeBytes:        packet.ExpectedBeforeSizeBytes,
		SourceHash:                     packet.SourceHash,
		SchemaURI:                      packet.SchemaURI,
		ValidatorPreflight:             packet.ValidatorPreflight,
		ProtectedPathCheck:             packet.ProtectedPathCheck,
		ProtectedPathFingerprintSHA256: packet.ProtectedPathFingerprintSHA256,
		RollbackAction:                 packet.RollbackAction,
		AcceptedPreimageSchemaStatus:   packet.AcceptedPreimageSchemaStatus,
		ExplicitApproval:               packet.ExplicitApproval,
		ApprovalActor:                  packet.ApprovalActor,
		ApprovalReason:                 packet.ApprovalReason,
		RequiredAuthorizationPhrase:    packet.RequiredAuthorizationPhrase,
	}
}

func statusProjectionApplyPacketSafetyFacts() map[string]bool {
	return map[string]bool{
		"read_only_preview":                  true,
		"apply_packet_generated":             true,
		"apply_command_executed":             false,
		"command_request_created":            false,
		"status_projection_written":          false,
		"project_write_attempted":            false,
		"execution_write_attempted":          false,
		"engine_call_attempted":              false,
		"commands_run":                       false,
		"worker_scheduled":                   false,
		"secrets_resolved":                   false,
		"network_used":                       false,
		"areamatrix_protected_paths_touched": false,
	}
}

func stringsTrim(value string) string {
	return strings.TrimSpace(value)
}
