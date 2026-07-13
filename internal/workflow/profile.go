package workflow

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

var (
	ErrProfileNotFound = errors.New("workflow profile not found")
	ErrInvalidProfile  = errors.New("invalid workflow profile")
)

var requiredWriteGuards = []string{
	"capability",
	"path_allowlist",
	"gate_result",
	"approval_record",
	"audit_event",
}

type Profile struct {
	ProfileID       string           `yaml:"profile_id" json:"profile_id"`
	ProfileVersion  int              `yaml:"profile_version" json:"profile_version"`
	DisplayName     string           `yaml:"display_name" json:"display_name"`
	Description     string           `yaml:"description" json:"description"`
	AdapterDefaults AdapterDefaults  `yaml:"adapter_defaults" json:"adapter_defaults"`
	VersionBinding  VersionBinding   `yaml:"version_binding" json:"version_binding"`
	ItemStates      []string         `yaml:"item_states" json:"item_states"`
	Stages          []Stage          `yaml:"stages" json:"stages"`
	Transitions     []Transition     `yaml:"transitions" json:"transitions"`
	Gates           []Gate           `yaml:"gates" json:"gates"`
	HardRules       []string         `yaml:"hard_rules" json:"hard_rules"`
	ArtifactPolicy  ArtifactPolicy   `yaml:"artifact_policy" json:"artifact_policy"`
	Permissions     PermissionPolicy `yaml:"permissions" json:"permissions"`
	Cutover         CutoverPolicy    `yaml:"cutover" json:"cutover"`
}

type AdapterDefaults struct {
	Adapter         string `yaml:"adapter" json:"adapter"`
	WorkflowProfile string `yaml:"workflow_profile" json:"workflow_profile"`
}

type VersionBinding struct {
	FreezeProfileHash bool   `yaml:"freeze_profile_hash" json:"freeze_profile_hash"`
	UpgradePolicy     string `yaml:"upgrade_policy" json:"upgrade_policy"`
}

type Stage struct {
	Name              string   `yaml:"name" json:"name"`
	Purpose           string   `yaml:"purpose" json:"purpose"`
	RequiredArtifacts []string `yaml:"required_artifacts" json:"required_artifacts"`
	AllowedOutputs    []string `yaml:"allowed_outputs" json:"allowed_outputs"`
	GateChecks        []string `yaml:"gate_checks" json:"gate_checks"`
	FailureRoutes     []string `yaml:"failure_routes" json:"failure_routes"`
}

type Transition struct {
	From         string `yaml:"from" json:"from"`
	To           string `yaml:"to" json:"to"`
	RequiredGate string `yaml:"required_gate" json:"required_gate,omitempty"`
}

type Gate struct {
	Name          string `yaml:"name" json:"name"`
	EarliestPhase string `yaml:"earliest_phase" json:"earliest_phase"`
	StatusSource  string `yaml:"status_source" json:"status_source"`
}

type ArtifactPolicy struct {
	MetadataSource        string `yaml:"metadata_source" json:"metadata_source"`
	ContentSource         string `yaml:"content_source" json:"content_source"`
	SourceDocsOwner       string `yaml:"source_docs_owner" json:"source_docs_owner"`
	GeneratedOutputOwner  string `yaml:"generated_output_owner" json:"generated_output_owner"`
	DefaultContentBackend string `yaml:"default_content_backend" json:"default_content_backend"`
}

type PermissionPolicy struct {
	DefaultMode   string   `yaml:"default_mode" json:"default_mode"`
	WriteRequires []string `yaml:"write_requires" json:"write_requires"`
}

type CutoverPolicy struct {
	Strategy              string `yaml:"strategy" json:"strategy"`
	V04Scope              string `yaml:"v0_4_scope" json:"v0_4_scope"`
	ExecutionCutoverPhase string `yaml:"execution_cutover_phase" json:"execution_cutover_phase"`
}

type LoadedProfile struct {
	Path     string
	SHA256   string
	Profile  Profile
	Warnings []string
}

func LoadProfile(path string) (LoadedProfile, error) {
	cleanPath := strings.TrimSpace(path)
	if cleanPath == "" {
		return LoadedProfile{}, fmt.Errorf("%w: path is required", ErrProfileNotFound)
	}
	content, err := os.ReadFile(cleanPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return LoadedProfile{}, fmt.Errorf("%w: %s", ErrProfileNotFound, cleanPath)
		}
		return LoadedProfile{}, fmt.Errorf("read workflow profile: %w", err)
	}
	var profile Profile
	decoder := yaml.NewDecoder(strings.NewReader(string(content)))
	decoder.KnownFields(true)
	if err := decoder.Decode(&profile); err != nil {
		return LoadedProfile{}, fmt.Errorf("%w: %w", ErrInvalidProfile, err)
	}
	warnings, err := ValidateProfile(profile)
	if err != nil {
		return LoadedProfile{}, err
	}
	sum := sha256.Sum256(content)
	return LoadedProfile{
		Path:     cleanPath,
		SHA256:   hex.EncodeToString(sum[:]),
		Profile:  profile,
		Warnings: warnings,
	}, nil
}

func LoadBuiltInProfile(root string, profileID string) (LoadedProfile, error) {
	id := strings.TrimSpace(profileID)
	if id == "" {
		return LoadedProfile{}, fmt.Errorf("%w: profile id is required", ErrProfileNotFound)
	}
	return LoadProfile(filepath.Join(root, "workflow", "profiles", id, "profile.yaml"))
}

func ListBuiltInProfiles(root string) ([]LoadedProfile, error) {
	profilesRoot := filepath.Join(root, "workflow", "profiles")
	entries, err := os.ReadDir(profilesRoot)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("%w: %s", ErrProfileNotFound, profilesRoot)
		}
		return nil, fmt.Errorf("read workflow profiles: %w", err)
	}
	profiles := []LoadedProfile{}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		loaded, err := LoadBuiltInProfile(root, entry.Name())
		if err != nil {
			return nil, err
		}
		profiles = append(profiles, loaded)
	}
	sort.Slice(profiles, func(i int, j int) bool {
		left := profiles[i].Profile.ProfileID
		right := profiles[j].Profile.ProfileID
		if left == right {
			return profiles[i].Profile.ProfileVersion < profiles[j].Profile.ProfileVersion
		}
		return left < right
	})
	return profiles, nil
}

func ValidateProfile(profile Profile) ([]string, error) {
	var failures []string
	var warnings []string
	if strings.TrimSpace(profile.ProfileID) == "" {
		failures = append(failures, "profile_id is required")
	}
	if profile.ProfileVersion < 0 {
		failures = append(failures, "profile_version must be non-negative")
	}
	if strings.TrimSpace(profile.AdapterDefaults.WorkflowProfile) != "" && profile.AdapterDefaults.WorkflowProfile != profile.ProfileID {
		failures = append(failures, "adapter_defaults.workflow_profile must match profile_id")
	}
	if len(profile.ItemStates) == 0 {
		failures = append(failures, "item_states must not be empty")
	}
	if len(profile.Stages) == 0 {
		failures = append(failures, "stages must not be empty")
	}
	if len(profile.Gates) == 0 {
		failures = append(failures, "gates must not be empty")
	}
	stageNames := map[string]bool{}
	for index, stage := range profile.Stages {
		name := strings.TrimSpace(stage.Name)
		if name == "" {
			failures = append(failures, fmt.Sprintf("stages[%d].name is required", index))
			continue
		}
		if stageNames[name] {
			failures = append(failures, "duplicate stage: "+name)
		}
		stageNames[name] = true
		if len(stage.RequiredArtifacts) == 0 {
			warnings = append(warnings, "stage has no required_artifacts: "+name)
		}
	}
	gateNames := map[string]bool{}
	for index, gate := range profile.Gates {
		name := strings.TrimSpace(gate.Name)
		if name == "" {
			failures = append(failures, fmt.Sprintf("gates[%d].name is required", index))
			continue
		}
		if gateNames[name] {
			failures = append(failures, "duplicate gate: "+name)
		}
		gateNames[name] = true
		if strings.TrimSpace(gate.StatusSource) == "" {
			warnings = append(warnings, "gate has no status_source: "+name)
		}
	}
	for index, transition := range profile.Transitions {
		if !stageNames[transition.From] {
			failures = append(failures, fmt.Sprintf("transitions[%d].from references unknown stage: %s", index, transition.From))
		}
		if !stageNames[transition.To] {
			failures = append(failures, fmt.Sprintf("transitions[%d].to references unknown stage: %s", index, transition.To))
		}
		if transition.RequiredGate != "" && !gateNames[transition.RequiredGate] {
			failures = append(failures, fmt.Sprintf("transitions[%d].required_gate references unknown gate: %s", index, transition.RequiredGate))
		}
	}
	for _, stage := range profile.Stages {
		for _, gateName := range stage.GateChecks {
			if !gateNames[gateName] {
				failures = append(failures, fmt.Sprintf("stage %s references unknown gate: %s", stage.Name, gateName))
			}
		}
		for _, route := range stage.FailureRoutes {
			if !stageNames[route] {
				failures = append(failures, fmt.Sprintf("stage %s references unknown failure route: %s", stage.Name, route))
			}
		}
	}
	if profile.VersionBinding.FreezeProfileHash && strings.TrimSpace(profile.VersionBinding.UpgradePolicy) == "" {
		failures = append(failures, "version_binding.upgrade_policy is required when freeze_profile_hash is true")
	}
	if strings.TrimSpace(profile.Permissions.DefaultMode) == "" {
		failures = append(failures, "permissions.default_mode is required")
	} else if profile.Permissions.DefaultMode != "readonly" {
		failures = append(failures, "permissions.default_mode must be readonly")
	}
	writeGuards := map[string]bool{}
	for _, guard := range profile.Permissions.WriteRequires {
		guard = strings.TrimSpace(guard)
		if guard != "" {
			writeGuards[guard] = true
		}
	}
	for _, required := range requiredWriteGuards {
		if !writeGuards[required] {
			failures = append(failures, "permissions.write_requires missing required guard: "+required)
		}
	}
	if len(failures) > 0 {
		return warnings, fmt.Errorf("%w: %s", ErrInvalidProfile, strings.Join(failures, "; "))
	}
	return warnings, nil
}
