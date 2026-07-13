package areamatrix

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

type Snapshot struct {
	Root             string
	Versions         []Version
	Residuals        []Residual
	Artifacts        []Artifact
	TaskSummary      TaskSummary
	StatusSummary    map[string]any
	StatusSourceHash string
}

type Version struct {
	Label          string
	Lifecycle      string
	SourcePath     string
	SourceHash     string
	Immutable      bool
	StatusSummary  map[string]any
	ArtifactCounts map[string]int
}

type Residual struct {
	Key               string
	VersionLabel      string
	Status            string
	Type              string
	Title             string
	SourcePath        string
	CurrentImpact     string
	ExecutableTask    bool
	PromotionRequired bool
	CloseCondition    string
	Immutable         bool
	Metadata          map[string]any
}

type Artifact struct {
	Type         string
	VersionLabel string
	SourcePath   string
	SHA256       string
	SizeBytes    int64
	ContentType  string
	Metadata     map[string]any
}

type TaskSummary struct {
	ActiveCount       int
	DoneCount         int
	BacklogPackages   int
	BacklogOpen       int
	BacklogClosed     int
	V1ExecutionTotal  int
	V1ExecutionDone   int
	V1ExecutionStatus map[string]int
}

type residualLedger struct {
	Items            []residualItem        `yaml:"items"`
	VersionResiduals []versionResidualLink `yaml:"version_residuals"`
}

type versionResidualLink struct {
	Version string `yaml:"version"`
	Source  string `yaml:"source"`
	Status  string `yaml:"status"`
	Summary string `yaml:"summary"`
}

type versionResidualLedger struct {
	VersionStatus map[string]any `yaml:"version_status"`
	Items         []residualItem `yaml:"items"`
}

type residualItem struct {
	ID                string         `yaml:"id"`
	Status            string         `yaml:"status"`
	Type              string         `yaml:"type"`
	Title             string         `yaml:"title"`
	Source            string         `yaml:"source"`
	CurrentImpact     string         `yaml:"current_impact"`
	ExecutableTask    bool           `yaml:"executable_task"`
	PromotionRequired bool           `yaml:"promotion_required"`
	CloseCondition    string         `yaml:"close_condition"`
	Owner             string         `yaml:"owner"`
	Blocker           string         `yaml:"blocker"`
	SupportingSources []string       `yaml:"supporting_sources"`
	Validation        []string       `yaml:"validation"`
	Notes             string         `yaml:"notes"`
	Metadata          map[string]any `yaml:",inline"`
}

type progressFile struct {
	Tasks map[string]progressTask `json:"tasks"`
}

type progressTask struct {
	Status string `json:"status"`
}

func Load(root string) (Snapshot, error) {
	snapshot := Snapshot{
		Root:          root,
		StatusSummary: map[string]any{},
	}

	globalLedger, err := readYAML[residualLedger](root, "workflow/residuals/residuals.yaml")
	if err != nil {
		return Snapshot{}, err
	}

	for _, item := range globalLedger.Items {
		snapshot.Residuals = append(snapshot.Residuals, convertResidual(item, ""))
	}

	for _, linked := range globalLedger.VersionResiduals {
		versionLedger, err := readYAML[versionResidualLedger](root, linked.Source)
		if err != nil {
			return Snapshot{}, err
		}
		sourceHash, _ := fileHash(root, linked.Source)
		version := Version{
			Label:         linked.Version,
			Lifecycle:     linked.Status,
			SourcePath:    linked.Source,
			SourceHash:    sourceHash,
			Immutable:     linked.Version == "v1-mvp",
			StatusSummary: versionLedger.VersionStatus,
		}
		version.ArtifactCounts = countVersionArtifacts(root, linked.Version)
		snapshot.Versions = append(snapshot.Versions, version)
		for _, item := range versionLedger.Items {
			residual := convertResidual(item, linked.Version)
			residual.Immutable = linked.Version == "v1-mvp"
			snapshot.Residuals = append(snapshot.Residuals, residual)
		}
	}

	if err := addTemplateVersion(root, &snapshot); err != nil {
		return Snapshot{}, err
	}
	if err := addArtifacts(root, &snapshot); err != nil {
		return Snapshot{}, err
	}

	taskSummary, err := loadTaskSummary(root)
	if err != nil {
		return Snapshot{}, err
	}
	snapshot.TaskSummary = taskSummary
	snapshot.StatusSummary = buildStatusSummary(snapshot)
	snapshot.StatusSourceHash = snapshotHash(snapshot)
	return snapshot, nil
}

func convertResidual(item residualItem, versionLabel string) Residual {
	metadata := map[string]any{}
	if item.Owner != "" {
		metadata["owner"] = item.Owner
	}
	if item.Blocker != "" {
		metadata["blocker"] = item.Blocker
	}
	if len(item.SupportingSources) > 0 {
		metadata["supporting_sources"] = item.SupportingSources
	}
	if len(item.Validation) > 0 {
		metadata["validation"] = item.Validation
	}
	if item.Notes != "" {
		metadata["notes"] = item.Notes
	}

	return Residual{
		Key:               item.ID,
		VersionLabel:      versionLabel,
		Status:            item.Status,
		Type:              item.Type,
		Title:             item.Title,
		SourcePath:        item.Source,
		CurrentImpact:     item.CurrentImpact,
		ExecutableTask:    item.ExecutableTask,
		PromotionRequired: item.PromotionRequired,
		CloseCondition:    item.CloseCondition,
		Metadata:          metadata,
	}
}

func addTemplateVersion(root string, snapshot *Snapshot) error {
	source := "workflow/versions/v-template/README.md"
	hash, err := fileHash(root, source)
	if err != nil {
		return nil
	}
	snapshot.Versions = append(snapshot.Versions, Version{
		Label:          "v-template",
		Lifecycle:      "template-only",
		SourcePath:     source,
		SourceHash:     hash,
		Immutable:      false,
		StatusSummary:  map[string]any{"template_only": true},
		ArtifactCounts: countVersionArtifacts(root, "v-template"),
	})
	return nil
}

func addArtifacts(root string, snapshot *Snapshot) error {
	candidates := []string{
		"workflow/residuals/residuals.yaml",
		"workflow/versions/v1-mvp/residuals/residuals.yaml",
		"workflow/versions/v1-mvp/execution/_shared/progress.json",
		"tasks/indexes/residuals.md",
		"tasks/backlog/README.md",
		"workflow/versions/v-template/README.md",
	}
	for _, path := range candidates {
		artifact, err := artifactFor(root, path)
		if err != nil {
			continue
		}
		artifact.Type = artifactType(path)
		artifact.VersionLabel = versionFromPath(path)
		snapshot.Artifacts = append(snapshot.Artifacts, artifact)
	}
	return nil
}

func loadTaskSummary(root string) (TaskSummary, error) {
	summary := TaskSummary{
		ActiveCount:       countDirs(filepath.Join(root, "tasks/active")),
		DoneCount:         countDoneTasks(filepath.Join(root, "tasks/done")),
		BacklogPackages:   countDirs(filepath.Join(root, "tasks/backlog/prompts")),
		V1ExecutionStatus: map[string]int{},
	}
	summary.BacklogClosed = summary.BacklogPackages

	progress, err := readJSON[progressFile](root, "workflow/versions/v1-mvp/execution/_shared/progress.json")
	if err != nil {
		return summary, err
	}
	summary.V1ExecutionTotal = len(progress.Tasks)
	for _, task := range progress.Tasks {
		summary.V1ExecutionStatus[task.Status]++
		if task.Status == "completed" {
			summary.V1ExecutionDone++
		}
	}
	return summary, nil
}

func buildStatusSummary(snapshot Snapshot) map[string]any {
	versions := make([]map[string]any, 0, len(snapshot.Versions))
	for _, version := range snapshot.Versions {
		if version.Label == "v-template" || version.Lifecycle == "template-only" {
			continue
		}
		versions = append(versions, map[string]any{
			"display_label":     version.Label,
			"version_kind":      "workflow_version",
			"lifecycle_status":  version.Lifecycle,
			"source_path":       version.SourcePath,
			"artifact_counts":   version.ArtifactCounts,
			"immutable_history": version.Immutable,
		})
	}
	status := map[string]any{
		"project":  "areamatrix",
		"versions": versions,
		"tasks": map[string]any{
			"active":           snapshot.TaskSummary.ActiveCount,
			"done":             snapshot.TaskSummary.DoneCount,
			"backlog_packages": snapshot.TaskSummary.BacklogPackages,
			"backlog_open":     snapshot.TaskSummary.BacklogOpen,
			"backlog_closed":   snapshot.TaskSummary.BacklogClosed,
		},
		"v1_execution": map[string]any{
			"total":  snapshot.TaskSummary.V1ExecutionTotal,
			"done":   snapshot.TaskSummary.V1ExecutionDone,
			"status": snapshot.TaskSummary.V1ExecutionStatus,
		},
		"residual_count": len(snapshot.Residuals),
		"version_count":  len(snapshot.Versions),
	}
	return status
}

func snapshotHash(snapshot Snapshot) string {
	content, _ := json.Marshal(snapshot.StatusSummary)
	sum := sha256.Sum256(content)
	return hex.EncodeToString(sum[:])
}

func readYAML[T any](root string, rel string) (T, error) {
	var out T
	content, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
	if err != nil {
		return out, fmt.Errorf("read %s: %w", rel, err)
	}
	if err := yaml.Unmarshal(content, &out); err != nil {
		return out, fmt.Errorf("parse %s: %w", rel, err)
	}
	return out, nil
}

func readJSON[T any](root string, rel string) (T, error) {
	var out T
	content, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
	if err != nil {
		return out, fmt.Errorf("read %s: %w", rel, err)
	}
	if err := json.Unmarshal(content, &out); err != nil {
		return out, fmt.Errorf("parse %s: %w", rel, err)
	}
	return out, nil
}

func countVersionArtifacts(root string, version string) map[string]int {
	counts := map[string]int{}
	base := filepath.Join(root, "workflow", "versions", version)
	entries, err := os.ReadDir(base)
	if err != nil {
		return counts
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		stage := entry.Name()
		counts[stage] = countFiles(filepath.Join(base, stage))
	}
	return counts
}

func countFiles(root string) int {
	count := 0
	_ = filepath.WalkDir(root, func(_ string, entry fs.DirEntry, err error) error {
		if err == nil && !entry.IsDir() {
			count++
		}
		return nil
	})
	return count
}

func countDirs(root string) int {
	entries, err := os.ReadDir(root)
	if err != nil {
		return 0
	}
	count := 0
	for _, entry := range entries {
		if entry.IsDir() {
			count++
		}
	}
	return count
}

func countDoneTasks(root string) int {
	count := 0
	_ = filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil || !entry.IsDir() || path == root {
			return nil
		}
		if strings.Contains(entry.Name(), ".") {
			count++
			return filepath.SkipDir
		}
		return nil
	})
	return count
}

func artifactFor(root string, rel string) (Artifact, error) {
	path := filepath.Join(root, filepath.FromSlash(rel))
	info, err := os.Stat(path)
	if err != nil {
		return Artifact{}, err
	}
	hash, err := fileHash(root, rel)
	if err != nil {
		return Artifact{}, err
	}
	return Artifact{
		SourcePath:  rel,
		SHA256:      hash,
		SizeBytes:   info.Size(),
		ContentType: contentType(rel),
		Metadata:    map[string]any{},
	}, nil
}

func fileHash(root string, rel string) (string, error) {
	content, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(content)
	return hex.EncodeToString(sum[:]), nil
}

func artifactType(path string) string {
	switch {
	case strings.HasSuffix(path, "progress.json"):
		return "progress"
	case strings.Contains(path, "residual"):
		return "residual_index"
	case strings.Contains(path, "backlog"):
		return "task_index"
	default:
		return "source_file"
	}
}

func versionFromPath(path string) string {
	parts := strings.Split(path, "/")
	for i, part := range parts {
		if part == "versions" && i+1 < len(parts) {
			return parts[i+1]
		}
	}
	return ""
}

func contentType(path string) string {
	switch {
	case strings.HasSuffix(path, ".yaml"), strings.HasSuffix(path, ".yml"):
		return "application/yaml"
	case strings.HasSuffix(path, ".json"):
		return "application/json"
	case strings.HasSuffix(path, ".md"):
		return "text/markdown"
	default:
		return "application/octet-stream"
	}
}
