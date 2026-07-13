package project

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const (
	statusProjectionAuthorizationMode = "status_projection_apply_authorization_preview_v1"
	statusProjectionSchemaURI         = "schemas/status-projection.schema.json"
	statusProjectionClaimScope        = "package_a_status_projection_preflight_only"
	maxStatusProjectionPreimageBytes  = 1024 * 1024
)

type StatusProjectionAuthorizationPreviewOptions struct {
	TargetURI   string
	GeneratedAt time.Time
}

type StatusProjectionAuthorizationPreview struct {
	Project                                       Record
	Status                                        string
	Mode                                          string
	ClaimScope                                    string
	NotReal100                                    bool
	Decision                                      string
	Message                                       string
	TargetKind                                    string
	TargetURI                                     string
	TargetPath                                    string
	SchemaURI                                     string
	ValidatorPreflight                            string
	ProtectedPathFingerprintSHA256                string
	SourceHash                                    string
	SummaryState                                  string
	RequiredAuthorizationPhrase                   string
	Permission                                    StatusProjectionAuthorizationPermission
	Preimage                                      StatusProjectionPreimage
	WriteSet                                      []StatusProjectionWriteSetEntry
	RequiredPreflight                             []string
	RequiredPacketFields                          []string
	RequiredCapabilities                          []string
	ProtectedPaths                                []string
	RollbackPlan                                  []string
	BlockedBy                                     []string
	Warnings                                      []string
	ForbiddenActions                              []string
	SafetyFacts                                   map[string]bool
	ApplyOpen                                     bool
	ApprovalRequired                              bool
	ApprovalStatus                                string
	WouldCreateCommandRequestAfterApproval        bool
	WouldCreateProjectStatusSnapshotAfterApproval bool
	WouldCreateStatusProjectionAfterApproval      bool
	WouldCreateEventAfterApproval                 bool
	WouldCreateAuditEventAfterApproval            bool
	WouldWriteProjectFileAfterApproval            bool
	WouldWriteExecutionAfterApproval              bool
	WouldRunEngineAfterApproval                   bool
	ProjectWriteAttempted                         bool
	ExecutionWriteAttempted                       bool
	EngineCallAttempted                           bool
	GeneratedAt                                   time.Time
}

type StatusProjectionAuthorizationPermission struct {
	Capability        string
	ResourceType      string
	TargetURI         string
	CapabilityAllowed bool
	PathAllowed       bool
	Allowed           bool
	Reason            string
}

type StatusProjectionPreimage struct {
	TargetPath               string
	Exists                   bool
	Readable                 bool
	SizeBytes                int64
	SHA256                   string
	SchemaStatus             string
	LegacyShape              bool
	MissingRequiredFields    []string
	UnexpectedTopLevelFields []string
	CompatibilityMissing     []string
	CompatibilityUnexpected  []string
	SourceSnapshotHash       string
	Message                  string
}

type StatusProjectionWriteSetEntry struct {
	TargetURI                string
	TargetPath               string
	Operation                string
	Capability               string
	ExpectedBeforeExists     bool
	ExpectedBeforeSHA256     string
	ExpectedBeforeSizeBytes  int64
	RequiresPreimageMatch    bool
	RequiresSchemaValidation bool
	RollbackAction           string
	ProtectedPath            bool
}

func (s Store) StatusProjectionAuthorizationPreview(ctx context.Context, record Record, options StatusProjectionAuthorizationPreviewOptions) (StatusProjectionAuthorizationPreview, error) {
	options = normalizeStatusProjectionAuthorizationPreviewOptions(options)
	snapshot, err := s.LatestImportSnapshot(ctx, record.ID)
	if err != nil {
		return StatusProjectionAuthorizationPreview{}, err
	}
	summaryJSON, err := json.Marshal(snapshot.Summary)
	if err != nil {
		return StatusProjectionAuthorizationPreview{}, fmt.Errorf("marshal status projection authorization summary: %w", err)
	}
	_ = summaryJSON

	targetPath, targetPathErr := statusProjectionTargetPath(record, options.TargetURI)
	preimage := inspectStatusProjectionPreimage(targetPath, targetPathErr)
	permission := s.statusProjectionAuthorizationPermission(ctx, record, options.TargetURI)
	preview := BuildStatusProjectionAuthorizationPreview(record, snapshot, options, preimage, permission)
	fingerprint, err := captureStatusProjectionProtectedPathFingerprint(record, options.TargetURI)
	if err != nil {
		preview.Status = "blocked"
		preview.Decision = "blocked"
		preview.Message = "status projection authorization preview is blocked by protected path fingerprint capture"
		preview.BlockedBy = uniqueStrings(append(preview.BlockedBy, "protected_path_fingerprint_unavailable", err.Error()))
		return preview, nil
	}
	preview.ProtectedPathFingerprintSHA256 = fingerprint.Hash
	return preview, nil
}

func normalizeStatusProjectionAuthorizationPreviewOptions(options StatusProjectionAuthorizationPreviewOptions) StatusProjectionAuthorizationPreviewOptions {
	options.TargetURI = strings.TrimSpace(options.TargetURI)
	if options.TargetURI == "" {
		options.TargetURI = ".areaflow/status.json"
	}
	if options.GeneratedAt.IsZero() {
		options.GeneratedAt = time.Now().UTC()
	}
	return options
}

func BuildStatusProjectionAuthorizationPreview(record Record, snapshot Snapshot, options StatusProjectionAuthorizationPreviewOptions, preimage StatusProjectionPreimage, permission StatusProjectionAuthorizationPermission) StatusProjectionAuthorizationPreview {
	options = normalizeStatusProjectionAuthorizationPreviewOptions(options)
	targetKind := statusProjectionTargetKind(options.TargetURI)
	validator := statusProjectionValidatorPreflight(preimage.TargetPath)
	preview := StatusProjectionAuthorizationPreview{
		Project:                        record,
		Status:                         "needs_approval",
		Mode:                           statusProjectionAuthorizationMode,
		ClaimScope:                     statusProjectionClaimScope,
		NotReal100:                     true,
		Decision:                       "needs_explicit_approval",
		Message:                        "status projection apply requires an explicit authorization packet before writing the managed project",
		TargetKind:                     targetKind,
		TargetURI:                      options.TargetURI,
		TargetPath:                     preimage.TargetPath,
		SchemaURI:                      statusProjectionSchemaURI,
		ValidatorPreflight:             validator,
		ProtectedPathFingerprintSHA256: "",
		SourceHash:                     snapshot.SourceHash,
		SummaryState:                   statusProjectionSummaryState(snapshot.Summary),
		Permission:                     permission,
		Preimage:                       preimage,
		WriteSet: []StatusProjectionWriteSetEntry{
			{
				TargetURI:                options.TargetURI,
				TargetPath:               preimage.TargetPath,
				Operation:                "replace_or_create",
				Capability:               "write_status",
				ExpectedBeforeExists:     preimage.Exists,
				ExpectedBeforeSHA256:     preimage.SHA256,
				ExpectedBeforeSizeBytes:  preimage.SizeBytes,
				RequiresPreimageMatch:    true,
				RequiresSchemaValidation: true,
				RollbackAction:           statusProjectionRollbackAction(preimage),
				ProtectedPath:            true,
			},
		},
		RequiredPreflight: []string{
			fmt.Sprintf("areaflow project status-projections %s --json", record.Key),
			validator,
			statusProjectionProtectedPathCheck(record),
		},
		RequiredPacketFields: []string{
			"target_uri",
			"schema_uri",
			"source_snapshot_hash",
			"expected_before_exists",
			"expected_before_sha256",
			"expected_before_size_bytes",
			"write_set",
			"validator_preflight",
			"protected_path_check",
			"protected_path_fingerprint_sha256",
			"rollback_plan",
			"explicit_approval",
			"approval_actor",
			"approval_reason",
		},
		RequiredCapabilities: []string{"write_status"},
		ProtectedPaths: []string{
			".areaflow/status.json",
			"workflow/README.md",
			"scripts/task_loop/console.py",
			"scripts/dev_tools/cli.py",
			"scripts/task_loop/runner.py",
			"scripts/areaflow_shim.py",
			"workflow/versions",
			"workflow/versions/v1-mvp/execution/_shared/progress.json",
		},
		RollbackPlan: []string{
			statusProjectionRollbackAction(preimage),
			"rerun the validator preflight after rollback",
			"rerun the protected path check and compare the preimage fingerprint",
		},
		BlockedBy: []string{"explicit_status_projection_apply_approval_missing"},
		ForbiddenActions: []string{
			"write_workflow_readme",
			"write_workflow_versions",
			"write_execution",
			"rewrite_progress_json",
			"start_task_loop",
			"run_engine",
			"resolve_secret",
			"network_call",
			"git_checkpoint",
		},
		SafetyFacts:      statusProjectionAuthorizationSafetyFacts(),
		ApplyOpen:        false,
		ApprovalRequired: true,
		ApprovalStatus:   "missing",
		GeneratedAt:      options.GeneratedAt,
	}
	if preimage.SchemaStatus == "legacy" || preimage.SchemaStatus == "mismatch" {
		preview.BlockedBy = append(preview.BlockedBy, "current_target_schema_mismatch_requires_preimage_review")
		preview.Warnings = append(preview.Warnings, "current target does not match stable status projection schema")
	}
	if preimage.SchemaStatus == "invalid_target" || preimage.SchemaStatus == "read_error" || preimage.SchemaStatus == "too_large" {
		preview.Status = "blocked"
		preview.Decision = "blocked"
		preview.Message = "status projection authorization preview is blocked by target preimage inspection"
		preview.BlockedBy = append(preview.BlockedBy, preimage.SchemaStatus)
	}
	if targetKind != "project_status_json" {
		preview.Status = "blocked"
		preview.Decision = "blocked"
		preview.Message = "status projection authorization preview is blocked by unsupported target"
		preview.BlockedBy = append(preview.BlockedBy, "unsupported_status_projection_target")
	}
	if !permission.Allowed {
		preview.Status = "blocked"
		preview.Decision = "blocked"
		preview.Message = "status projection authorization preview is blocked by project permission policy"
		if permission.Reason != "" {
			preview.BlockedBy = append(preview.BlockedBy, permission.Reason)
		} else {
			preview.BlockedBy = append(preview.BlockedBy, "write_status_permission_not_allowed")
		}
	}
	if preview.Status != "blocked" {
		preview.WouldCreateCommandRequestAfterApproval = true
		preview.WouldCreateProjectStatusSnapshotAfterApproval = true
		preview.WouldCreateStatusProjectionAfterApproval = true
		preview.WouldCreateEventAfterApproval = true
		preview.WouldCreateAuditEventAfterApproval = true
		preview.WouldWriteProjectFileAfterApproval = true
	}
	preview.RequiredAuthorizationPhrase = statusProjectionRequiredAuthorizationPhrase(preview)
	preview.BlockedBy = uniqueStrings(preview.BlockedBy)
	preview.Warnings = uniqueStrings(preview.Warnings)
	return preview
}

func (s Store) statusProjectionAuthorizationPermission(ctx context.Context, record Record, targetURI string) StatusProjectionAuthorizationPermission {
	permission := StatusProjectionAuthorizationPermission{
		Capability:   "write_status",
		ResourceType: "path",
		TargetURI:    targetURI,
		Reason:       "not evaluated",
	}
	rows, err := s.pool.Query(ctx, `
SELECT effect, capability, resource_type, pattern
FROM project_permissions
WHERE project_id = $1 AND resource_type IN ('capability', 'path')
ORDER BY id`,
		record.ID,
	)
	if err != nil {
		permission.Reason = fmt.Sprintf("load project permissions: %v", err)
		return permission
	}
	defer rows.Close()

	for rows.Next() {
		var effect, capability, resourceType, pattern string
		if err := rows.Scan(&effect, &capability, &resourceType, &pattern); err != nil {
			permission.Reason = fmt.Sprintf("scan project permission: %v", err)
			return permission
		}
		if effect == "deny" && resourceType == "path" && globMatch(pattern, targetURI) {
			permission.Reason = "path denied by forbidden path"
			return permission
		}
		if resourceType == "capability" && capability == "write_status" && effect == "allow" {
			permission.CapabilityAllowed = true
		}
		if resourceType == "path" && capability == "write_status" && effect == "allow" && globMatch(pattern, targetURI) {
			permission.PathAllowed = true
		}
	}
	if err := rows.Err(); err != nil {
		permission.Reason = fmt.Sprintf("iterate project permissions: %v", err)
		return permission
	}
	if !permission.CapabilityAllowed {
		permission.Reason = "capability not allowed"
		return permission
	}
	if !permission.PathAllowed {
		permission.Reason = "path not allowed"
		return permission
	}
	permission.Allowed = true
	permission.Reason = "allowed"
	return permission
}

func statusProjectionTargetPath(record Record, targetURI string) (string, error) {
	if strings.TrimSpace(record.RootPath) == "" {
		return "", fmt.Errorf("project root is empty")
	}
	relative := normalizeProjectRelativePath(targetURI)
	if relative == "" {
		return "", fmt.Errorf("target URI must stay under project root")
	}
	rootAbs, err := filepath.Abs(record.RootPath)
	if err != nil {
		return "", fmt.Errorf("resolve project root: %w", err)
	}
	target := filepath.Join(rootAbs, filepath.FromSlash(relative))
	rel, err := filepath.Rel(rootAbs, target)
	if err != nil {
		return "", fmt.Errorf("compare target path: %w", err)
	}
	if rel == "." || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || filepath.IsAbs(rel) {
		return "", fmt.Errorf("target URI escapes project root")
	}
	return target, nil
}

func inspectStatusProjectionPreimage(targetPath string, targetPathErr error) StatusProjectionPreimage {
	preimage := StatusProjectionPreimage{
		TargetPath:   targetPath,
		SchemaStatus: "missing",
		Message:      "target does not exist",
		Readable:     false,
		Exists:       false,
	}
	if targetPathErr != nil {
		preimage.SchemaStatus = "invalid_target"
		preimage.Message = targetPathErr.Error()
		return preimage
	}
	info, err := os.Stat(targetPath)
	if os.IsNotExist(err) {
		return preimage
	}
	if err != nil {
		preimage.SchemaStatus = "read_error"
		preimage.Message = fmt.Sprintf("stat target: %v", err)
		return preimage
	}
	if info.IsDir() {
		preimage.Exists = true
		preimage.SchemaStatus = "read_error"
		preimage.Message = "target is a directory"
		return preimage
	}
	preimage.Exists = true
	preimage.SizeBytes = info.Size()
	if info.Size() > maxStatusProjectionPreimageBytes {
		preimage.SchemaStatus = "too_large"
		preimage.Message = fmt.Sprintf("target exceeds %d bytes", maxStatusProjectionPreimageBytes)
		return preimage
	}
	content, err := os.ReadFile(targetPath)
	if err != nil {
		preimage.SchemaStatus = "read_error"
		preimage.Message = fmt.Sprintf("read target: %v", err)
		return preimage
	}
	preimage.Readable = true
	preimage.SizeBytes = int64(len(content))
	preimage.SHA256 = sha256Hex(content)
	var document map[string]any
	if err := json.Unmarshal(content, &document); err != nil {
		preimage.SchemaStatus = "mismatch"
		preimage.Message = "target is not valid JSON"
		return preimage
	}
	annotateStatusProjectionSchemaShape(&preimage, document)
	return preimage
}

func annotateStatusProjectionSchemaShape(preimage *StatusProjectionPreimage, document map[string]any) {
	required := []string{
		"schema_version",
		"project_id",
		"project_name",
		"area_flow_url",
		"cutover_phase",
		"active_versions",
		"last_synced_at",
		"source_snapshot_hash",
		"compatibility",
	}
	allowed := map[string]bool{}
	for _, key := range required {
		allowed[key] = true
		if _, ok := document[key]; !ok {
			preimage.MissingRequiredFields = append(preimage.MissingRequiredFields, key)
		}
	}
	for key := range document {
		if !allowed[key] {
			preimage.UnexpectedTopLevelFields = append(preimage.UnexpectedTopLevelFields, key)
		}
	}
	legacyFields := map[string]bool{
		"version":      true,
		"generated_at": true,
		"project":      true,
		"source":       true,
		"source_hash":  true,
		"summary":      true,
	}
	for _, key := range preimage.UnexpectedTopLevelFields {
		if legacyFields[key] {
			preimage.LegacyShape = true
			break
		}
	}
	compatibility, ok := document["compatibility"].(map[string]any)
	preimage.SourceSnapshotHash = strings.TrimSpace(stringValue(document["source_snapshot_hash"]))
	if ok {
		requiredCompatibility := []string{"shim_lifecycle_state", "offline_source", "blocked_commands"}
		allowedCompatibility := map[string]bool{}
		for _, key := range requiredCompatibility {
			allowedCompatibility[key] = true
			if _, ok := compatibility[key]; !ok {
				preimage.CompatibilityMissing = append(preimage.CompatibilityMissing, key)
			}
		}
		for key := range compatibility {
			if !allowedCompatibility[key] {
				preimage.CompatibilityUnexpected = append(preimage.CompatibilityUnexpected, key)
			}
		}
	} else if _, exists := document["compatibility"]; exists {
		preimage.CompatibilityMissing = append(preimage.CompatibilityMissing, "compatibility_object")
	}
	sort.Strings(preimage.MissingRequiredFields)
	sort.Strings(preimage.UnexpectedTopLevelFields)
	sort.Strings(preimage.CompatibilityMissing)
	sort.Strings(preimage.CompatibilityUnexpected)
	if len(preimage.MissingRequiredFields) == 0 && len(preimage.UnexpectedTopLevelFields) == 0 && len(preimage.CompatibilityMissing) == 0 && len(preimage.CompatibilityUnexpected) == 0 {
		preimage.SchemaStatus = "stable"
		preimage.Message = "target matches stable status projection shape"
		return
	}
	if preimage.LegacyShape {
		preimage.SchemaStatus = "legacy"
		preimage.Message = "target uses legacy status projection shape"
		return
	}
	preimage.SchemaStatus = "mismatch"
	preimage.Message = "target does not match stable status projection shape"
}

func statusProjectionValidatorPreflight(targetPath string) string {
	if targetPath == "" {
		targetPath = "<target-status-json>"
	}
	return fmt.Sprintf("python3 scripts/validate-status-projection-schema.py %s %s", statusProjectionSchemaURI, targetPath)
}

func statusProjectionProtectedPathCheck(record Record) string {
	root := strings.TrimSpace(record.RootPath)
	if root == "" {
		root = "<managed-project-root>"
	}
	return fmt.Sprintf("git -C %s status --short -- workflow/README.md .areaflow/status.json scripts/task_loop/console.py scripts/dev_tools/cli.py scripts/task_loop/runner.py scripts/areaflow_shim.py workflow/versions workflow/versions/v1-mvp/execution/_shared/progress.json", root)
}

func statusProjectionRollbackAction(preimage StatusProjectionPreimage) string {
	if preimage.Exists {
		return "restore the captured preimage bytes for .areaflow/status.json"
	}
	return "delete .areaflow/status.json if apply created it"
}

func statusProjectionAuthorizationSafetyFacts() map[string]bool {
	return map[string]bool{
		"read_only_preview":                  true,
		"authorization_packet_preview":       true,
		"apply_open":                         false,
		"command_request_created":            false,
		"status_projection_written":          false,
		"project_write_attempted":            false,
		"execution_write_attempted":          false,
		"engine_call_attempted":              false,
		"commands_run":                       false,
		"worker_scheduled":                   false,
		"secrets_resolved":                   false,
		"network_used":                       false,
		"workflow_readme_written":            false,
		"workflow_versions_written":          false,
		"legacy_progress_written":            false,
		"legacy_logs_written":                false,
		"legacy_checkpoint_written":          false,
		"areamatrix_protected_paths_touched": false,
	}
}
