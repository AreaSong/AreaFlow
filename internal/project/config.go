package project

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Version       int           `yaml:"version"`
	Project       ProjectConfig `yaml:"project"`
	Ownership     Ownership     `yaml:"ownership"`
	ArtifactStore ArtifactStore `yaml:"artifact_store"`
	Permissions   Permissions   `yaml:"permissions"`
	Commands      Commands      `yaml:"commands"`
	Scheduling    Scheduling    `yaml:"scheduling"`
	Engines       Engines       `yaml:"engines"`
	StatusExport  StatusExport  `yaml:"status_export"`
	Migration     Migration     `yaml:"migration"`

	SourcePath string `yaml:"-"`
	SourceHash string `yaml:"-"`
}

type ProjectConfig struct {
	ID              string `yaml:"id"`
	Name            string `yaml:"name"`
	Root            string `yaml:"root"`
	Kind            string `yaml:"kind"`
	Adapter         string `yaml:"adapter"`
	WorkflowProfile string `yaml:"workflow_profile"`
	DefaultBranch   string `yaml:"default_branch"`
}

type Ownership struct {
	Mode          string        `yaml:"mode"`
	SourceOfTruth SourceOfTruth `yaml:"source_of_truth"`
	Cutover       Cutover       `yaml:"cutover"`
}

type SourceOfTruth struct {
	ProductDocs   string `yaml:"product_docs"`
	SourceCode    string `yaml:"source_code"`
	Workflow      string `yaml:"workflow"`
	Execution     string `yaml:"execution"`
	StatusSummary string `yaml:"status_summary"`
}

type Cutover struct {
	Enabled            bool   `yaml:"enabled"`
	NewVersionsOwnedBy string `yaml:"new_versions_owned_by"`
	LegacyVersionsMode string `yaml:"legacy_versions_mode"`
	ExecutionOwnedBy   string `yaml:"execution_owned_by"`
}

type ArtifactStore struct {
	Backend string `yaml:"backend"`
	Root    string `yaml:"root"`
}

type Permissions struct {
	Capabilities  map[string]bool `yaml:"capabilities"`
	ReadPaths     []string        `yaml:"read_paths"`
	WritePaths    []string        `yaml:"write_paths"`
	ForbiddenPath []string        `yaml:"forbidden_paths"`
}

type Commands struct {
	Allowed   []string `yaml:"allowed"`
	Forbidden []string `yaml:"forbidden"`
}

type Scheduling struct {
	Priority             int      `yaml:"priority"`
	MaxParallelTasks     int      `yaml:"max_parallel_tasks"`
	AgentRole            string   `yaml:"agent_role"`
	RequiredCapabilities []string `yaml:"required_capabilities"`
	EngineProfile        string   `yaml:"engine_profile"`
}

type Engines struct {
	Default  string                `yaml:"default"`
	Profiles []EngineProfileConfig `yaml:"profiles"`
}

type EngineProfileConfig struct {
	ID             string         `yaml:"id"`
	Provider       string         `yaml:"provider"`
	SecretRef      string         `yaml:"secret_ref"`
	Enabled        bool           `yaml:"enabled"`
	ResourceLimits map[string]any `yaml:"resource_limits"`
}

type StatusExport struct {
	Enabled      bool         `yaml:"enabled"`
	Path         string       `yaml:"path"`
	HumanSummary HumanSummary `yaml:"human_summary"`
}

type HumanSummary struct {
	Enabled     bool   `yaml:"enabled"`
	Path        string `yaml:"path"`
	BlockMarker string `yaml:"block_marker"`
}

type Migration struct {
	Strategy         string   `yaml:"strategy"`
	Phase            string   `yaml:"phase"`
	ImportedVersions []string `yaml:"imported_versions"`
	ImmutableImports []string `yaml:"immutable_imports"`
}

func LoadConfig(path string) (Config, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("read project config %s: %w", path, err)
	}

	var cfg Config
	if err := yaml.Unmarshal(content, &cfg); err != nil {
		return Config{}, fmt.Errorf("parse project config %s: %w", path, err)
	}
	if err := cfg.Validate(); err != nil {
		return Config{}, fmt.Errorf("invalid project config %s: %w", path, err)
	}
	cfg.Ownership = NormalizeOwnership(cfg.Ownership)
	cfg.Scheduling = NormalizeScheduling(cfg.Scheduling)
	cfg.Engines = NormalizeEngines(cfg.Engines)
	cfg.StatusExport = NormalizeStatusExport(cfg.StatusExport)
	cfg.Migration = NormalizeMigration(cfg.Migration)
	cfg.SourcePath = absoluteConfigPath(path)
	cfg.SourceHash = sha256Hex(content)
	if err := cfg.ValidateNormalized(); err != nil {
		return Config{}, fmt.Errorf("invalid project config %s: %w", path, err)
	}
	return cfg, nil
}

func sha256Hex(content []byte) string {
	sum := sha256.Sum256(content)
	return hex.EncodeToString(sum[:])
}

func absoluteConfigPath(path string) string {
	absolutePath, err := filepath.Abs(path)
	if err != nil {
		return path
	}
	return absolutePath
}

func (cfg Config) Validate() error {
	if cfg.Version != 1 {
		return fmt.Errorf("unsupported config version %d", cfg.Version)
	}
	if cfg.Project.ID == "" {
		return fmt.Errorf("project.id is required")
	}
	if cfg.Project.Name == "" {
		return fmt.Errorf("project.name is required")
	}
	if cfg.Project.Root == "" {
		return fmt.Errorf("project.root is required")
	}
	if cfg.Project.Adapter == "" {
		return fmt.Errorf("project.adapter is required")
	}
	if cfg.Project.WorkflowProfile == "" {
		return fmt.Errorf("project.workflow_profile is required")
	}
	return nil
}

func (cfg Config) ValidateNormalized() error {
	if !knownOwnershipMode(cfg.Ownership.Mode) {
		return fmt.Errorf("ownership.mode %q is not supported", cfg.Ownership.Mode)
	}
	if err := validateOwner("ownership.source_of_truth.product_docs", cfg.Ownership.SourceOfTruth.ProductDocs); err != nil {
		return err
	}
	if err := validateOwner("ownership.source_of_truth.source_code", cfg.Ownership.SourceOfTruth.SourceCode); err != nil {
		return err
	}
	if err := validateOwner("ownership.source_of_truth.workflow", cfg.Ownership.SourceOfTruth.Workflow); err != nil {
		return err
	}
	if err := validateOwner("ownership.source_of_truth.execution", cfg.Ownership.SourceOfTruth.Execution); err != nil {
		return err
	}
	if err := validateOwner("ownership.source_of_truth.status_summary", cfg.Ownership.SourceOfTruth.StatusSummary); err != nil {
		return err
	}
	if err := validateOwner("ownership.cutover.new_versions_owned_by", cfg.Ownership.Cutover.NewVersionsOwnedBy); err != nil {
		return err
	}
	if err := validateOwner("ownership.cutover.execution_owned_by", cfg.Ownership.Cutover.ExecutionOwnedBy); err != nil {
		return err
	}
	if !knownMigrationPhase(cfg.Migration.Phase) {
		return fmt.Errorf("migration.phase %q is not supported", cfg.Migration.Phase)
	}
	if cfg.Migration.Strategy != "import_mirror_shadow_cutover_archive" {
		return fmt.Errorf("migration.strategy %q is not supported", cfg.Migration.Strategy)
	}
	if cfg.StatusExport.Enabled && strings.TrimSpace(cfg.StatusExport.Path) == "" {
		return fmt.Errorf("status_export.path is required when status export is enabled")
	}
	if cfg.StatusExport.HumanSummary.Enabled {
		if strings.TrimSpace(cfg.StatusExport.HumanSummary.Path) == "" {
			return fmt.Errorf("status_export.human_summary.path is required when human summary is enabled")
		}
		if strings.TrimSpace(cfg.StatusExport.HumanSummary.BlockMarker) == "" {
			return fmt.Errorf("status_export.human_summary.block_marker is required when human summary is enabled")
		}
	}
	return nil
}

func NormalizeOwnership(ownership Ownership) Ownership {
	ownership.Mode = strings.TrimSpace(ownership.Mode)
	if ownership.Mode == "" {
		ownership.Mode = "import"
	}
	ownership.SourceOfTruth.ProductDocs = defaultOwner(ownership.SourceOfTruth.ProductDocs, "project")
	ownership.SourceOfTruth.SourceCode = defaultOwner(ownership.SourceOfTruth.SourceCode, "project")
	ownership.SourceOfTruth.Workflow = defaultOwner(ownership.SourceOfTruth.Workflow, "project")
	ownership.SourceOfTruth.Execution = defaultOwner(ownership.SourceOfTruth.Execution, "project")
	ownership.SourceOfTruth.StatusSummary = defaultOwner(ownership.SourceOfTruth.StatusSummary, "areaflow")
	ownership.Cutover.NewVersionsOwnedBy = defaultOwner(ownership.Cutover.NewVersionsOwnedBy, "project")
	ownership.Cutover.ExecutionOwnedBy = defaultOwner(ownership.Cutover.ExecutionOwnedBy, "project")
	ownership.Cutover.LegacyVersionsMode = strings.TrimSpace(ownership.Cutover.LegacyVersionsMode)
	if ownership.Cutover.LegacyVersionsMode == "" {
		ownership.Cutover.LegacyVersionsMode = "project_owned"
	}
	return ownership
}

func NormalizeScheduling(scheduling Scheduling) Scheduling {
	if scheduling.Priority <= 0 {
		scheduling.Priority = 100
	}
	if scheduling.MaxParallelTasks <= 0 {
		scheduling.MaxParallelTasks = 1
	}
	if scheduling.AgentRole == "" {
		scheduling.AgentRole = "local_worker"
	}
	if len(scheduling.RequiredCapabilities) == 0 {
		scheduling.RequiredCapabilities = []string{"read_project"}
	}
	scheduling.RequiredCapabilities = normalizeCapabilityList(scheduling.RequiredCapabilities)
	return scheduling
}

func NormalizeEngines(engines Engines) Engines {
	engines.Default = strings.TrimSpace(engines.Default)
	normalizedProfiles := make([]EngineProfileConfig, 0, len(engines.Profiles))
	for _, profile := range engines.Profiles {
		profile.ID = strings.TrimSpace(profile.ID)
		profile.Provider = strings.TrimSpace(profile.Provider)
		profile.SecretRef = strings.TrimSpace(profile.SecretRef)
		if profile.ID == "" {
			continue
		}
		if profile.Provider == "" {
			profile.Provider = profile.ID
		}
		if profile.SecretRef == "" {
			profile.SecretRef = "none"
		}
		if profile.ResourceLimits == nil {
			profile.ResourceLimits = map[string]any{}
		}
		normalizedProfiles = append(normalizedProfiles, profile)
	}
	if engines.Default == "" && len(normalizedProfiles) > 0 {
		engines.Default = normalizedProfiles[0].ID
	}
	engines.Profiles = normalizedProfiles
	return engines
}

func NormalizeStatusExport(status StatusExport) StatusExport {
	status.Path = strings.TrimSpace(status.Path)
	if status.Enabled && status.Path == "" {
		status.Path = ".areaflow/status.json"
	}
	status.HumanSummary.Path = strings.TrimSpace(status.HumanSummary.Path)
	if status.HumanSummary.Enabled && status.HumanSummary.Path == "" {
		status.HumanSummary.Path = "workflow/README.md"
	}
	status.HumanSummary.BlockMarker = strings.TrimSpace(status.HumanSummary.BlockMarker)
	if status.HumanSummary.Enabled && status.HumanSummary.BlockMarker == "" {
		status.HumanSummary.BlockMarker = "AREAFLOW_STATUS"
	}
	return status
}

func NormalizeMigration(migration Migration) Migration {
	migration.Strategy = strings.TrimSpace(migration.Strategy)
	if migration.Strategy == "" {
		migration.Strategy = "import_mirror_shadow_cutover_archive"
	}
	migration.Phase = strings.TrimSpace(migration.Phase)
	if migration.Phase == "" {
		migration.Phase = "import"
	}
	migration.ImportedVersions = normalizeConfigStringList(migration.ImportedVersions)
	migration.ImmutableImports = normalizeConfigStringList(migration.ImmutableImports)
	return migration
}

func engineProfileByID(engines Engines, id string) (EngineProfileConfig, bool) {
	id = strings.TrimSpace(id)
	if id == "" {
		id = engines.Default
	}
	for _, profile := range engines.Profiles {
		if profile.ID == id {
			return profile, true
		}
	}
	return EngineProfileConfig{}, false
}

func defaultOwner(value string, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	return value
}

func validateOwner(name string, value string) error {
	if value != "project" && value != "areaflow" && value != "external" {
		return fmt.Errorf("%s %q is not supported", name, value)
	}
	return nil
}

func knownOwnershipMode(mode string) bool {
	switch mode {
	case "import", "mirror", "shadow", "cutover", "archived":
		return true
	default:
		return false
	}
}

func knownMigrationPhase(phase string) bool {
	switch phase {
	case "import", "mirror", "shadow", "cutover", "authoring_cutover", "execution_beta", "execution_cutover", "archive", "shim_retirement":
		return true
	default:
		return false
	}
}

func normalizeConfigStringList(values []string) []string {
	normalized := make([]string, 0, len(values))
	seen := map[string]bool{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		normalized = append(normalized, value)
	}
	return normalized
}
