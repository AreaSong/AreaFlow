package status

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"time"

	"github.com/areasong/areaflow/internal/project"
)

type ExportDocument struct {
	SchemaVersion      int                  `json:"schema_version"`
	ProjectID          string               `json:"project_id"`
	ProjectName        string               `json:"project_name"`
	AreaFlowURL        string               `json:"area_flow_url"`
	CutoverPhase       string               `json:"cutover_phase"`
	ActiveVersions     []ActiveVersion      `json:"active_versions"`
	LastSyncedAt       string               `json:"last_synced_at"`
	SourceSnapshotHash string               `json:"source_snapshot_hash"`
	Compatibility      CompatibilitySummary `json:"compatibility"`
}

type CompatibilitySummary struct {
	ShimLifecycleState string   `json:"shim_lifecycle_state"`
	OfflineSource      string   `json:"offline_source"`
	BlockedCommands    []string `json:"blocked_commands"`
}

type ActiveVersion struct {
	DisplayLabel    string        `json:"display_label"`
	VersionKind     string        `json:"version_kind"`
	LifecycleStatus string        `json:"lifecycle_status"`
	RoughProgress   RoughProgress `json:"rough_progress"`
}

type RoughProgress struct {
	Percent int    `json:"percent"`
	Label   string `json:"label"`
	Blocked bool   `json:"blocked"`
}

type WriteResult struct {
	Target                    string
	Hash                      string
	Size                      int64
	RootContained             bool
	StableProjectionValidated bool
	AtomicReplaceUsed         bool
}

func Write(record project.Record, snapshot project.Snapshot, relPath string) (string, error) {
	result, err := WriteWithResult(record, snapshot, relPath)
	if err != nil {
		return "", err
	}
	return result.Target, nil
}

func WriteWithResult(record project.Record, snapshot project.Snapshot, relPath string) (WriteResult, error) {
	target, root, err := statusTargetPath(record, relPath)
	if err != nil {
		return WriteResult{}, err
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return WriteResult{}, fmt.Errorf("create status export directory: %w", err)
	}
	if err := ensureStatusTargetParentContained(root, target); err != nil {
		return WriteResult{}, err
	}

	now := time.Now().UTC().Format(time.RFC3339)
	phase := cutoverPhase(snapshot.Summary)
	document := ExportDocument{
		SchemaVersion:      1,
		ProjectID:          record.Key,
		ProjectName:        projectName(record),
		AreaFlowURL:        areaFlowURL(record, snapshot.Summary),
		CutoverPhase:       phase,
		ActiveVersions:     activeVersions(snapshot.Summary),
		LastSyncedAt:       now,
		SourceSnapshotHash: snapshot.SourceHash,
		Compatibility: CompatibilitySummary{
			ShimLifecycleState: shimLifecycleState(phase),
			OfflineSource:      relPath,
			BlockedCommands: []string{
				"./task-loop run",
				"promotion apply",
				"write execution",
			},
		},
	}
	content, err := marshalStableExportDocument(document)
	if err != nil {
		return WriteResult{}, fmt.Errorf("marshal status export: %w", err)
	}
	if existing, err := os.ReadFile(target); err == nil {
		var existingDocument ExportDocument
		if err := json.Unmarshal(existing, &existingDocument); err == nil && sameDocumentExceptGenerated(existingDocument, document) {
			return WriteResult{
				Target:                    target,
				Hash:                      hashContent(existing),
				Size:                      int64(len(existing)),
				RootContained:             true,
				StableProjectionValidated: true,
				AtomicReplaceUsed:         false,
			}, nil
		}
	}
	if err := writeFileAtomically(target, content, 0o644); err != nil {
		return WriteResult{}, fmt.Errorf("write status export: %w", err)
	}
	return WriteResult{
		Target:                    target,
		Hash:                      hashContent(content),
		Size:                      int64(len(content)),
		RootContained:             true,
		StableProjectionValidated: true,
		AtomicReplaceUsed:         true,
	}, nil
}

func marshalStableExportDocument(document ExportDocument) ([]byte, error) {
	if err := validateExportDocument(document); err != nil {
		return nil, err
	}
	content, err := json.MarshalIndent(document, "", "  ")
	if err != nil {
		return nil, err
	}
	content = append(content, '\n')
	if err := validateStableExportJSON(content); err != nil {
		return nil, err
	}
	return content, nil
}

func validateExportDocument(document ExportDocument) error {
	if document.SchemaVersion != 1 {
		return fmt.Errorf("schema_version must be 1")
	}
	required := map[string]string{
		"project_id":           document.ProjectID,
		"project_name":         document.ProjectName,
		"area_flow_url":        document.AreaFlowURL,
		"cutover_phase":        document.CutoverPhase,
		"last_synced_at":       document.LastSyncedAt,
		"source_snapshot_hash": document.SourceSnapshotHash,
	}
	for key, value := range required {
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("%s is required", key)
		}
	}
	for index, version := range document.ActiveVersions {
		if strings.TrimSpace(version.DisplayLabel) == "" {
			return fmt.Errorf("active_versions[%d].display_label is required", index)
		}
		if strings.TrimSpace(version.VersionKind) == "" {
			return fmt.Errorf("active_versions[%d].version_kind is required", index)
		}
		if strings.TrimSpace(version.LifecycleStatus) == "" {
			return fmt.Errorf("active_versions[%d].lifecycle_status is required", index)
		}
		if version.RoughProgress.Percent < 0 || version.RoughProgress.Percent > 100 {
			return fmt.Errorf("active_versions[%d].rough_progress.percent must be between 0 and 100", index)
		}
		if strings.TrimSpace(version.RoughProgress.Label) == "" {
			return fmt.Errorf("active_versions[%d].rough_progress.label is required", index)
		}
	}
	if strings.TrimSpace(document.Compatibility.ShimLifecycleState) == "" {
		return fmt.Errorf("compatibility.shim_lifecycle_state is required")
	}
	if strings.TrimSpace(document.Compatibility.OfflineSource) == "" {
		return fmt.Errorf("compatibility.offline_source is required")
	}
	if len(document.Compatibility.BlockedCommands) == 0 {
		return fmt.Errorf("compatibility.blocked_commands is required")
	}
	for index, command := range document.Compatibility.BlockedCommands {
		if strings.TrimSpace(command) == "" {
			return fmt.Errorf("compatibility.blocked_commands[%d] is required", index)
		}
	}
	return nil
}

func validateStableExportJSON(content []byte) error {
	var document map[string]any
	if err := json.Unmarshal(content, &document); err != nil {
		return fmt.Errorf("stable projection JSON must parse: %w", err)
	}
	if err := validateExactKeys("status projection", document, []string{
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
	versions, ok := document["active_versions"].([]any)
	if !ok {
		return fmt.Errorf("active_versions must be an array")
	}
	for index, rawVersion := range versions {
		version, ok := rawVersion.(map[string]any)
		if !ok {
			return fmt.Errorf("active_versions[%d] must be an object", index)
		}
		if err := validateExactKeys(fmt.Sprintf("active_versions[%d]", index), version, []string{
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
		if err := validateExactKeys(fmt.Sprintf("active_versions[%d].rough_progress", index), progress, []string{
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
	return validateExactKeys("compatibility", compatibility, []string{
		"shim_lifecycle_state",
		"offline_source",
		"blocked_commands",
	})
}

func validateExactKeys(scope string, document map[string]any, allowed []string) error {
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

func statusTargetPath(record project.Record, relPath string) (string, string, error) {
	if strings.TrimSpace(record.RootPath) == "" {
		return "", "", fmt.Errorf("project %s has no root path", record.Key)
	}
	cleanRel := filepath.Clean(filepath.FromSlash(strings.TrimSpace(relPath)))
	if cleanRel == "." || cleanRel == "" {
		return "", "", fmt.Errorf("status export target path is required")
	}
	if filepath.IsAbs(cleanRel) {
		return "", "", fmt.Errorf("status export target must be relative to the project root")
	}
	root, err := filepath.Abs(record.RootPath)
	if err != nil {
		return "", "", fmt.Errorf("resolve project root: %w", err)
	}
	target, err := filepath.Abs(filepath.Join(root, cleanRel))
	if err != nil {
		return "", "", fmt.Errorf("resolve status export target: %w", err)
	}
	rel, err := filepath.Rel(root, target)
	if err != nil {
		return "", "", fmt.Errorf("compare status export target with project root: %w", err)
	}
	if rel == "." || rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
		return "", "", fmt.Errorf("status export target escapes the project root")
	}
	return target, root, nil
}

func ensureStatusTargetParentContained(root string, target string) error {
	resolvedRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		return fmt.Errorf("resolve project root symlinks: %w", err)
	}
	resolvedParent, err := filepath.EvalSymlinks(filepath.Dir(target))
	if err != nil {
		return fmt.Errorf("resolve status export parent symlinks: %w", err)
	}
	rel, err := filepath.Rel(resolvedRoot, resolvedParent)
	if err != nil {
		return fmt.Errorf("compare status export parent with project root: %w", err)
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
		return fmt.Errorf("status export parent escapes the project root")
	}
	return nil
}

func writeFileAtomically(target string, content []byte, mode os.FileMode) error {
	dir := filepath.Dir(target)
	temp, err := os.CreateTemp(dir, "."+filepath.Base(target)+".tmp-*")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
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
		return fmt.Errorf("write temp file: %w", err)
	}
	if err := temp.Chmod(mode); err != nil {
		_ = temp.Close()
		return fmt.Errorf("chmod temp file: %w", err)
	}
	if err := temp.Sync(); err != nil {
		_ = temp.Close()
		return fmt.Errorf("sync temp file: %w", err)
	}
	if err := temp.Close(); err != nil {
		return fmt.Errorf("close temp file: %w", err)
	}
	if err := os.Rename(tempName, target); err != nil {
		return fmt.Errorf("replace status file: %w", err)
	}
	keepTemp = false
	syncDirBestEffort(dir)
	return nil
}

func syncDirBestEffort(dir string) {
	handle, err := os.Open(dir)
	if err != nil {
		return
	}
	defer handle.Close()
	_ = handle.Sync()
}

func sameDocumentExceptGenerated(a ExportDocument, b ExportDocument) bool {
	a.LastSyncedAt = ""
	b.LastSyncedAt = ""
	return reflect.DeepEqual(a, b)
}

func hashContent(content []byte) string {
	return fmt.Sprintf("%x", sha256.Sum256(content))
}

func projectName(record project.Record) string {
	if record.Name != "" {
		return record.Name
	}
	return record.Key
}

func areaFlowURL(record project.Record, summary map[string]any) string {
	if value := stringValue(summary, "area_flow_url"); value != "" {
		return value
	}
	return fmt.Sprintf("http://127.0.0.1:3847/projects/%s", record.Key)
}

func cutoverPhase(summary map[string]any) string {
	for _, key := range []string{"cutover_phase", "shim_lifecycle_state", "migration_phase"} {
		if value := stringValue(summary, key); value != "" {
			return value
		}
	}
	return "import_mirror"
}

func shimLifecycleState(phase string) string {
	switch phase {
	case "read_only_shim", "execution_forwarding", "retired_thin_entry":
		return phase
	default:
		return "not_installed"
	}
}

func activeVersions(summary map[string]any) []ActiveVersion {
	versions := []ActiveVersion{}
	for _, item := range sliceValue(summary["versions"]) {
		version, ok := item.(map[string]any)
		if !ok {
			continue
		}
		label := stringValue(version, "display_label")
		if label == "" {
			continue
		}
		lifecycle := stringValue(version, "lifecycle_status")
		if lifecycle == "" {
			lifecycle = "unknown"
		}
		kind := stringValue(version, "version_kind")
		if kind == "" {
			kind = "workflow_version"
		}
		versions = append(versions, ActiveVersion{
			DisplayLabel:    label,
			VersionKind:     kind,
			LifecycleStatus: lifecycle,
			RoughProgress:   roughProgress(label, lifecycle, summary),
		})
	}
	if len(versions) == 0 && intValue(nestedValue(summary, "v1_execution", "total")) > 0 {
		lifecycle := "imported"
		versions = append(versions, ActiveVersion{
			DisplayLabel:    "v1-mvp",
			VersionKind:     "workflow_version",
			LifecycleStatus: lifecycle,
			RoughProgress:   roughProgress("v1-mvp", lifecycle, summary),
		})
	}
	return versions
}

func roughProgress(label string, lifecycle string, summary map[string]any) RoughProgress {
	total := intValue(nestedValue(summary, "v1_execution", "total"))
	done := intValue(nestedValue(summary, "v1_execution", "done"))
	if label == "v1-mvp" && total > 0 {
		return RoughProgress{
			Percent: clampPercent(done * 100 / total),
			Label:   fmt.Sprintf("%d/%d v1 execution tasks completed", done, total),
			Blocked: lifecycleBlocked(lifecycle),
		}
	}
	percent := 0
	if lifecycle == "archived" {
		percent = 100
	}
	return RoughProgress{
		Percent: percent,
		Label:   fmt.Sprintf("workflow version %s", lifecycle),
		Blocked: lifecycleBlocked(lifecycle),
	}
}

func lifecycleBlocked(value string) bool {
	switch value {
	case "blocked", "blocked-external", "blocked-decision", "mixed-blocked":
		return true
	default:
		return false
	}
}

func clampPercent(value int) int {
	if value < 0 {
		return 0
	}
	if value > 100 {
		return 100
	}
	return value
}

func stringValue(data map[string]any, key string) string {
	value, ok := data[key]
	if !ok {
		return ""
	}
	text, ok := value.(string)
	if !ok {
		return ""
	}
	return text
}

func sliceValue(value any) []any {
	if items, ok := value.([]any); ok {
		return items
	}
	if maps, ok := value.([]map[string]any); ok {
		items := make([]any, 0, len(maps))
		for _, item := range maps {
			items = append(items, item)
		}
		return items
	}
	return nil
}

func nestedValue(data map[string]any, outer string, inner string) any {
	nested, ok := data[outer].(map[string]any)
	if !ok {
		return nil
	}
	return nested[inner]
}

func intValue(value any) int {
	switch typed := value.(type) {
	case int:
		return typed
	case int64:
		return int(typed)
	case float64:
		return int(typed)
	case json.Number:
		result, _ := typed.Int64()
		return int(result)
	default:
		return 0
	}
}
