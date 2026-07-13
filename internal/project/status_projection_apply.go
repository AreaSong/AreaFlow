package project

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

const statusProjectionApplyCommandType = "project.status_projection.apply"

type StatusProjectionWriter func(ctx context.Context, record Record, snapshot Snapshot, targetURI string) (StatusProjectionWriteResult, error)

type StatusProjectionWriteResult struct {
	Target                    string
	Hash                      string
	Size                      int64
	RootContained             bool
	StableProjectionValidated bool
	AtomicReplaceUsed         bool
}

type statusProjectionFilePreimage struct {
	TargetPath string
	Exists     bool
	Content    []byte
	SHA256     string
	SizeBytes  int64
}

type ApplyStatusProjectionOptions struct {
	TargetURI      string
	IdempotencyKey string
	Actor          string
	Reason         string
	Writer         StatusProjectionWriter
	Gate           StatusProjectionApplyGateOptions
}

type ApplyStatusProjectionResult struct {
	Project                   Record
	Status                    string
	Decision                  string
	Message                   string
	Blockers                  []string
	EventID                   int64
	AuditEventID              int64
	SnapshotID                int64
	StatusProjectionID        int64
	TargetKind                string
	TargetURI                 string
	WrittenTarget             string
	WriteHash                 string
	WriteSize                 int64
	PreimageCaptured          bool
	PreimageExists            bool
	PreimageSHA256            string
	PreimageSize              int64
	PostWriteVerified         bool
	PostWriteSHA256           string
	PostWriteSize             int64
	ProtectedPathsVerified    bool
	ProtectedPathBeforeHash   string
	ProtectedPathAfterHash    string
	ExpectedProtectedPathHash string
	RootContained             bool
	StableProjectionValid     bool
	AtomicReplaceUsed         bool
	RollbackCompensation      bool
	SourceHash                string
	SummaryState              string
	ApplyGateStatus           string
	ApplyGateDecision         string
	ApplyGateApprovalStatus   string
	ApplyCommandEligible      bool
	IdempotencyKey            string
	Created                   bool
	GeneratedAt               time.Time
	ProjectWriteAttempted     bool
	ExecutionWriteAttempted   bool
	EngineCallAttempted       bool
}

func (s Store) ApplyStatusProjection(ctx context.Context, record Record, options ApplyStatusProjectionOptions) (ApplyStatusProjectionResult, error) {
	options = normalizeApplyStatusProjectionOptions(options)
	if options.Writer == nil {
		return ApplyStatusProjectionResult{}, fmt.Errorf("status projection writer is required")
	}

	snapshot, err := s.LatestImportSnapshot(ctx, record.ID)
	if err != nil {
		return ApplyStatusProjectionResult{}, err
	}
	summaryJSON, err := json.Marshal(snapshot.Summary)
	if err != nil {
		return ApplyStatusProjectionResult{}, fmt.Errorf("marshal status projection summary: %w", err)
	}
	requestHash, err := statusProjectionApplyRequestHash(record, options, snapshot, summaryJSON)
	if err != nil {
		return ApplyStatusProjectionResult{}, err
	}
	if options.IdempotencyKey == "" {
		options.IdempotencyKey = statusProjectionApplyIdempotencyKey(record, options, snapshot, summaryJSON)
	}
	gate, err := s.StatusProjectionApplyGate(ctx, record, options.Gate)
	if err != nil {
		return ApplyStatusProjectionResult{}, err
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return ApplyStatusProjectionResult{}, fmt.Errorf("begin status projection apply: %w", err)
	}
	defer tx.Rollback(ctx)

	created, err := reserveCommandRequest(ctx, tx, record.ID, statusProjectionApplyCommandType, options.IdempotencyKey, requestHash)
	if err != nil {
		return ApplyStatusProjectionResult{}, err
	}
	if !created {
		result, err := loadStatusProjectionApplyByCommandResponse(ctx, tx, record, options.IdempotencyKey)
		if err != nil {
			return ApplyStatusProjectionResult{}, err
		}
		if err := tx.Commit(ctx); err != nil {
			return ApplyStatusProjectionResult{}, fmt.Errorf("commit idempotent status projection apply: %w", err)
		}
		result.Created = false
		return result, nil
	}

	result := evaluateStatusProjectionApply(ctx, tx, record, snapshot, options, gate)
	var preimage statusProjectionFilePreimage
	wroteProjectFile := false
	if result.Decision == "allowed" {
		preimage, wroteProjectFile, err = runStatusProjectionApplyWriter(ctx, record, snapshot, options.TargetURI, options.Gate, options.Writer, &result)
		if err != nil {
			return ApplyStatusProjectionResult{}, err
		}
	}
	handlePostWriteError := func(err error) (ApplyStatusProjectionResult, error) {
		if wroteProjectFile {
			if rollbackErr := rollbackStatusProjectionApplyFile(preimage, result.WriteHash); rollbackErr != nil {
				return ApplyStatusProjectionResult{}, fmt.Errorf("%w; rollback status projection file: %v", err, rollbackErr)
			}
		}
		return ApplyStatusProjectionResult{}, err
	}

	snapshotID, err := insertStatusProjectionSnapshot(ctx, tx, record, snapshot, options, result, summaryJSON)
	if err != nil {
		return handlePostWriteError(err)
	}
	result.SnapshotID = snapshotID
	projectionID, err := insertAppliedStatusProjection(ctx, tx, record, snapshot, options, result, summaryJSON)
	if err != nil {
		return handlePostWriteError(err)
	}
	result.StatusProjectionID = projectionID
	eventID, err := insertStatusProjectionApplyEvent(ctx, tx, result, options)
	if err != nil {
		return handlePostWriteError(err)
	}
	result.EventID = eventID
	auditEventID, err := insertStatusProjectionApplyAuditEvent(ctx, tx, result, options)
	if err != nil {
		return handlePostWriteError(err)
	}
	result.AuditEventID = auditEventID
	result.IdempotencyKey = options.IdempotencyKey
	result.Created = true
	if err := completeCommandRequestResponse(ctx, tx, record.ID, statusProjectionApplyCommandType, options.IdempotencyKey, statusProjectionApplyCommandResponse(result)); err != nil {
		return handlePostWriteError(err)
	}
	if err := tx.Commit(ctx); err != nil {
		return ApplyStatusProjectionResult{}, fmt.Errorf("commit status projection apply: %w", err)
	}
	return result, nil
}

func normalizeApplyStatusProjectionOptions(options ApplyStatusProjectionOptions) ApplyStatusProjectionOptions {
	options.TargetURI = strings.TrimSpace(options.TargetURI)
	options.IdempotencyKey = strings.TrimSpace(options.IdempotencyKey)
	options.Actor = strings.TrimSpace(options.Actor)
	options.Reason = strings.TrimSpace(options.Reason)
	options.Gate = normalizeStatusProjectionApplyGateOptions(options.Gate)
	if options.TargetURI == "" {
		options.TargetURI = ".areaflow/status.json"
	}
	if options.Gate.TargetURI == "" {
		options.Gate.TargetURI = options.TargetURI
	}
	if options.Gate.TargetURI != options.TargetURI {
		options.Gate.TargetURI = options.TargetURI
	}
	if options.Actor == "" {
		options.Actor = "local-user"
	}
	if options.Reason == "" {
		options.Reason = "status projection apply"
	}
	return options
}

func statusProjectionApplyRequestHash(record Record, options ApplyStatusProjectionOptions, snapshot Snapshot, summaryJSON []byte) (string, error) {
	payload, err := json.Marshal(map[string]any{
		"command_type": statusProjectionApplyCommandType,
		"project_id":   record.ID,
		"project_key":  record.Key,
		"project_root": record.RootPath,
		"target_kind":  statusProjectionTargetKind(options.TargetURI),
		"target_uri":   options.TargetURI,
		"source_hash":  snapshot.SourceHash,
		"summary":      json.RawMessage(summaryJSON),
		"actor":        options.Actor,
		"reason":       options.Reason,
		"gate_packet":  statusProjectionApplyGateRequestHashPayload(options.Gate),
		"protected":    true,
	})
	if err != nil {
		return "", fmt.Errorf("marshal status projection apply request hash payload: %w", err)
	}
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:]), nil
}

func statusProjectionApplyIdempotencyKey(record Record, options ApplyStatusProjectionOptions, snapshot Snapshot, summaryJSON []byte) string {
	return fmt.Sprintf("project.status_projection.apply:%s:%s:%s:%s:%s:%s",
		record.Key,
		options.TargetURI,
		shortSHA256Hex([]byte(strings.TrimSpace(record.RootPath))),
		strings.TrimSpace(snapshot.SourceHash),
		statusProjectionApplyGatePacketHash(options.Gate),
		shortSHA256Hex(summaryJSON),
	)
}

func statusProjectionApplyGatePacketHash(options StatusProjectionApplyGateOptions) string {
	payload, err := json.Marshal(statusProjectionApplyGateRequestHashPayload(options))
	if err != nil {
		return shortSHA256Hex([]byte("invalid-gate-packet"))
	}
	return shortSHA256Hex(payload)
}

func statusProjectionApplyGateRequestHashPayload(options StatusProjectionApplyGateOptions) map[string]any {
	payload := map[string]any{
		"target_uri":                        options.TargetURI,
		"expected_before_sha256":            options.ExpectedBeforeSHA256,
		"source_hash":                       options.SourceHash,
		"schema_uri":                        options.SchemaURI,
		"validator_preflight":               options.ValidatorPreflight,
		"protected_path_check":              options.ProtectedPathCheck,
		"protected_path_fingerprint_sha256": options.ProtectedPathFingerprintSHA256,
		"rollback_action":                   options.RollbackAction,
		"accepted_preimage_schema_status":   options.AcceptedPreimageSchemaStatus,
		"explicit_approval":                 options.ExplicitApproval,
		"approval_actor":                    options.ApprovalActor,
		"approval_reason":                   options.ApprovalReason,
	}
	if options.ExpectedBeforeExists != nil {
		payload["expected_before_exists"] = *options.ExpectedBeforeExists
	}
	if options.ExpectedBeforeSizeBytes != nil {
		payload["expected_before_size_bytes"] = *options.ExpectedBeforeSizeBytes
	}
	return payload
}

func statusProjectionTargetKind(targetURI string) string {
	switch strings.TrimSpace(targetURI) {
	case ".areaflow/status.json":
		return "project_status_json"
	case "workflow/README.md":
		return "workflow_readme"
	default:
		return "unknown"
	}
}

func evaluateStatusProjectionApply(ctx context.Context, tx pgx.Tx, record Record, snapshot Snapshot, options ApplyStatusProjectionOptions, gate StatusProjectionApplyGate) ApplyStatusProjectionResult {
	result := ApplyStatusProjectionResult{
		Project:                 record,
		Status:                  "written",
		Decision:                "allowed",
		Message:                 "status projection written",
		TargetKind:              statusProjectionTargetKind(options.TargetURI),
		TargetURI:               options.TargetURI,
		SourceHash:              snapshot.SourceHash,
		SummaryState:            statusProjectionSummaryState(snapshot.Summary),
		ApplyGateStatus:         gate.Status,
		ApplyGateDecision:       gate.Decision,
		ApplyGateApprovalStatus: gate.ApprovalStatus,
		ApplyCommandEligible:    gate.ApplyCommandEligible,
		GeneratedAt:             time.Now().UTC(),
		ProjectWriteAttempted:   false,
		ExecutionWriteAttempted: false,
		EngineCallAttempted:     false,
	}
	blockers := []string{}
	if options.TargetURI != ".areaflow/status.json" {
		blockers = append(blockers, "status projection apply currently only supports .areaflow/status.json")
	}
	if gate.Status != "pass" || gate.Decision != "go" || !gate.ApplyCommandEligible {
		blockers = append(blockers, statusProjectionApplyGateBlockers(gate)...)
	}
	allowed, reason, err := canWritePathTx(ctx, tx, record.ID, "write_status", options.TargetURI)
	if err != nil {
		blockers = append(blockers, err.Error())
	} else if !allowed {
		blockers = append(blockers, reason)
	}
	if len(blockers) > 0 {
		result.Status = "blocked"
		result.Decision = "denied"
		result.Message = "status projection apply blocked by protected boundary"
		result.Blockers = blockers
		return result
	}
	result.ProjectWriteAttempted = true
	return result
}

func statusProjectionApplyGateBlockers(gate StatusProjectionApplyGate) []string {
	blockers := []string{"status_projection_apply_gate_blocked"}
	for _, item := range gate.Items {
		if item.Status == "pass" {
			continue
		}
		if len(item.BlockedBy) == 0 {
			blockers = append(blockers, item.Key)
			continue
		}
		blockers = append(blockers, item.BlockedBy...)
	}
	return uniqueStrings(blockers)
}

func canWritePathTx(ctx context.Context, tx pgx.Tx, projectID int64, capability string, path string) (bool, string, error) {
	rows, err := tx.Query(ctx, `
SELECT effect, capability, resource_type, pattern
FROM project_permissions
WHERE project_id = $1 AND resource_type IN ('capability', 'path')
ORDER BY id`,
		projectID,
	)
	if err != nil {
		return false, "", fmt.Errorf("load project permissions: %w", err)
	}
	defer rows.Close()

	capabilityAllowed := false
	pathAllowed := false
	for rows.Next() {
		var effect, permissionCapability, resourceType, pattern string
		if err := rows.Scan(&effect, &permissionCapability, &resourceType, &pattern); err != nil {
			return false, "", fmt.Errorf("scan project permission: %w", err)
		}
		if effect == "deny" && resourceType == "path" && globMatch(pattern, path) {
			return false, "path denied by forbidden path", nil
		}
		if resourceType == "capability" && permissionCapability == capability && effect == "allow" {
			capabilityAllowed = true
		}
		if resourceType == "path" && permissionCapability == capability && effect == "allow" && globMatch(pattern, path) {
			pathAllowed = true
		}
	}
	if err := rows.Err(); err != nil {
		return false, "", fmt.Errorf("iterate project permissions: %w", err)
	}
	if !capabilityAllowed {
		return false, "capability not allowed", nil
	}
	if !pathAllowed {
		return false, "path not allowed", nil
	}
	return true, "allowed", nil
}

func runStatusProjectionApplyWriter(ctx context.Context, record Record, snapshot Snapshot, targetURI string, gateOptions StatusProjectionApplyGateOptions, writer StatusProjectionWriter, result *ApplyStatusProjectionResult) (statusProjectionFilePreimage, bool, error) {
	preimage, err := captureStatusProjectionApplyPreimage(record, targetURI)
	if err != nil {
		result.Decision = "denied"
		result.Status = "blocked"
		result.Message = "status projection preimage capture failed"
		result.Blockers = append(result.Blockers, err.Error())
		result.ProjectWriteAttempted = false
		return statusProjectionFilePreimage{}, false, nil
	}

	result.PreimageCaptured = true
	result.PreimageExists = preimage.Exists
	result.PreimageSHA256 = preimage.SHA256
	result.PreimageSize = preimage.SizeBytes
	if blockers := statusProjectionApplyPreimageRecheckBlockers(preimage, gateOptions); len(blockers) > 0 {
		result.Decision = "denied"
		result.Status = "blocked"
		result.Message = "status projection preimage changed before write"
		result.Blockers = append(result.Blockers, blockers...)
		result.ProjectWriteAttempted = false
		return preimage, false, nil
	}
	protectedBefore, err := captureStatusProjectionProtectedPathFingerprint(record, targetURI)
	if err != nil {
		result.Decision = "denied"
		result.Status = "blocked"
		result.Message = "status projection protected path fingerprint failed before write"
		result.Blockers = append(result.Blockers, err.Error())
		result.ProjectWriteAttempted = false
		return preimage, false, nil
	}
	result.ProtectedPathBeforeHash = protectedBefore.Hash
	result.ExpectedProtectedPathHash = strings.TrimSpace(gateOptions.ProtectedPathFingerprintSHA256)
	if result.ExpectedProtectedPathHash == "" || protectedBefore.Hash != result.ExpectedProtectedPathHash {
		result.Decision = "denied"
		result.Status = "blocked"
		result.Message = "status projection protected path fingerprint changed before write"
		result.Blockers = append(result.Blockers, "protected_path_fingerprint_missing_or_mismatch_before_write")
		result.ProjectWriteAttempted = false
		return preimage, false, nil
	}

	writeResult, err := writer(ctx, record, snapshot, targetURI)
	if err != nil {
		result.Decision = "denied"
		result.Status = "blocked"
		result.Message = "status projection write failed"
		result.Blockers = append(result.Blockers, err.Error())
		verification, rolledBack, rollbackErr := rollbackStatusProjectionApplyPartialWriterError(preimage)
		result.PostWriteVerified = false
		result.PostWriteSHA256 = verification.SHA256
		result.PostWriteSize = verification.Size
		if rolledBack {
			result.RollbackCompensation = true
		}
		if rollbackErr != nil {
			return preimage, false, fmt.Errorf("status projection write failed: %w; rollback status projection file: %v", err, rollbackErr)
		}
		return preimage, false, nil
	}

	result.WrittenTarget = writeResult.Target
	result.WriteHash = writeResult.Hash
	result.WriteSize = writeResult.Size
	result.RootContained = writeResult.RootContained
	result.StableProjectionValid = writeResult.StableProjectionValidated
	result.AtomicReplaceUsed = writeResult.AtomicReplaceUsed
	result.RollbackCompensation = true
	verification, err := verifyStatusProjectionApplyWrittenFile(writeResult, preimage.TargetPath, snapshot.SourceHash)
	result.PostWriteVerified = verification.Verified
	result.PostWriteSHA256 = verification.SHA256
	result.PostWriteSize = verification.Size
	result.RootContained = verification.RootContained
	result.StableProjectionValid = verification.StableProjectionValidated
	if err != nil {
		result.Decision = "denied"
		result.Status = "blocked"
		result.Message = "status projection post-write verification failed"
		result.Blockers = append(result.Blockers, err.Error())
		rollbackHash := result.PostWriteSHA256
		if rollbackHash == "" {
			rollbackHash = result.WriteHash
		}
		if rollbackErr := rollbackStatusProjectionApplyFile(preimage, rollbackHash); rollbackErr != nil {
			return preimage, false, fmt.Errorf("status projection post-write verification failed: %w; rollback status projection file: %v", err, rollbackErr)
		}
		return preimage, false, nil
	}
	protectedAfter, err := captureStatusProjectionProtectedPathFingerprint(record, targetURI)
	result.ProtectedPathAfterHash = protectedAfter.Hash
	if err != nil {
		result.Decision = "denied"
		result.Status = "blocked"
		result.Message = "status projection protected path fingerprint failed after write"
		result.Blockers = append(result.Blockers, err.Error())
		rollbackHash := result.PostWriteSHA256
		if rollbackHash == "" {
			rollbackHash = result.WriteHash
		}
		if rollbackErr := rollbackStatusProjectionApplyFile(preimage, rollbackHash); rollbackErr != nil {
			return preimage, false, fmt.Errorf("status projection protected path fingerprint failed after write: %w; rollback status projection file: %v", err, rollbackErr)
		}
		return preimage, false, nil
	}
	if protectedBefore.Hash != protectedAfter.Hash {
		result.Decision = "denied"
		result.Status = "blocked"
		result.Message = "status projection protected paths changed during write"
		result.Blockers = append(result.Blockers, "protected_path_fingerprint_changed_after_write")
		rollbackHash := result.PostWriteSHA256
		if rollbackHash == "" {
			rollbackHash = result.WriteHash
		}
		if rollbackErr := rollbackStatusProjectionApplyFile(preimage, rollbackHash); rollbackErr != nil {
			return preimage, false, fmt.Errorf("status projection protected paths changed during write; rollback status projection file: %v", rollbackErr)
		}
		return preimage, false, nil
	}
	result.ProtectedPathsVerified = true
	return preimage, true, nil
}

func statusProjectionApplyPreimageRecheckBlockers(preimage statusProjectionFilePreimage, gateOptions StatusProjectionApplyGateOptions) []string {
	gateOptions = normalizeStatusProjectionApplyGateOptions(gateOptions)
	blockers := []string{}
	if gateOptions.ExpectedBeforeExists == nil {
		blockers = append(blockers, "expected_before_exists_missing_before_write")
	} else if preimage.Exists != *gateOptions.ExpectedBeforeExists {
		blockers = append(blockers, "expected_before_exists_changed_before_write")
	}
	if gateOptions.ExpectedBeforeSizeBytes == nil {
		blockers = append(blockers, "expected_before_size_bytes_missing_before_write")
	} else if preimage.SizeBytes != *gateOptions.ExpectedBeforeSizeBytes {
		blockers = append(blockers, "expected_before_size_bytes_changed_before_write")
	}
	expectedSHA := strings.TrimSpace(gateOptions.ExpectedBeforeSHA256)
	if preimage.Exists {
		if expectedSHA == "" {
			blockers = append(blockers, "expected_before_sha256_missing_before_write")
		} else if preimage.SHA256 != expectedSHA {
			blockers = append(blockers, "expected_before_sha256_changed_before_write")
		}
	} else if expectedSHA != "" {
		blockers = append(blockers, "expected_before_sha256_changed_before_write")
	}
	return uniqueStrings(blockers)
}

func rollbackStatusProjectionApplyPartialWriterError(preimage statusProjectionFilePreimage) (statusProjectionPostWriteVerification, bool, error) {
	verification := statusProjectionPostWriteVerification{}
	info, err := os.Stat(preimage.TargetPath)
	if os.IsNotExist(err) {
		if !preimage.Exists {
			return verification, false, nil
		}
		if err := restoreStatusProjectionPreimageAtomically(preimage.TargetPath, preimage.Content); err != nil {
			return verification, true, fmt.Errorf("restore missing status projection preimage after writer error: %w", err)
		}
		return verification, true, nil
	}
	if err != nil {
		return verification, false, fmt.Errorf("stat status projection after writer error: %w", err)
	}
	if info.IsDir() {
		return verification, false, fmt.Errorf("status projection target is a directory after writer error")
	}
	content, err := os.ReadFile(preimage.TargetPath)
	if err != nil {
		return verification, false, fmt.Errorf("read status projection after writer error: %w", err)
	}
	verification.SHA256 = sha256Hex(content)
	verification.Size = int64(len(content))
	if preimage.Exists && verification.SHA256 == preimage.SHA256 {
		return verification, false, nil
	}
	if err := rollbackStatusProjectionApplyFile(preimage, verification.SHA256); err != nil {
		return verification, true, err
	}
	return verification, true, nil
}

func captureStatusProjectionApplyPreimage(record Record, targetURI string) (statusProjectionFilePreimage, error) {
	targetPath, err := statusProjectionTargetPath(record, targetURI)
	if err != nil {
		return statusProjectionFilePreimage{}, err
	}
	preimage := statusProjectionFilePreimage{TargetPath: targetPath}
	info, err := os.Stat(targetPath)
	if os.IsNotExist(err) {
		return preimage, nil
	}
	if err != nil {
		return statusProjectionFilePreimage{}, fmt.Errorf("stat status projection preimage: %w", err)
	}
	if info.IsDir() {
		return statusProjectionFilePreimage{}, fmt.Errorf("status projection target is a directory")
	}
	if info.Size() > maxStatusProjectionPreimageBytes {
		return statusProjectionFilePreimage{}, fmt.Errorf("status projection preimage exceeds %d bytes", maxStatusProjectionPreimageBytes)
	}
	content, err := os.ReadFile(targetPath)
	if err != nil {
		return statusProjectionFilePreimage{}, fmt.Errorf("read status projection preimage: %w", err)
	}
	preimage.Exists = true
	preimage.Content = content
	preimage.SHA256 = sha256Hex(content)
	preimage.SizeBytes = int64(len(content))
	return preimage, nil
}

type statusProjectionPostWriteVerification struct {
	Verified                  bool
	SHA256                    string
	Size                      int64
	RootContained             bool
	StableProjectionValidated bool
}

func verifyStatusProjectionApplyWrittenFile(writeResult StatusProjectionWriteResult, expectedTarget string, expectedSourceHash string) (statusProjectionPostWriteVerification, error) {
	verification := statusProjectionPostWriteVerification{}
	if strings.TrimSpace(writeResult.Target) == "" {
		return verification, fmt.Errorf("status projection writer returned empty target")
	}
	actualTarget, err := filepath.Abs(writeResult.Target)
	if err != nil {
		return verification, fmt.Errorf("resolve written status projection target: %w", err)
	}
	expectedTarget, err = filepath.Abs(expectedTarget)
	if err != nil {
		return verification, fmt.Errorf("resolve expected status projection target: %w", err)
	}
	if actualTarget != expectedTarget {
		return verification, fmt.Errorf("status projection post-write target mismatch: actual %s != expected %s", actualTarget, expectedTarget)
	}
	if strings.TrimSpace(writeResult.Hash) == "" {
		return verification, fmt.Errorf("status projection writer returned empty hash")
	}
	content, err := os.ReadFile(actualTarget)
	if err != nil {
		return verification, fmt.Errorf("read written status projection for post-write verification: %w", err)
	}
	verification.SHA256 = sha256Hex(content)
	verification.Size = int64(len(content))
	if verification.SHA256 != writeResult.Hash {
		return verification, fmt.Errorf("status projection post-write hash mismatch: actual %s != reported %s", verification.SHA256, writeResult.Hash)
	}
	if verification.Size != writeResult.Size {
		return verification, fmt.Errorf("status projection post-write size mismatch: actual %d != reported %d", verification.Size, writeResult.Size)
	}
	verification.RootContained = true
	if err := verifyStableStatusProjectionJSON(content, expectedSourceHash); err != nil {
		return verification, err
	}
	verification.StableProjectionValidated = true
	verification.Verified = true
	return verification, nil
}

func verifyStableStatusProjectionJSON(content []byte, expectedSourceHash string) error {
	var document map[string]any
	if err := json.Unmarshal(content, &document); err != nil {
		return fmt.Errorf("stable projection JSON must parse: %w", err)
	}
	if err := statusProjectionVerifyExactKeys("status projection", document, []string{
		"schema_version",
		"project_id",
		"project_name",
		"area_flow_url",
		"cutover_phase",
		"active_versions",
		"last_synced_at",
		"source_snapshot_hash",
		"compatibility",
	}); err != nil {
		return err
	}
	if got := strings.TrimSpace(stringValue(document["source_snapshot_hash"])); got == "" || got != strings.TrimSpace(expectedSourceHash) {
		return fmt.Errorf("source_snapshot_hash mismatch: actual %s != expected %s", got, strings.TrimSpace(expectedSourceHash))
	}
	versions, ok := document["active_versions"].([]any)
	if !ok {
		return fmt.Errorf("active_versions must be an array")
	}
	for index, rawVersion := range versions {
		version, ok := rawVersion.(map[string]any)
		if !ok {
			return fmt.Errorf("active_versions[%d] must be an object", index)
		}
		if err := statusProjectionVerifyExactKeys(fmt.Sprintf("active_versions[%d]", index), version, []string{
			"display_label",
			"version_kind",
			"lifecycle_status",
			"rough_progress",
		}); err != nil {
			return err
		}
		progress, ok := version["rough_progress"].(map[string]any)
		if !ok {
			return fmt.Errorf("active_versions[%d].rough_progress must be an object", index)
		}
		if err := statusProjectionVerifyExactKeys(fmt.Sprintf("active_versions[%d].rough_progress", index), progress, []string{
			"percent",
			"label",
			"blocked",
		}); err != nil {
			return err
		}
	}
	compatibility, ok := document["compatibility"].(map[string]any)
	if !ok {
		return fmt.Errorf("compatibility must be an object")
	}
	return statusProjectionVerifyExactKeys("compatibility", compatibility, []string{
		"shim_lifecycle_state",
		"offline_source",
		"blocked_commands",
	})
}

type statusProjectionProtectedPathFingerprint struct {
	Hash       string
	EntryCount int
}

func captureStatusProjectionProtectedPathFingerprint(record Record, targetURI string) (statusProjectionProtectedPathFingerprint, error) {
	root, err := filepath.Abs(record.RootPath)
	if err != nil {
		return statusProjectionProtectedPathFingerprint{}, fmt.Errorf("resolve project root for protected path fingerprint: %w", err)
	}
	targetPath, err := statusProjectionTargetPath(record, targetURI)
	if err != nil {
		return statusProjectionProtectedPathFingerprint{}, err
	}
	targetPath, err = filepath.Abs(targetPath)
	if err != nil {
		return statusProjectionProtectedPathFingerprint{}, fmt.Errorf("resolve target path for protected path fingerprint: %w", err)
	}
	entries := []string{}
	for _, protectedPath := range statusProjectionProtectedPathFingerprintPaths() {
		absolute := filepath.Join(root, filepath.FromSlash(protectedPath))
		if samePath(absolute, targetPath) {
			continue
		}
		pathEntries, err := fingerprintStatusProjectionProtectedPath(root, absolute)
		if err != nil {
			return statusProjectionProtectedPathFingerprint{}, err
		}
		entries = append(entries, pathEntries...)
	}
	payload := strings.Join(entries, "\n")
	sum := sha256.Sum256([]byte(payload))
	return statusProjectionProtectedPathFingerprint{
		Hash:       hex.EncodeToString(sum[:]),
		EntryCount: len(entries),
	}, nil
}

func statusProjectionProtectedPathFingerprintPaths() []string {
	return []string{
		"workflow/README.md",
		".areaflow/status.json",
		"scripts/task_loop/console.py",
		"scripts/dev_tools/cli.py",
		"scripts/task_loop/runner.py",
		"scripts/areaflow_shim.py",
		"workflow/versions",
		"workflow/versions/v1-mvp/execution/_shared/progress.json",
	}
}

func fingerprintStatusProjectionProtectedPath(root string, path string) ([]string, error) {
	info, err := os.Lstat(path)
	if os.IsNotExist(err) {
		rel, relErr := protectedPathFingerprintRel(root, path)
		if relErr != nil {
			return nil, relErr
		}
		return []string{rel + "\tmissing"}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("stat protected path %s: %w", path, err)
	}
	if !info.IsDir() {
		entry, err := fingerprintStatusProjectionProtectedPathEntry(root, path, info)
		if err != nil {
			return nil, err
		}
		return []string{entry}, nil
	}
	entries := []string{}
	err = filepath.WalkDir(path, func(current string, dirEntry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		info, err := os.Lstat(current)
		if err != nil {
			return err
		}
		entry, err := fingerprintStatusProjectionProtectedPathEntry(root, current, info)
		if err != nil {
			return err
		}
		entries = append(entries, entry)
		_ = dirEntry
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walk protected path %s: %w", path, err)
	}
	return entries, nil
}

func fingerprintStatusProjectionProtectedPathEntry(root string, path string, info os.FileInfo) (string, error) {
	rel, err := protectedPathFingerprintRel(root, path)
	if err != nil {
		return "", err
	}
	mode := info.Mode()
	if mode.IsRegular() {
		content, err := os.ReadFile(path)
		if err != nil {
			return "", fmt.Errorf("read protected path %s: %w", path, err)
		}
		return fmt.Sprintf("%s\tfile\t%d\t%s", rel, len(content), sha256Hex(content)), nil
	}
	if mode.IsDir() {
		return rel + "\tdir", nil
	}
	if mode&os.ModeSymlink != 0 {
		target, err := os.Readlink(path)
		if err != nil {
			return "", fmt.Errorf("read protected path symlink %s: %w", path, err)
		}
		return fmt.Sprintf("%s\tsymlink\t%s", rel, target), nil
	}
	return fmt.Sprintf("%s\tother\t%s\t%d", rel, mode.String(), info.Size()), nil
}

func protectedPathFingerprintRel(root string, path string) (string, error) {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return "", fmt.Errorf("compare protected path with project root: %w", err)
	}
	if rel == "." {
		return ".", nil
	}
	return filepath.ToSlash(rel), nil
}

func samePath(left string, right string) bool {
	leftAbs, leftErr := filepath.Abs(left)
	rightAbs, rightErr := filepath.Abs(right)
	if leftErr != nil || rightErr != nil {
		return left == right
	}
	return leftAbs == rightAbs
}

func statusProjectionVerifyExactKeys(scope string, document map[string]any, allowed []string) error {
	allowedKeys := map[string]bool{}
	for _, key := range allowed {
		allowedKeys[key] = true
		if _, ok := document[key]; !ok {
			return fmt.Errorf("%s missing required key %s", scope, key)
		}
	}
	for key := range document {
		if !allowedKeys[key] {
			return fmt.Errorf("%s has unexpected key %s", scope, key)
		}
	}
	return nil
}

func stringValue(value any) string {
	text, ok := value.(string)
	if !ok {
		return ""
	}
	return text
}

func rollbackStatusProjectionApplyFile(preimage statusProjectionFilePreimage, writeHash string) error {
	if strings.TrimSpace(writeHash) == "" {
		return fmt.Errorf("write hash missing")
	}
	current, err := os.ReadFile(preimage.TargetPath)
	if err != nil {
		return fmt.Errorf("read written status projection: %w", err)
	}
	if sha256Hex(current) != writeHash {
		return fmt.Errorf("written status projection hash changed before rollback")
	}
	if preimage.Exists {
		if err := restoreStatusProjectionPreimageAtomically(preimage.TargetPath, preimage.Content); err != nil {
			return fmt.Errorf("restore status projection preimage: %w", err)
		}
		return nil
	}
	if err := os.Remove(preimage.TargetPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove created status projection: %w", err)
	}
	return nil
}

func restoreStatusProjectionPreimageAtomically(targetPath string, content []byte) error {
	dir := filepath.Dir(targetPath)
	temp, err := os.CreateTemp(dir, "."+filepath.Base(targetPath)+".rollback-*")
	if err != nil {
		return fmt.Errorf("create rollback temp file: %w", err)
	}
	tempName := temp.Name()
	keepTemp := true
	defer func() {
		if keepTemp {
			_ = os.Remove(tempName)
		}
	}()
	if _, err := temp.Write(content); err != nil {
		_ = temp.Close()
		return fmt.Errorf("write rollback temp file: %w", err)
	}
	if err := temp.Chmod(0o644); err != nil {
		_ = temp.Close()
		return fmt.Errorf("chmod rollback temp file: %w", err)
	}
	if err := temp.Sync(); err != nil {
		_ = temp.Close()
		return fmt.Errorf("sync rollback temp file: %w", err)
	}
	if err := temp.Close(); err != nil {
		return fmt.Errorf("close rollback temp file: %w", err)
	}
	if err := os.Rename(tempName, targetPath); err != nil {
		return fmt.Errorf("replace rollback target: %w", err)
	}
	keepTemp = false
	syncStatusProjectionDirBestEffort(dir)
	return nil
}

func syncStatusProjectionDirBestEffort(dir string) {
	handle, err := os.Open(dir)
	if err != nil {
		return
	}
	defer handle.Close()
	_ = handle.Sync()
}

func insertStatusProjectionSnapshot(ctx context.Context, tx pgx.Tx, record Record, snapshot Snapshot, options ApplyStatusProjectionOptions, result ApplyStatusProjectionResult, summaryJSON []byte) (int64, error) {
	var snapshotID int64
	if err := tx.QueryRow(ctx, `
INSERT INTO project_status_snapshots (project_id, snapshot_kind, summary, source_hash, export_path)
VALUES ($1, 'mirror_export', $2::jsonb, $3, $4)
RETURNING id`,
		record.ID,
		string(summaryJSON),
		snapshot.SourceHash,
		options.TargetURI,
	).Scan(&snapshotID); err != nil {
		return 0, fmt.Errorf("insert status projection snapshot: %w", err)
	}
	_ = result
	return snapshotID, nil
}

func insertAppliedStatusProjection(ctx context.Context, tx pgx.Tx, record Record, snapshot Snapshot, options ApplyStatusProjectionOptions, result ApplyStatusProjectionResult, summaryJSON []byte) (int64, error) {
	metadata, err := json.Marshal(map[string]any{
		"command_type":                  statusProjectionApplyCommandType,
		"actor":                         options.Actor,
		"reason":                        options.Reason,
		"decision":                      result.Decision,
		"blockers":                      result.Blockers,
		"written_target":                result.WrittenTarget,
		"write_hash":                    result.WriteHash,
		"write_size":                    result.WriteSize,
		"preimage_captured":             result.PreimageCaptured,
		"preimage_exists":               result.PreimageExists,
		"preimage_sha256":               result.PreimageSHA256,
		"preimage_size":                 result.PreimageSize,
		"post_write_verified":           result.PostWriteVerified,
		"post_write_sha256":             result.PostWriteSHA256,
		"post_write_size":               result.PostWriteSize,
		"protected_paths_verified":      result.ProtectedPathsVerified,
		"protected_path_before_hash":    result.ProtectedPathBeforeHash,
		"protected_path_after_hash":     result.ProtectedPathAfterHash,
		"expected_protected_path_hash":  result.ExpectedProtectedPathHash,
		"root_contained":                result.RootContained,
		"stable_projection_validated":   result.StableProjectionValid,
		"atomic_replace_used":           result.AtomicReplaceUsed,
		"rollback_compensation_enabled": result.RollbackCompensation,
		"apply_gate_status":             result.ApplyGateStatus,
		"apply_gate_decision":           result.ApplyGateDecision,
		"apply_gate_approval_status":    result.ApplyGateApprovalStatus,
		"apply_command_eligible":        result.ApplyCommandEligible,
		"project_write_attempted":       result.ProjectWriteAttempted,
		"execution_write_attempted":     result.ExecutionWriteAttempted,
		"engine_call_attempted":         result.EngineCallAttempted,
	})
	if err != nil {
		return 0, fmt.Errorf("marshal status projection metadata: %w", err)
	}
	writeState := "written"
	var writtenAt any = time.Now().UTC()
	if result.Decision != "allowed" {
		writeState = "blocked"
		writtenAt = nil
	}
	var projectionID int64
	if err := tx.QueryRow(ctx, `
INSERT INTO status_projections (
    project_id,
    target_kind,
    target_uri,
    summary_state,
    payload_json,
    source_hash,
    write_state,
    written_at,
    metadata
)
VALUES ($1, $2, $3, $4, $5::jsonb, $6, $7, $8, $9::jsonb)
RETURNING id`,
		record.ID,
		result.TargetKind,
		options.TargetURI,
		statusProjectionSummaryState(snapshot.Summary),
		string(summaryJSON),
		snapshot.SourceHash,
		writeState,
		writtenAt,
		string(metadata),
	).Scan(&projectionID); err != nil {
		return 0, fmt.Errorf("insert status projection apply record: %w", err)
	}
	return projectionID, nil
}

func insertStatusProjectionApplyEvent(ctx context.Context, tx pgx.Tx, result ApplyStatusProjectionResult, options ApplyStatusProjectionOptions) (int64, error) {
	metadata, err := json.Marshal(map[string]any{
		"target_kind":                   result.TargetKind,
		"target_uri":                    result.TargetURI,
		"status_projection_id":          result.StatusProjectionID,
		"snapshot_id":                   result.SnapshotID,
		"source_hash":                   result.SourceHash,
		"decision":                      result.Decision,
		"blockers":                      result.Blockers,
		"actor":                         options.Actor,
		"reason":                        options.Reason,
		"idempotency_key":               options.IdempotencyKey,
		"preimage_captured":             result.PreimageCaptured,
		"preimage_exists":               result.PreimageExists,
		"preimage_sha256":               result.PreimageSHA256,
		"preimage_size":                 result.PreimageSize,
		"post_write_verified":           result.PostWriteVerified,
		"post_write_sha256":             result.PostWriteSHA256,
		"post_write_size":               result.PostWriteSize,
		"protected_paths_verified":      result.ProtectedPathsVerified,
		"protected_path_before_hash":    result.ProtectedPathBeforeHash,
		"protected_path_after_hash":     result.ProtectedPathAfterHash,
		"expected_protected_path_hash":  result.ExpectedProtectedPathHash,
		"root_contained":                result.RootContained,
		"stable_projection_validated":   result.StableProjectionValid,
		"atomic_replace_used":           result.AtomicReplaceUsed,
		"rollback_compensation_enabled": result.RollbackCompensation,
		"apply_gate_status":             result.ApplyGateStatus,
		"apply_gate_decision":           result.ApplyGateDecision,
		"apply_gate_approval_status":    result.ApplyGateApprovalStatus,
		"apply_command_eligible":        result.ApplyCommandEligible,
		"project_write_attempted":       result.ProjectWriteAttempted,
		"execution_write_attempted":     result.ExecutionWriteAttempted,
		"engine_call_attempted":         result.EngineCallAttempted,
	})
	if err != nil {
		return 0, fmt.Errorf("marshal status projection apply event metadata: %w", err)
	}
	severity := "info"
	eventType := "project.status_projection.apply.completed"
	message := "Status projection apply completed"
	if result.Decision != "allowed" {
		severity = "warn"
		eventType = "project.status_projection.apply.blocked"
		message = "Status projection apply blocked"
	}
	var eventID int64
	if err := tx.QueryRow(ctx, `
INSERT INTO events (project_id, event_type, severity, message, metadata)
VALUES ($1, $2, $3, $4, $5::jsonb)
RETURNING id`,
		result.Project.ID,
		eventType,
		severity,
		message,
		string(metadata),
	).Scan(&eventID); err != nil {
		return 0, fmt.Errorf("insert status projection apply event: %w", err)
	}
	return eventID, nil
}

func insertStatusProjectionApplyAuditEvent(ctx context.Context, tx pgx.Tx, result ApplyStatusProjectionResult, options ApplyStatusProjectionOptions) (int64, error) {
	metadata, err := json.Marshal(map[string]any{
		"target_kind":                   result.TargetKind,
		"target_uri":                    result.TargetURI,
		"status_projection_id":          result.StatusProjectionID,
		"snapshot_id":                   result.SnapshotID,
		"source_hash":                   result.SourceHash,
		"written_target":                result.WrittenTarget,
		"write_hash":                    result.WriteHash,
		"write_size":                    result.WriteSize,
		"preimage_captured":             result.PreimageCaptured,
		"preimage_exists":               result.PreimageExists,
		"preimage_sha256":               result.PreimageSHA256,
		"preimage_size":                 result.PreimageSize,
		"post_write_verified":           result.PostWriteVerified,
		"post_write_sha256":             result.PostWriteSHA256,
		"post_write_size":               result.PostWriteSize,
		"protected_paths_verified":      result.ProtectedPathsVerified,
		"protected_path_before_hash":    result.ProtectedPathBeforeHash,
		"protected_path_after_hash":     result.ProtectedPathAfterHash,
		"expected_protected_path_hash":  result.ExpectedProtectedPathHash,
		"root_contained":                result.RootContained,
		"stable_projection_validated":   result.StableProjectionValid,
		"atomic_replace_used":           result.AtomicReplaceUsed,
		"rollback_compensation_enabled": result.RollbackCompensation,
		"blockers":                      result.Blockers,
		"actor":                         options.Actor,
		"idempotency_key":               options.IdempotencyKey,
		"apply_gate_status":             result.ApplyGateStatus,
		"apply_gate_decision":           result.ApplyGateDecision,
		"apply_gate_approval_status":    result.ApplyGateApprovalStatus,
		"apply_command_eligible":        result.ApplyCommandEligible,
		"project_write_attempted":       result.ProjectWriteAttempted,
		"execution_write_attempted":     result.ExecutionWriteAttempted,
		"engine_call_attempted":         result.EngineCallAttempted,
	})
	if err != nil {
		return 0, fmt.Errorf("marshal status projection apply audit metadata: %w", err)
	}
	var auditEventID int64
	if err := tx.QueryRow(ctx, `
INSERT INTO audit_events (project_id, action, capability, resource_type, resource, decision, reason, metadata)
VALUES ($1, 'project.status_projection.apply', 'write_status', 'path', $2, $3, $4, $5::jsonb)
RETURNING id`,
		result.Project.ID,
		result.TargetURI,
		result.Decision,
		options.Reason,
		string(metadata),
	).Scan(&auditEventID); err != nil {
		return 0, fmt.Errorf("insert status projection apply audit event: %w", err)
	}
	return auditEventID, nil
}

func statusProjectionApplyCommandResponse(result ApplyStatusProjectionResult) map[string]any {
	return map[string]any{
		"project_id":                    result.Project.ID,
		"project_key":                   result.Project.Key,
		"status":                        result.Status,
		"decision":                      result.Decision,
		"message":                       result.Message,
		"blockers":                      result.Blockers,
		"event_id":                      result.EventID,
		"audit_event_id":                result.AuditEventID,
		"snapshot_id":                   result.SnapshotID,
		"status_projection_id":          result.StatusProjectionID,
		"target_kind":                   result.TargetKind,
		"target_uri":                    result.TargetURI,
		"written_target":                result.WrittenTarget,
		"write_hash":                    result.WriteHash,
		"write_size":                    result.WriteSize,
		"preimage_captured":             result.PreimageCaptured,
		"preimage_exists":               result.PreimageExists,
		"preimage_sha256":               result.PreimageSHA256,
		"preimage_size":                 result.PreimageSize,
		"post_write_verified":           result.PostWriteVerified,
		"post_write_sha256":             result.PostWriteSHA256,
		"post_write_size":               result.PostWriteSize,
		"protected_paths_verified":      result.ProtectedPathsVerified,
		"protected_path_before_hash":    result.ProtectedPathBeforeHash,
		"protected_path_after_hash":     result.ProtectedPathAfterHash,
		"expected_protected_path_hash":  result.ExpectedProtectedPathHash,
		"root_contained":                result.RootContained,
		"stable_projection_validated":   result.StableProjectionValid,
		"atomic_replace_used":           result.AtomicReplaceUsed,
		"rollback_compensation_enabled": result.RollbackCompensation,
		"source_hash":                   result.SourceHash,
		"summary_state":                 result.SummaryState,
		"apply_gate_status":             result.ApplyGateStatus,
		"apply_gate_decision":           result.ApplyGateDecision,
		"apply_gate_approval_status":    result.ApplyGateApprovalStatus,
		"apply_command_eligible":        result.ApplyCommandEligible,
		"idempotency_key":               result.IdempotencyKey,
		"generated_at":                  result.GeneratedAt.Format(time.RFC3339),
		"project_write_attempted":       result.ProjectWriteAttempted,
		"execution_write_attempted":     result.ExecutionWriteAttempted,
		"engine_call_attempted":         result.EngineCallAttempted,
	}
}

func loadStatusProjectionApplyByCommandResponse(ctx context.Context, tx pgx.Tx, record Record, idempotencyKey string) (ApplyStatusProjectionResult, error) {
	response, err := loadCommandResponse(ctx, tx, record.ID, statusProjectionApplyCommandType, idempotencyKey)
	if err != nil {
		return ApplyStatusProjectionResult{}, err
	}
	generatedAt := time.Now().UTC()
	if raw := metadataString(response, "generated_at"); raw != "" {
		if parsed, err := time.Parse(time.RFC3339, raw); err == nil {
			generatedAt = parsed
		}
	}
	return ApplyStatusProjectionResult{
		Project:                   record,
		Status:                    metadataString(response, "status"),
		Decision:                  metadataString(response, "decision"),
		Message:                   metadataString(response, "message"),
		Blockers:                  metadataStringSlice(response, "blockers"),
		EventID:                   metadataInt64(response, "event_id"),
		AuditEventID:              metadataInt64(response, "audit_event_id"),
		SnapshotID:                metadataInt64(response, "snapshot_id"),
		StatusProjectionID:        metadataInt64(response, "status_projection_id"),
		TargetKind:                metadataString(response, "target_kind"),
		TargetURI:                 metadataString(response, "target_uri"),
		WrittenTarget:             metadataString(response, "written_target"),
		WriteHash:                 metadataString(response, "write_hash"),
		WriteSize:                 metadataInt64(response, "write_size"),
		PreimageCaptured:          metadataBool(response, "preimage_captured"),
		PreimageExists:            metadataBool(response, "preimage_exists"),
		PreimageSHA256:            metadataString(response, "preimage_sha256"),
		PreimageSize:              metadataInt64(response, "preimage_size"),
		PostWriteVerified:         metadataBool(response, "post_write_verified"),
		PostWriteSHA256:           metadataString(response, "post_write_sha256"),
		PostWriteSize:             metadataInt64(response, "post_write_size"),
		ProtectedPathsVerified:    metadataBool(response, "protected_paths_verified"),
		ProtectedPathBeforeHash:   metadataString(response, "protected_path_before_hash"),
		ProtectedPathAfterHash:    metadataString(response, "protected_path_after_hash"),
		ExpectedProtectedPathHash: metadataString(response, "expected_protected_path_hash"),
		RootContained:             metadataBool(response, "root_contained"),
		StableProjectionValid:     metadataBool(response, "stable_projection_validated"),
		AtomicReplaceUsed:         metadataBool(response, "atomic_replace_used"),
		RollbackCompensation:      metadataBool(response, "rollback_compensation_enabled"),
		SourceHash:                metadataString(response, "source_hash"),
		SummaryState:              metadataString(response, "summary_state"),
		ApplyGateStatus:           metadataString(response, "apply_gate_status"),
		ApplyGateDecision:         metadataString(response, "apply_gate_decision"),
		ApplyGateApprovalStatus:   metadataString(response, "apply_gate_approval_status"),
		ApplyCommandEligible:      metadataBool(response, "apply_command_eligible"),
		IdempotencyKey:            idempotencyKey,
		GeneratedAt:               generatedAt,
		ProjectWriteAttempted:     metadataBool(response, "project_write_attempted"),
		ExecutionWriteAttempted:   metadataBool(response, "execution_write_attempted"),
		EngineCallAttempted:       metadataBool(response, "engine_call_attempted"),
	}, nil
}
